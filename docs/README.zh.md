# AI Assistant 后端

[**English**](../README.md) | [**日本語**](README.ja.md)

Go 语言实现的后台服务，提供 AI 聊天 API（SSE 流式）、MCP（Model Context Protocol）桥接管理、多模型配置管理，以及 Blender/GIMP 插件支持。

## 部署

### 环境要求
- Go 1.26+
- Node.js 22+（构建前端时需要）

### 构建与运行

```bash
# 构建后端
cd gosrc && go build -o ../exe/aiass.exe .

# 构建前端（可选，独立部署时需要）
cd web && npm install && npm run build

# 运行
cd exe && ./aiass.exe
```

服务默认启动于 `http://localhost:41400`，数据库文件为 `./data/assdata.db`。

### 全平台打包

```bash
bash scripts/package.sh                          # 当前平台
bash scripts/package.sh linux-amd64 darwin-arm64  # 指定目标
```

输出至 `publs/` 目录：
- Windows: `aiass-{os}-{arch}.zip`（使用 7z）
- 其他平台: `aiass-{os}-{arch}.tar.gz`

每个包包含：
- `exe/aiass.exe` — 后端可执行文件
- `exe/static/` — 前端静态资源
- `data/mcp_bin/` — 对应平台的 MCP 二进制
- `data/plugin/` — Blender 和 GIMP 插件脚本
