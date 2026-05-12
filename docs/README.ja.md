# AI Assistant バックエンド

[**English**](../README.md) | [**中文**](README.zh.md)

AI Assistant アプリケーションのバックエンドサービス。AI チャット API（SSE ストリーミング）、MCP ブリッジ管理、マルチモデル構成、Blender/GIMP プラグインサポートを提供します。

> **フロントエンドリポジトリ**: [github.com/poorboy/aiassistant_f](https://github.com/poorboy/aiassistant_f)

## 機能

- **AI チャット**: OpenAI API 互換の SSE ストリーミングチャット。MCP ツール呼び出し（Function Calling）に対応。
- **マルチモデル管理**: 複数のプロバイダーとモデル（DeepSeek, OpenAI, Anthropic, Google 等）を設定可能。各モデルに API キー、ベース URL、プロキシ設定を個別に設定。
- **MCP ブリッジ**: Blender および GIMP の MCP サービス接続/切断、ツール一覧表示、リアルタイムログ表示。
- **ロールプロンプト**: システムロールプロンプトの CRUD。AI の動作をカスタマイズ。
- **プラグインサポート**: Blender アドオンおよび GIMP プラグインスクリプトを内蔵。

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
