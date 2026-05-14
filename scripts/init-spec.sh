#!/bin/bash
# init-spec.sh
# Purpose: Mechanical scaffold of a per-feature spec folder per pattern-spec-driven-development.
#          Vault-rooted: requires vault entry before scaffolding (SSOT discipline).
#
# Usage:
#   ./init-spec.sh <feature-id> [--task <task-id>] [--force-no-vault]
#
# Mechanical only. For Socratic proposal filling (Q1-Q6), use /spec fill in an agent.
# See: $VAULT_PATH/00_meta/skills/spec/SKILL.md for the full workflow.

set -euo pipefail

usage() {
    cat <<EOF
Usage: init-spec.sh <feature-id> [--task <task-id>] [--force-no-vault]

  <feature-id>           e.g. AI-001-ollama-public or 2026-05-13-foo
  --task <task-id>       override which task ID to look up in vault 11-tasks.md
  --force-no-vault       skip vault context check (NOT RECOMMENDED — violates SSOT)
  -h, --help             show this help
EOF
}

# --- Parse args ---
FEATURE_ID=""
TASK_ID=""
FORCE_NO_VAULT=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --task) TASK_ID="$2"; shift 2 ;;
        --force-no-vault) FORCE_NO_VAULT=1; shift ;;
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

# --- Validate id ---
if [[ ! "$FEATURE_ID" =~ ^([A-Z]+-[0-9]+(-[a-z0-9-]+)?|[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9-]+)$ ]]; then
    printf '[ERROR] Invalid feature-id: %s\n' "$FEATURE_ID" >&2
    printf '        Expected: TICKET-NNN[-slug] (e.g. AI-001-ollama-public) or YYYY-MM-DD-slug\n' >&2
    exit 1
fi

# --- Resolve paths ---
VAULT_PATH="${VAULT_PATH:-$HOME/Projects/knowledge}"
TEMPLATES_DIR="$VAULT_PATH/00_meta/templates"

if [[ ! -d "$TEMPLATES_DIR" ]]; then
    printf '[ERROR] Vault templates not found at: %s\n' "$TEMPLATES_DIR" >&2
    # shellcheck disable=SC2016  # intentional: '$VAULT_PATH' is the literal env var name we want to show
    printf '        Set %s env var to vault root, or clone the vault.\n' '$VAULT_PATH' >&2
    exit 2
fi

if ! REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
    printf '[ERROR] Not in a git repo. cd into a repo first.\n' >&2
    exit 1
fi
REPO_NAME="$(basename "$REPO_ROOT")"
SPEC_DIR="$REPO_ROOT/specs/$FEATURE_ID"

# --- No clobber ---
if [[ -d "$SPEC_DIR" ]]; then
    printf '[ERROR] Already exists: %s\n' "$SPEC_DIR" >&2
    exit 1
fi
if [[ -d "$REPO_ROOT/specs/archive/$FEATURE_ID" ]]; then
    printf '[WARN] %s exists in specs/archive/. Possibly reviving.\n' "$FEATURE_ID" >&2
fi

# --- Vault context check ---
VAULT_LINE=""
TASKS_FILE="$VAULT_PATH/10_projects/$REPO_NAME/11-tasks.md"

if [[ "$FORCE_NO_VAULT" -eq 0 ]]; then
    SEARCH_ID="${TASK_ID:-$FEATURE_ID}"
    if [[ -f "$TASKS_FILE" ]]; then
        VAULT_LINE="$(grep -E "^- \[[ x-]\] \*\*${SEARCH_ID}\*\*" "$TASKS_FILE" 2>/dev/null || true)"
    fi
    if [[ -z "$VAULT_LINE" ]]; then
        cat >&2 <<EOF
[ERROR] No vault entry found for "$SEARCH_ID" in $TASKS_FILE.

Spec must be downstream of a vault entry (backlog/ADR/roadmap) per SSOT discipline.

Options:
  (a) Cancel. Add an entry to 11-tasks.md, then re-run.
  (b) Re-run with --force-no-vault (NOT RECOMMENDED).
EOF
        exit 3
    fi
    printf '[INFO] Vault context found in %s\n' "$TASKS_FILE"
fi

# --- Scaffold ---
mkdir -p "$SPEC_DIR"
TODAY="$(date -u +%Y-%m-%d)"
TITLE="$FEATURE_ID"

for tpl in proposal tasks verification; do
    src="$TEMPLATES_DIR/spec-${tpl}.md"
    dst="$SPEC_DIR/${tpl}.md"
    if [[ ! -f "$src" ]]; then
        printf '[ERROR] Template missing: %s\n' "$src" >&2
        exit 2
    fi
    sed -e "s|<feature-id>|$FEATURE_ID|g" \
        -e "s|{TITLE}|$TITLE|g" \
        -e "s|{{date:YYYY-MM-DD}}|$TODAY|g" \
        "$src" > "$dst"
done

# --- Inject vault context comment in proposal Why ---
if [[ -n "$VAULT_LINE" ]]; then
    proposal="$SPEC_DIR/proposal.md"
    CONTEXT_TEXT="$(echo "$VAULT_LINE" | sed -E 's/^- \[[^]]+\] \*\*[^*]+\*\*:? *//')"
    awk -v line="<!-- from 11-tasks.md: $CONTEXT_TEXT -->" '
        /^## Why$/ {print; print ""; print line; next}
        {print}
    ' "$proposal" > "${proposal}.tmp" && mv "${proposal}.tmp" "$proposal"
fi

# --- Output ---
printf '\n[OK] Created: %s\n' "$SPEC_DIR"
printf '     proposal.md, tasks.md, verification.md\n'
if [[ -n "$VAULT_LINE" ]]; then
    printf '     Vault context linked from %s\n' "$TASKS_FILE"
fi
printf '\nNext: fill proposal.md interactively ("/spec fill %s" in an agent)\n' "$FEATURE_ID"
printf '      or edit by hand. Do not skip the Why.\n'
