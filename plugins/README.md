# Plugins Directory

This directory stores **local plugins** for development and self-hosted deployments.

## Structure

```
plugins/
├── local/              # Local plugin store (mounted to containers)
│   ├── backend/        # Backend WASM binaries, migrations, manifests
│   └── frontend/       # Frontend JS/CSS bundles
└── README.md           # This file
```

## Local Plugin Store

The `local/` directory is the **local plugin store** where you install plugins for development or self-hosted deployments.

### Directory Layout

```
plugins/local/
  backend/
    <plugin-id>/
      plugin.json          ← plugin manifest
      backend.wasm         ← compiled WASM binary
      migrations/
        0001_*.sql
  frontend/
    <plugin-id>/
      assets/
        remoteEntry.js     ← module-federation entry point
        ...                ← other hashed JS/CSS chunks
```

### Mount Points

| Sub-directory | Mounted to | Purpose |
|---|---|---|
| `backend/` | API container at `/plugins` | WASM binary + SQL migrations + manifest |
| `frontend/` | Gateway container at `/var/www/plugins` | Built JS/CSS bundles only |

### Installing a Plugin

See the [Local Plugins README](./local/README.md) for detailed instructions on building and installing plugins locally.

## Plugin SDKs

The Plugin SDKs have been moved to their own repositories:

- **Backend SDK (Go)**: [github.com/Paca-AI/plugin-sdk-go](https://github.com/Paca-AI/plugin-sdk-go)
- **Frontend SDK (React/TypeScript)**: [github.com/Paca-AI/plugin-sdk-react](https://github.com/Paca-AI/plugin-sdk-react)

## Documentation

For plugin development documentation, see:

- [Plugin System Overview](../docs/plugins/overview.md)
- [Plugin Developer Guide](../docs/plugins/developer-guide.md)
- [SDK Reference](../docs/plugins/sdk-reference.md)
- [First-Party Plugins](../docs/plugins/first-party-plugins.md)
