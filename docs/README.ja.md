# AI Assistant バックエンド

[**English**](../README.md) | [**中文**](README.zh.md)

Go 製バックエンド。SSE ストリーミング AI チャット、Blender/GIMP MCP ブリッジ、マルチプロバイダーモデル管理、ロールプロンプト、プラグイン展開を提供します。

> **フロントエンドリポジトリ**: [github.com/poorboy/aiassistant_f](https://github.com/poorboy/aiassistant_f)
>
> **ユーザーマニュアル**: [English](MANUAL.en.md) | [中文](MANUAL.zh.md) | [日本語](MANUAL.ja.md)

## デプロイ

### 前提条件
- Go 1.26+
- フロントエンド構築には Node.js 22+ と [フロントエンドプロジェクト](https://github.com/poorboy/aiassistant_f) が必要

### ビルドと実行

```bash
# バックエンドのビルド
cd gosrc && go build -o ../exe/aiass.exe .

# フロントエンドのビルド
git clone https://github.com/poorboy/aiassistant_f.git
cd aiassistant_f && npm install && npm run build

# 実行
cd exe && ./aiass.exe
```
サービスは `http://localhost:41400` で起動。デフォルトデータベース: `./data/assdata.db`。
