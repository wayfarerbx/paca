# ── Production Dockerfile for services/realtime ──────────────────────────────
#
# Multi-stage build:
#   1. deps  — install production dependencies with a frozen lockfile.
#   2. final — copy only what is needed to keep the image lean.
#
# Bun runs TypeScript natively so no compile step is required.

FROM oven/bun:1-alpine AS deps
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --production

# ── Final image ───────────────────────────────────────────────────────────────

FROM oven/bun:1-alpine
WORKDIR /app

# Non-root user supplied by the base image.
USER bun

COPY --from=deps /app/node_modules node_modules
COPY package.json ./
COPY src/ src/

EXPOSE 3001
ENV PORT=3001 \
    NODE_ENV=production

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO/dev/null --tries=1 http://127.0.0.1:${PORT}/healthz || exit 1

CMD ["bun", "run", "src/index.ts"]
