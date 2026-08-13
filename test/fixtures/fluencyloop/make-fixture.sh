#!/usr/bin/env bash
# make-fixture.sh — builds the deterministic FluencyLoop test repo used by CI's
# byte-identity job. Prints the fixture repo path on stdout.
#
# The repo mirrors the shape the original slice-context.sh expects:
#   .fluencyloop/state.json              (feature + last journaled session)
#   docs/fluencyloop/store/features/...  (committed journal store)
#   main.txt, src/util.js                (tracked working files)
#   newfile.txt                          (untracked, part of the slice)
#
# Usage: make-fixture.sh [dir]
set -euo pipefail

# Deterministic commits: fixed identities and dates make the resulting tree,
# blob, and commit hashes identical on every runner, so the byte-identity
# comparison is meaningful across Linux and Windows CI.
export GIT_AUTHOR_NAME="FluencyLoop CI" GIT_AUTHOR_EMAIL="ci@example.com"
export GIT_COMMITTER_NAME="FluencyLoop CI" GIT_COMMITTER_EMAIL="ci@example.com"
export GIT_AUTHOR_DATE="2001-02-03T04:05:06Z"
export GIT_COMMITTER_DATE="2001-02-03T04:05:06Z"

DIR="${1:-$(mktemp -d)}"
mkdir -p "$DIR"
git -C "$DIR" init -q -b main
git -C "$DIR" config user.name "FluencyLoop CI"
git -C "$DIR" config user.email "ci@example.com"
git -C "$DIR" config core.autocrlf false

mkdir -p "$DIR/.fluencyloop" \
         "$DIR/docs/fluencyloop/store/features" \
         "$DIR/docs/fluencyloop/features/042-rate-limit/sessions"

printf '%s\n' '{"feature":"042-rate-limit","base_ref":"main","last_session":"001-wire-cache"}' > "$DIR/.fluencyloop/state.json"
printf '%s\n' '{"schema_version":"1","type":"decision","feature":"042-rate-limit","session":"001-wire-cache","commit":"abc123"}' > "$DIR/docs/fluencyloop/store/features/042-rate-limit.jsonl"
printf '%s\n' 'first' > "$DIR/main.txt"

git -C "$DIR" add -A
git -C "$DIR" commit -q -m init

printf '%s\n' 'code' > "$DIR/main.txt"
mkdir -p "$DIR/src"
printf '%s\n' 'export function inc(x) { if (x > 0) { return x + 1; } return x; }' > "$DIR/src/util.js"
git -C "$DIR" add -A
git -C "$DIR" commit -q -m change

printf '%s\n' 'hello' > "$DIR/newfile.txt"

printf '%s' "$DIR"
