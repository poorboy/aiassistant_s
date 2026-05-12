#!/bin/bash
echo "Stopping aiass.exe..."
taskkill -f -im aiass.exe 2>/dev/null && echo "✅ Stopped" || echo "⚠️ No aiass.exe running"
