#!/bin/bash
# Pipe Action wrapper
set -e

echo "$PIPE_SCRIPT" > /tmp/pipe-script

[ -n "$PIPE_PROVIDER" ] && sed -i "1i ai_provider \"$PIPE_PROVIDER\"" /tmp/pipe-script || true

echo "=== Pipe Script ==="
cat /tmp/pipe-script
echo "===================="

if ! command -v pipe &>/dev/null; then
  cd "$GITHUB_WORKSPACE" && make build 2>/dev/null && cp bin/pipe /usr/local/bin/pipe || {
    cd "$GITHUB_WORKSPACE" && go build -o /usr/local/bin/pipe ./cmd/pipe 2>/dev/null
  }
fi

pipe ${PIPE_FLAGS:--vm -q} /tmp/pipe-script
