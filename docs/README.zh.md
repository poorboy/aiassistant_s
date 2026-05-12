# AI Assistant 后端

[**English**](../README.md) | [**日本語**](README.ja.md)

AI Assistant 应用的后端服务。提供 AI 聊天 API（SSE 流式）、MCP 桥接管理、多模型配置，以及 Blender/GIMP 插件支持。

> **前端仓库**: [github.com/poorboy/aiassistant_f](https://github.com/poorboy/aiassistant_f)

## 功能

- **AI 聊天**: 兼容 OpenAI API 的 SSE 流式聊天，支持 MCP 工具调用（Function Calling）。
- **多模型管理**: 配置多个供应商和模型（DeepSeek、OpenAI、Anthropic、Google 等），每个模型可独立设置 API Key、Base URL 和代理地址。
- **MCP 桥接**: 连接/断开 Blender 和 GIMP MCP 服务，查看可用工具列表，实时查看运行日志。
- **角色提示词**: 管理系统角色提示词（CRUD），自定义 AI 行为。
- **插件支持**: 内置 Blender 插件和 GIMP 插件脚本。

## 部署

### 环境要求
- Go 1.26+
- 构建前端需要 Node.js 22+ 和 [前端项目](https://github.com/poorboy/aiassistant_f)

### 构建与运行

```bash
# 构建后端
cd gosrc && go build -o ../exe/aiass.exe .

# 构建前端（从前端仓库克隆）
git clone https://github.com/poorboy/aiassistant_f.git
cd aiassistant_f && npm install && npm run build

# 运行
cd exe && ./aiass.exe
```
服务启动于 `http://localhost:41400`，默认数据库文件 `./data/assdata.db`。
