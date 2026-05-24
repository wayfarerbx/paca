FROM python:3.12-slim

# Install Node.js via NodeSource (includes npx, required for MCP servers)
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && node --version && npm --version && npx --version

# Install uv
RUN pip install uv

WORKDIR /app

COPY pyproject.toml .
RUN uv pip install --system -e .

COPY src/ ./src/

ENV PYTHONPATH=/app
CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8080"]
