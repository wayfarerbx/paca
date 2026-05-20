FROM python:3.12-slim

# Install Node.js (required for the built-in paca MCP server via npx)
RUN apt-get update && apt-get install -y --no-install-recommends nodejs npm \
    && rm -rf /var/lib/apt/lists/*

# Install uv
RUN pip install uv

WORKDIR /app

COPY pyproject.toml .
RUN uv pip install --system -e .

COPY src/ ./src/

ENV PYTHONPATH=/app
CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8080"]
