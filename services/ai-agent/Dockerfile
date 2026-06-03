# Stage 1: Build the Paca MCP server (Node.js ESM bundle).
# The build context for this Dockerfile is the paca/ root so that both
# services/ai-agent/ and apps/mcp/ are accessible.
FROM node:22-alpine AS mcp-builder
WORKDIR /mcp-src
COPY apps/mcp/package*.json ./
RUN npm ci
COPY apps/mcp/ ./
RUN npm run build

# Stage 2: Production AI-agent image.
FROM python:3.12-slim

# Install uv
RUN pip install uv

WORKDIR /app

COPY services/ai-agent/pyproject.toml .
RUN uv pip install --system -e .

COPY services/ai-agent/src/ ./src/

# Copy the pre-built MCP bundle so the agent can launch it via
# `node /mcp/build/index.js` without an external volume mount.
COPY --from=mcp-builder /mcp-src/build /mcp/build
COPY --from=mcp-builder /mcp-src/package.json /mcp/package.json

ENV PYTHONPATH=/app
CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8080"]
