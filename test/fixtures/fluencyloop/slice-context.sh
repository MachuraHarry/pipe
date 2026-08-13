#!/usr/bin/env bash
# slice-context.sh — reference copy of FluencyLoop's original for byte-identity
# testing of examples/fluencyloop-slice-context.pipe.
#
# Original: https://github.com/baokhang83/fluencyloop
#   plugins/fluencyloop/scripts/bash/slice-context.sh
# Copied under its project license; see that repository.
#
# Change vs. original (verified locally, needed under mawk 1.3.4 on Ubuntu):
#   split(p,"/",a)  ->  split(p,a,"/")   (correct awk argument order)
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
require_fluency
cd "$(repo_root)"

JSON_MODE=false
for arg in "$@"; do
    case "$arg" in
        --json) JSON_MODE=true ;;
        *) echo "Unknown option: $arg" >&2; exit 1 ;;
    esac
done

EXCLUDE=(-- . ':!.fluencyloop' ':!docs/fluencyloop')

FEATURE="$(state_get feature)"; [ -z "$FEATURE" ] && FEATURE="$(current_feature_slug)"
BASE_REF="$(state_get base_ref)"; [ -z "$BASE_REF" ] && BASE_REF="main"

SINCE=""; BASE_KIND="base-ref"
if [ -n "$FEATURE" ] && [ -n "$(state_get last_session)" ]; then
    STORE="$(feature_store_path "$FEATURE")"
    if [ -f "$STORE" ]; then
        LAST_JOURNAL="$(git log -1 --format=%H -- "$STORE" 2>/dev/null || true)"
    else
        SDIR="$(feature_path "$FEATURE")/sessions"
        LAST_JOURNAL="$(git log -1 --format=%H -- "$SDIR" 2>/dev/null || true)"
    fi
    [ -n "$LAST_JOURNAL" ] && { SINCE="$LAST_JOURNAL"; BASE_KIND="last-session"; }
fi
[ -z "$SINCE" ] && SINCE="$BASE_REF"
if ! git rev-parse --verify --quiet "$SINCE" >/dev/null 2>&1; then
    if git rev-parse --verify --quiet HEAD >/dev/null 2>&1; then
        SINCE="HEAD"; BASE_KIND="head"
    else
        SINCE=""; BASE_KIND="unborn"
    fi
fi

untracked_files() {
    git ls-files --others --exclude-standard -z "${EXCLUDE[@]}"
}

untracked_diff() {
    untracked_files | while IFS= read -r -d '' f; do
        git diff --no-index -- /dev/null "$f" 2>/dev/null || true
    done
}

unborn_files() {
    git ls-files --cached --others --exclude-standard -z "${EXCLUDE[@]}"
}

unborn_diff() {
    unborn_files | while IFS= read -r -d '' f; do
        [ -f "$f" ] || continue
        git diff --no-index -- /dev/null "$f" 2>/dev/null || true
    done
}

unborn_numstat() {
    unborn_files | while IFS= read -r -d '' f; do
        [ -f "$f" ] || continue
        git diff --no-index --numstat -- /dev/null "$f" 2>/dev/null || true
    done
}

if [ -n "$SINCE" ]; then
    DIFF="$( git diff "$SINCE" "${EXCLUDE[@]}"; untracked_diff )"
    read -r INS DEL TRACKED_FILES <<EOF
$(git diff --numstat "$SINCE" "${EXCLUDE[@]}" | awk '{i+=($1=="-"?0:$1); d+=($2=="-"?0:$2); n++} END{print i+0, d+0, n+0}')
EOF
else
    DIFF="$(unborn_diff)"
    read -r INS DEL _ <<EOF
$(unborn_numstat | awk '{i+=($1=="-"?0:$1); d+=($2=="-"?0:$2)} END{print i+0, d+0, 0}')
EOF
    TRACKED_FILES="$(git diff --cached --name-status "${EXCLUDE[@]}" | awk 'END{print NR+0}')"
fi

UNTRACKED_COUNT="$(untracked_files | tr '\0' '\n' | awk 'END{print NR+0}')"
FILES_CHANGED=$((TRACKED_FILES + UNTRACKED_COUNT))
if [ -n "$SINCE" ]; then
    SHORT="$(git rev-parse --short "$SINCE" 2>/dev/null || printf '%s' "$SINCE")"
else
    SHORT="unborn"
fi

HEUR="$(printf '%s\n' "$DIFF" | awk '
    function base(p,   n,a){ n=split(p,a,"/"); return a[n] }
    /^\+\+\+ / { f=$2; sub(/^b\//,"",f); bn=base(f)
        manifest = (bn ~ /^(package\.json|pom\.xml|go\.mod|Cargo\.toml|Gemfile|composer\.json|pyproject\.toml|requirements[^ ]*\.txt|Pipfile|build\.gradle(\.kts)?)$/)
        next }
    /^(---|diff |index |@@|new file|deleted file|similarity|rename|Binary)/ { next }
    /^\+/ {
        t=substr($0,2); gsub(/^[[:space:]]+/,"",t)
        if (t=="") next
        iscomment = (t ~ /^(\/\/|#|\*|\/\*|--|<!--)/)
        if (!iscomment) code++
        if (t ~ /^(import|from|#include|using|use)[[:space:]]/ || t ~ /require[[:space:]]*\(/) imp++
        if (manifest && !iscomment) dep++
        if (t ~ /^(export|public)[[:space:]]/ \
            || t ~ /^(export[[:space:]]+)?(default[[:space:]]+)?(async[[:space:]]+)?function[[:space:]]/ \
            || t ~ /^(public[[:space:]]+|export[[:space:]]+|abstract[[:space:]]+)*(class|interface|enum|trait|struct)[[:space:]]/ \
            || t ~ /^(pub[[:space:]]+)?fn[[:space:]]/ || t ~ /^func[[:space:]]/ || t ~ /^def[[:space:]]/ \
            || t ~ /^@[A-Za-z_.]*(route|mapping|[GgPpDd][a-z]+)[(:]?/) api++
        if (t ~ /(^|[^A-Za-z_])(if|else|for|while|switch|case|try|catch|except|match)([^A-Za-z_]|$)/) ctl++
    }
    END {
        score=0; sig=""
        if (imp>0 || dep>0)   { score+=2; sig=sig "dep-or-import " }
        if (api>0)            { score+=2; sig=sig "new-api " }
        if (ctl>0 && code>=8) { score+=2; sig=sig "control-flow " }
        if (code>=15)         { score+=1; sig=sig "size " }
        if (code>=40)         { score+=1 }
        if (code==0)          { score=0; sig="trivial" }
        sub(/[[:space:]]+$/,"",sig)
        printf "%d %s %s\n", score, (score>=2 ? "true" : "false"), sig
    }
')"
read -r DEC_SCORE DEC_LIKELY DEC_SIGNALS <<<"$HEUR" || true
: "${DEC_SCORE:=0}" "${DEC_LIKELY:=false}"
DEC_SIGNALS_JSON="$(printf '%s' "${DEC_SIGNALS:-}" | awk '{for(i=1;i<=NF;i++) printf "%s\"%s\"",(i>1?",":""),$i}')"

if $JSON_MODE; then
    if [ -n "$SINCE" ]; then
        file_status="$(git diff --name-status "$SINCE" "${EXCLUDE[@]}")"
    else
        file_status="$(git diff --cached --name-status "${EXCLUDE[@]}")"
    fi
    files="$(printf '%s\n' "$file_status" | awk -F'\t' '
        NF>=2 { p=$NF; gsub(/\\/,"\\\\",p); gsub(/"/,"\\\"",p)
                printf "%s{\"status\":\"%s\",\"path\":\"%s\"}", (n++?",":""), $1, p }')"
    untracked="$(untracked_files | tr '\0' '\n' | awk '
        { p=$0; gsub(/\\/,"\\\\",p); gsub(/"/,"\\\"",p); printf "%s\"%s\"", (n++?",":""), p }')"
    diff_esc="$(printf '%s' "$DIFF" | awk '
        { s=$0; gsub(/\\/,"\\\\",s); gsub(/"/,"\\\"",s); gsub(/\t/,"\\t",s); gsub(/\r/,"\\r",s)
          printf "%s%s", (NR>1?"\\n":""), s }')"
    printf '{"feature":"%s","base_kind":"%s","base":"%s","files_changed":%s,"insertions":%s,"deletions":%s,"likely_decision":%s,"decision_score":%s,"decision_signals":[%s],"files":[%s],"untracked":[%s],"diff":"%s"}\n' \
        "$(json_escape "$FEATURE")" "$BASE_KIND" "$(json_escape "$SHORT")" \
        "$FILES_CHANGED" "$INS" "$DEL" "$DEC_LIKELY" "$DEC_SCORE" "$DEC_SIGNALS_JSON" \
        "$files" "$untracked" "$diff_esc"
else
    echo "# Slice context — feature: ${FEATURE:-<none>} (since $BASE_KIND $SHORT)"
    echo "# $FILES_CHANGED file(s), +$INS -$DEL"
    echo "# likely_decision: $DEC_LIKELY (score $DEC_SCORE${DEC_SIGNALS:+: $DEC_SIGNALS})"
    echo
    printf '%s\n' "$DIFF"
fi
