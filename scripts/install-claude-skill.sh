#!/usr/bin/env bash
# Paca Claude Code Skill Installer
# Installs /paca and /paca-setup slash commands into ~/.claude/commands/
# so they are available in every Claude Code session.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Paca-AI/paca/master/scripts/install-claude-skill.sh | bash
#   OR (from a local clone):
#   bash scripts/install-claude-skill.sh

set -euo pipefail

REPO="Paca-AI/paca"
BRANCH="master"
BASE_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/.claude/commands"
DEST_DIR="${HOME}/.claude/commands"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()    { echo -e "${CYAN}[paca]${NC} $*"; }
success() { echo -e "${GREEN}[paca]${NC} $*"; }
warn()    { echo -e "${YELLOW}[paca]${NC} $*"; }

echo ""
echo "  🦙 Paca Claude Code Skill Installer"
echo "  ────────────────────────────────────"
echo ""

# Detect if running from a local clone (the script lives in scripts/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || echo "")"
LOCAL_COMMANDS=""
if [[ -n "${SCRIPT_DIR}" && -d "${SCRIPT_DIR}/../.claude/commands" ]]; then
  LOCAL_COMMANDS="${SCRIPT_DIR}/../.claude/commands"
  info "Local clone detected — installing from ${LOCAL_COMMANDS}"
else
  info "Installing from GitHub (${REPO}@${BRANCH})"
  # Check for curl or wget
  if ! command -v curl &>/dev/null && ! command -v wget &>/dev/null; then
    echo "Error: curl or wget is required. Install one and re-run." >&2
    exit 1
  fi
fi

# Create destination directory
mkdir -p "${DEST_DIR}"
info "Installing skills to: ${DEST_DIR}"
echo ""

install_skill() {
  local name="$1"
  local dest="${DEST_DIR}/${name}.md"

  if [[ -n "${LOCAL_COMMANDS}" ]]; then
    cp "${LOCAL_COMMANDS}/${name}.md" "${dest}"
  elif command -v curl &>/dev/null; then
    curl -fsSL "${BASE_URL}/${name}.md" -o "${dest}"
  else
    wget -qO "${dest}" "${BASE_URL}/${name}.md"
  fi

  success "Installed: /claude commands/${name}.md"
}

install_skill "paca"
install_skill "paca-setup"

echo ""
success "Installation complete!"
echo ""
echo "  Available commands in Claude Code:"
echo "  ┌─────────────────────────────────────────────────────────────────────┐"
echo "  │  /paca <request>   — Manage tasks, docs, sprints via Paca           │"
echo "  │  /paca-setup       — Configure the Paca MCP server connection        │"
echo "  └─────────────────────────────────────────────────────────────────────┘"
echo ""
echo "  Next step: configure the Paca MCP server."
echo "  In a Claude Code session, run:  /paca-setup"
echo ""
echo "  Or add the MCP server manually:"
echo ""
echo "    claude mcp add paca \\"
echo "      --env PACA_API_KEY=<your-api-key> \\"
echo "      --env PACA_API_URL=<your-paca-url> \\"
echo "      -- npx -y @paca-ai/paca-mcp"
echo ""
echo "  Docs: https://github.com/${REPO}/blob/${BRANCH}/docs/guides/claude-code-skill.md"
echo ""
