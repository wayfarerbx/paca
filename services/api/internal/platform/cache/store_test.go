package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/paca/api/internal/platform/cache"
)

// newTestStore starts a miniredis instance and returns a Store backed by it.
// The server is stopped and the client closed when the test ends.
func newTestStore(t *testing.T, ns string) *cache.Store {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return cache.NewStore(client, ns)
}

func TestStore_GetMiss(t *testing.T) {
	st := newTestStore(t, "t:")
	ctx := context.Background()

	var got string
	hit, err := st.Get(ctx, "missing", &got)
	if err != nil {
		t.Fatalf("Get on missing key: unexpected error: %v", err)
	}
	if hit {
		t.Fatal("Get on missing key: expected cache miss, got hit")
	}
}

func TestStore_SetThenGet(t *testing.T) {
	st := newTestStore(t, "t:")
	ctx := context.Background()

	type payload struct {
		Name string
		Age  int
	}
	want := payload{Name: "alice", Age: 30}

	if err := st.Set(ctx, "user", want, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var got payload
	hit, err := st.Get(ctx, "user", &got)
	if err != nil {
		t.Fatalf("Get after Set: %v", err)
	}
	if !hit {
		t.Fatal("Get after Set: expected cache hit, got miss")
	}
	if got != want {
		t.Fatalf("Get after Set: want %+v, got %+v", want, got)
	}
}

func TestStore_SetWithZeroTTL(t *testing.T) {
	st := newTestStore(t, "t:")
	ctx := context.Background()

	if err := st.Set(ctx, "nettl", "value", 0); err != nil {
		t.Fatalf("Set with zero TTL: %v", err)
	}

	var got string
	hit, err := st.Get(ctx, "nettl", &got)
	if err != nil {
		t.Fatalf("Get after zero-TTL Set: %v", err)
	}
	if !hit {
		t.Fatal("Get after zero-TTL Set: expected hit (no expiry)")
	}
}

func TestStore_Delete(t *testing.T) {
	st := newTestStore(t, "t:")
	ctx := context.Background()

	if err := st.Set(ctx, "k1", "v1", time.Minute); err != nil {
		t.Fatalf("Set k1: %v", err)
	}
	if err := st.Set(ctx, "k2", "v2", time.Minute); err != nil {
		t.Fatalf("Set k2: %v", err)
	}

	if err := st.Delete(ctx, "k1", "k2"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	for _, key := range []string{"k1", "k2"} {
		var s string
		hit, err := st.Get(ctx, key, &s)
		if err != nil {
			t.Fatalf("Get %q after Delete: %v", key, err)
		}
		if hit {
			t.Fatalf("Get %q after Delete: expected miss, got hit", key)
		}
	}
}

func TestStore_DeleteNoKeys(t *testing.T) {
	st := newTestStore(t, "t:")
	ctx := context.Background()

	// Calling Delete with no keys should be a no-op.
	if err := st.Delete(ctx); err != nil {
		t.Fatalf("Delete with no keys: %v", err)
	}
}

func TestStore_NamespaceIsolation(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	storeA := cache.NewStore(client, "a:")
	storeB := cache.NewStore(client, "b:")
	ctx := context.Background()

	if err := storeA.Set(ctx, "key", "from-a", time.Minute); err != nil {
		t.Fatalf("storeA.Set: %v", err)
	}

	// storeB uses a different namespace, so the key should be missing.
	var got string
	hit, err := storeB.Get(ctx, "key", &got)
	if err != nil {
		t.Fatalf("storeB.Get: %v", err)
	}
	if hit {
		t.Fatal("namespace isolation broken: storeB hit key written by storeA")
	}

	// storeA should still see it.
	hit, err = storeA.Get(ctx, "key", &got)
	if err != nil {
		t.Fatalf("storeA.Get: %v", err)
	}
	if !hit {
		t.Fatal("storeA lost its own key after storeB lookup")
	}
	if got != "from-a" {
		t.Fatalf("storeA.Get: want %q, got %q", "from-a", got)
	}
}
