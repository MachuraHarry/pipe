#!/bin/sh
# Pipe (SPR) installer
#
# Downloads the latest (or pinned) Pipe release binary from GitHub,
# verifies its SHA256 checksum and installs it into your PATH.
#
# Usage:
#   curl -fsSL https://pipe-lang.com/install.sh | bash
#
# Environment overrides:
#   PIPE_VERSION  Release tag to install, e.g. v1.0.0 (default: latest)
#   PIPE_DIR      Install directory (default: ~/.local/bin, or /usr/local/bin when root)
#   PREFIX        Alias for PIPE_DIR
#   PIPE_OS       Force platform override for testing (linux|darwin|windows)
#   PIPE_ARCH     Force architecture override for testing (amd64|arm64)
#
# Uninstall:
#   install.sh --uninstall

set -eu

REPO="MachuraHarry/pipe"
BIN_NAME="pipe"
DEFAULT_VERSION="latest"

die() {
  echo "error: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

uninstall() {
  dir="${PIPE_DIR:-${PREFIX:-}}"
  if [ -z "$dir" ]; then
    if [ "$(id -u 2>/dev/null || echo 0)" = "0" ]; then
      dir="/usr/local/bin"
    else
      dir="${HOME}/.local/bin"
    fi
  fi
  rm -f "$dir/$BIN_NAME" && echo "Removed $dir/$BIN_NAME" || die "nothing to remove"
}

if [ "$(basename "$0")" = "install.sh" ] && [ "${1:-}" = "--uninstall" ]; then
  uninstall
  exit 0
fi

# --- detect platform -------------------------------------------------------
detect_os() {
  os="${PIPE_OS:-}"
  if [ -n "$os" ]; then
    case "$os" in
      linux|darwin|windows) ;;
      *) die "unsupported PIPE_OS: $os (expected linux, darwin or windows)" ;;
    esac
    echo "$os"
    return
  fi

  uname_os="$(uname -s)"
  case "$uname_os" in
    Linux)      echo "linux" ;;
    Darwin)     echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT) echo "windows" ;;
    *) die "unsupported OS: $uname_os" ;;
  esac
}

detect_arch() {
  arch="${PIPE_ARCH:-}"
  if [ -n "$arch" ]; then
    case "$arch" in
      amd64|arm64) ;;
      *) die "unsupported PIPE_ARCH: $arch (expected amd64 or arm64)" ;;
    esac
    echo "$arch"
    return
  fi

  uname_arch="$(uname -m)"
  case "$uname_arch" in
    x86_64|amd64|AMD64) echo "amd64" ;;
    aarch64|arm64|ARM64) echo "arm64" ;;
    *) die "unsupported architecture: $uname_arch (expected x86_64 or aarch64)" ;;
  esac
}

install_dir() {
  if [ -n "${PIPE_DIR:-}" ] || [ -n "${PREFIX:-}" ]; then
    echo "${PIPE_DIR:-${PREFIX}}"
    return
  fi
  if [ "$(id -u 2>/dev/null || echo 0)" = "0" ]; then
    echo "/usr/local/bin"
  else
    echo "${HOME}/.local/bin"
  fi
}

detect_sha() {
  if command -v sha256sum >/dev/null 2>&1; then
    SHA256SUM="sha256sum"
  elif command -v shasum >/dev/null 2>&1; then
    SHA256SUM="shasum -a 256"
  else
    die "neither sha256sum nor shasum is available"
  fi
}

verify_checksum() {
  artifact="$1"
  sha_file="$2"
  dir="$3"
  (
    cd "$dir" && $SHA256SUM -c "$sha_file" >/dev/null 2>&1
  ) || die "SHA256 verification failed for $artifact"
  echo "SHA256 verified"
}

main() {
  need_cmd curl
  need_cmd tar

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="${PIPE_VERSION:-$DEFAULT_VERSION}"
  destdir="$(install_dir)"

  if [ "$os" = "windows" ]; then
    die "install.sh is for Linux/macOS. On Windows use: irm https://pipe-lang.com/install.ps1 | iex"
  fi

  detect_sha

  artifact="pipe-${os}-${arch}.tar.gz"
  base="https://github.com/${REPO}/releases"
  if [ "$version" = "latest" ] || [ "$version" = "$DEFAULT_VERSION" ]; then
    url="${base}/latest/download/${artifact}"
  else
    url="${base}/download/${version}/${artifact}"
  fi

  tmp="$(mktemp -d "${TMPDIR:-/tmp}/pipe-install.XXXXXX")"
  trap 'rm -rf "$tmp"' EXIT HUP INT TERM

  echo "Downloading Pipe ${version} (${os}/${arch})"
  curl -fsSL -o "$tmp/$artifact" "$url"
  echo "Downloaded $url"

  if curl -fsSL -o "$tmp/$artifact.sha256" "$url.sha256" 2>/dev/null; then
    verify_checksum "$artifact" "$artifact.sha256" "$tmp"
  else
    echo "warning: no SHA256 checksum available, skipping verification"
  fi

  tar -xzf "$tmp/$artifact" -C "$tmp"

  binary="$tmp/$BIN_NAME"
  if [ ! -f "$binary" ]; then
    die "archive did not contain a $BIN_NAME binary"
  fi

  mkdir -p "$destdir"
  install -m 755 "$binary" "$destdir/$BIN_NAME"

  echo ""
  echo "Pipe installed: $destdir/$BIN_NAME"
  echo ""
  case ":$PATH:" in
    *":$destdir:"*) ;;
    *)
      echo "NOTE: $destdir is not in your PATH."
      if [ "$destdir" = "/usr/local/bin" ]; then
        echo "Open a new shell, or add it: export PATH=\"$destdir:\$PATH\""
      else
        echo "Add it to your shell profile, e.g. echo 'export PATH=\"$destdir:\$PATH\"' >> ~/.bashrc"
      fi
      echo ""
      ;;
  esac

  if "$destdir/$BIN_NAME" -h >/dev/null 2>&1; then
    echo "Verified: $("$destdir/$BIN_NAME" -h | head -n 1)"
  else
    "$destdir/$BIN_NAME" -h 2>&1 | head -n 1 || true
  fi
}

main "$@"
