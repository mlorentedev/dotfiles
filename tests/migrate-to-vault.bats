#!/usr/bin/env bats
# SDD-008 (#141): scripts/skills/migrate-to-vault.sh — transactional repatriation
# of ai/skills -> vault 00_meta/skills. Runs against isolated fixture git repos;
# never touches the real vault.

setup() {
    SCRIPT="$BATS_TEST_DIRNAME/../scripts/skills/migrate-to-vault.sh"
    TMP="$(mktemp -d)"

    # fixture dotfiles repo with ai/skills + schema
    REPO="$TMP/repo"
    mkdir -p "$REPO/ai/skills" "$REPO/harness"
    cd "$REPO" || exit 1
    git init -qb main; git config user.email t@t; git config user.name t; git config commit.gpgsign false
    printf '{ "required": ["name", "description"] }\n' > "$REPO/harness/skill-frontmatter.schema.json"
    _mk() { mkdir -p "$REPO/ai/skills/$1"; printf -- '---\nname: %s\ndescription: %s\n---\n\n# %s\n' "$1" "$2" "$1" > "$REPO/ai/skills/$1/SKILL.md"; }
    _mk good-one "First skill."
    _mk good-two "Second skill."
    git add -A; git commit -qm seed

    # fixture vault repo with an empty 00_meta/skills
    VAULT="$TMP/vault"
    mkdir -p "$VAULT/00_meta/skills"
    git -C "$VAULT" init -qb main
    git -C "$VAULT" config user.email t@t; git -C "$VAULT" config user.name t; git -C "$VAULT" config commit.gpgsign false
    git -C "$VAULT" commit -qm seed --allow-empty
}

teardown() { cd / || true; rm -rf "$TMP"; }

@test "migrate: copies new skills into the vault and commits once" {
    run env VAULT_PATH="$VAULT" bash "$SCRIPT"
    [ "$status" -eq 0 ]
    [ -f "$VAULT/00_meta/skills/good-one/SKILL.md" ]
    [ -f "$VAULT/00_meta/skills/good-two/SKILL.md" ]
    git -C "$VAULT" log -1 --format=%s | grep -q 'migrate 2 skills'
}

@test "migrate: idempotent — second run skips existing, no new vault commit" {
    run env VAULT_PATH="$VAULT" bash "$SCRIPT"; [ "$status" -eq 0 ]
    before="$(git -C "$VAULT" rev-parse HEAD)"
    run env VAULT_PATH="$VAULT" bash "$SCRIPT"; [ "$status" -eq 0 ]
    [[ "$output" == *"already present"* ]]
    [ "$(git -C "$VAULT" rev-parse HEAD)" = "$before" ]
}

@test "migrate: --dry-run writes nothing to the vault" {
    run env VAULT_PATH="$VAULT" bash "$SCRIPT" --dry-run
    [ "$status" -eq 0 ]
    [ ! -d "$VAULT/00_meta/skills/good-one" ]
}

@test "migrate: transactional — an invalid skill rolls back ALL additions, non-zero exit" {
    # zzz-* sorts AFTER the good skills, so good-one/good-two are copied first,
    # then validation fails on zzz-bad and rollback must remove them.
    mkdir -p "$REPO/ai/skills/zzz-bad"
    printf -- '---\ndescription: no name field\n---\n# bad\n' > "$REPO/ai/skills/zzz-bad/SKILL.md"
    run env VAULT_PATH="$VAULT" bash "$SCRIPT"
    [ "$status" -ne 0 ]
    [ ! -d "$VAULT/00_meta/skills/good-one" ]
    [ ! -d "$VAULT/00_meta/skills/good-two" ]
    [ ! -d "$VAULT/00_meta/skills/zzz-bad" ]
}
