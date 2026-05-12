# AI Assistant バックエンド

[**English**](../README.md) | [**中文**](README.zh.md)

AI チャット API（SSE ストリーミング）、MCP（Model Context Protocol）ブリッジ管理、マルチモデル構成、Blender/GIMP プラグインサポートを提供する Go バックエンドサービス。

## デプロイ

### 前提条件
- Go 1.26+
- Node.js 22+（フロントエンド構築時）

### ビルドと実行

```bash
# バックエンドのビルド
cd gosrc && go build -o ../exe/aiass.exe .

# フロントエンドのビルド（スタンドアロンデプロイ時）
cd web && npm install && npm run build

# 実行
cd exe && ./aiass.exe
```

サービスは `http://localhost:41400` で起動します。デフォルトのデータベース: `./data/assdata.db`。

### マルチプラットフォームパッケージ

```bash
bash scripts/package.sh                          # 現在のプラットフォーム
bash scripts/package.sh linux-amd64 darwin-arm64  # ターゲット指定
```

出力先: `publs/` ディレクトリ
- Windows: `aiass-{os}-{arch}.zip`（7z 使用）
- その他: `aiass-{os}-{arch}.tar.gz`

各パッケージの内容:
- `exe/aiass.exe` — バックエンドバイナリ
- `exe/static/` — フロントエンドアセット
- `data/mcp_bin/` — 対象プラットフォームの MCP バイナリ
- `data/plugin/` — Blender および GIMP プラグイン
