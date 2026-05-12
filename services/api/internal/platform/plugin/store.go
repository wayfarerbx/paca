package plugin

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StoreConfig parameterises where plugin WASM binaries are loaded from.
type StoreConfig struct {
	// Store selects the backend: "local" or "s3".
	Store string
	// WASMDir is the root directory for local WASM files.
	// Only used when Store == "local".
	WASMDir string
	// S3Bucket is the bucket name used when Store == "s3".
	S3Bucket string
	// S3Prefix is the key prefix under which plugins are stored.
	// Binaries are fetched at {S3Prefix}/{pluginName}/backend.wasm.
	S3Prefix string
	// S3Region is the AWS region for S3 operations.
	S3Region string
}

// Store is responsible for loading raw WASM bytes for a named plugin.
type Store struct {
	cfg StoreConfig
	s3  *s3.Client // nil when Store == "local"
}

// NewStore creates a Store using the given configuration.
// When StoreConfig.Store is "s3", an AWS SDK S3 client is initialised using
// the default credential chain (environment, instance profile, etc.).
// Returns an error if Store is not "local" or "s3".
func NewStore(ctx context.Context, cfg StoreConfig) (*Store, error) {
	// Validate store type to prevent silent fallback to local disk.
	if cfg.Store != "local" && cfg.Store != "s3" {
		return nil, fmt.Errorf("plugin store: invalid Store value %q (must be \"local\" or \"s3\")", cfg.Store)
	}

	st := &Store{cfg: cfg}
	if cfg.Store == "s3" {
		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.S3Region))
		if err != nil {
			return nil, fmt.Errorf("plugin store: load AWS config: %w", err)
		}
		st.s3 = s3.NewFromConfig(awsCfg)
	}
	return st, nil
}

// LoadWASM fetches the compiled WASM binary for the plugin identified by name
// (e.g. "com.paca.bdd").  The binary is expected at:
//
//   - local: {WASMDir}/{name}/backend.wasm
//   - s3:    s3://{S3Bucket}/{S3Prefix}/{name}/backend.wasm
func (s *Store) LoadWASM(ctx context.Context, name string) ([]byte, error) {
	switch s.cfg.Store {
	case "s3":
		return s.loadFromS3(ctx, name)
	default:
		return s.loadFromDisk(name)
	}
}

func (s *Store) loadFromDisk(name string) ([]byte, error) {
	path := filepath.Join(s.cfg.WASMDir, name, "backend.wasm")
	data, err := os.ReadFile(path) // #nosec G304 — path components from trusted config + DB
	if err != nil {
		return nil, fmt.Errorf("plugin store: read %q: %w", path, err)
	}
	return data, nil
}

func (s *Store) loadFromS3(ctx context.Context, name string) ([]byte, error) {
	key := s.cfg.S3Prefix + "/" + name + "/backend.wasm"
	out, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("plugin store: s3 get %q: %w", key, err)
	}
	defer func() { _ = out.Body.Close() }()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("plugin store: read s3 body %q: %w", key, err)
	}
	return data, nil
}

// LoadPluginJSON fetches the plugin.json manifest file for the given plugin.
// Layout mirrors LoadWASM: {WASMDir}/{name}/plugin.json or s3 equivalent.
func (s *Store) LoadPluginJSON(ctx context.Context, name string) ([]byte, error) {
	switch s.cfg.Store {
	case "s3":
		key := s.cfg.S3Prefix + "/" + name + "/plugin.json"
		out, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.cfg.S3Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return nil, fmt.Errorf("plugin store: s3 get plugin.json %q: %w", key, err)
		}
		defer func() { _ = out.Body.Close() }()
		data, err := io.ReadAll(out.Body)
		if err != nil {
			return nil, fmt.Errorf("plugin store: read s3 plugin.json %q: %w", key, err)
		}
		return data, nil
	default:
		path := filepath.Join(s.cfg.WASMDir, name, "plugin.json")
		data, err := os.ReadFile(path) // #nosec G304 — trusted config + DB
		if err != nil {
			return nil, fmt.Errorf("plugin store: read plugin.json %q: %w", path, err)
		}
		return data, nil
	}
}

// ListMigrations returns the SQL migration files for the given plugin in
// lexicographic order.  On disk they live at {WASMDir}/{name}/migrations/*.sql.
// On S3 they are listed under {S3Prefix}/{name}/migrations/.
func (s *Store) ListMigrations(ctx context.Context, name string) ([]MigrationFile, error) {
	switch s.cfg.Store {
	case "s3":
		return s.listMigrationsS3(ctx, name)
	default:
		return s.listMigrationsLocal(name)
	}
}

// MigrationFile holds a migration filename and its SQL content.
type MigrationFile struct {
	Name string // e.g. "0001_create_scenarios.sql"
	SQL  string
}

func (s *Store) listMigrationsLocal(name string) ([]MigrationFile, error) {
	dir := filepath.Join(s.cfg.WASMDir, name, "migrations")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil // plugin has no migrations
	}
	if err != nil {
		return nil, fmt.Errorf("plugin store: list migrations for %q: %w", name, err)
	}
	var files []MigrationFile
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		content, err := os.ReadFile(path) // #nosec G304 — trusted config + DB
		if err != nil {
			return nil, fmt.Errorf("plugin store: read migration %q: %w", path, err)
		}
		files = append(files, MigrationFile{Name: e.Name(), SQL: string(content)})
	}
	return files, nil
}

func (s *Store) listMigrationsS3(ctx context.Context, name string) ([]MigrationFile, error) {
	prefix := s.cfg.S3Prefix + "/" + name + "/migrations/"
	paginator := s3.NewListObjectsV2Paginator(s.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.cfg.S3Bucket),
		Prefix: aws.String(prefix),
	})
	var files []MigrationFile
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("plugin store: list s3 migrations for %q: %w", name, err)
		}
		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if filepath.Ext(key) != ".sql" {
				continue
			}
			out, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(s.cfg.S3Bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				return nil, fmt.Errorf("plugin store: get s3 migration %q: %w", key, err)
			}
			data, err := io.ReadAll(out.Body)
			_ = out.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("plugin store: read s3 migration %q: %w", key, err)
			}
			baseName := filepath.Base(key)
			files = append(files, MigrationFile{Name: baseName, SQL: string(data)})
		}
	}
	return files, nil
}
