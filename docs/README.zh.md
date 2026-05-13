# AI Assistant 后端

[**English**](../README.md) | [**日本語**](README.ja.md)

Go 语言实现的后端服务，提供 SSE 流式 AI 聊天、Blender/GIMP MCP 桥接、多供应商模型配置管理、角色提示词系统，以及插件部署支持。

> **前端仓库**: [github.com/poorboy/aiassistant_f](https://github.com/poorboy/aiassistant_f)
>
> **用户手册**: [English](MANUAL.en.md) | [中文](MANUAL.zh.md) | [日本語](MANUAL.ja.md)

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
