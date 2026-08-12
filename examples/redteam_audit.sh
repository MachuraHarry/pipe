#!/usr/bin/env bash
# Red-team sandbox escape test runner.
#
# Reads the DeepSeek API key from the DEEPSEEK_API_KEY environment variable
# (never hardcoded), builds the pipe binary and runs one of the red-team
# scripts in a throwaway working directory.
#
# Variants:
#   a  redteam.pipe          — locked profile (exec:false, temp-only,
#                              whitelist api.github.com) + MCP gates
#   b  redteam_ratchet.pipe  — unlocked profile; probes the sandbox "ratchet"
#                              (switch to more-permissive profiles)
#
# Usage:
#   DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh [a|b] [max_rounds]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
if [[ -z "${DEEPSEEK_API_KEY:-}" ]]; then
    echo "error: DEEPSEEK_API_KEY is not set" >&2
    echo "usage: DEEPSEEK_API_KEY=sk-... ./examples/redteam_audit.sh [a|b] [max_rounds]" >&2
    exit 1
fi

VARIANT="${1:-a}"
ROUNDS="${2:-20}"
case "$VARIANT" in
    a) SCRIPT="redteam.pipe"; LABEL="locked" ;;
    b) SCRIPT="redteam_ratchet.pipe"; LABEL="ratchet-unlocked" ;;
    *) echo "error: unknown variant '$VARIANT' (use a or b)" >&2; exit 1 ;;
esac

OUT="$ROOT/../docs/tests/sandbox-audit/out"
mkdir -p "$OUT"

# 1. Build the binary.
echo "==> building pipe binary"
go build -o "$OUT/pipe" "$ROOT/../cmd/pipe"

# 2. Throwaway working directory: the temp-only profile redirects writes into
#    <wd>/.pipe_sandbox, keeping the repo clean.
WD="$OUT/wd-$VARIANT"
rm -rf "$WD"
mkdir -p "$WD"

# 3. Run the escape attempt. The process CWD determines the default working
#    dir of the profile, so pipe is launched inside the throwaway dir.
echo "==> running red-team escape attempt (variant $VARIANT/$LABEL, max rounds: $ROUNDS)"
RUN_FILE="$OUT/run-$VARIANT-$(date +%Y%m%d-%H%M%S).txt"
(
    cd "$WD"
    "$OUT/pipe" "$ROOT/$SCRIPT"
) 2>&1 | tee "$RUN_FILE"
EXIT="${PIPESTATUS[0]}"
echo "==> pipe exit code: $EXIT"

# 4. Produce a short summary report.
{
    echo "# Red-Team Run Summary ($LABEL)"
    echo
    echo "- variant: $VARIANT ($LABEL)"
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
} > "$OUT/report-$VARIANT.md"

echo "==> done: $RUN_FILE"
echo "==> summary: $OUT/report-$VARIANT.md"
echo "==> audit evidence: $(grep -c '| tool_call |' "$RUN_FILE") tool calls logged"
