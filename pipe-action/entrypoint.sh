#!/bin/sh
set -euo pipefail

if [ -z "${INPUT_SCRIPT:-}" ]; then
  echo "Error: 'script' input is required"
  exit 1
fi

# Build flags
flags=""
if [ "${INPUT_SANDBOX:-false}" = "true" ]; then
  flags="$flags --sandbox"
fi
if [ "${INPUT_ALLOW_AI:-false}" = "true" ]; then
  flags="$flags --allow-ai"
fi
if [ -n "${INPUT_TIMEOUT:-}" ]; then
  flags="$flags --timeout $INPUT_TIMEOUT"
fi
if [ -n "${INPUT_AI_PROVIDER:-}" ]; then
  flags="$flags --ai-provider $INPUT_AI_PROVIDER"
fi

tmpfile=$(mktemp)
echo "$INPUT_SCRIPT" > "$tmpfile"
# shellcheck disable=SC2086
pipe $flags "$tmpfile"
rc=$?
rm -f "$tmpfile"
exit $rc
