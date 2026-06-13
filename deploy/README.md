# Deploy

This directory contains deployment assets for two distinct use cases:

- contributor-friendly local development;
- production container deployment for self-hosters.

## Contents

| File | Description |
|---|---|
| `docker-compose.dev.yml` | Local development stack: PostgreSQL, Valkey, MinIO, and optional app containers |
| `docker-compose.prod.yml` | Production stack: pulls pre-built images from DockerHub, no source checkout required |
| `docker-compose.e2e.yml` | End-to-end test stack mirroring production topology with fixed test credentials |
| `.env.dev.example` | Optional environment file for `docker-compose.dev.yml` (tunnel / custom domain) |
| `.env.production.example` | Example environment file for manual production deployments |

Service container definitions live with each service:
- [`services/api/Dockerfile`](../services/api/Dockerfile)
- [`apps/web/Dockerfile`](../apps/web/Dockerfile)

## Production Deployment

### Recommended: install script

The easiest way to run Paca without cloning the repository is via the install script
published with each release. It downloads the compose file and nginx config, walks you
through configuration interactively (database, storage, AI agent), generates a `.env`
with strong random secrets, and starts the stack.

```bash
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/install.sh -o install.sh
bash install.sh
```

The installer supports:

| Option | Description |
|---|---|
| Bundled PostgreSQL | Starts a postgres container (default) |
| External PostgreSQL | Supply a `DATABASE_URL`; postgres container is suppressed |
| Self-hosted MinIO | Starts a MinIO container for S3-compatible file storage (default) |
| AWS S3 | Supply AWS credentials; MinIO container is suppressed |
| AI Agent | Enabled by default; can be skipped to reduce resource usage |

### Manual setup

Download the two required files from the latest release:

```bash
mkdir -p paca/nginx && cd paca
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/docker-compose.yml -o docker-compose.yml
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/gateway.conf      -o nginx/gateway.conf
```

Download the example environment file and edit it:

```bash
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/docker-compose.yml -o docker-compose.yml
# Or use the .env.production.example from the repo as a reference:
# https://github.com/Paca-AI/paca/blob/master/deploy/.env.production.example
```

Create a `.env` with the required variables:

```bash
# Required: generate with 'openssl rand -hex 32'
JWT_SECRET=<strong-random-secret>
ADMIN_PASSWORD=<strong-password>
POSTGRES_PASSWORD=<strong-random-password>
# Required when using AI agent: generate with 'openssl rand -hex 32'
AGENT_API_KEY=<strong-random-secret>
INTERNAL_API_KEY=<strong-random-secret>
# Required for plugin secrets at rest: generate with 'openssl rand -hex 32'
ENCRYPTION_KEY=<64-char-hex>
PUBLIC_URL=http://your-domain-or-ip
```

Start the full stack (bundled PostgreSQL + MinIO):

```bash
docker compose --env-file .env up -d
```

**With external PostgreSQL** (suppress the bundled container):

```bash
# Set DATABASE_URL in .env to your managed connection string.
docker compose --env-file .env up -d --scale postgres=0
```

**With AWS S3** (suppress MinIO):

```bash
# Set STORAGE_PROVIDER=s3 and real AWS credentials in .env.
docker compose --env-file .env up -d --scale minio=0
```

**Without the AI agent**:

```bash
docker compose --env-file .env up -d --scale ai-agent=0
```

Flags can be combined:

```bash
docker compose --env-file .env up -d --scale postgres=0 --scale minio=0
```

### Upgrading to a new version

Pull the latest images and restart the stack:

```bash
docker compose pull
docker compose --env-file .env up -d
```

Database migrations run automatically on API startup — no manual steps are required.

> **Before upgrading:** check the [CHANGELOG](../CHANGELOG.md) for breaking changes or release-specific migration steps.

---

### Upgrading from an earlier installation

The compose project was renamed from `paca-prod` to `paca` in this release.
Docker Compose namespaces volumes by project name, so existing volumes
(`paca-prod_postgres_data`, `paca-prod_minio_data`, etc.) are **not** automatically
attached to the new stack. To migrate:

```bash
# 1. Stop the old stack (volumes are preserved on disk).
docker compose -p paca-prod --env-file .env down

# 2. Rename each volume you want to keep.
docker volume create paca_postgres_data
docker run --rm \
  -v paca-prod_postgres_data:/from \
  -v paca_postgres_data:/to \
  alpine sh -c "cp -av /from/. /to/"
docker volume rm paca-prod_postgres_data

# Repeat for minio_data, valkey_data, and plugin volumes as needed.

# 3. Start the new stack.
docker compose --env-file .env up -d
```

If you are doing a fresh install (no data to keep), no migration is needed.

### Pinning a release version

Set the image variables in `.env` to lock to a specific release:

```bash
PACA_API_IMAGE=pacaai/paca-api:1.2.3
PACA_WEB_IMAGE=pacaai/paca-web:1.2.3
PACA_REALTIME_IMAGE=pacaai/paca-realtime:1.2.3
PACA_AI_AGENT_IMAGE=pacaai/paca-ai-agent:1.2.3
```

## Development Compose

Use [`docker-compose.dev.yml`](./docker-compose.dev.yml) for local development and contributor onboarding.

When exposing the stack through a tunnel or reverse proxy, copy the example env file and set the public host:

```bash
cp deploy/.env.dev.example deploy/.env.dev
# Edit PUBLIC_HOST and VITE_ALLOWED_HOST in deploy/.env.dev
docker compose --env-file deploy/.env.dev -f deploy/docker-compose.dev.yml up -d
```

Start the full local stack in containers (no tunnel, plain localhost):

```bash
docker compose -f deploy/docker-compose.dev.yml up -d
```

Start only shared dependencies:

```bash
docker compose -f deploy/docker-compose.dev.yml up -d postgres valkey
```

For day-to-day coding, contributors can run the application services directly on the host
and use Docker Compose only for PostgreSQL and Valkey.

### Development service ports

| Service | Port | Notes |
|---|---|---|
| PostgreSQL | 5432 | Local database for development |
| Valkey | 6379 | Local cache / event streams |
| API | 8080 | Containerized Go service |
| Web | 3000 | Containerized React app |
| MinIO S3 API | 9000 | Local object store (S3-compatible) |
| MinIO Console | 9001 | MinIO web UI (credentials: `minioadmin` / `minioadmin`) |

Stop the development stack:

```bash
docker compose -f deploy/docker-compose.dev.yml down
```

Remove the Postgres volume as well:

```bash
docker compose -f deploy/docker-compose.dev.yml down -v
```

## E2E Compose

Use [`docker-compose.e2e.yml`](./docker-compose.e2e.yml) to spin up a full production-like
stack with fixed, test-safe credentials for running end-to-end tests:

```bash
docker compose -f deploy/docker-compose.e2e.yml up -d --build --wait
docker compose -f deploy/docker-compose.e2e.yml down -v
```

All secrets are intentionally weak and public — never use them outside a local E2E environment.
