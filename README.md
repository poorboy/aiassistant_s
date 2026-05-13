# AI Assistant Backend

[**中文**](docs/README.zh.md) | [**日本語**](docs/README.ja.md)

This Go backend powers the AI Assistant with SSE streaming AI chat, MCP (Model Context Protocol) bridge for Blender & GIMP, multi-model management across providers, system role prompts, and plugin deployment.

> **Frontend repository**: [github.com/poorboy/aiassistant_f](https://github.com/poorboy/aiassistant_f)
>
> **User Manual**: [English](docs/MANUAL.en.md) | [中文](docs/MANUAL.zh.md) | [日本語](docs/MANUAL.ja.md)

## Deployment

### Prerequisites
- Go 1.26+
- Frontend build requires Node.js 22+ and the [frontend project](https://github.com/poorboy/aiassistant_f)

### Build & Run

```bash
# Build backend
cd gosrc && go build -o ../exe/aiass.exe .

# Build frontend (from frontend repo)
git clone https://github.com/poorboy/aiassistant_f.git
cd aiassistant_f && npm install && npm run build

# Run
cd exe && ./aiass.exe
```
Service starts at `http://localhost:41400`. Default database: `./data/assdata.db`.
