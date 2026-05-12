#!/bin/bash
cd "$(dirname "$0")/../web"
echo "Building frontend..."
npm run build
echo "✅ Frontend built to exe/static/"
