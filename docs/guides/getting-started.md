# Getting Started

## Option 1 — Install Script (recommended)

Runs on any Linux server with Docker. Downloads the release assets, walks you through configuration interactively, and starts the full stack.

```bash
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/install.sh | bash
```

Open `http://your-server-ip` when it finishes.

---

## Option 2 — Docker Compose (manual)

Pulls pre-built images. No repository clone required.

```bash
# Download compose file and nginx config
mkdir paca && cd paca
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/docker-compose.yml -o docker-compose.yml
mkdir -p nginx
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/gateway.conf -o nginx/gateway.conf

# Create an environment file
cat > .env <<'EOF'
JWT_SECRET=<run: openssl rand -hex 32>
ADMIN_PASSWORD=<your-admin-password>
POSTGRES_PASSWORD=<run: openssl rand -hex 32>
AGENT_API_KEY=<run: openssl rand -hex 32>
INTERNAL_API_KEY=<run: openssl rand -hex 32>
ENCRYPTION_KEY=<run: openssl rand -hex 32>
PUBLIC_URL=http://localhost
EOF

# Start the stack
docker compose --env-file .env up -d
```

Open `http://localhost` — log in with `admin` and the password you set.

---

## Option 3 — Local Development

For contributors. Clone the repo, then start everything with one command:

```bash
git clone https://github.com/Paca-AI/paca.git && cd paca
docker compose -f deploy/docker-compose.dev.yml up -d
```

All services start with hot-reload — the API, web app, and realtime service all watch your local source files and rebuild automatically. Open `http://localhost` when the stack is healthy.

See [local-development.md](local-development.md) for details on the dev stack and running services on the host.

---

## Upgrading to a new version

Pull the latest images and restart the stack. Run these commands from the directory where your `docker-compose.yml` lives:

```bash
docker compose pull
docker compose --env-file .env up -d
```

Database migrations run automatically on API startup — no manual steps are required.

> **Before upgrading:** check [CHANGELOG.md](../../CHANGELOG.md) for breaking changes or release-specific migration steps.

---

## Connect an AI Agent via MCP

After Paca is running:

1. Generate an API key: **Settings → API Keys → New Key**
2. Add the Paca MCP server to your agent config (Claude Desktop example):

```json
{
  "mcpServers": {
    "paca": {
      "command": "npx",
      "args": ["-y", "@paca-ai/paca-mcp"],
      "env": {
        "PACA_API_KEY": "your-api-key-here",
        "PACA_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

See [mcp-server-setup.md](mcp-server-setup.md) for platform-specific instructions and advanced configuration.

---

## What to Read Next

| Document | When to read it |
|---|---|
| [local-development.md](local-development.md) | Setting up a contributor environment |
| [mcp-server-setup.md](mcp-server-setup.md) | Connecting AI agents via MCP |
| [../architecture/overview.md](../architecture/overview.md) | Understanding the system architecture |
| [../plugins/overview.md](../plugins/overview.md) | Writing or installing plugins |
| [../../deploy/README.md](../../deploy/README.md) | Production deployment reference |
| [../../CHANGELOG.md](../../CHANGELOG.md) | Release history |
