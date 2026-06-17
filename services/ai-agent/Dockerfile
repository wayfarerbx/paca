FROM python:3.12-slim

# Install uv
RUN pip install uv

WORKDIR /app

COPY pyproject.toml .
RUN uv pip install --system -e .

COPY src/ ./src/
COPY data/ ./data/

ENV PYTHONPATH=/app
CMD ["uvicorn", "src.main:app", "--host", "0.0.0.0", "--port", "8080"]
