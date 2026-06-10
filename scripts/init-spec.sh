#!/bin/bash
# init-spec.sh
# Purpose: Mechanical scaffold of a per-feature spec folder per pattern-spec-driven-development.
#          Work-gated: requires an OPEN GitHub issue (bitacora) before scaffolding (ADR-018).
#
# Usage:
#   ./init-spec.sh <feature-id> --issue <number> [--force-no-gate]
#
# Mechanical only. For Socratic proposal filling (Q1-Q6), use /spec fill in an agent.
# See: $VAULT_PATH/00_meta/skills/spec/SKILL.md for the full workflow.

set -euo pipefail

usage() {
    cat <<EOF
Usage: init-spec.sh <feature-id> --issue <number> [--force-no-gate]

  <feature-id>           e.g. AI-001-ollama-public or 2026-05-13-foo
  --issue <number>       GitHub issue that gates this work (must exist and be OPEN)
  --force-no-gate        skip the work-gate check (NOT RECOMMENDED — gate is the SSOT)
  --force-no-vault       deprecated alias of --force-no-gate
  -h, --help             show this help
EOF
}

# --- Parse args ---
FEATURE_ID=""
ISSUE_NUM=""
FORCE_NO_GATE=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --issue) ISSUE_NUM="$2"; shift 2 ;;
        --force-no-gate) FORCE_NO_GATE=1; shift ;;
        --force-no-vault)
            printf '[WARN] --force-no-vault is deprecated; use --force-no-gate (ADR-018).\n' >&2
            FORCE_NO_GATE=1; shift ;;
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
# The optional single letter after the number ([a-z]?) admits the sub-id
# convention (SDD-012b, WIN-002a) that check-backlog-integrity.sh already treats
# as a distinct ticket. Without it, init-spec rejected ids the rest of the system
# accepts — sibling defect to BUG-024 in the same gate.
if [[ ! "$FEATURE_ID" =~ ^([A-Z]+-[0-9]+[a-z]?(-[a-z0-9-]+)?|[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9-]+)$ ]]; then
    printf '[ERROR] Invalid feature-id: %s\n' "$FEATURE_ID" >&2
    printf '        Expected: TICKET-NNN[letter][-slug] (e.g. AI-001-ollama-public, SDD-012b-guard) or YYYY-MM-DD-slug\n' >&2
    exit 1
fi

if [[ -n "$ISSUE_NUM" && ! "$ISSUE_NUM" =~ ^[0-9]+$ ]]; then
    printf '[ERROR] --issue expects a number, got: %s\n' "$ISSUE_NUM" >&2
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
SPEC_DIR="$REPO_ROOT/specs/$FEATURE_ID"

# --- No clobber ---
if [[ -d "$SPEC_DIR" ]]; then
    printf '[ERROR] Already exists: %s\n' "$SPEC_DIR" >&2
    exit 1
fi
if [[ -d "$REPO_ROOT/specs/archive/$FEATURE_ID" ]]; then
    printf '[WARN] %s exists in specs/archive/. Possibly reviving.\n' "$FEATURE_ID" >&2
fi

# --- Work-gate check (ADR-018: an OPEN GitHub issue, not a vault entry) ---
ISSUE_TITLE=""

if [[ "$FORCE_NO_GATE" -eq 0 ]]; then
    if [[ -z "$ISSUE_NUM" ]]; then
        cat >&2 <<EOF
[ERROR] No work-gate given. Pass --issue <number>.

Per ADR-018 every spec is downstream of an OPEN GitHub issue on the bitacora
Project — that issue is the work-gate (the vault no longer holds task state).

Options:
  (a) Open (or find) the issue, then re-run with --issue <number>.
  (b) Re-run with --force-no-gate (NOT RECOMMENDED).
EOF
        exit 3
    fi
    if ! command -v gh >/dev/null 2>&1; then
        printf '[ERROR] gh CLI not found — cannot verify the work-gate issue #%s.\n' "$ISSUE_NUM" >&2
        printf '        Install gh, or re-run with --force-no-gate (NOT RECOMMENDED).\n' >&2
        exit 3
    fi
    GATE_INFO=""
    if ! GATE_INFO="$(gh issue view "$ISSUE_NUM" --json state,title --jq '.state + "\t" + .title' 2>&1)"; then
        printf '[ERROR] Work-gate issue #%s not found (or gh failed):\n' "$ISSUE_NUM" >&2
        printf '        %s\n' "$GATE_INFO" >&2
        exit 3
    fi
    ISSUE_STATE="${GATE_INFO%%$'\t'*}"
    ISSUE_TITLE="${GATE_INFO#*$'\t'}"
    if [[ "$ISSUE_STATE" != "OPEN" ]]; then
        printf '[ERROR] Work-gate issue #%s is not open (state: %s).\n' "$ISSUE_NUM" "$ISSUE_STATE" >&2
        printf '        The work-gate is an OPEN issue. Reopen it or pick the right one.\n' >&2
        exit 3
    fi
    printf '[INFO] Work-gate OK: issue #%s is open — %s\n' "$ISSUE_NUM" "$ISSUE_TITLE"
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

# --- Inject issue context comment in proposal Why ---
if [[ -n "$ISSUE_TITLE" ]]; then
    proposal="$SPEC_DIR/proposal.md"
    awk -v line="<!-- from issue #$ISSUE_NUM: $ISSUE_TITLE -->" '
        /^## Why$/ {print; print ""; print line; next}
        {print}
    ' "$proposal" > "${proposal}.tmp" && mv "${proposal}.tmp" "$proposal"
fi

# --- Output ---
printf '\n[OK] Created: %s\n' "$SPEC_DIR"
printf '     proposal.md, tasks.md, verification.md\n'
if [[ -n "$ISSUE_TITLE" ]]; then
    printf '     Work-gate linked: issue #%s\n' "$ISSUE_NUM"
fi
printf '\nNext: fill proposal.md interactively ("/spec fill %s" in an agent)\n' "$FEATURE_ID"
printf '      or edit by hand. Do not skip the Why.\n'
