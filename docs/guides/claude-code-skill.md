# Paca Skill for Claude Code

Use Paca directly from Claude Code CLI with the `/paca` and `/paca-setup` slash commands. Once installed, Claude will use your Paca workspace for tasks, documentation, and sprint management — instead of creating local files.

## Install

Run the installer (works on macOS and Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/Paca-AI/paca/master/scripts/install-claude-skill.sh | bash
```

Or, from a local clone of this repo:

```bash
bash scripts/install-claude-skill.sh
```

The installer copies two skill files to `~/.claude/commands/`, making `/paca` and `/paca-setup` available in every Claude Code session.

## Configure the MCP server

The skill requires the Paca MCP server to be connected. After installing the skill, run `/paca-setup` inside a Claude Code session for an interactive setup walkthrough, or follow the quick steps below.

### Quick setup — Claude Code CLI

```bash
claude mcp add paca \
  --env PACA_API_KEY=<your-api-key> \
  --env PACA_API_URL=<your-paca-url> \
  -- npx -y @paca-ai/paca-mcp
```

Replace `<your-api-key>` (from Paca → Settings → API Keys) and `<your-paca-url>` (e.g. `http://localhost:8080` or your hosted URL).

### Project-level setup (recommended for teams)

Create `.claude/mcp.json` in your project root:

```json
{
  "mcpServers": {
    "paca": {
      "command": "npx",
      "args": ["-y", "@paca-ai/paca-mcp"],
      "env": {
        "PACA_API_KEY": "<your-api-key>",
        "PACA_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

> **Security:** Do not commit API keys. Add `.claude/mcp.json` to `.gitignore` or inject `PACA_API_KEY` from your shell environment.

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "paca": {
      "command": "npx",
      "args": ["-y", "@paca-ai/paca-mcp"],
      "env": {
        "PACA_API_KEY": "<your-api-key>",
        "PACA_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

Restart Claude Desktop after saving.

## Commands

### `/paca <request>`

Interact with your Paca workspace in natural language. Claude will route your request to the right Paca MCP tools.

No special syntax or prefixes — just describe what you want in plain English:

```
/paca Fix the login redirect bug, assign to sprint 3
/paca Write the API authentication guide
/paca I need to review the PR, update staging, and write release notes
/paca What's in the current sprint?
/paca Mark task #42 as done
```

You can also reference an existing task by ID, prefixed ID, or URL and Claude will look it up automatically:

```
/paca #42 is done
/paca ABC-17 is blocked — add a comment: needs design review
/paca http://localhost/projects/abc-uuid/tasks/def-uuid move to next sprint
```

Claude reads your intent and picks the right Paca tool automatically:

| You say... | Claude uses... |
|---|---|
| Describe a bug / feature / chore | `create_task` → Paca Board |
| Ask to write a guide / spec / design | `create_document` → Paca Docs |
| Reference `#42`, `ABC-42`, or a task URL | `get_task_by_number` or `get_task` → then act on it |
| List multiple to-dos in one message | `create_task` per item → Paca Board |
| Ask about sprint / board / status | `list_tasks` + `list_sprints` |
| Say "mark X as done" / "close X" | `update_task` → sets status |

### `/paca-setup`

Interactive setup wizard. Walks you through connecting Claude Code to your Paca instance, verifying the connection, and installing the global skill.

```
/paca-setup
```

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `PACA_API_KEY` | Yes | — | API key (Paca → Settings → API Keys) |
| `PACA_API_URL` | No | `http://localhost:8080` | Your Paca instance URL |
| `PACA_PROJECT_ID` | No | — | Scope Claude to a single project |
| `PACA_AGENT_ID` | No | — | Agent identity for agent-mode API keys |

## Make Paca the default for your project

To make Claude always prefer Paca tools in a project (without needing to type `/paca` every time), add this to your project's `CLAUDE.md`:

```markdown
## Project management

This project uses Paca for all project management. When working in this codebase:

- **Tasks and to-dos** → use `create_task` / `list_tasks` via the Paca MCP tools. Do not create local TODO files or add TODO comments.
- **Documentation** → use `create_document` / `update_document` via Paca MCP. Do not create standalone `.md` docs unless they belong in the repository (e.g. README, CONTRIBUTING).
- **Sprint planning** → use `create_sprint` / `list_sprints` via Paca MCP.

If Paca MCP tools are not available, say so and ask the user to run `/paca-setup`.
```

## Uninstall

```bash
rm ~/.claude/commands/paca.md ~/.claude/commands/paca-setup.md
```

## Available tools

The Paca MCP server provides **81 tools** across 16 categories. See [mcp-server-setup.md](mcp-server-setup.md) for the complete reference.
