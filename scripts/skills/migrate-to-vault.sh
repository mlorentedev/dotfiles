#!/usr/bin/env bash
# migrate-to-vault.sh — SDD-008: repatriate dotfiles/ai/skills/<name>/ into the
# vault SSOT 00_meta/skills/<name>/ (compile-once-deploy-everywhere).
#
# Transactional (AC2): a pre-run snapshot of newly-added dirs is rolled back on
# any failure, so no half-migrated state survives. Idempotent: skills already in
# the vault are skipped. Validates frontmatter (reuses the schema) before copying.
# Does NOT delete ai/skills — that git rm is a separate, reviewable PR step; this
# script only POPULATES the vault SSOT and commits it.
#
# Usage: migrate-to-vault.sh [--dry-run]
#   VAULT_PATH overrides the vault root (default ~/Projects/knowledge).
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
SRC_DIR="$REPO_ROOT/ai/skills"
VAULT_PATH="${VAULT_PATH:-$HOME/Projects/knowledge}"
DST_DIR="$VAULT_PATH/00_meta/skills"
SCHEMA="$REPO_ROOT/harness/skill-frontmatter.schema.json"
DRY_RUN=0
[ "${1:-}" = "--dry-run" ] && DRY_RUN=1

command -v jq >/dev/null 2>&1 || { printf '[ERROR] jq required\n' >&2; exit 2; }
[ -d "$SRC_DIR" ] || { printf '[migrate] nothing to do: %s absent\n' "$SRC_DIR"; exit 0; }
[ -d "$DST_DIR" ] || { printf '[ERROR] vault skills dir not found: %s (clone the vault)\n' "$DST_DIR" >&2; exit 2; }

# Validate name+description present and non-empty in a SKILL.md (schema-driven).
validate_fm() {
    local f="$1" req
    [ "$(sed -n '1p' "$f")" = "---" ] || { printf '[ERROR] %s:1: missing frontmatter\n' "$f" >&2; return 1; }
    while IFS= read -r req; do
        awk -v k="$req" '/^---[[:space:]]*$/{n++; next} n==1 && $0 ~ ("^" k ":[[:space:]]*[^[:space:]]"){ok=1} n>=2{exit} END{exit ok?0:1}' "$f" \
            || { printf '[ERROR] %s: required frontmatter "%s" missing/empty\n' "$f" "$req" >&2; return 1; }
    done < <(jq -r '.required[]' "$SCHEMA")
}

ADDED=()
rollback() { local d; for d in ${ADDED[@]+"${ADDED[@]}"}; do rm -rf "${DST_DIR:?}/${d:?}"; done; printf '[migrate] ROLLED BACK %s dir(s)\n' "${#ADDED[@]}" >&2; }
trap 'rollback' ERR

migrated=0 skipped=0
for skill in "$SRC_DIR"/*/; do
    [ -f "$skill/SKILL.md" ] || continue
    name="$(basename "$skill")"
    validate_fm "$skill/SKILL.md"
    if [ -d "$DST_DIR/$name" ]; then
        printf '[migrate] skip (already in vault): %s\n' "$name"
        skipped=$((skipped + 1))
        continue
    fi
    if [ "$DRY_RUN" = "1" ]; then
        printf '[migrate] DRY-RUN would add: %s\n' "$name"
        continue
    fi
    cp -r "$skill" "$DST_DIR/$name"
    ADDED+=("$name")
    migrated=$((migrated + 1))
    printf '[migrate] vault <- %s\n' "$name"
done
trap - ERR

printf '[migrate] %s migrated, %s already present\n' "$migrated" "$skipped"
[ "$DRY_RUN" = "1" ] && exit 0
[ "$migrated" -eq 0 ] && { printf '[migrate] vault unchanged; no commit\n'; exit 0; }

# Commit the new SSOT entries to the vault. Origin recorded in the message
# (cross-repo git mv cannot preserve history; dotfiles git retains it).
ORIGIN="$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"
git -C "$VAULT_PATH" add "00_meta/skills"
git -C "$VAULT_PATH" commit -q -m "skills(SSOT): migrate ${migrated} skills from mlorentedev/dotfiles ai/skills/ (SDD-008, origin ${ORIGIN})" \
    && printf '[migrate] committed %s skills to the vault\n' "$migrated"
