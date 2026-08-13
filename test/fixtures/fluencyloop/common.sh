#!/usr/bin/env bash
# common.sh — trimmed reference copy of FluencyLoop's shared helpers, kept to
# the subset slice-context.sh uses, for byte-identity testing of
# examples/fluencyloop-slice-context.pipe.
#
# Original: https://github.com/baokhang83/fluencyloop
#   plugins/fluencyloop/scripts/bash/common.sh
# Copied under its project license; see that repository.
set -euo pipefail

repo_root() {
    git rev-parse --show-toplevel 2>/dev/null || true
}

fluency_dir() {
    local root; root="$(repo_root)"
    if [ -n "$root" ]; then printf '%s/.fluencyloop' "$root"; fi
}

docs_dir() {
    local root; root="$(repo_root)"
    if [ -n "$root" ]; then printf '%s/docs/fluencyloop' "$root"; fi
}

require_fluency() {
    local dir; dir="$(fluency_dir)"
    if [ -z "$dir" ] || [ ! -d "$dir" ]; then
        echo "Error: FluencyLoop is not initialised here. Run 'fluencyloop init' first." >&2
        exit 1
    fi
}

state_path() { local d; d="$(fluency_dir)"; if [ -n "$d" ]; then printf '%s/state.json' "$d"; fi; }

state_get() {
    local f; f="$(state_path)"
    [ -f "$f" ] || return 0
    sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$f" | head -n1
}

current_feature_slug() {
    local b; b="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
    case "$b" in
        feature/*) printf '%s' "${b#feature/}" ;;
        *) printf '' ;;
    esac
}

feature_path() {
    local slug="$1" new old state_feature stored
    state_feature="$(state_get feature)"
    if [ -n "$state_feature" ] && [ "$state_feature" = "$slug" ]; then
        stored="$(state_get feature_dir)"
        if [ -n "$stored" ]; then
            printf '%s/%s' "$(repo_root)" "$stored"
            return
        fi
    fi
    new="$(docs_dir)/features/$slug"
    old="$(fluency_dir)/features/$slug"
    if [ ! -d "$new" ] && [ -d "$old" ]; then
        printf '%s' "$old"
    else
        printf '%s' "$new"
    fi
}

store_dir() { local d; d="$(docs_dir)"; if [ -n "$d" ]; then printf '%s/store' "$d"; fi; }

feature_store_path() { printf '%s/features/%s.jsonl' "$(store_dir)" "$1"; }

json_escape() {
    printf '%s' "$1" | sed -E 's/\\/\\\\/g; s/"/\\"/g' | awk 'BEGIN{ORS=""} {print (NR>1?"\\n":"") $0}'
}
