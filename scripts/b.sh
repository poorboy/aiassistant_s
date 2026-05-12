#!/bin/bash
cd "$(dirname "$0")/../gosrc"
echo "Building Go backend..."
go clean
go build -o aiass.exe
rm -f ../exe/aiass.exe
mv aiass.exe ../exe/
echo "✅ Built to exe/aiass.exe"
