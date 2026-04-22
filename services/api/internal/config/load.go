// Package config loads runtime configuration from environment variables and
// optional .env files.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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

	redisURL, err := requireEnv("REDIS_URL")
	if err != nil {
		errs = append(errs, err)
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
		},
		Database: DatabaseConfig{
			DSN: dsn,
		},
		Redis: RedisConfig{
			URL: redisURL,
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
		GitHub: GitHubConfig{
			// GITHUB_ENCRYPTION_KEY must be a 64-character lowercase hex string
			// representing 32 bytes (AES-256).  It is optional; when absent the
			// GitHub integration endpoints will fail with a clear error at runtime.
			EncryptionKey: env("GITHUB_ENCRYPTION_KEY", ""),
			// WebhookURL is derived from PUBLIC_URL by appending the webhook path.
			// Set PUBLIC_URL to the externally reachable base URL of this service,
			// e.g. "https://api.example.com".  Leave empty in development to skip
			// webhook creation when linking GitHub repositories.
			WebhookURL: buildWebhookURL(env("PUBLIC_URL", "")),
		},
	}, nil
}

// buildWebhookURL appends the standard GitHub webhook path to the public base URL.
// Returns an empty string when publicURL is empty (disables automatic webhook creation).
func buildWebhookURL(publicURL string) string {
	if publicURL == "" {
		return ""
	}
	return strings.TrimRight(publicURL, "/") + "/api/v1/github/webhook"
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
