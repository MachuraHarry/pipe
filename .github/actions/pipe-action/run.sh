#!/bin/bash
# Pipe Action wrapper — receives script content via stdin
set -e

# Read everything from stdin into the script file
cat > /tmp/pipe-script

# Prepend provider if specified
[ -n "$PIPE_PROVIDER" ] && sed -i "1i ai_provider \"$PIPE_PROVIDER\"" /tmp/pipe-script || true

echo "=== Pipe Script ==="
cat /tmp/pipe-script
echo "===================="

# Download pipe if not available
if ! command -v pipe &>/dev/null; then
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)
  [ "$ARCH" = "x86_64" ] && ARCH="amd64"
  [ "$ARCH" = "aarch64" ] && ARCH="arm64"
  URL="https://github.com/MachuraHarry/pipe/releases/latest/download/pipe-${OS}-${ARCH}.tar.gz"
  if curl -fsSL "$URL" -o pipe.tar.gz 2>/dev/null; then
    tar -xzf pipe.tar.gz && rm pipe.tar.gz
  else
    TMP=$(mktemp -d)
    git clone --depth 1 https://github.com/MachuraHarry/pipe "$TMP" 2>/dev/null
    cd "$TMP" && go build -o pipe ./cmd/pipe && cp pipe /usr/local/bin/
    cd / && rm -rf "$TMP"
  fi
  chmod +x pipe
fi

pipe ${PIPE_FLAGS:--vm -q} /tmp/pipe-script
