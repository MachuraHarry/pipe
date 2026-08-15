#!/usr/bin/env bash
# Start the docs-pipe web dashboard and open it in the browser.
# Usage: ./scripts/docs-dashboard.sh
# Env:  PIPE_DOCS_DIR  (default docs/en)   PORT (default 8090)
#       OPEN_BROWSER=0 to skip auto-opening the browser.
set -euo pipefail
cd "$(dirname "$0")/.."

PORT="${PORT:-8090}"
DOCS_DIR="${PIPE_DOCS_DIR:-docs/en}"
LOG="${TMPDIR:-/tmp}/docs-dashboard.log"

if [ ! -x bin/pipe ]; then
    echo "Building pipe ..."
    make build
fi

echo "Indexing '$DOCS_DIR' and starting the dashboard on port $PORT ..."
PIPE_DOCS_DIR="$DOCS_DIR" ./bin/pipe examples/docs_pipe_dashboard.pipe >"$LOG" 2>&1 &
PID=$!

cleanup() { kill "$PID" 2>/dev/null || true; }
trap cleanup EXIT INT TERM

for _ in $(seq 1 120); do
    if curl -s -o /dev/null "http://localhost:$PORT/health"; then
        echo ""
        echo "  Dashboard ready:  http://localhost:$PORT"
        if [ "${OPEN_BROWSER:-1}" != "0" ]; then
            xdg-open "http://localhost:$PORT" >/dev/null 2>&1 || true
        fi
        echo "  Press Ctrl+C to stop."
        wait "$PID"
        exit 0
    fi
    if ! kill -0 "$PID" 2>/dev/null; then
        echo "The dashboard exited unexpectedly. Log:"
        cat "$LOG"
        exit 1
    fi
    sleep 1
done

echo "Timed out waiting for the server. Log:"
cat "$LOG"
exit 1
