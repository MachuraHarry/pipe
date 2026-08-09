#!/usr/bin/env bash
# Red-team sandbox escape test runner.
#
# Reads the DeepSeek API key from the DEEPSEEK_API_KEY environment variable
# (never hardcoded), builds the pipe binary, runs redteam.pipe in a throwaway
# working directory and writes the full output + a summary report to
# out/run<N>.txt and out/report.md.
#
# Usage:
#   DEEPSEEK_API_KEY=sk-... ./runner.sh [max_rounds]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
if [[ -z "${DEEPSEEK_API_KEY:-}" ]]; then
    echo "error: DEEPSEEK_API_KEY is not set" >&2
    echo "usage: DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh" >&2
    exit 1
fi

ROUNDS="${1:-20}"
OUT="$ROOT/../docs/tests/sandbox-audit/out"
mkdir -p "$OUT"

# 1. Build the binary.
echo "==> building pipe binary"
go build -o "$OUT/pipe" "$ROOT/../cmd/pipe"

# 2. Throwaway working directory: the temp-only profile redirects writes into
#    <wd>/.pipe_sandbox, keeping the repo clean.
WD="$OUT/wd"
rm -rf "$WD"
mkdir -p "$WD"

# 3. Run the escape attempt. The process CWD determines the default working
#    dir of the profile, so pipe is launched inside the throwaway dir.
echo "==> running red-team escape attempt (max rounds: $ROUNDS)"
RUN_FILE="$OUT/run-$(date +%Y%m%d-%H%M%S).txt"
(
    cd "$WD"
    "$OUT/pipe" "$ROOT/redteam.pipe"
) 2>&1 | tee "$RUN_FILE"
EXIT="${PIPESTATUS[0]}"
echo "==> pipe exit code: $EXIT"

# 4. Produce a short summary report.
{
    echo "# Red-Team Run Summary"
    echo
    echo "- time: $(date -Iseconds)"
    echo "- max_rounds: $ROUNDS"
    echo "- model: deepseek-v4-pro (thinking enabled, effort high)"
    echo "- exit: $EXIT"
    echo
    echo "## Tool-call audit trail"
    grep "| tool_call |" "$RUN_FILE" | sort | uniq -c | sed 's/^/    /' | head -40
    echo
    echo "## Escape verdicts (model-reported)"
    grep -iE "blocked|failed|escape" "$RUN_FILE" | grep -iv "audit" | tail -30 | sed 's/^/    /'
} > "$OUT/report.md"

echo "==> done: $RUN_FILE"
echo "==> summary: $OUT/report.md"
echo "==> audit evidence: $(grep -c '| tool_call |' "$RUN_FILE") tool calls logged"
