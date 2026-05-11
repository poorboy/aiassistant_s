# AI Assistant - Server

Go + Echo 后端服务，提供 AI 聊天 + MCP 协议控制 Blender/GIMP。

## 技术栈

- Go 1.26
- Echo v4
- SQLite3

## 快速开始

```bash
cd server
go run .
```

## 构建

```bash
go build -ldflags "-s -w" -o output.exe .
```

## API

| Method   | Path                               | 说明         |
| -------- | ---------------------------------- | ------------ |
| GET      | `/health`                          | 健康检查     |
| GET      | `/api/chat/stream`                 | SSE 流式聊天 |
| GET/POST | `/api/chat/conversations`          | 会话管理     |
| GET/PUT  | `/api/settings`                    | 系统设置     |
| GET      | `/api/mcp/connections`             | MCP 连接列表 |
| POST     | `/api/mcp/connections/:id/connect` | 连接 MCP     |
