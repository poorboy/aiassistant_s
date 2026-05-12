# AI Assistant Backend

[**中文**](docs/README.zh.md) | [**日本語**](docs/README.ja.md)

Backend service for the AI Assistant application. Provides AI chat API with SSE streaming, MCP (Model Context Protocol) bridge management, multi-model configuration, and Blender/GIMP plugin support.

> **Frontend repository**: [github.com/poorboy/aiassistant_f](https://github.com/poorboy/aiassistant_f)

## Features

- **AI Chat**: SSE streaming chat compatible with OpenAI API. Supports tool calling (Function Calling) for MCP tools.
- **Multi-Model Management**: Configure multiple providers and models (DeepSeek, OpenAI, Anthropic, Google, etc.). Each model can have its own API key, base URL, and proxy settings.
- **MCP Bridge**: Connect/disconnect Blender and GIMP MCP services, list tools, view real-time logs.
- **Role Prompts**: CRUD for system role prompts to customize AI behavior.
- **Plugin Support**: Includes Blender addon and GIMP plugin scripts.

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
