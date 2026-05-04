// Package config defines the typed configuration model for the API service.
package config

import "time"

// Config holds all runtime configuration for the service.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Cache    CacheConfig
	JWT      JWTConfig
	Admin    AdminConfig
	Storage  StorageConfig
	GitHub   GitHubConfig
	Env      string // development | production
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         string
	CookieSecure bool // set Secure flag on auth cookies; enable when behind an SSL-terminating proxy
}

// AdminConfig holds the default administrator credentials seeded on first startup.
type AdminConfig struct {
	Username string
	Password string
}

// DatabaseConfig holds the primary database connection settings.
type DatabaseConfig struct {
	DSN string

	// Driver selects the database backend: "postgres" (default) or "dsql"
	// (Amazon Aurora DSQL with IAM authentication).
	Driver string

	// AWSRegion is the AWS region used when Driver is "dsql".
	// It must match the region where the Aurora DSQL cluster is deployed.
	AWSRegion string
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	URL string
}

// CacheConfig holds TTL settings for the different cache categories.
//
// Each TTL controls how long the corresponding data is served from Valkey/Redis
// before a fresh database read is made.  Set a TTL to zero to disable caching
// for that category entirely.
//
// Environment variables (loaded by config.Load):
//
//	CACHE_PROJECT_TTL  – project + member data          (default: 5m)
//	CACHE_CONFIG_TTL   – task types, statuses, custom
//	                     field definitions, and roles    (default: 10m)
//	CACHE_SPRINT_TTL   – sprints and views               (default: 2m)
type CacheConfig struct {
	// ProjectTTL is the cache duration for project detail and member list data.
	ProjectTTL time.Duration
	// ConfigTTL is the cache duration for infrequently-changing project
	// configuration: task types, task statuses, custom field definitions, and
	// project roles.  Global roles also use this TTL.
	ConfigTTL time.Duration
	// SprintTTL is the cache duration for sprint and view configuration data.
	SprintTTL time.Duration
}

// JWTConfig holds JWT signing and expiry settings.
type JWTConfig struct {
	Secret            string
	AccessTTL         time.Duration
	RefreshTTL        time.Duration // persistent session (remember me = true)
	RefreshSessionTTL time.Duration // ephemeral session (remember me = false)
}

// StorageConfig holds object-storage settings.
// When Provider is "s3" the service connects to AWS S3 using the Region field.
// When Provider is "minio" (default) it targets the Endpoint URL.
// The bucket is created automatically on startup if it does not exist.
type StorageConfig struct {
	Provider        string // "s3" | "minio"  (default: "minio")
	Endpoint        string // MinIO URL, e.g. "minio:9000"; ignored for AWS S3
	PublicURL       string // public-facing base URL for presigned URLs, e.g. "http://localhost/storage"
	Region          string // AWS region; used for S3; also supplied to MinIO (can be any value)
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool // set true when Endpoint is HTTPS
}

// GitHubConfig holds settings for the GitHub integration feature.
type GitHubConfig struct {
	// EncryptionKey is a 32-byte AES-256 key (hex-encoded) used to encrypt
	// GitHub personal access tokens and webhook secrets at rest.
	// Required when the GitHub integration feature is in use.
	EncryptionKey string

	// WebhookURL is the fully-qualified URL GitHub will POST webhook events to,
	// derived from the PUBLIC_URL environment variable by appending
	// "/api/v1/github/webhook".  Empty in local development (skips webhook creation).
	WebhookURL string
}
