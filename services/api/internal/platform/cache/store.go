package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store wraps *redis.Client with typed JSON get/set/delete helpers and a
// namespace prefix so multiple features can share the same Redis instance
// without key conflicts.
type Store struct {
	client *redis.Client
	ns     string // namespace prefix, e.g. "paca:"
}

// NewStore returns a Store that prepends ns to every Redis key.
func NewStore(client *redis.Client, ns string) *Store {
	return &Store{client: client, ns: ns}
}

// Get retrieves the JSON-encoded value at key and unmarshals it into dest.
// It returns (true, nil) on a cache hit, (false, nil) on a miss, and
// (false, err) on any other error.
func (s *Store) Get(ctx context.Context, key string, dest any) (bool, error) {
	data, err := s.client.Get(ctx, s.ns+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache: get %q: %w", key, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("cache: unmarshal %q: %w", key, err)
	}
	return true, nil
}

// Set JSON-encodes val and stores it at key with the given TTL.
// A zero TTL stores the value without expiry.
func (s *Store) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("cache: marshal %q: %w", key, err)
	}
	if err := s.client.Set(ctx, s.ns+key, data, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set %q: %w", key, err)
	}
	return nil
}

// Delete removes one or more keys from the cache. It is a no-op when keys is
// empty.
func (s *Store) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	full := make([]string, len(keys))
	for i, k := range keys {
		full[i] = s.ns + k
	}
	if err := s.client.Del(ctx, full...).Err(); err != nil {
		return fmt.Errorf("cache: delete: %w", err)
	}
	return nil
}
