#!/usr/bin/env bash
# Paca – upgrade script
#
# Updates an existing Paca installation (created by install.sh, or set up
# manually per deploy/README.md) to a new release: refreshes
# docker-compose.yml and the Caddyfile, re-pins image versions in .env when a
# specific version is requested, backfills any .env variables introduced
# since the install was created, then pulls and restarts the stack.
#
# Run this from the directory that holds your docker-compose.yml and .env
# (the directory install.sh created, or wherever you set things up manually).
#
# ── Recommended (interactive) ────────────────────────────────────────────────
#   cd /path/to/your/paca/install
#   curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/upgrade.sh -o upgrade.sh
#   bash upgrade.sh
#
# ── One-liner (non-interactive, upgrades to latest) ───────────────────────────
#   bash <(curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/upgrade.sh)
#
# ── Environment variable overrides ───────────────────────────────────────────
#   PACA_DIR         Installation directory to upgrade   (default: .)
#   PACA_VERSION     Release tag to upgrade to           (default: latest)
#   PACA_YES         Skip prompts, use defaults          (set to 1)
#
# Extra arguments are passed through to the final `docker compose up -d`,
# e.g. to keep the same service scaling you used originally:
#   bash upgrade.sh --scale web=0 --scale minio=0

set -euo pipefail

# ── Colours ───────────────────────────────────────────────────────────────────

BOLD='\033[1m'; DIM='\033[2m'
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'
RESET='\033[0m'

info()    { echo -e "${GREEN}✔${RESET}  $*"; }
warn()    { echo -e "${YELLOW}!${RESET}  $*"; }
error()   { echo -e "${RED}✖${RESET}  $*" >&2; }
die()     { error "$*"; exit 1; }
heading() { echo -e "\n${BOLD}${CYAN}── $* ${RESET}${DIM}$(printf '─%.0s' {1..40})${RESET}"; }
bold()    { echo -e "${BOLD}$*${RESET}"; }

# ── Helpers ───────────────────────────────────────────────────────────────────

# ask VAR "Question" "default"
# Reads from /dev/tty when stdin is a pipe (curl | bash).
ask() {
    local _var="$1"
    local question="$2"
    local default="${3:-}"
    local prompt

    if [[ -n "$default" ]]; then
        prompt="${BOLD}→${RESET} ${question} ${DIM}[${default}]${RESET}: "
    else
        prompt="${BOLD}→${RESET} ${question}: "
    fi

    local _input=""
    if [[ "${PACA_YES:-0}" == "1" ]]; then
        printf -v "$_var" %s "${default}"
        return
    fi
    if [[ -t 0 ]]; then
        read -r -p "$(echo -e "$prompt")" _input
    elif [[ -e /dev/tty ]]; then
        read -r -p "$(echo -e "$prompt")" _input </dev/tty
    else
        printf -v "$_var" %s "${default}"
        return
    fi
    printf -v "$_var" %s "${_input:-$default}"
}

# yes_no VAR "Question" "y|n"
yes_no() {
    local _var="$1"
    local question="$2"
    local default="${3:-y}"
    local answer=""
    ask answer "$question" "$default"
    local answer_lower
    answer_lower="$(printf '%s' "$answer" | tr '[:upper:]' '[:lower:]')"
    case "$answer_lower" in
        y|yes) printf -v "$_var" %s "yes" ;;
        *)     printf -v "$_var" %s "no"  ;;
    esac
}

# download URL DEST
download() {
    local url="$1" dest="$2"
    if command -v curl &>/dev/null; then
        curl -fsSL --retry 3 "$url" -o "$dest"
    elif command -v wget &>/dev/null; then
        wget -q --tries=3 -O "$dest" "$url"
    else
        die "Neither curl nor wget found. Install one and retry."
    fi
}

# set_env_var FILE VAR VALUE
# Replaces an existing "VAR=..." line in FILE, or appends it if absent.
# Goes through a temp file rather than `sed -i` to avoid GNU/BSD differences.
set_env_var() {
    local file="$1" var="$2" value="$3" tmp
    tmp="$(mktemp)"
    if grep -q "^${var}=" "$file" 2>/dev/null; then
        awk -v var="$var" -v val="$value" -F= '
            $1 == var { print var "=" val; next }
            { print }
        ' "$file" > "$tmp"
    else
        cp "$file" "$tmp"
        printf '%s=%s\n' "$var" "$value" >> "$tmp"
    fi
    mv "$tmp" "$file"
}

# has_env_var FILE VAR
has_env_var() {
    grep -q "^${2}=" "$1" 2>/dev/null
}

# get_env_var FILE VAR
get_env_var() {
    grep "^${2}=" "$1" 2>/dev/null | head -1 | cut -d= -f2-
}

# ── Version / URL resolution ──────────────────────────────────────────────────

PACA_VERSION="${PACA_VERSION:-latest}"

if [[ "$PACA_VERSION" == "latest" ]]; then
    RELEASE_BASE="https://github.com/Paca-AI/paca/releases/latest/download"
else
    RELEASE_BASE="https://github.com/Paca-AI/paca/releases/download/${PACA_VERSION}"
fi

# Strip leading 'v' for Docker image tags (v1.2.3 → 1.2.3).
IMAGE_TAG="${PACA_VERSION#v}"

# ── Preflight ─────────────────────────────────────────────────────────────────

echo ""
bold "╔══════════════════════════════════════════════════════════╗"
bold "║         Paca  –  upgrade an existing installation        ║"
bold "╚══════════════════════════════════════════════════════════╝"
echo ""

if ! command -v docker &>/dev/null; then
    die "Docker is not installed. Get it at https://docs.docker.com/get-docker/"
fi
if ! docker info &>/dev/null 2>&1; then
    die "Docker daemon is not running. Start Docker Desktop (or the daemon) and retry."
fi

COMPOSE_CMD=""
if docker compose version &>/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
elif command -v docker-compose &>/dev/null; then
    COMPOSE_CMD="docker-compose"
else
    die "Docker Compose not found. Install it from https://docs.docker.com/compose/install/"
fi

info "Docker OK  (compose: $COMPOSE_CMD)"

PACA_DIR="${PACA_DIR:-.}"
cd "${PACA_DIR}"
info "Installation directory: $(pwd)"

if [[ ! -f docker-compose.yml || ! -f .env ]]; then
    die "No existing Paca installation found here (missing docker-compose.yml and/or .env). Use install.sh for a fresh install, or set PACA_DIR to point at your install directory."
fi

# ── Current vs target version ─────────────────────────────────────────────────

heading "Version"

CURRENT_TAG="unknown"
if grep -q "^PACA_API_IMAGE=" .env; then
    CURRENT_TAG="$(grep "^PACA_API_IMAGE=" .env | head -1 | sed -e 's/^PACA_API_IMAGE=//' -e 's/.*://')"
fi

info "Current version: ${CURRENT_TAG}"
info "Target version:  ${IMAGE_TAG}"

if [[ -f nginx/gateway.conf ]]; then
    heading "Gateway migration"
    warn "This installation predates the nginx → Caddy gateway migration."
    info "caddy/Caddyfile will be downloaded; nginx/ is no longer referenced by docker-compose.yml."
    info "SITE_ADDRESS and GATEWAY_HTTPS_PORT will be added to .env so Caddy can serve HTTPS automatically."
    info "Once you've confirmed the upgraded stack works, the nginx/ directory can be removed."
fi

PROCEED="yes"
yes_no PROCEED "Proceed with upgrade?" "y"
if [[ "$PROCEED" != "yes" ]]; then
    warn "Upgrade cancelled."
    exit 0
fi

mkdir -p caddy

# ── Backup and refresh infrastructure files ───────────────────────────────────

heading "Backing up and refreshing infrastructure files"

TS="$(date +%s)"

cp docker-compose.yml "docker-compose.yml.bak.${TS}"
info "Backed up docker-compose.yml → docker-compose.yml.bak.${TS}"
download "${RELEASE_BASE}/docker-compose.yml" docker-compose.yml
info "Downloaded the latest docker-compose.yml."

if [[ -f caddy/Caddyfile ]]; then
    cp caddy/Caddyfile "caddy/Caddyfile.bak.${TS}"
    info "Backed up caddy/Caddyfile → caddy/Caddyfile.bak.${TS}"
fi
download "${RELEASE_BASE}/Caddyfile" caddy/Caddyfile
info "Downloaded the latest caddy/Caddyfile."

ENV_BACKED_UP=0
backup_env_once() {
    if [[ "$ENV_BACKED_UP" == "0" ]]; then
        cp .env ".env.bak.${TS}"
        info "Backed up .env → .env.bak.${TS}"
        ENV_BACKED_UP=1
    fi
}

# Only re-pin image tags in .env when a specific version was requested.
# Installs left on the default ":latest" floating tag are already upgraded
# by the pull below — rewriting them here would silently switch a
# deliberately-pinned install onto floating tags, or vice versa.
if [[ "$PACA_VERSION" != "latest" ]]; then
    backup_env_once
    for var in PACA_API_IMAGE PACA_WEB_IMAGE PACA_REALTIME_IMAGE PACA_AI_AGENT_IMAGE; do
        image_name="$(echo "$var" | sed -e 's/^PACA_//' -e 's/_IMAGE$//' | tr '[:upper:]' '[:lower:]' | sed 's/_/-/g')"
        set_env_var .env "$var" "pacaai/paca-${image_name}:${IMAGE_TAG}"
    done
    info "Pinned image versions in .env to ${IMAGE_TAG}."
else
    info "Using floating :latest images — no image version changes needed."
fi

# Backfill variables introduced by the nginx → Caddy gateway migration.
# Installations from before that release have neither in .env. SITE_ADDRESS
# defaults to the hostname already in PUBLIC_URL so Caddy requests a
# certificate for the address Paca is actually reachable at, rather than
# silently leaving the upgraded gateway on plain HTTP.
GATEWAY_VARS_ADDED=0
if ! has_env_var .env SITE_ADDRESS; then
    backup_env_once
    _PUBLIC_URL="$(get_env_var .env PUBLIC_URL)"
    _SITE_ADDRESS="${_PUBLIC_URL#http://}"
    _SITE_ADDRESS="${_SITE_ADDRESS#https://}"
    _SITE_ADDRESS="${_SITE_ADDRESS%%/*}"
    _SITE_ADDRESS="${_SITE_ADDRESS%%:*}"
    _SITE_ADDRESS="${_SITE_ADDRESS:-localhost}"
    set_env_var .env SITE_ADDRESS "$_SITE_ADDRESS"
    info "Added SITE_ADDRESS=${_SITE_ADDRESS} to .env (derived from your existing PUBLIC_URL)."
    GATEWAY_VARS_ADDED=1
fi
if ! has_env_var .env GATEWAY_HTTPS_PORT; then
    backup_env_once
    set_env_var .env GATEWAY_HTTPS_PORT "443"
    info "Added GATEWAY_HTTPS_PORT=443 to .env."
    GATEWAY_VARS_ADDED=1
fi
if [[ "$GATEWAY_VARS_ADDED" == "1" ]]; then
    warn "Ports 80 and 443 must both be reachable from the internet for Let's Encrypt to succeed."
    info "Already behind another TLS terminator (a load balancer, Cloudflare, etc.)? Set SITE_ADDRESS=:80 in .env to keep this gateway on plain HTTP."
fi

# Backfill variables for the db-backup service introduced after this install
# was created. Installations from before that release have neither in .env.
# Only relevant for the bundled postgres container — a non-blank DATABASE_URL
# means an external/managed database is in use, which is assumed to already
# have its own backup mechanism (mirrors the bundled-vs-external check in
# install.sh).
SCALE_OPTS=()
if [[ -z "$(get_env_var .env DATABASE_URL)" ]]; then
    if ! has_env_var .env BACKUP_DIR; then
        backup_env_once
        set_env_var .env BACKUP_DIR "./backups"
        info "Added BACKUP_DIR=./backups to .env."
    fi
    if ! has_env_var .env BACKUP_RETENTION_DAYS; then
        backup_env_once
        set_env_var .env BACKUP_RETENTION_DAYS "7"
        info "Added BACKUP_RETENTION_DAYS=7 to .env."
    fi
    if ! has_env_var .env BACKUP_CRON; then
        backup_env_once
        set_env_var .env BACKUP_CRON "0 2 * * *"
        info "Added BACKUP_CRON=0 2 * * * to .env (runs daily at 02:00 UTC)."
    fi
    _BACKUP_DIR="$(get_env_var .env BACKUP_DIR)"
    _BACKUP_DIR="${_BACKUP_DIR:-./backups}"
    mkdir -p "$_BACKUP_DIR"
    warn "A new db-backup service now writes a daily database dump to ${_BACKUP_DIR}. Disable with --scale db-backup=0 if you already back up this database elsewhere."
else
    SCALE_OPTS+=(--scale db-backup=0)
    info "Using an external database (DATABASE_URL is set) — skipping the automated db-backup service."
fi

# ── Pull and restart ──────────────────────────────────────────────────────────

heading "Pulling images and restarting"

# shellcheck disable=SC2086
$COMPOSE_CMD --env-file .env pull
# shellcheck disable=SC2086
$COMPOSE_CMD --env-file .env up -d --remove-orphans ${SCALE_OPTS[@]+"${SCALE_OPTS[@]}"} "$@"

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
bold "╔══════════════════════════════════════════════════════════╗"
bold "║              Paca has been upgraded!                     ║"
bold "╚══════════════════════════════════════════════════════════╝"
echo ""
info "Version: ${IMAGE_TAG}"
echo ""
echo -e "${DIM}Database migrations run automatically on API startup.${RESET}"
echo ""
echo -e "  ${BOLD}Check status:${RESET}  ${COMPOSE_CMD} --env-file .env ps"
echo -e "  ${BOLD}View logs:${RESET}     ${COMPOSE_CMD} --env-file .env logs -f"
echo ""
