#!/bin/bash
# Usage: GITHUB_TOKEN=ghp_xxx bash scripts/release.sh [tag]
# Builds all platform packages and publishes them to GitHub Releases.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$PROJECT_DIR/publs"

TOKEN="${GITHUB_TOKEN}"
REPO="poorboy/aiassistant_s"
TAG="${1:-v1.0.0}"
VERSION="${TAG#v}"

if [ -z "$TOKEN" ]; then
  echo "❌ GITHUB_TOKEN not set"
  exit 1
fi

# Step 1: Build all platform packages via package.sh
echo ">>> Building all platform packages..."
bash "$SCRIPT_DIR/package.sh" windows-386 windows-amd64 windows-arm64 linux-386 linux-amd64 linux-arm64 darwin-amd64 darwin-arm64

# Step 2: Create release
echo ""
echo ">>> Creating release ${TAG}..."
RELEASE_JSON=$(curl -s -X POST "https://api.github.com/repos/${REPO}/releases" \
  -H "Authorization: token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "$(cat <<EOF
{
  "tag_name": "${TAG}",
  "target_commitish": "main",
  "name": "v${VERSION}",
  "body": "## AI Assistant Backend v${VERSION}\n\nMulti-platform packages for the Go backend service.\n\n### Platforms\n- Windows (386 / amd64 / arm64)\n- Linux (386 / amd64 / arm64)\n- macOS / Darwin (amd64 / arm64)",
  "draft": false,
  "prerelease": false
}
EOF
)")

RELEASE_ID=$(echo "$RELEASE_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || \
  echo "$RELEASE_JSON" | python -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null)

if [ -z "$RELEASE_ID" ]; then
  echo "❌ Failed to create release:"
  echo "$RELEASE_JSON"
  exit 1
fi
echo "✅ Release created, ID: ${RELEASE_ID}"

# Step 3: Upload packages as assets
echo ""
echo ">>> Uploading assets..."
EXTENSIONS=("zip" "tar.gz")
for ext in "${EXTENSIONS[@]}"; do
  for f in "$BUILD_DIR"/*."$ext"; do
    [ -f "$f" ] || continue
    name=$(basename "$f")
    echo "  Uploading ${name}..."
    curl -s -X POST "https://uploads.github.com/repos/${REPO}/releases/${RELEASE_ID}/assets?name=${name}" \
      -H "Authorization: token ${TOKEN}" \
      -H "Content-Type: application/octet-stream" \
      --data-binary @"${f}" > /dev/null
  done
done

echo ""
echo "✅ All assets uploaded to https://github.com/${REPO}/releases/tag/${TAG}"
