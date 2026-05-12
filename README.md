# AI Assistant Backend

[**中文**](docs/README.zh.md) | [**日本語**](docs/README.ja.md)

A Go backend service providing AI chat API with SSE streaming, MCP (Model Context Protocol) bridge management, multi-model configuration, and plugin support for Blender & GIMP.

## Deployment

### Prerequisites
- Go 1.26+
- Node.js 22+ (for frontend build)

### Build & Run

```bash
# Build backend
cd gosrc && go build -o ../exe/aiass.exe .

# Build frontend (optional, for standalone deployment)
cd web && npm install && npm run build

# Run
cd exe && ./aiass.exe
```

The service starts at `http://localhost:41400`. Default database: `./data/assdata.db`.

### Packaging (All Platforms)

```bash
bash scripts/package.sh                      # current platform
bash scripts/package.sh linux-amd64 darwin-arm64  # specific targets
```

Output: `publs/aiass-{os}-{arch}.zip` (Windows) or `.tar.gz` (others).

Each package contains:
- `exe/aiass.exe` — backend binary
- `exe/static/` — frontend assets
- `data/mcp_bin/` — MCP binaries for target platform
- `data/plugin/` — Blender & GIMP plugins
