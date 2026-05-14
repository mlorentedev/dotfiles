#!/bin/bash
# archive-spec.sh
# Purpose: Mechanical archive of a per-feature spec folder.
#          Tag-check pre-flight enforced unless --force-with-drafts.
#          No vault promotion — do that interactively via /spec archive in an agent.
#
# Usage:
#   ./archive-spec.sh <feature-id> [--pr <url>] [--abandoned] [--force-with-drafts]
#
# See: $VAULT_PATH/00_meta/skills/spec/SKILL.md for the full archive workflow.

set -euo pipefail

usage() {
    cat <<EOF
Usage: archive-spec.sh <feature-id> [--pr <url>] [--abandoned] [--force-with-drafts]

  <feature-id>             e.g. AI-001-ollama-public
  --pr <url>               record PR URL in proposal.md (informational)
  --abandoned              route to specs/archive/_abandoned/<id>/, set status abandoned
  --force-with-drafts      allow archive even with unresolved [AGENT-DRAFT] tags
  -h, --help               show this help
EOF
}

# --- Parse args ---
FEATURE_ID=""
PR_URL=""
ABANDONED=0
FORCE_DRAFTS=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --pr) PR_URL="$2"; shift 2 ;;
        --abandoned) ABANDONED=1; shift ;;
        --force-with-drafts) FORCE_DRAFTS=1; shift ;;
        -h|--help) usage; exit 0 ;;
        -*) printf '[ERROR] Unknown option: %s\n' "$1" >&2; usage >&2; exit 1 ;;
        *)
            if [[ -z "$FEATURE_ID" ]]; then
                FEATURE_ID="$1"; shift
            else
                printf '[ERROR] Multiple feature ids given\n' >&2; exit 1
            fi ;;
    esac
done

if [[ -z "$FEATURE_ID" ]]; then
    usage >&2
    exit 1
fi

# --- Resolve paths ---
if ! REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
    printf '[ERROR] Not in a git repo.\n' >&2
    exit 1
fi
SPEC_DIR="$REPO_ROOT/specs/$FEATURE_ID"

if [[ ! -d "$SPEC_DIR" ]]; then
    printf '[ERROR] Spec not found: %s\n' "$SPEC_DIR" >&2
    exit 1
fi

# --- Pre-flight: tag check ---
if [[ "$FORCE_DRAFTS" -eq 0 ]]; then
    if matches="$(grep -rnE '\[AGENT-(DRAFT|SUGGESTION)\]' "$SPEC_DIR" 2>/dev/null)"; then
        printf '[ERROR] Unresolved tags found:\n' >&2
        echo "$matches" >&2
        printf '\nResolve them (accept/edit/delete) before archiving, or use --force-with-drafts.\n' >&2
        exit 4
    fi
fi

# --- Determine target ---
if [[ "$ABANDONED" -eq 1 ]]; then
    TARGET_DIR="$REPO_ROOT/specs/archive/_abandoned/$FEATURE_ID"
    NEW_STATUS="abandoned"
else
    TARGET_DIR="$REPO_ROOT/specs/archive/$FEATURE_ID"
    NEW_STATUS="archived"
fi

if [[ -d "$TARGET_DIR" ]]; then
    printf '[ERROR] Already in archive: %s\n' "$TARGET_DIR" >&2
    exit 1
fi

# --- Move ---
mkdir -p "$(dirname "$TARGET_DIR")"
mv "$SPEC_DIR" "$TARGET_DIR"

# --- Update status in proposal.md frontmatter ---
PROPOSAL="$TARGET_DIR/proposal.md"
if [[ -f "$PROPOSAL" ]]; then
    awk -v new_status="$NEW_STATUS" '
        BEGIN {in_fm=0; fm_count=0}
        /^---$/ {
            fm_count++
            if (fm_count == 1) {in_fm=1}
            else if (fm_count == 2) {in_fm=0}
            print
            next
        }
        in_fm && /^status: / {print "status: " new_status; next}
        {print}
    ' "$PROPOSAL" > "${PROPOSAL}.tmp" && mv "${PROPOSAL}.tmp" "$PROPOSAL"
fi

# --- Record PR URL if provided ---
if [[ -n "$PR_URL" ]]; then
    printf '\n<!-- archived %s — PR: %s -->\n' "$(date -u +%Y-%m-%d)" "$PR_URL" >> "$PROPOSAL"
fi

# --- Output ---
printf '\n[OK] Archived: %s -> %s\n' "$SPEC_DIR" "$TARGET_DIR"
printf '     status: %s\n' "$NEW_STATUS"
if [[ -n "$PR_URL" ]]; then
    printf '     PR: %s\n' "$PR_URL"
fi
printf '\nVault promotion (lessons/ADR/pattern) and 11-tasks.md tick must be done\n'
printf 'separately (via /spec archive in an agent, or by hand).\n'
