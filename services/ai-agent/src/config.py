from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    # Service
    port: int = 8080
    log_level: str = "INFO"

    # Valkey / Redis
    valkey_url: str = "redis://valkey:6379/0"

    # Database
    database_url: str

    # Service-to-service
    internal_api_key: str
    api_base_url: str = "http://api:8080"

    # Built-in Paca MCP — the API key used by the AI agent's hardcoded paca MCP
    # server.  Set this to the same value as AGENT_API_KEY on the api service.
    # When empty the built-in paca MCP server is not injected.
    paca_api_key: str = ""

    # Docker
    docker_socket: str = "/var/run/docker.sock"
    agent_server_image: str = "ghcr.io/openhands/agent-server:latest-python"
    conversation_persistence_root: str = "/data/conversations"

    # Port pool for Docker containers
    port_pool_start: int = 10000
    port_pool_size: int = 100

    # Worker
    worker_concurrency: int = 10


settings = Settings()
