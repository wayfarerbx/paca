# Realtime Service

Socket.IO server that fans out real-time events to web clients by subscribing
to the `paca.events` Valkey Pub/Sub channel published by `services/api`.

## Responsibilities

- Accept authenticated Socket.IO client connections (JWT via cookie or handshake auth).
- Delegate access-token verification to `services/api` (`GET /api/v1/users/me/global-permissions`) — no local JWT validation.
- Subscribe to the `paca.events` Valkey Pub/Sub channel.
- Route incoming events to namespace-scoped Socket.IO rooms (`project:<projectId>:tasks`, `project:<projectId>:docs`).
- Expose a `/healthz` endpoint for container health checks.

## Stack

| Concern       | Tool                        |
|---------------|-----------------------------|
| Runtime       | [Bun](https://bun.sh) 1.x   |
| Language      | TypeScript (strict)         |
| WebSockets    | [Socket.IO](https://socket.io) 4.x |
| Valkey client | [ioredis](https://github.com/redis/ioredis) 5.x |
| Logging       | [pino](https://getpino.io)  |

## Local development

### Outside Docker (fastest feedback loop)

```sh
# 1. Copy environment file and adjust values if needed.
cp .env.example .env

# 2. Install dependencies.
bun install

# 3. Start with hot-reload (requires Valkey running locally on port 6379).
bun run dev
```

### Via Docker Compose (full stack)

```sh
# From the repo root — starts postgres, valkey, api, web, realtime, and gateway.
docker compose -f deploy/docker-compose.dev.yml up -d
```

The service is reachable through the nginx gateway at `http://localhost/ws/`.

## Environment variables

| Variable       | Required | Default               | Description                                      |
|----------------|----------|-----------------------|--------------------------------------------------|
| `PORT`         | No       | `3001`                | HTTP port the server listens on.                 |
| `REDIS_URL`    | **Yes**  | —                     | Valkey/Redis connection URL (same as API).       |
| `API_URL`      | **Yes**  | —                     | Internal base URL of `services/api` used for token verification and permissions. |
| `CORS_ORIGINS` | No       | `http://localhost:3000` | Comma-separated allowed origins.               |
| `LOG_LEVEL`    | No       | `info`                | Pino log level (`trace`…`fatal`).                |
| `NODE_ENV`     | No       | `development`         | `production` enables structured JSON logging.    |

## Client-side connection

```ts
import { io } from "socket.io-client";

const socket = io("http://localhost", {
  path: "/ws/socket.io",   // nginx strips the /ws prefix
  withCredentials: true,   // sends the access_token cookie automatically
});

// Join a project room to receive scoped events.
socket.emit("join", { projectId: "<uuid>" });

// Listen for API events.
socket.on("event", ({ type, payload }) => {
  console.log(type, payload);
});
```

## Event format

Every event received from Valkey (and re-emitted to Socket.IO clients) follows
the shape published by `services/api`:

```json
{
  "type": "task.comment.added",
  "payload": {
    "id": "...",
    "task_id": "...",
    "project_id": "...",
    "activity_type": "...",
    "content": "...",
    "actor_id": "...",
    "created_at": "...",
    "updated_at": "..."
  }
}
```

Events with a `project_id` field are delivered only to sockets that joined the
corresponding `project:<id>` room.

## Linting & type checking

```sh
bun run lint    # biome check
bun run check   # tsc --noEmit
```
