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

@test "setup-linux.sh runs compile-harness --refresh during deploy" {
    grep -q 'compile-harness.sh" --refresh' "$BATS_TEST_DIRNAME/../setup-linux.sh"
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

# --- SDD-008: kind: render (skills) ---

seed_skills_fixture() {
    cat > "$REPO/harness/manifest.json" <<'EOF'
{ "version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [], "targets": [],
  "skills": { "vault_subpath": "00_meta/skills", "record_dir": "harness/skills",
    "agents": [ { "agent": "claude",   "render": "skill",   "out_dir": "out/claude" },
                { "agent": "opencode", "render": "command", "out_dir": "out/opencode" } ] } }
EOF
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

@test "render: --refresh renders a vault skill to claude + opencode with provenance" {
    seed_skills_fixture
    run_refresh
    [ "$status" -eq 0 ]
    [ -f "$REPO/harness/skills/demo-skill/SKILL.md" ]
    # claude output keeps name:, carries provenance
    grep -q '^name: demo-skill' "$REPO/out/claude/demo-skill/SKILL.md"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$REPO/out/claude/demo-skill/SKILL.md"
    grep -q '^generated_from: 00_meta/skills/demo-skill/SKILL.md' "$REPO/out/claude/demo-skill/SKILL.md"
    # opencode command drops name:, keeps description + provenance
    [ -f "$REPO/out/opencode/demo-skill.md" ]
    ! grep -q '^name:' "$REPO/out/opencode/demo-skill.md"
    grep -q '^description:' "$REPO/out/opencode/demo-skill.md"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$REPO/out/opencode/demo-skill.md"
}

@test "render: --check passes after refresh, fails after a rendered output is hand-edited" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check; [ "$status" -eq 0 ]
    sed -i 's/Body line one./TAMPERED/' "$REPO/out/claude/demo-skill/SKILL.md"
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
    [[ "$output" == *"DRIFT"* ]]
}

@test "render: --check works offline (no vault) from the committed record" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    run env VAULT_PATH="$TMP/nonexistent" "$SCRIPT" --check
    [ "$status" -eq 0 ]
}

@test "render: per-skill targets[] limits which agents receive output" {
    seed_skills_fixture
    mkdir -p "$VAULT/00_meta/skills/claude-only"
    cat > "$VAULT/00_meta/skills/claude-only/SKILL.md" <<'EOF'
---
name: claude-only
description: Only for claude.
targets: [claude]
---

# Claude Only
EOF
    run_refresh; [ "$status" -eq 0 ]
    [ -f "$REPO/out/claude/claude-only/SKILL.md" ]
    [ ! -f "$REPO/out/opencode/claude-only.md" ]
}
