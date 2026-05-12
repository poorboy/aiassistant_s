#!/bin/bash
SCRIPT_DIR="$(dirname "$0")"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Stopping existing processes on port 41400..."
netstat -ano | grep :41400 | awk '{print $5}' | xargs -r taskkill -f -pid 2>/dev/null || true

echo "Stopping existing aiass processes..."
taskkill -f -im aiass.exe 2>/dev/null || true

echo "Building Go project..."
cd "$PROJECT_DIR/gosrc"
go clean
go build -o aiass.exe

echo "Moving executable..."
rm -f "$PROJECT_DIR/exe/aiass.exe"
mv aiass.exe "$PROJECT_DIR/exe/"

echo "Starting server..."
cd "$PROJECT_DIR/exe"
./aiass.exe > "$PROJECT_DIR/testLog.log" 2>&1 &

sleep 3
if curl -s http://localhost:41400/health > /dev/null 2>&1; then
    echo "✅ Server started successfully"
    echo "Log file: testLog.log"
else
    echo "❌ Server failed to start"
    tail -n 20 "$PROJECT_DIR/testLog.log"
fi
