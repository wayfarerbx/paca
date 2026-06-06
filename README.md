<p align="center">
  <img src="docs/assets/paca-logo.svg" alt="Paca logo" width="256" />
</p>

<h1 align="center">Paca</h1>

<p align="center"><strong>AI-native. Free. Lightweight. Open-source.<br/>The fully customizable alternative to Jira, Trello, ClickUp, and Monday.</strong></p>

<p align="center">
  <a href="https://github.com/Paca-AI/paca/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="License" /></a>
  <a href="https://github.com/Paca-AI/paca/releases"><img src="https://img.shields.io/github/v/release/Paca-AI/paca" alt="Latest Release" /></a>
  <a href="https://github.com/Paca-AI/paca/stargazers"><img src="https://img.shields.io/github/stars/Paca-AI/paca?style=social" alt="Stars" /></a>
</p>

<p align="center">
  <a href="#getting-started">Getting Started</a>
  ·
  <a href="docs/architecture/overview.md">Architecture</a>
  ·
  <a href="CONTRIBUTING.md">Contributing</a>
  ·
  <a href="ROADMAP.md">Roadmap</a>
</p>

---

## What is Paca?

Paca is a **self-hosted project management platform** where AI agents and humans collaborate as equal teammates inside a Scrum team — not as chatbots bolted on the side.

Jira gives you a backlog. ClickUp gives you automations. Monday gives you dashboards. **Paca gives your AI agents a seat at the table.** They join sprint planning, pick up tasks from the board, write BDD specs, and adapt alongside humans in real time.

Everything about Paca — its workflow, its data model, its UI — is **configurable and extendable via plugins**.

---

## Why Paca?

| | Jira / Trello / ClickUp / Monday | **Paca** |
|:--|:--|:--|
| **AI integration** | Chatbot add-ons, peripheral automation | AI agents as first-class Scrum teammates |
| **Collaboration model** | Human-only by default | Human + AI, side by side on the same board |
| **Hosting** | Vendor cloud (your data, their servers) | Self-hosted, you own everything |
| **Cost** | $8–$20+ per seat/month | **Free forever** |
| **Customization** | Limited; locked behind enterprise tiers | **Fully open: configuration + plugins** |
| **Weight** | Bloated feature sprawl | Lightweight core; extend only what you need |
| **Source** | Closed / proprietary | **100% open-source (Apache 2.0)** |

---

## Core Idea: Humans and AI Agents, One Scrum Team

The central insight behind Paca is that **AI agents should participate in the Scrum process**, not just generate output in isolation.

In Paca, AI agents:

- Are **assigned to sprints** and appear on the Scrumban board alongside human teammates
- **Pick up tasks** from the backlog and update their status in real time
- **Collaborate on BDD specs** — helping Product Owners and BAs write Gherkin scenarios
- **Contribute to System Design Documents** — keeping the architecture visible to the whole team
- **Probe, sense, and respond** to emerging complexity, just like a human would

This is not automation. It is **genuine collaboration** — rooted in the Cynefin / Stacey framework's recognition that complex domains require teams, not pipelines.

<p align="center">
  <video src="https://github.com/Paca-AI/paca/releases/download/v0.3.4/demo-assign-task.mp4" autoplay muted loop playsinline width="720"></video>
</p>

---

## Fully Customizable — Configuration and Plugins

Paca ships as a small, focused core. Everything else is optional.

**Configuration-driven:** workflows, statuses, field definitions, board layouts, sprint rules, and agent behavior are all driven by project-level configuration files. No code needed to adapt Paca to your team's process.

**Plugin system:** extend or replace any part of Paca via plugins. Plugins are compiled to **WebAssembly (WASM)** for the backend (write in Go, Rust, AssemblyScript — anything with a WASM target) and standard module bundles for the frontend. Plugins run in a sandboxed environment with a capability-based permission model; they declare exactly what host functions they need, and nothing more.

```
plugins/
├── backend/        # WASM modules — add custom routes, logic, data models
└── frontend/       # UI modules — add custom pages, board views, widgets
```

Browse and install community plugins directly from the **Plugin Marketplace** inside the Paca UI — no command line required. Go to **Settings → Plugins → Marketplace**, find a plugin, and click **Install**.

<p align="center">
  <video src="https://github.com/Paca-AI/paca/releases/download/v0.3.4/demo-plugin-marketplace.mp4" autoplay muted loop playsinline width="720"></video>
</p>

For local development or custom plugins, you can also install from the filesystem:

```bash
./scripts/install-local-plugin.sh ./my-plugin --api-key <your-api-key>
```

---

## The P-A-C-A Cycle

Paca structures team collaboration around four phases that mirror both Scrum and the scientific method:

```
Plan  →  Act  →  Check  →  Adapt
  ↑                             |
  └─────────────────────────────┘
```

| Phase | What happens |
|:--|:--|
| **Plan** | POs, BAs, and AI agents collaboratively refine the backlog. BDD scenarios and SDD designs are written together. |
| **Act** | Sprint is live. Humans and AI agents pull tasks from the board, execute, and post updates. |
| **Check** | QA agents run automated verification. Humans review AI output. The board reflects reality. |
| **Adapt** | Data from the sprint informs the next cycle. The team — human and AI — retrospects together. |

---

## Key Features

- **Unified Scrumban Board** — humans and AI agents share a single real-time board; no separate "AI workspace"
- **BDD Collaboration** — Gherkin scenario editor co-authored by POs, BAs, and AI agents
- **System Design Documents (SDD)** — living architecture docs that keep AI agents contextually grounded
- **MCP Server** — connect Claude, custom agents, or any MCP-compatible tool directly into Paca's data layer
- **Real-time updates** — Socket.IO delivery; everyone sees changes the moment they happen
- **OpenHands-powered agents** — AI agents run on the [OpenHands](https://github.com/All-Hands-AI/OpenHands) SDK; each agent executes inside its own isolated sandbox container so your host environment is never touched
- **WASM plugin sandbox** — extend Paca safely; plugins cannot escape their declared permissions
- **Self-hosted** — runs on a single Docker Compose command; your data never leaves your infrastructure
- **Lightweight by default** — minimal core, no feature bloat; add only what your team actually needs

---

## Getting Started

### Option 1 — Interactive install script (recommended for production)

Runs on any Linux server with Docker. No repository clone required.

```bash
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/install.sh | bash
```

The script walks you through configuration interactively and starts the full stack. Open `http://your-server-ip` when it finishes.

<p align="center">
  <video src="https://github.com/Paca-AI/paca/releases/download/v0.3.4/demo-install.mp4" autoplay muted loop playsinline width="720"></video>
</p>

---

### Option 2 — Docker Compose (manual)

```bash
# 1. Create a directory and download the compose file
mkdir paca && cd paca
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/docker-compose.yml -o docker-compose.yml
mkdir -p nginx
curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/gateway.conf -o nginx/gateway.conf

# 2. Create your environment file
cat > .env <<'EOF'
JWT_SECRET=<run: openssl rand -hex 32>
ADMIN_PASSWORD=<your-admin-password>
POSTGRES_PASSWORD=<run: openssl rand -hex 32>
AGENT_API_KEY=<run: openssl rand -hex 32>
INTERNAL_API_KEY=<run: openssl rand -hex 32>
ENCRYPTION_KEY=<run: openssl rand -hex 32>
PUBLIC_URL=http://localhost
EOF

# 3. Start the stack
docker compose --env-file .env up -d
```

Open `http://localhost` — log in with `admin` and the password you set.

> **Customizing the stack:** scale down services you don't need.
>
> ```bash
> # External PostgreSQL (supply DATABASE_URL in .env)
> docker compose --env-file .env up -d --scale postgres=0
>
> # AWS S3 instead of MinIO (set STORAGE_PROVIDER=s3 in .env)
> docker compose --env-file .env up -d --scale minio=0
>
> # Without the AI agent (reduces resource usage)
> docker compose --env-file .env up -d --scale ai-agent=0
> ```

---

### Option 3 — Local development

```bash
# Clone the repository
git clone https://github.com/Paca-AI/paca.git && cd paca

# Start infrastructure dependencies (PostgreSQL + Valkey)
docker compose -f deploy/docker-compose.dev.yml up -d postgres valkey

# Or start the full dev stack in containers
docker compose -f deploy/docker-compose.dev.yml up -d
```

See [docs/guides/local-development.md](docs/guides/local-development.md) for running services on the host for active development.

---

## MCP Server — Connect Any AI Agent to Paca

Paca ships an [MCP (Model Context Protocol)](https://modelcontextprotocol.io) server that gives any compatible AI agent direct, structured access to your workspace — projects, tasks, sprints, documents, members, and more. No scraping, no custom APIs to wire up.

The server is published as **`@paca-ai/paca-mcp`** on npm. You run it with `npx`; your MCP client handles the rest.

### Quick Setup — Claude Desktop

1. Open (or create) the Claude Desktop config file:
   - **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

2. Add the `paca` entry:

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

3. Restart Claude Desktop. Claude now has access to all Paca tools and can answer requests like:
   - *"List all active sprints in project X"*
   - *"Create a task for implementing OAuth and assign it to sprint 3"*
   - *"Add a comment to task #42 with my progress update"*

### Environment Variables

| Variable | Required | Default | Description |
|:--|:--|:--|:--|
| `PACA_API_KEY` | Yes | — | API key from your Paca instance (Settings → API Keys) |
| `PACA_API_URL` | No | `http://localhost:8080` | URL of your Paca API |
| `PACA_PROJECT_ID` | No | — | Scope the agent to a single project |
| `PACA_AGENT_ID` | No | — | Identity for agent-mode (requires `PACA_PROJECT_ID`) |

### Other MCP-Compatible Clients

Any client that speaks MCP works. Typical configuration:

```json
{
  "name": "paca",
  "command": "npx",
  "args": ["-y", "@paca-ai/paca-mcp"],
  "env": {
    "PACA_API_KEY": "your-api-key-here",
    "PACA_API_URL": "http://your-paca-instance:8080"
  }
}
```

### Available Tools

The server exposes tools across these categories:

| Category | Tools |
|:--|:--|
| Projects | `list_projects`, `get_project`, `create_project`, `update_project`, `delete_project` |
| Tasks | `list_tasks`, `get_task`, `create_task`, `update_task`, `delete_task`, + more |
| Sprints | `list_sprints`, `create_sprint`, `update_sprint`, `complete_sprint`, + more |
| Documents | `list_docs`, `read_doc`, `write_doc`, `delete_doc`, `move_doc` |
| Members & Roles | `list_project_members`, `add_project_member`, `list_project_roles`, + more |
| Task Types & Statuses | `list_task_types`, `create_task_type`, `list_task_statuses`, + more |
| Views & Custom Fields | `list_views`, `create_view`, `list_custom_fields`, `create_custom_field`, + more |
| Attachments | `list_task_attachments`, `get_attachment_download_url`, `delete_task_attachment` |
| Activity & Comments | `list_task_activities`, `add_task_comment`, `update_task_comment`, `delete_task_comment` |
| Plugin tools | Installed plugins can register additional tools at runtime |

For a complete reference and advanced configuration (agent-mode, plugin tools, programmatic usage), see [docs/guides/mcp-server-setup.md](docs/guides/mcp-server-setup.md).

---

## Architecture

```
apps/web          React + TanStack Start + shadcn/ui — user interface
apps/mcp          @paca-ai/paca-mcp — MCP server for AI agent integration
services/api      Go + Gin — core business logic and REST API
services/realtime Node.js + Socket.IO — real-time event fan-out
services/ai-agent Python + FastAPI + OpenHands SDK — AI agent orchestration
apps/e2e          Playwright — end-to-end test suite

PostgreSQL        Persistent store
Valkey            Cache + async event streams between services
```

See [docs/architecture/overview.md](docs/architecture/overview.md) for detail.

---

## The "Paca" Story

The name is a small pun on the Japanese word **"Baka" (ばか)** — "silly."

In the early days, we jokingly called our AI assistants "silly" when they hallucinated. And building a serious project management platform as a free, open-source alternative to multi-billion-dollar tools might also seem a bit silly.

But Paca is built from conviction: human-AI collaboration in a real Scrum team should be accessible to every team, everywhere — not locked behind a vendor's pricing model. We think that's worth being a little foolish about. 🦙✨

---

## Documentation

| Document | Description |
|:--|:--|
| [docs/architecture/overview.md](docs/architecture/overview.md) | High-level system architecture |
| [docs/guides/getting-started.md](docs/guides/getting-started.md) | Getting started (install, Docker, local dev) |
| [docs/guides/local-development.md](docs/guides/local-development.md) | Contributor dev environment setup |
| [docs/guides/mcp-server-setup.md](docs/guides/mcp-server-setup.md) | Connect AI agents via MCP |
| [docs/plugins/](docs/plugins/) | Plugin system: backend (WASM) and frontend |
| [deploy/README.md](deploy/README.md) | Full deployment reference |
| [CHANGELOG.md](CHANGELOG.md) | Release history |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute |
| [SECURITY.md](SECURITY.md) | Security policy |

---

## License

Distributed under the **Apache License 2.0**. See [LICENSE](LICENSE) for details.
