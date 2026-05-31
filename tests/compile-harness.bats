#!/usr/bin/env bats
# Tests for scripts/compile-harness.sh — ENGINE-001 agent-artifact deploy engine.
# Each test runs the real script against an isolated temp git repo + fake vault,
# so the real AGENTS.md / CLAUDE.md are never touched.

setup() {
    SCRIPT="$BATS_TEST_DIRNAME/../scripts/compile-harness.sh"
    TMP="/tmp/bats_harness_$$_${BATS_TEST_NUMBER:-0}"
    mkdir -p "$TMP"

    # fake vault with one extractable section
    VAULT="$TMP/vault"
    mkdir -p "$VAULT/00_meta/patterns"
    cat > "$VAULT/00_meta/patterns/test-pattern.md" <<'EOF'
# Test Pattern

## 1. Demo Rule
- rule line one
- rule line two

## 2. Next Section
- unrelated
EOF

    # repo fixture
    REPO="$TMP/repo"
    mkdir -p "$REPO/harness"
    cd "$REPO" || exit 1
    git init -q -b main
    git config user.email t@t; git config user.name t; git config commit.gpgsign false

    # Point the real script at this fixture tree (the script defaults to its own
    # SCRIPT_DIR — the live repo — so without this every mode would operate on the
    # real harness/ instead of the fixture). Inherited by every `run` below.
    export HARNESS_REPO_ROOT="$REPO"

    cat > "$REPO/harness/manifest.json" <<'EOF'
{ "version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [ { "id": "demo", "source": "test-pattern.md#1-demo-rule" } ],
  "targets":  [ { "agent": "t", "kind": "native", "file": "TARGET.md", "inject": ["demo"] } ] }
EOF

    printf 'intro\n\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n\noutro\n' > "$REPO/TARGET.md"
    git add -A; git commit -q -m seed
}

teardown() { cd / || true; rm -rf "$TMP"; }

run_refresh() { run env VAULT_PATH="$VAULT" "$SCRIPT" --refresh; }

@test "--help exits 0 and prints usage" {
    run "$SCRIPT" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage"* ]]
}

@test "unknown argument exits 2" {
    run "$SCRIPT" --bogus
    [ "$status" -eq 2 ]
}

@test "AC2/AC7: --refresh writes source-of-record + injects content with sha marker" {
    run_refresh
    [ "$status" -eq 0 ]
    [ -f "$REPO/harness/enforced/demo.md" ]
    grep -q "rule line one" "$REPO/TARGET.md"
    grep -q "rule line two" "$REPO/TARGET.md"
    grep -qE "BEGIN HARNESS GENERATED \(sha256:[0-9a-f]{16}\)" "$REPO/TARGET.md"
}

@test "AC2: --refresh is idempotent" {
    run_refresh; [ "$status" -eq 0 ]
    cp "$REPO/TARGET.md" "$TMP/first"
    run_refresh; [ "$status" -eq 0 ]
    diff "$TMP/first" "$REPO/TARGET.md"
}

@test "AC1: --check passes on a freshly refreshed tree" {
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check
    [ "$status" -eq 0 ]
    [[ "$output" == *"no harness drift"* ]]
}

@test "AC1: --check fails after a deployed block is hand-edited" {
    run_refresh; [ "$status" -eq 0 ]
    sed -i 's/rule line one/TAMPERED/' "$REPO/TARGET.md"
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
    [[ "$output" == *"DRIFT"* ]]
}

@test "AC3: --check works offline (no vault) from the committed record" {
    run_refresh; [ "$status" -eq 0 ]
    run env VAULT_PATH="$TMP/nonexistent" "$SCRIPT" --check
    [ "$status" -eq 0 ]
}

@test "AC3: --check fails when the source-of-record is missing" {
    run_refresh; [ "$status" -eq 0 ]
    rm -f "$REPO/harness/enforced/demo.md"
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
}

@test "AC5: missing END marker makes --refresh fail loudly (no silent append)" {
    printf 'intro\n<!-- BEGIN HARNESS GENERATED -->\nno end here\n' > "$REPO/TARGET.md"
    run_refresh
    [ "$status" -ne 0 ]
    [[ "$output" == *"marker"* ]]
}

@test "AC6: healthcheck.sh wires the offline harness drift check" {
    grep -q 'compile-harness.sh" --check' "$BATS_TEST_DIRNAME/../scripts/healthcheck.sh"
}

@test "AC1: healthcheck.sh asserts deployed skills are symlink-free (SDD-008)" {
    grep -qF 'deployed skill path(s) are symlinks' "$BATS_TEST_DIRNAME/../scripts/healthcheck.sh"
    grep -qF '.claude/skills' "$BATS_TEST_DIRNAME/../scripts/healthcheck.sh"
    grep -qF '.config/opencode/commands' "$BATS_TEST_DIRNAME/../scripts/healthcheck.sh"
}

@test "setup-linux.sh runs compile-harness --refresh during deploy" {
    grep -q 'compile-harness.sh" --refresh' "$BATS_TEST_DIRNAME/../setup-linux.sh"
}

@test "setup-linux.sh runs compile-harness --deploy to render skills from records" {
    grep -q 'compile-harness.sh" --deploy' "$BATS_TEST_DIRNAME/../setup-linux.sh"
}

@test "setup-linux.sh no longer deploys skills from the removed ai/skills tree" {
    # assert no ACTIVE code path reads the deleted sources (historical mentions
    # in comments are allowed; the deploy is now compile-harness.sh --deploy)
    ! grep -qF '"$CURRENT_DIR/ai/skills/' "$BATS_TEST_DIRNAME/../setup-linux.sh"
    ! grep -qF '"$CURRENT_DIR/ai/opencode/commands' "$BATS_TEST_DIRNAME/../setup-linux.sh"
    ! grep -qF 'OPENCODE_CMDS_SRC=' "$BATS_TEST_DIRNAME/../setup-linux.sh"
}

@test "AC4: injection past the line cap fails (ai/claude/CLAUDE.md)" {
    mkdir -p "$REPO/ai/claude"
    cat > "$REPO/harness/manifest.json" <<'EOF'
{ "version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [ { "id": "demo", "source": "test-pattern.md#1-demo-rule" } ],
  "targets":  [ { "agent": "claude", "kind": "pointer", "file": "ai/claude/CLAUDE.md", "inject": ["demo"] } ] }
EOF
    { printf 'line %d\n' $(seq 1 99); printf '<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n'; } > "$REPO/ai/claude/CLAUDE.md"
    run_refresh
    [ "$status" -ne 0 ]
    [[ "$output" == *"cap"* ]]
}

# --- SDD-008: kind: render (skills), option A (records + render-at-deploy) ---
# --refresh writes committed records ONLY (vault SKILL.md -> harness/skills/).
# --deploy renders records to $HOME per the manifest deploy[] (de-symlinking
# first) and injects the copilot catalog into the $HOME instructions file.
# --check validates each committed record renders cleanly (offline, no vault).

seed_skills_fixture() {
    FAKEHOME="$TMP/home"
    mkdir -p "$FAKEHOME"
    cat > "$REPO/harness/manifest.json" <<'EOF'
{ "version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [], "targets": [],
  "skills": { "vault_subpath": "00_meta/skills", "record_dir": "harness/skills",
    "schema": "harness/skill-frontmatter.schema.json",
    "deploy": [ { "agent": "claude",   "render": "skill",   "dir": ".claude/skills" },
                { "agent": "opencode", "render": "command", "dir": ".config/opencode/commands" },
                { "agent": "agy",      "render": "skill",   "dir": ".gemini/skills" },
                { "agent": "agy",      "render": "prompt",  "dir": ".gemini/prompts" } ],
    "catalog": { "agent": "copilot", "file": ".copilot/copilot-instructions.md" } } }
EOF
    cat > "$REPO/harness/skill-frontmatter.schema.json" <<'EOF'
{ "required": ["name", "description"] }
EOF
    # copilot base file in the fake HOME, with markers --deploy injects into
    mkdir -p "$FAKEHOME/.copilot"
    printf 'intro\n\n## Skills\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n\noutro\n' > "$FAKEHOME/.copilot/copilot-instructions.md"
    mkdir -p "$VAULT/00_meta/skills/demo-skill"
    cat > "$VAULT/00_meta/skills/demo-skill/SKILL.md" <<'EOF'
---
name: demo-skill
description: Demo skill for the render pipeline.
---

# Demo Skill

Body line one.
EOF
}

seed_claude_only() {
    mkdir -p "$VAULT/00_meta/skills/claude-only"
    cat > "$VAULT/00_meta/skills/claude-only/SKILL.md" <<'EOF'
---
name: claude-only
description: Only for claude.
targets: [claude]
---

# Claude Only
EOF
}

run_deploy() { run env HOME="$FAKEHOME" "$SCRIPT" --deploy; }

@test "render: --refresh writes a verbatim record only (no provenance, no \$HOME write)" {
    seed_skills_fixture
    run_refresh
    [ "$status" -eq 0 ]
    [ -f "$REPO/harness/skills/demo-skill/SKILL.md" ]
    # record is a byte-for-byte copy of the vault source; provenance is added at deploy
    diff "$VAULT/00_meta/skills/demo-skill/SKILL.md" "$REPO/harness/skills/demo-skill/SKILL.md"
    ! grep -q '^generated' "$REPO/harness/skills/demo-skill/SKILL.md"
    # option A: --refresh never renders to $HOME
    [ ! -d "$FAKEHOME/.claude" ]
}

@test "render: --deploy renders records to \$HOME with provenance (claude + opencode)" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    # claude native skill keeps name:, carries provenance
    grep -q '^name: demo-skill' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md"
    grep -q '^generated_from: 00_meta/skills/demo-skill/SKILL.md' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md"
    # opencode command drops name:, keeps description + provenance
    [ -f "$FAKEHOME/.config/opencode/commands/demo-skill.md" ]
    ! grep -q '^name:' "$FAKEHOME/.config/opencode/commands/demo-skill.md"
    grep -q '^description:' "$FAKEHOME/.config/opencode/commands/demo-skill.md"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$FAKEHOME/.config/opencode/commands/demo-skill.md"
}

@test "AC1: --deploy replaces a pre-existing vault symlink with a regular copy" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    # simulate the BUG-100 fragility: a vault symlink already in place
    mkdir -p "$FAKEHOME/.claude/skills"
    ln -s "$VAULT/00_meta/skills/demo-skill" "$FAKEHOME/.claude/skills/demo-skill"
    run_deploy; [ "$status" -eq 0 ]
    [ ! -L "$FAKEHOME/.claude/skills/demo-skill" ]
    [ -f "$FAKEHOME/.claude/skills/demo-skill/SKILL.md" ]
    # AC1: no deployed skill path is a symlink
    [ -z "$(find "$FAKEHOME/.claude/skills" "$FAKEHOME/.config/opencode/commands" "$FAKEHOME/.gemini/skills" -type l 2>/dev/null)" ]
}

@test "AC4: --deploy carries auxiliary skill files verbatim (dir-based renders)" {
    seed_skills_fixture
    printf '# reference doc\n' > "$VAULT/00_meta/skills/demo-skill/reference.md"
    run_refresh; [ "$status" -eq 0 ]
    # the record keeps the aux file
    [ -f "$REPO/harness/skills/demo-skill/reference.md" ]
    run_deploy; [ "$status" -eq 0 ]
    # claude (dir render) gets aux file; opencode (single-file command) does not
    [ -f "$FAKEHOME/.claude/skills/demo-skill/reference.md" ]
    [ ! -f "$FAKEHOME/.config/opencode/commands/reference.md" ]
}

@test "render: agy native skill keeps frontmatter; agy flat prompt strips it (AC6)" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    grep -q '^name: demo-skill' "$FAKEHOME/.gemini/skills/demo-skill/SKILL.md"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$FAKEHOME/.gemini/skills/demo-skill/SKILL.md"
    [ -f "$FAKEHOME/.gemini/prompts/demo-skill.md" ]
    ! grep -q '^name:' "$FAKEHOME/.gemini/prompts/demo-skill.md"
    ! grep -q '^description:' "$FAKEHOME/.gemini/prompts/demo-skill.md"
    grep -q 'sha256:' "$FAKEHOME/.gemini/prompts/demo-skill.md"
    grep -q 'Body line one.' "$FAKEHOME/.gemini/prompts/demo-skill.md"
}

@test "render: --deploy injects the copilot catalog into the \$HOME file (AC6)" {
    seed_skills_fixture
    seed_claude_only
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    grep -qF -- '**demo-skill**' "$FAKEHOME/.copilot/copilot-instructions.md"
    grep -qF -- 'Demo skill for the render pipeline.' "$FAKEHOME/.copilot/copilot-instructions.md"
    grep -qE 'BEGIN HARNESS GENERATED \(sha256:[0-9a-f]{16}\)' "$FAKEHOME/.copilot/copilot-instructions.md"
    # claude-only opt-out absent from the copilot catalog
    ! grep -q 'claude-only' "$FAKEHOME/.copilot/copilot-instructions.md"
}

@test "AC6: per-skill targets[] limits which agents receive deployed output" {
    seed_skills_fixture
    seed_claude_only
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    [ -f "$FAKEHOME/.claude/skills/claude-only/SKILL.md" ]
    [ ! -f "$FAKEHOME/.config/opencode/commands/claude-only.md" ]
    [ ! -d "$FAKEHOME/.gemini/skills/claude-only" ]
    [ ! -f "$FAKEHOME/.gemini/prompts/claude-only.md" ]
}

@test "AC1: --deploy prunes an output whose skill dropped this agent from targets[]" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    [ -f "$FAKEHOME/.config/opencode/commands/demo-skill.md" ]
    # demo-skill becomes claude-only; re-refresh + re-deploy must drop the opencode output
    printf -- '---\nname: demo-skill\ndescription: now claude only.\ntargets: [claude]\n---\n\n# Demo\n' > "$VAULT/00_meta/skills/demo-skill/SKILL.md"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    [ ! -f "$FAKEHOME/.config/opencode/commands/demo-skill.md" ]
    [ -f "$FAKEHOME/.claude/skills/demo-skill/SKILL.md" ]
}

@test "AC3: --check validates records render, offline (no vault)" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    run env VAULT_PATH="$TMP/nonexistent" "$SCRIPT" --check
    [ "$status" -eq 0 ]
    [[ "$output" == *"no harness drift"* ]]
}

@test "AC3: --check fails when a committed record has invalid frontmatter" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    printf -- '---\ndescription: no name on purpose.\n---\n\n# x\n' > "$REPO/harness/skills/demo-skill/SKILL.md"
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
    [[ "$output" == *"name"* ]]
}

@test "schema: a skill missing required 'name' fails --refresh with file context (AC5)" {
    seed_skills_fixture
    cat > "$VAULT/00_meta/skills/demo-skill/SKILL.md" <<'EOF'
---
description: Missing name on purpose.
---

# Demo
EOF
    run_refresh
    [ "$status" -ne 0 ]
    [[ "$output" == *"name"* ]]
    [[ "$output" == *"SKILL.md"* ]]
}

@test "schema: unterminated frontmatter fails --refresh (AC5)" {
    seed_skills_fixture
    printf -- '---\nname: x\ndescription: y\n' > "$VAULT/00_meta/skills/demo-skill/SKILL.md"
    run_refresh
    [ "$status" -ne 0 ]
    [[ "$output" == *"frontmatter"* ]]
}
