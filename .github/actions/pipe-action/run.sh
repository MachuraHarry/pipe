#!/bin/bash
# Pipe Action wrapper
set -e

if [ -z "$PIPE_FILE" ] && [ -z "$PIPE_SCRIPT" ]; then
  echo "pipe-action: either 'script' or 'file' input is required" >&2
  exit 1
fi

if [ -n "$PIPE_FILE" ]; then
  if [ ! -f "$PIPE_FILE" ]; then
    echo "pipe-action: file not found: $PIPE_FILE" >&2
    exit 1
  fi
  echo "=== Pipe File: $PIPE_FILE ==="
else
  echo "$PIPE_SCRIPT" > /tmp/pipe-script
  if [ -n "$PIPE_PROVIDER" ]; then
    sed -i "1i ai_provider \"$PIPE_PROVIDER\"" /tmp/pipe-script
  fi
  echo "=== Pipe Script ==="
  cat /tmp/pipe-script
  echo "===================="
fi

install_pipe() {
  local os arch url version="${PIPE_VERSION:-latest}" artifact

  case "${RUNNER_OS:-$(uname -s)}" in
    Linux)        os=linux ;;
    macOS|Darwin) os=darwin ;;
    Windows|MINGW*|MSYS*|CYGWIN*) os=windows ;;
    *) echo "pipe-action: unsupported OS: ${RUNNER_OS:-$(uname -s)}" >&2; exit 1 ;;
  esac

  case "${RUNNER_ARCH:-$(uname -m)}" in
    ARM64|arm64|aarch64) arch=arm64 ;;
    X64|amd64|x86_64)    arch=amd64 ;;
    *) echo "pipe-action: unsupported architecture: ${RUNNER_ARCH:-$(uname -m)}" >&2; exit 1 ;;
  esac

  artifact="pipe-${os}-${arch}.tar.gz"
  if [ "$version" = "latest" ]; then
    url="https://github.com/MachuraHarry/pipe/releases/latest/download/${artifact}"
  else
    url="https://github.com/MachuraHarry/pipe/releases/download/${version}/${artifact}"
  fi

  echo "pipe-action: downloading Pipe ${version} (${os}/${arch})"
  curl -fsSL --retry 5 --retry-all-errors --retry-delay 2 -o "/tmp/${artifact}" "$url"

  # Verify SHA256 when the release ships one (releases after v0.7.0 do); warn and continue otherwise.
  if curl -fsSL --retry 5 --retry-all-errors --retry-delay 2 -o "/tmp/${artifact}.sha256" "$url.sha256" 2>/dev/null; then
    (cd /tmp && sha256sum -c "${artifact}.sha256" >/dev/null) || {
      echo "pipe-action: SHA256 verification failed" >&2
      exit 1
    }
    echo "pipe-action: SHA256 verified"
  else
    echo "pipe-action: warning — no SHA256 checksum available, skipping verification"
  fi

  tar -xzf "/tmp/${artifact}" -C /tmp

  mkdir -p "$HOME/.local/bin"
  if [ -f /tmp/pipe.exe ]; then
    mv /tmp/pipe.exe "$HOME/.local/bin/pipe.exe"
    chmod +x "$HOME/.local/bin/pipe.exe"
  else
    mv /tmp/pipe "$HOME/.local/bin/pipe"
    chmod +x "$HOME/.local/bin/pipe"
  fi
  export PATH="$HOME/.local/bin:$PATH"
}

if ! command -v pipe &>/dev/null; then
  install_pipe
fi

# In file mode the provider is passed as a flag (inline scripts use sed injection above).
provider_flag=""
if [ -n "$PIPE_PROVIDER" ] && [ -n "$PIPE_FILE" ]; then
  provider_flag=" --ai-provider $PIPE_PROVIDER"
fi

cmd="pipe${provider_flag} ${PIPE_FLAGS:--vm -q}"
if [ -n "$PIPE_FILE" ]; then
  cmd="$cmd $PIPE_FILE"
else
  cmd="$cmd /tmp/pipe-script"
fi

echo "=== Pipe command ==="
echo "$cmd"
echo "===================="

# shellcheck disable=SC2086
eval "$cmd"
