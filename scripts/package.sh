#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$PROJECT_DIR/publs"
EXE_DIR="$PROJECT_DIR/exe"
EX_DATA_DIR="$PROJECT_DIR/ex_data"
GOSRC_DIR="$PROJECT_DIR/gosrc"
WEB_DIR="$PROJECT_DIR/web"

mkdir -p "$OUTPUT_DIR"

rm -rf "$OUTPUT_DIR"/*

TARGETS=()
if [ $# -eq 0 ]; then
  TARGETS=("windows-386" "windows-amd64" "windows-arm64" "linux-386" "linux-amd64" "linux-arm64" "darwin-amd64" "darwin-arm64")
else
  for arg in "$@"; do
    TARGETS+=("$arg")
  done
fi

build_frontend() {
  echo ">>> Building frontend..."
  cd "$WEB_DIR"
  npm run build 2>&1 | tail -1
}

build_backend() {
  local os="$1" arch="$2" out="$3"
  echo ">>> Building aiass.exe for ${os}-${arch}..."
  cd "$GOSRC_DIR"
  GOOS="$os" GOARCH="$arch" go build -o "$out" .
}

build_package() {
  local os="$1" arch="$2"
  local tag="${os}-${arch}"
  local pkg_dir="$OUTPUT_DIR/aiass-${tag}"
  local pkg_top_dir="$pkg_dir/aiass"
  local pkg_data_dir="$pkg_top_dir/data"

  echo ""
  echo "=========================================="
  echo "  Packaging for ${tag}"
  echo "=========================================="

  rm -rf "$pkg_dir" "$OUTPUT_DIR/aiass-${tag}.zip" "$OUTPUT_DIR/aiass-${tag}.tar.gz"
  mkdir -p "$pkg_top_dir" "$pkg_data_dir" "$pkg_data_dir/mcp_bin" "$pkg_data_dir/log" "$pkg_data_dir/plugin"
  mkdir -p "$pkg_top_dir/help"

  # 1. Build go backend
  build_backend "$os" "$arch" "$pkg_top_dir/aiass.exe"

  # 2. Copy frontend static
  cp -r "$EXE_DIR/static" "$pkg_top_dir/"

  # 3. Copy MCP binaries for this platform (flat under data/mcp_bin/)
  if [ -d "$EX_DATA_DIR/mcp_bin" ]; then
    for mcp in blender gimp; do
      src_dir="$EX_DATA_DIR/mcp_bin/${mcp}/${tag}"
      if [ -d "$src_dir" ]; then
        for f in "$src_dir"/*; do
          [ -f "$f" ] && cp "$f" "$pkg_data_dir/mcp_bin/"
        done
      fi
    done
  fi

  # 4. Copy help docs
  if [ -d "$PROJECT_DIR/help" ]; then
    cp -r "$PROJECT_DIR/help"/* "$pkg_top_dir/help/"
  fi

  # 5. Copy plugin files
  if [ -d "$EXE_DIR/data/plugin" ]; then
    cp "$EXE_DIR/data/plugin"/*.py "$pkg_data_dir/plugin/"
  fi

  # 5. Create package
  cd "$OUTPUT_DIR"
  if [ "$os" = "windows" ]; then
    archive="aiass-${tag}.zip"
    echo ">>> Creating ${archive} (7z)..."
    7z a -tzip "${archive}" "aiass-${tag}" > /dev/null
  else
    archive="aiass-${tag}.tar.gz"
    echo ">>> Creating ${archive}..."
    tar -czf "${archive}" "aiass-${tag}"
  fi

  rm -rf "$pkg_dir"
  echo "✅ Done: ${OUTPUT_DIR}/${archive}"
}

# Ensure frontend is built
build_frontend

for target in "${TARGETS[@]}"; do
  os="${target%%-*}"
  arch="${target##*-}"
  build_package "$os" "$arch"
done

echo ""
echo "=========================================="
echo "  All packages built in: ${OUTPUT_DIR}"
ls -lh "$OUTPUT_DIR"/*.zip "$OUTPUT_DIR"/*.tar.gz 2>/dev/null || true
echo "=========================================="
