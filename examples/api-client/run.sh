#!/usr/bin/env bash
# Example: call the pipe-api from anywhere
set -euo pipefail

API="${1:-http://localhost:3001}"

echo "=== health ==="
curl -s "$API/health" | jq .

echo ""
echo "=== run: print ==="
curl -s -X POST "$API/run" \
  -H 'Content-Type: application/json' \
  -d '{"code": "print (2 + 2)"}' | jq .

echo ""
echo "=== run: summarize ==="
curl -s -X POST "$API/run" \
  -H 'Content-Type: application/json' \
  -d '{"code": "ai_provider \"openai\"; summarize \"Pipe is a programming language where AI operations are first-class citizens. It has 25 built-in AI operations.\" | print"}' | jq .
