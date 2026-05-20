// Package config loads runtime configuration from environment variables and
// optional .env files.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Load reads .env (if present) then environment variables and returns a
// validated Config.  Missing required keys cause a non-nil error that names
// every absent variable so operators see all gaps at once.
func Load() (*Config, error) {
	// .env is optional; ignore "file not found" error.
	_ = godotenv.Load()

	accessTTL, err := parseDuration(env("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("config: JWT_ACCESS_TTL: %w", err)
	}
	refreshTTL, err := parseDuration(env("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("config: JWT_REFRESH_TTL: %w", err)
	}
	refreshSessionTTL, err := parseDuration(env("JWT_REFRESH_SESSION_TTL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("config: JWT_REFRESH_SESSION_TTL: %w", err)
	}

	cookieSecure, err := strconv.ParseBool(env("COOKIE_SECURE", "false"))
	if err != nil {
		return nil, fmt.Errorf("config: COOKIE_SECURE: %w", err)
	}

	// Collect all missing required keys before returning so the caller sees
	// every problem in a single error rather than one failure at a time.
	var errs []error

	secret, err := requireEnv("JWT_SECRET")
	if err != nil {
		errs = append(errs, err)
	}

	dsn, err := requireEnv("DATABASE_URL")
	if err != nil {
		errs = append(errs, err)
	}

	dbDriver := env("DATABASE_DRIVER", "postgres")
	switch dbDriver {
	case "postgres", "dsql":
		// valid values
	default:
		errs = append(errs, fmt.Errorf("config: DATABASE_DRIVER must be one of: postgres, dsql"))
	}

	dbAWSRegion := env("DATABASE_AWS_REGION", "")
	if dbDriver == "dsql" && dbAWSRegion == "" {
		errs = append(errs, fmt.Errorf("config: DATABASE_AWS_REGION must be set when DATABASE_DRIVER=dsql"))
	}

	redisURL, err := requireEnv("REDIS_URL")
	if err != nil {
		errs = append(errs, err)
	}

	cacheProjectTTL, err := parseDuration(env("CACHE_PROJECT_TTL", "5m"))
	if err != nil {
		return nil, fmt.Errorf("config: CACHE_PROJECT_TTL: %w", err)
	}
	cacheConfigTTL, err := parseDuration(env("CACHE_CONFIG_TTL", "10m"))
	if err != nil {
		return nil, fmt.Errorf("config: CACHE_CONFIG_TTL: %w", err)
	}
	cacheSprintTTL, err := parseDuration(env("CACHE_SPRINT_TTL", "2m"))
	if err != nil {
		return nil, fmt.Errorf("config: CACHE_SPRINT_TTL: %w", err)
	}

	marketplaceTimeout, err := parseDuration(env("PLUGINS_MARKETPLACE_TIMEOUT", "20s"))
	if err != nil {
		return nil, fmt.Errorf("config: PLUGINS_MARKETPLACE_TIMEOUT: %w", err)
	}

	adminUser, err := requireEnv("ADMIN_USERNAME")
	if err != nil {
		errs = append(errs, err)
	}

	adminPass, err := requireEnv("ADMIN_PASSWORD")
	if err != nil {
		errs = append(errs, err)
	}

	storageAccessKey, err := requireEnv("STORAGE_ACCESS_KEY_ID")
	if err != nil {
		errs = append(errs, err)
	}

	storageSecretKey, err := requireEnv("STORAGE_SECRET_ACCESS_KEY")
	if err != nil {
		errs = append(errs, err)
	}

	storageUseSSL, err := strconv.ParseBool(env("STORAGE_USE_SSL", "false"))
	if err != nil {
		return nil, fmt.Errorf("config: STORAGE_USE_SSL: %w", err)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &Config{
		Env: env("ENV", "development"),
		Server: ServerConfig{
			Port:         env("PORT", "8080"),
			CookieSecure: cookieSecure,
			PublicURL:    env("PUBLIC_URL", ""),
		},
		Database: DatabaseConfig{
			DSN:       dsn,
			Driver:    dbDriver,
			AWSRegion: dbAWSRegion,
		},
		Redis: RedisConfig{
			URL: redisURL,
		},
		Cache: CacheConfig{
			ProjectTTL: cacheProjectTTL,
			ConfigTTL:  cacheConfigTTL,
			SprintTTL:  cacheSprintTTL,
		},
		JWT: JWTConfig{
			Secret:            secret,
			AccessTTL:         accessTTL,
			RefreshTTL:        refreshTTL,
			RefreshSessionTTL: refreshSessionTTL,
		},
		Admin: AdminConfig{
			Username: adminUser,
			Password: adminPass,
		},
		Storage: StorageConfig{
			Provider:        env("STORAGE_PROVIDER", "minio"),
			Endpoint:        env("STORAGE_ENDPOINT", "minio:9000"),
			PublicURL:       env("STORAGE_PUBLIC_URL", ""),
			Region:          env("STORAGE_REGION", "us-east-1"),
			Bucket:          env("STORAGE_BUCKET", "paca"),
			AccessKeyID:     storageAccessKey,
			SecretAccessKey: storageSecretKey,
			UseSSL:          storageUseSSL,
		},
		Security: SecurityConfig{
			// ENCRYPTION_KEY should be a 64-character lowercase hex string
			// representing 32 bytes (AES-256).
			// Backward compatibility: fall back to legacy GITHUB_ENCRYPTION_KEY.
			EncryptionKey: env("ENCRYPTION_KEY", env("GITHUB_ENCRYPTION_KEY", "")),
			// AGENT_API_KEY is optional; when set the API accepts it as a
			// static service key for the AI agent without a DB lookup.
			AgentAPIKey: env("AGENT_API_KEY", ""),
		},
		Plugins: PluginsConfig{
			// PLUGINS_STORE controls where WASM binaries are loaded from.
			// "local" reads from the local filesystem; "s3" reads from the
			// object-storage bucket configured via STORAGE_* variables.
			Store:                 env("PLUGINS_STORE", "local"),
			WASMDir:               env("PLUGINS_WASM_DIR", "./plugins/local/backend"),
			FrontendDir:           env("PLUGINS_FRONTEND_DIR", "./plugins/local/frontend"),
			MCPDir:                env("PLUGINS_MCP_DIR", "./plugins/local/mcp"),
			S3Prefix:              env("PLUGINS_S3_PREFIX", "plugins"),
			MarketplaceCatalogURL: env("PLUGINS_MARKETPLACE_CATALOG_URL", "https://raw.githubusercontent.com/Paca-AI/paca-plugins/master/catalog/plugins.json"),
			MarketplaceTimeout:    marketplaceTimeout,
		},
		AIAgentURL: env("AI_AGENT_URL", "http://ai-agent:8080"),
	}, nil
}

// env returns the environment variable value or a fallback default.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// requireEnv returns the value of the named environment variable, or an error
// if the variable is unset or empty.
func requireEnv(key string) (string, error) {
	if v := os.Getenv(key); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("config: %s must be set", key)
}

func parseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	return d, nil
}
