#!/usr/bin/env bash
# Paca – interactive install script
#
# Downloads the pre-built release artifacts and walks you through setup.
#
# ── Recommended (interactive) ────────────────────────────────────────────────
#   curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/install.sh -o install.sh
#   bash install.sh
#
# ── One-liner (non-interactive, all defaults + auto-generated secrets) ────────
#   bash <(curl -fsSL https://github.com/Paca-AI/paca/releases/latest/download/install.sh)
#
# ── Environment variable overrides ───────────────────────────────────────────
#   PACA_DIR         Installation directory      (default: ./paca)
#   PACA_VERSION     Release tag to install      (default: latest)
#   PACA_YES         Skip prompts, use defaults  (set to 1)

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

# head -c closes the pipe before tr finishes, causing tr to exit with SIGPIPE
# (status 141). pipefail would propagate that non-zero status and kill the
# script. Run each pipeline in a subshell with pipefail disabled so the exit
# code is taken from head (always 0) rather than tr.
rand_hex()    { ( set +o pipefail; LC_ALL=C tr -dc 'a-f0-9'    </dev/urandom | head -c "${1:-32}"; ); }
rand_alnum()  { ( set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c "${1:-24}"; ); }

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

# ask_secret VAR "Question"  (no echo, no default shown)
ask_secret() {
    local _var="$1"
    local question="$2"
    local _input=""
    local prompt="${BOLD}→${RESET} ${question} ${DIM}(hidden)${RESET}: "

    if [[ "${PACA_YES:-0}" == "1" ]]; then
        printf -v "$_var" %s ""
        return
    fi
    if [[ -t 0 ]]; then
        read -r -s -p "$(echo -e "$prompt")" _input; echo
    elif [[ -e /dev/tty ]]; then
        read -r -s -p "$(echo -e "$prompt")" _input </dev/tty; echo
    fi
    printf -v "$_var" %s "$_input"
}

# ask_choice VAR "Question" option1 option2 ...
# Returns the chosen option string.
ask_choice() {
    local _var="$1"
    local question="$2"
    shift 2
    local options=("$@")
    local i

    echo -e "\n${question}"
    for i in "${!options[@]}"; do
        echo -e "  ${BOLD}$((i+1))${RESET}) ${options[$i]}"
    done

    local choice=""
    ask choice "Choice" "1"
    local idx=$(( choice - 1 ))
    if (( idx < 0 || idx >= ${#options[@]} )); then
        warn "Invalid choice, using default (1)"
        idx=0
    fi
    printf -v "$_var" %s "${options[$idx]}"
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
bold "║         Paca  –  open-source AI-native project mgmt     ║"
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

# ── Installation directory ────────────────────────────────────────────────────

heading "Installation directory"

PACA_DIR="${PACA_DIR:-./paca}"
ask PACA_DIR "Where should Paca be installed?" "$PACA_DIR"

mkdir -p "${PACA_DIR}/nginx"
cd "${PACA_DIR}"
info "Working directory: $(pwd)"

# ── Admin credentials ─────────────────────────────────────────────────────────

heading "Admin account"

ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ask ADMIN_USERNAME "Admin username" "$ADMIN_USERNAME"

_GENERATED_PW="$(rand_alnum 16)"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
ask_secret ADMIN_PASSWORD "Admin password (leave blank to auto-generate)"
if [[ -z "$ADMIN_PASSWORD" ]]; then
    ADMIN_PASSWORD="$_GENERATED_PW"
    ADMIN_PASSWORD_GENERATED=1
else
    ADMIN_PASSWORD_GENERATED=0
fi

# ── Encryption key ────────────────────────────────────────────────────────────

heading "Encryption key"

echo "  This key encrypts plugin secrets (OAuth tokens, API keys) stored in the"
echo "  database. If you are connecting to an existing Paca database you MUST"
echo "  supply the original key — a different key makes all existing encrypted"
echo "  values permanently unreadable."
echo ""

ENCRYPTION_KEY=""
ENCRYPTION_KEY_GENERATED=0
ask_secret ENCRYPTION_KEY "Encryption key — 64-char hex (leave blank to generate)"

if [[ -z "$ENCRYPTION_KEY" ]]; then
    ENCRYPTION_KEY="$(rand_hex 64)"
    ENCRYPTION_KEY_GENERATED=1
    info "Encryption key generated."
else
    if [[ ! "$ENCRYPTION_KEY" =~ ^[a-f0-9]{64}$ ]]; then
        die "Invalid encryption key: must be exactly 64 lowercase hex characters (32 bytes). Generate one with: openssl rand -hex 32"
    fi
    info "Using the provided encryption key."
fi

# ── Database ──────────────────────────────────────────────────────────────────

heading "Database"

DB_CHOICE=""
ask_choice DB_CHOICE "How should Paca store data?" \
    "Bundled PostgreSQL container (recommended)" \
    "External / managed PostgreSQL (bring your own)"

SCALE_POSTGRES=""
DATABASE_URL_OVERRIDE=""
POSTGRES_PASSWORD_VALUE=""

if [[ "$DB_CHOICE" == *"External"* ]]; then
    SCALE_POSTGRES="--scale postgres=0"
    ask DATABASE_URL_OVERRIDE "PostgreSQL connection URL" "postgres://user:pass@host:5432/dbname"
    # Set a placeholder so the compose's ${POSTGRES_PASSWORD:-changeme} default doesn't matter.
    POSTGRES_PASSWORD_VALUE="not-used-external-db"
    info "External PostgreSQL will be used."
else
    POSTGRES_PASSWORD_VALUE="$(rand_alnum 20)"
    info "A bundled PostgreSQL container will be started."
fi

# ── Object storage ────────────────────────────────────────────────────────────

heading "Object storage"

STORAGE_CHOICE=""
ask_choice STORAGE_CHOICE "Where should file attachments be stored?" \
    "Self-hosted MinIO (recommended, no cloud account needed)" \
    "AWS S3"

SCALE_MINIO=""
STORAGE_PROVIDER="minio"
STORAGE_ENDPOINT="minio:9000"
STORAGE_USE_SSL="false"
STORAGE_REGION="us-east-1"
STORAGE_BUCKET="paca"
STORAGE_ACCESS_KEY_ID="$(rand_alnum 16)"
STORAGE_SECRET_ACCESS_KEY="$(rand_alnum 32)"

if [[ "$STORAGE_CHOICE" == *"AWS"* ]]; then
    SCALE_MINIO="--scale minio=0"
    STORAGE_PROVIDER="s3"
    STORAGE_ENDPOINT=""
    STORAGE_USE_SSL="true"

    ask STORAGE_REGION    "AWS region"          "us-east-1"
    ask STORAGE_BUCKET    "S3 bucket name"      ""
    ask STORAGE_ACCESS_KEY_ID    "AWS access key ID"     ""
    ask_secret STORAGE_SECRET_ACCESS_KEY "AWS secret access key"

    if [[ -z "$STORAGE_BUCKET" ]]; then
        die "S3 bucket name is required."
    fi
    if [[ -z "$STORAGE_ACCESS_KEY_ID" || -z "$STORAGE_SECRET_ACCESS_KEY" ]]; then
        die "AWS credentials are required."
    fi
    info "AWS S3 will be used (bucket: ${STORAGE_BUCKET}, region: ${STORAGE_REGION})."
else
    info "Self-hosted MinIO will be started."
fi

# ── Network ───────────────────────────────────────────────────────────────────

heading "Network"

GATEWAY_PORT="80"
ask GATEWAY_PORT "Gateway port (the port Paca will be accessible on)" "80"

# Derive a sensible default public URL from the port.
if [[ "$GATEWAY_PORT" == "80" ]]; then
    _DEFAULT_PUBLIC_URL="http://localhost"
elif [[ "$GATEWAY_PORT" == "443" ]]; then
    _DEFAULT_PUBLIC_URL="https://localhost"
else
    _DEFAULT_PUBLIC_URL="http://localhost:${GATEWAY_PORT}"
fi

PUBLIC_URL=""
ask PUBLIC_URL "Public URL (full URL where Paca will be accessible, no trailing slash)" "$_DEFAULT_PUBLIC_URL"
PUBLIC_URL="${PUBLIC_URL%/}"  # strip trailing slash

# Set COOKIE_SECURE based on whether the URL uses HTTPS.
if [[ "$PUBLIC_URL" == https://* ]]; then
    COOKIE_SECURE="true"
else
    COOKIE_SECURE="false"
fi

# Compute storage public URL only for MinIO (S3 presigned URLs are self-contained).
if [[ "$STORAGE_PROVIDER" == "minio" ]]; then
    STORAGE_PUBLIC_URL="${PUBLIC_URL}/storage"
else
    STORAGE_PUBLIC_URL=""
fi

# ── Web app ───────────────────────────────────────────────────────────────────

heading "Web application"

WEB_CHOICE=""
ask_choice WEB_CHOICE "How do you want to serve the web app?" \
    "Bundled container (recommended – nginx serves the built React SPA)" \
    "External hosting (S3, CloudFront, Vercel, etc. – only API services run here)"

SCALE_WEB=""
if [[ "$WEB_CHOICE" == *"External"* ]]; then
    SCALE_WEB="--scale web=0"
    echo ""
    warn "The web container will be skipped."
    warn "Build the SPA from source and deploy the dist/ folder to your CDN."
    warn "Point your CDN's API proxy to: ${_DEFAULT_PUBLIC_URL:-http://localhost}/api"
    echo ""
    info "The gateway will still serve /api/, /ws/, and /storage/ routes."
else
    info "Bundled web container will be started."
fi

# ── AI Agent ──────────────────────────────────────────────────────────────────

heading "AI Agent (optional)"

echo "  The AI agent enables autonomous task execution."
echo "  It requires access to the Docker socket on the host machine."
echo ""

INCLUDE_AI_AGENT="yes"
yes_no INCLUDE_AI_AGENT "Include the AI agent service?" "y"

SCALE_AI_AGENT=""
AGENT_API_KEY="$(rand_hex 32)"
INTERNAL_API_KEY="$(rand_hex 32)"

if [[ "$INCLUDE_AI_AGENT" == "no" ]]; then
    SCALE_AI_AGENT="--scale ai-agent=0"
    info "AI agent will be skipped."
else
    info "AI agent will be included."
fi

# ── Download release assets ───────────────────────────────────────────────────

heading "Downloading release assets"

if [[ -f docker-compose.yml ]]; then
    warn "docker-compose.yml already exists — skipping download."
else
    info "Downloading docker-compose.yml..."
    download "${RELEASE_BASE}/docker-compose.yml" docker-compose.yml
fi

if [[ -f nginx/gateway.conf ]]; then
    warn "nginx/gateway.conf already exists — skipping download."
else
    info "Downloading nginx/gateway.conf..."
    download "${RELEASE_BASE}/gateway.conf" nginx/gateway.conf
fi

# ── Generate .env ─────────────────────────────────────────────────────────────

heading "Generating .env"

if [[ -f .env ]]; then
    warn ".env already exists."
    KEEP_ENV="yes"
    yes_no KEEP_ENV "Keep existing .env?" "y"
    if [[ "$KEEP_ENV" == "yes" ]]; then
        warn "Keeping existing .env. Delete it and re-run to regenerate."
    else
        mv .env ".env.bak.$(date +%s)"
        warn "Old .env backed up."
        KEEP_ENV="no"
    fi
else
    KEEP_ENV="no"
fi

if [[ "$KEEP_ENV" == "no" ]]; then
    JWT_SECRET="$(rand_hex 32)"

    cat >.env <<EOF
# ── Paca environment ──────────────────────────────────────────────────────────
# Generated by install.sh  $(date -u '+%Y-%m-%dT%H:%M:%SZ')
#
# To reconfigure: edit this file, then run:
#   ${COMPOSE_CMD} --env-file .env up -d
# ─────────────────────────────────────────────────────────────────────────────

# ── Image versions ────────────────────────────────────────────────────────────
PACA_API_IMAGE=pacaai/paca-api:${IMAGE_TAG}
PACA_WEB_IMAGE=pacaai/paca-web:${IMAGE_TAG}
PACA_REALTIME_IMAGE=pacaai/paca-realtime:${IMAGE_TAG}
PACA_AI_AGENT_IMAGE=pacaai/paca-ai-agent:${IMAGE_TAG}

ENVIRONMENT=production
GATEWAY_PORT=${GATEWAY_PORT}

# ── Public URL ────────────────────────────────────────────────────────────────
PUBLIC_URL=${PUBLIC_URL}

# ── Admin credentials ────────────────────────────────────────────────────────
ADMIN_USERNAME=${ADMIN_USERNAME}
ADMIN_PASSWORD=${ADMIN_PASSWORD}

# ── JWT ───────────────────────────────────────────────────────────────────────
JWT_SECRET=${JWT_SECRET}
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
JWT_REFRESH_SESSION_TTL=24h
COOKIE_SECURE=${COOKIE_SECURE}

# ── Database ─────────────────────────────────────────────────────────────────
POSTGRES_DB=paca
POSTGRES_USER=paca
POSTGRES_PASSWORD=${POSTGRES_PASSWORD_VALUE}
# Override: leave blank to use the bundled postgres container above.
DATABASE_URL=${DATABASE_URL_OVERRIDE}

# ── Cache (Valkey) ────────────────────────────────────────────────────────────
# Leave blank to use the bundled Valkey container.
REDIS_URL=

# ── Object storage ────────────────────────────────────────────────────────────
STORAGE_PROVIDER=${STORAGE_PROVIDER}
STORAGE_ENDPOINT=${STORAGE_ENDPOINT}
STORAGE_PUBLIC_URL=${STORAGE_PUBLIC_URL}
STORAGE_REGION=${STORAGE_REGION}
STORAGE_BUCKET=${STORAGE_BUCKET}
STORAGE_ACCESS_KEY_ID=${STORAGE_ACCESS_KEY_ID}
STORAGE_SECRET_ACCESS_KEY=${STORAGE_SECRET_ACCESS_KEY}
STORAGE_USE_SSL=${STORAGE_USE_SSL}

# ── Encryption ────────────────────────────────────────────────────────────────
# 64-char hex string used to encrypt plugin secrets at rest.
ENCRYPTION_KEY=${ENCRYPTION_KEY}

# ── AI Agent ─────────────────────────────────────────────────────────────────
# Both keys below must match across api and ai-agent services.
AGENT_API_KEY=${AGENT_API_KEY}
INTERNAL_API_KEY=${INTERNAL_API_KEY}
AI_AGENT_PORT=8082
AGENT_SERVER_IMAGE=ghcr.io/openhands/agent-server:latest-python
PORT_POOL_START=10000
PORT_POOL_SIZE=100
WORKER_CONCURRENCY=10

# ── Logging ───────────────────────────────────────────────────────────────────
LOG_LEVEL=info
EOF
    info ".env written."
fi

# ── Confirm and start ─────────────────────────────────────────────────────────

heading "Summary"

echo ""
echo -e "  ${BOLD}Directory   ${RESET}$(pwd)"
echo -e "  ${BOLD}Version     ${RESET}${PACA_VERSION}"
echo -e "  ${BOLD}Public URL  ${RESET}${PUBLIC_URL}"
echo -e "  ${BOLD}Database    ${RESET}$( [[ -n "$SCALE_POSTGRES" ]] && echo "External PostgreSQL" || echo "Bundled PostgreSQL container" )"
echo -e "  ${BOLD}Storage     ${RESET}$( [[ "$STORAGE_PROVIDER" == "s3" ]] && echo "AWS S3 (${STORAGE_BUCKET})" || echo "Self-hosted MinIO" )"
echo -e "  ${BOLD}Web app     ${RESET}$( [[ -n "$SCALE_WEB" ]] && echo "External / CDN (container skipped)" || echo "Bundled container" )"
echo -e "  ${BOLD}AI Agent    ${RESET}$( [[ -n "$SCALE_AI_AGENT" ]] && echo "Disabled" || echo "Enabled" )"
echo -e "  ${BOLD}Admin user  ${RESET}${ADMIN_USERNAME}"
echo ""

START="yes"
yes_no START "Pull images and start Paca now?" "y"

if [[ "$START" != "yes" ]]; then
    echo ""
    warn "Installation files are ready. Start Paca manually with:"
    echo ""
    bold "  cd $(pwd)"
    bold "  ${COMPOSE_CMD} --env-file .env up -d ${SCALE_POSTGRES} ${SCALE_MINIO} ${SCALE_WEB} ${SCALE_AI_AGENT} --pull always"
    echo ""
    exit 0
fi

# ── Start the stack ───────────────────────────────────────────────────────────

heading "Starting Paca"

SCALE_OPTS=()
[[ -n "$SCALE_POSTGRES"  ]] && SCALE_OPTS+=($SCALE_POSTGRES)
[[ -n "$SCALE_MINIO"     ]] && SCALE_OPTS+=($SCALE_MINIO)
[[ -n "$SCALE_WEB"       ]] && SCALE_OPTS+=($SCALE_WEB)
[[ -n "$SCALE_AI_AGENT"  ]] && SCALE_OPTS+=($SCALE_AI_AGENT)

# shellcheck disable=SC2086
$COMPOSE_CMD --env-file .env up -d ${SCALE_OPTS[@]+"${SCALE_OPTS[@]}"} --pull always

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
bold "╔══════════════════════════════════════════════════════════╗"
bold "║               Paca is starting up!                      ║"
bold "╚══════════════════════════════════════════════════════════╝"
echo ""
info "Web UI:     ${PUBLIC_URL}"
info "Admin user: ${ADMIN_USERNAME}"

if [[ "$ADMIN_PASSWORD_GENERATED" == "1" ]]; then
    echo ""
    warn "Your admin password was auto-generated. Save it now:"
    bold "    ADMIN_PASSWORD=${ADMIN_PASSWORD}"
    echo ""
    warn "It is also stored in $(pwd)/.env"
fi

if [[ "$ENCRYPTION_KEY_GENERATED" == "1" ]]; then
    echo ""
    warn "Your encryption key was auto-generated. Back it up — you will need"
    warn "this exact key to access encrypted data if you ever migrate or restore"
    warn "the database:"
    bold "    ENCRYPTION_KEY=${ENCRYPTION_KEY}"
    echo ""
    warn "It is also stored in $(pwd)/.env"
fi

echo ""
echo -e "${DIM}Services may take up to a minute to become healthy.${RESET}"
echo ""
echo -e "  ${BOLD}Check status:${RESET}  ${COMPOSE_CMD} --env-file .env ps"
echo -e "  ${BOLD}View logs:${RESET}     ${COMPOSE_CMD} --env-file .env logs -f"
echo -e "  ${BOLD}Stop:${RESET}          ${COMPOSE_CMD} --env-file .env down"
echo ""
