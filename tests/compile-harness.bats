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

# Stub `dotf` on PATH so the agent render can resolve a model tier without a Go
# toolchain in this job.
#
# The split is deliberate, not a shortcut. This layer's contract with the
# resolver is "call it, substitute its stdout, honour its exit status", and that
# is what these tests pin. The other half — that the real `dotf harness
# resolve-tier` writes the model id to STDOUT rather than stderr — is pinned in
# cli/internal/cmd/stdout_contract_test.go, where a Go toolchain exists. Neither
# side takes the other's word for it.
#
# Do NOT "fix" this by building dotf here: ADR-020 keeps the Go and shell layers
# on two loops, and the bats job installs no Go.
seed_dotf_stub() {
    STUB_BIN="$TMP/stubbin"
    mkdir -p "$STUB_BIN"
    cat > "$STUB_BIN/dotf" <<'STUB'
#!/usr/bin/env bash
# The capability probe: the render greps this for the subcommand name to decide
# whether the binary is new enough. cli/internal/cmd pins the real help's shape.
if [ "$1 $2" = "harness --help" ]; then
    printf 'Available Commands:\n  resolve-tier          Resolve a neutral model tier\n  resolve-capabilities  Resolve neutral capabilities\n  triggers              x\n'
    exit 0
fi
if [ "$1 $2" = "harness resolve-capabilities" ]; then
    if [ -n "${STUB_CAP_LINE-unset}" ] && [ "${STUB_CAP_LINE-unset}" != "unset" ]; then
        printf '%s\n' "$STUB_CAP_LINE"; exit 0
    fi
    if [ "${STUB_CAP_LINE-unset}" = "unset" ]; then
        printf 'tools: Read, Glob, Bash\n'; exit 0
    fi
    printf 'capability "%s" is not mapped for harness "%s"\n' "$3" "$5" >&2
    exit 1
fi
# Only the subcommands the render calls; anything else is a test bug.
[ "$1 $2" = "harness resolve-tier" ] || { printf 'stub: unexpected: %s\n' "$*" >&2; exit 127; }
if [ -n "${STUB_TIER_MODEL:-}" ]; then
    printf '%s\n' "$STUB_TIER_MODEL"
    exit 0
fi
# Mirrors the real command's failure shape: nothing on stdout, both the tier and
# the harness named on stderr, non-zero exit.
printf 'tier "%s" declares no model for harness "%s"\n' "$3" "$5" >&2
exit 1
STUB
    chmod +x "$STUB_BIN/dotf"
}

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

# --- HARNESS-072: coverage, not just consistency -------------------------------
# The region diff renders its expected side from the target's OWN inject list, so
# an id missing from that list is missing from both sides and the target reports
# OK. These tests pin the separate coverage assertion that does catch it.

seed_second_surface() {
    printf 'intro\n\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n\noutro\n' > "$REPO/TARGET2.md"
}

# A second surface that the `demo` region is NOT injected into. $1 = the JSON
# object for the single `demo` enforced entry, so each test varies only the
# opt_out. Nothing is injected into TARGET2.md and its inject list is empty, so
# the region diff is genuinely consistent there — only coverage has anything to
# say about it.
write_two_surface_manifest() {
    cat > "$REPO/harness/manifest.json" <<EOF
{ "version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [ $1 ],
  "targets":  [ { "agent": "t",  "kind": "native", "file": "TARGET.md",  "inject": ["demo"] },
                { "agent": "t2", "kind": "native", "file": "TARGET2.md", "inject": [] } ] }
EOF
}

@test "HARNESS-072: --check fails when a region reaches one surface but not another" {
    seed_second_surface
    write_two_surface_manifest '{ "id": "demo", "source": "test-pattern.md#1-demo-rule" }'
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
    [[ "$output" == *"GAP"* ]]
    [[ "$output" == *"TARGET2.md"* ]]
    # The point of the guard: the region diff is PERFECTLY HAPPY with TARGET2.md,
    # because it renders what it expects from that target's own (empty) inject
    # list. Consistency says OK; only coverage sees the surface was skipped.
    [[ "$output" == *"[check] OK -> TARGET2.md"* ]]
    # And an orphan check would miss it too — the id is in use on TARGET.md.
    [[ "$output" == *"[check] OK -> TARGET.md"* ]]
}

@test "HARNESS-072: a declared opt_out with a reason satisfies coverage" {
    seed_second_surface
    write_two_surface_manifest '{ "id": "demo", "source": "test-pattern.md#1-demo-rule",
        "opt_out": { "TARGET2.md": "this surface states the rule in hand-written prose" } }'
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check
    [ "$status" -eq 0 ]
    [[ "$output" == *"excluded from TARGET2.md"* ]]
    [[ "$output" == *"hand-written prose"* ]]
}

@test "HARNESS-072: an opt_out with an empty reason is still a gap" {
    seed_second_surface
    write_two_surface_manifest '{ "id": "demo", "source": "test-pattern.md#1-demo-rule",
        "opt_out": { "TARGET2.md": "" } }'
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
    [[ "$output" == *"GAP"* ]]
}

# A manifest carrying a `doctrine` section, so check_coverage's surface list
# includes the reserved "doctrine" key alongside the targets[] files. $1 = the
# JSON for the single `demo` enforced entry, $2 = the doctrine inject array.
#
# Round-2 review finding: the three cases above all use write_two_surface_manifest,
# whose manifest has no doctrine section — so check_coverage's doctrine branch
# (it reads .doctrine.inject rather than a targets[] entry) had no fixture at all.
# The real tree passes because pr-sizing happens to be in doctrine.inject, which
# is precisely the kind of accident that hides a broken branch.
write_doctrine_manifest() {
    cat > "$REPO/harness/manifest.json" <<EOF
{ "version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [ $1 ],
  "targets":  [ { "agent": "t", "kind": "native", "file": "TARGET.md", "inject": ["demo"] } ],
  "doctrine": { "inject": $2,
                "deploy": [ { "agent": "d", "file": ".d/D.md", "char_cap": 12000 } ] } }
EOF
}

@test "HARNESS-072: a region missing from doctrine.inject is a coverage gap" {
    write_doctrine_manifest '{ "id": "demo", "source": "test-pattern.md#1-demo-rule" }' '[]'
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
    [[ "$output" == *"GAP"* ]]
    [[ "$output" == *"doctrine"* ]]
    # Same shape as the targets[] case: the region diff is happy, because
    # TARGET.md does inject `demo` and renders consistently. Only coverage can
    # see that the doctrine payloads were skipped.
    [[ "$output" == *"[check] OK -> TARGET.md"* ]]
}

@test "HARNESS-072: a region present in doctrine.inject satisfies coverage" {
    write_doctrine_manifest '{ "id": "demo", "source": "test-pattern.md#1-demo-rule" }' '["demo"]'
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check
    [ "$status" -eq 0 ]
    [[ "$output" != *"GAP"* ]]
}

@test "HARNESS-072: an opt_out naming the doctrine surface satisfies coverage" {
    write_doctrine_manifest '{ "id": "demo", "source": "test-pattern.md#1-demo-rule",
        "opt_out": { "doctrine": "the compact payload cannot carry the full prose" } }' '[]'
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check
    [ "$status" -eq 0 ]
    [[ "$output" == *"excluded from doctrine"* ]]
    [[ "$output" == *"cannot carry the full prose"* ]]
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

# AC6 (offline harness drift gate) + AC1 (deployed skills are symlink-free)
# moved from healthcheck.sh to cli/internal/doctor (checkHarnessDrift), covered
# by go test (TestCheckHarnessDrift). The structural .sh greps are retired with
# the script; the behavioral `compile-harness.sh --check` gate is still
# exercised end-to-end below.

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
                { "agent": "agy",      "render": "prompt",  "dir": ".gemini/prompts" },
                { "agent": "copilot",  "render": "skill",   "dir": ".copilot/skills" } ],
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

# STUB_TIER_MODEL defaults to the id the shipped map really resolves `top` to for
# claude, so a passing fixture and the real deploy agree on the value.
# A test that wants the unresolvable path sets STUB_TIER_MODEL="" before calling.
run_deploy() {
    run env HOME="$FAKEHOME" PATH="${DEPLOY_PATH:-${STUB_BIN:-}:$PATH}" \
        STUB_TIER_MODEL="${STUB_TIER_MODEL-opus}" \
        STUB_CAP_LINE="${STUB_CAP_LINE-unset}" "$SCRIPT" --deploy
}

@test "render: --refresh stamps the committed record with its own provenance (HARNESS-069), no \$HOME write" {
    seed_skills_fixture
    run_refresh
    [ "$status" -eq 0 ]
    [ -f "$REPO/harness/skills/demo-skill/SKILL.md" ]
    REC="$REPO/harness/skills/demo-skill/SKILL.md"
    # the record's body and name/description are unchanged from the vault source
    grep -q '^name: demo-skill' "$REC"
    grep -qF 'Body line one.' "$REC"
    # but it now says what it was refreshed from — not hand-authored
    grep -q '^generated: true' "$REC"
    grep -q '^generated_from: 00_meta/skills/demo-skill/SKILL.md' "$REC"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$REC"
    # option A: --refresh never renders to $HOME
    [ ! -d "$FAKEHOME/.claude" ]
}

@test "render: --deploy renders records to \$HOME with one set of provenance fields, not stacked (HARNESS-069)" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    # claude native skill keeps name:, carries provenance
    grep -q '^name: demo-skill' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md"
    grep -q '^generated_from: 00_meta/skills/demo-skill/SKILL.md' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md"
    # the record's OWN provenance (added at --refresh) must not survive into the
    # deployed copy alongside deploy's own — exactly one set, describing $HOME's
    # relationship to the record, not two describing two different relationships
    [ "$(grep -c '^generated:' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md")" -eq 1 ]
    [ "$(grep -c '^generated_from:' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md")" -eq 1 ]
    [ "$(grep -c '^generated_sha:' "$FAKEHOME/.claude/skills/demo-skill/SKILL.md")" -eq 1 ]
    # opencode command drops name:, keeps description + provenance
    [ -f "$FAKEHOME/.config/opencode/commands/demo-skill.md" ]
    ! grep -q '^name:' "$FAKEHOME/.config/opencode/commands/demo-skill.md"
    grep -q '^description:' "$FAKEHOME/.config/opencode/commands/demo-skill.md"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$FAKEHOME/.config/opencode/commands/demo-skill.md"
    [ "$(grep -c '^generated_from:' "$FAKEHOME/.config/opencode/commands/demo-skill.md")" -eq 1 ]
    # copilot uses the Agent Skills directory format and keeps the whole record
    grep -q '^name: demo-skill' "$FAKEHOME/.copilot/skills/demo-skill/SKILL.md"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$FAKEHOME/.copilot/skills/demo-skill/SKILL.md"
    [ "$(grep -c '^generated_from:' "$FAKEHOME/.copilot/skills/demo-skill/SKILL.md")" -eq 1 ]
}

@test "skills: --deploy drops neutral/store-only keys (paths, keywords, requires, id, etc.) from native frontmatter" {
    seed_skills_fixture
    mkdir -p "$VAULT/00_meta/skills/full-skill"
    cat > "$VAULT/00_meta/skills/full-skill/SKILL.md" <<'EOF'
---
id: full-skill
type: skill
status: active
created: '2026-05-31'
owner: manu
name: full-skill
description: Full skill with metadata.
allowed-tools: [Bash, Read, Edit, Write]
keywords: [full, test]
paths: ['**/test/**', src/**]
requires: [other-skill]
targets: [claude, opencode]
---

# Full Skill

Body content.
EOF
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]

    F="$FAKEHOME/.claude/skills/full-skill/SKILL.md"
    [ -f "$F" ]
    grep -q '^name: full-skill' "$F"
    grep -q '^description: ' "$F"
    grep -q '^allowed-tools: ' "$F"
    grep -q '^generated: true' "$F"
    grep -q '^generated_from: 00_meta/skills/full-skill/SKILL.md' "$F"

    # neutral/store-only keys must NOT leak into deployed frontmatter
    ! grep -qE '^(id|type|status|created|owner|paths|keywords|requires|targets):' "$F"

    # opencode command drops name, keeps description + allowed-tools, drops neutral keys
    OC="$FAKEHOME/.config/opencode/commands/full-skill.md"
    [ -f "$OC" ]
    ! grep -q '^name:' "$OC"
    grep -q '^description: ' "$OC"
    grep -q '^allowed-tools: ' "$OC"
    ! grep -qE '^(id|type|status|created|owner|paths|keywords|requires|targets):' "$OC"
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
    # dir renders get aux files; opencode (single-file command) does not
    [ -f "$FAKEHOME/.claude/skills/demo-skill/reference.md" ]
    [ -f "$FAKEHOME/.copilot/skills/demo-skill/reference.md" ]
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
    [ ! -d "$FAKEHOME/.copilot/skills/claude-only" ]
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

@test "HARNESS-053: --deploy warns on an unmarked copy of a managed skill and never deletes it" {
    seed_skills_fixture
    # the skill is fenced to another agent, so a re-deploy never overwrites the
    # residue on agy — exactly the case the prune cannot reach
    printf -- '---\nname: demo-skill\ndescription: claude only.\ntargets: [claude]\n---\n\n# Demo\n' \
        > "$VAULT/00_meta/skills/demo-skill/SKILL.md"
    run_refresh; [ "$status" -eq 0 ]
    # residue from a pre-provenance deploy: right name, no `generated:` marker
    mkdir -p "$FAKEHOME/.gemini/skills/demo-skill"
    printf -- '---\nname: demo-skill\ndescription: stale copy.\n---\n\n# Old body\n' \
        > "$FAKEHOME/.gemini/skills/demo-skill/SKILL.md"
    # a third-party skill owning a name we do not manage must stay silent
    mkdir -p "$FAKEHOME/.gemini/skills/vendor-skill"
    printf -- '---\nname: vendor-skill\ndescription: not ours.\n---\n\n# Vendor\n' \
        > "$FAKEHOME/.gemini/skills/vendor-skill/SKILL.md"

    run_deploy; [ "$status" -eq 0 ]
    [[ "$output" == *"WARN unmanaged copy"* ]]
    [[ "$output" == *"demo-skill"* ]]
    [[ "$output" != *"vendor-skill"* ]]
    [ -f "$FAKEHOME/.gemini/skills/vendor-skill/SKILL.md" ]
    # reported, never deleted — the marker is the only proof of ownership
    [ -f "$FAKEHOME/.gemini/skills/demo-skill/SKILL.md" ]
    grep -q 'stale copy' "$FAKEHOME/.gemini/skills/demo-skill/SKILL.md"
}

@test "HARNESS-053: a marked output still prunes when its skill drops the agent" {
    seed_skills_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    [ -d "$FAKEHOME/.gemini/skills/demo-skill" ]
    printf -- '---\nname: demo-skill\ndescription: now claude only.\ntargets: [claude]\n---\n\n# Demo\n' \
        > "$VAULT/00_meta/skills/demo-skill/SKILL.md"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    [ ! -d "$FAKEHOME/.gemini/skills/demo-skill" ]
    [[ "$output" == *"pruned stale"* ]]
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

# --- ENGINE-002: follow-up hardening from the ENGINE-001 adversarial review ---

@test "ENGINE-002: extract_section keeps deeper sub-headings inside an enforced section" {
    # Regression guard for the #156 truncation class: a future deeper sub-heading
    # (###) under an enforced section must NOT cut the rule short. The extractor
    # stops only at a same-or-higher-level heading, so the next ## section is the
    # real boundary; the nested ### and its content stay in the record.
    cat > "$VAULT/00_meta/patterns/test-pattern.md" <<'EOF'
# Test Pattern

## 1. Demo Rule
- rule line one
### 1.1 Nested detail
- nested rule line
- rule line two

## 2. Next Section
- unrelated
EOF
    run_refresh
    [ "$status" -eq 0 ]
    grep -qF '### 1.1 Nested detail' "$REPO/harness/enforced/demo.md"
    grep -qF 'nested rule line'      "$REPO/harness/enforced/demo.md"
    grep -qF 'rule line two'         "$REPO/harness/enforced/demo.md"
    # the next same-level section is the boundary and must not leak in
    ! grep -qF 'unrelated'        "$REPO/harness/enforced/demo.md"
    ! grep -qF 'Next Section'     "$REPO/harness/enforced/demo.md"
}

@test "ENGINE-002: --refresh aborts loudly when the manifest anchor is missing (FM D3)" {
    # Anchor typo / renamed section: the extractor must abort with a clear error
    # rather than silently writing an empty record.
    cat > "$REPO/harness/manifest.json" <<'EOF'
{ "version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [ { "id": "demo", "source": "test-pattern.md#9-does-not-exist" } ],
  "targets":  [ { "agent": "t", "kind": "native", "file": "TARGET.md", "inject": ["demo"] } ] }
EOF
    run_refresh
    [ "$status" -ne 0 ]
    [[ "$output" == *"section"* ]]
    [[ "$output" == *"not found"* ]]
    # no broken record left behind
    [ ! -f "$REPO/harness/enforced/demo.md" ]
}

@test "ENGINE-002: AC6 behavioral - the drift gate dotf doctor wires passes clean, fails tampered" {
    # `dotf doctor` gates on `if compile-harness.sh --check; then pass; else fail`
    # (checkHarnessDrift). Running the full doctor here can't isolate that gate
    # (unrelated tool/vault checks would dominate its exit code), so we exercise the
    # exact command it wires and assert BOTH branches: clean tree -> exit 0 (pass),
    # tampered block -> exit !=0 (fail). Complements TestCheckHarnessDrift (go test).
    run_refresh; [ "$status" -eq 0 ]
    run "$SCRIPT" --check
    [ "$status" -eq 0 ]
    sed -i 's/rule line one/TAMPERED/' "$REPO/TARGET.md"
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
    [[ "$output" == *"DRIFT"* ]]
}

# --- ADR-027 / HARNESS-043: agents (curator dogfood slice) ---
# --refresh writes committed AGENT.md records (vault definitions -> harness/agents).
# --deploy renders each record to its harness-native agent path (agent-md = single
# file) and enforces forced-skill PRESENCE by injecting an AGENT-PRESENCE marked
# region into each harness's always-loaded instructions file (uniform across
# claude/opencode/pi/copilot), coexisting with the patterns region. --check
# validates the record renders offline.

seed_agents_fixture() {
    FAKEHOME="$TMP/home"
    mkdir -p "$FAKEHOME"
    seed_dotf_stub
    cat > "$REPO/harness/manifest.json" <<'EOF'
{ "version": 1, "vault_subpath": "00_meta/patterns",
  "enforced": [], "targets": [],
  "agents": { "vault_subpath": "00_meta/agents/definitions", "record_dir": "harness/agents",
    "schema": "harness/agent-frontmatter.schema.json",
    "deploy": [ { "agent": "claude", "render": "agent-md", "dir": ".claude/agents" } ],
    "presence": [
      { "agent": "claude",   "file": ".claude/CLAUDE.md" },
      { "agent": "opencode", "file": ".config/opencode/AGENTS.md" },
      { "agent": "pi",       "file": ".pi/agent/AGENTS.md" },
      { "agent": "copilot",  "file": ".copilot/copilot-instructions.md" } ] } }
EOF
    cat > "$REPO/harness/agent-frontmatter.schema.json" <<'EOF'
{ "required": ["name", "description", "kind"] }
EOF
    mkdir -p "$VAULT/00_meta/agents/definitions/curator"
    cat > "$VAULT/00_meta/agents/definitions/curator/AGENT.md" <<'EOF'
---
name: curator
description: Crystallize-phase persona.
kind: invocable
model: top
capabilities: [read, search, edit]
skills: [vault-doctor, crystallize, genre-picker]
targets: [claude, opencode, pi, copilot]
---

# Curator

Body line one.
EOF
}

# Seed a harness instructions file with a pre-existing patterns region + user
# content, so a presence injection must coexist with (never disturb) both.
seed_instructions_file() {
    mkdir -p "$(dirname "$1")"
    printf 'user intro\n\n<!-- BEGIN HARNESS GENERATED -->\npatterns content\n<!-- END HARNESS GENERATED -->\n\nuser outro\n' > "$1"
}

# HARNESS-054: the two surfaces that cannot take a full instructions file get the
# compact doctrine payload instead. Extends the agents fixture with a doctrine
# block and the enforced record its payload renders from.
seed_doctrine_fixture() {
    seed_agents_fixture
    # a persona with no targets[] is universal, so presence reaches every surface
    # including the two this fixture adds
    sed -i '/^targets: /d' "$VAULT/00_meta/agents/definitions/curator/AGENT.md"
    mkdir -p "$REPO/harness/enforced"
    printf -- '- rule one\n- rule two\n' > "$REPO/harness/enforced/demo.md"
    local tmp
    tmp="$(mktemp)"
    jq '.doctrine = {
          "inject": ["demo"],
          "deploy": [
            { "agent": "agy",   "file": ".gemini/GEMINI.md",  "char_cap": 12000 },
            { "agent": "codex", "file": ".codex/AGENTS.md",
              "shadowed_by": ".codex/AGENTS.override.md", "char_cap": 32768 } ] }' \
        "$REPO/harness/manifest.json" > "$tmp" && mv "$tmp" "$REPO/harness/manifest.json"
}

@test "HARNESS-054: --deploy creates the doctrine file for a surface that has none" {
    seed_doctrine_fixture
    run_refresh; [ "$status" -eq 0 ]
    [ ! -f "$FAKEHOME/.gemini/GEMINI.md" ]
    [ ! -f "$FAKEHOME/.codex/AGENTS.md" ]
    run_deploy; [ "$status" -eq 0 ]
    for f in "$FAKEHOME/.gemini/GEMINI.md" "$FAKEHOME/.codex/AGENTS.md"; do
        [ -f "$f" ]
        grep -q 'rule one' "$f"                      # enforced rules travelled
        grep -q 'MUST consume' "$f"                   # presence travelled
        grep -q 'BEGIN HARNESS GENERATED' "$f"
    done
}

@test "HARNESS-054: doctrine injection preserves user content and is idempotent" {
    seed_doctrine_fixture
    run_refresh; [ "$status" -eq 0 ]
    mkdir -p "$FAKEHOME/.gemini"
    printf 'my own gemini rules\n' > "$FAKEHOME/.gemini/GEMINI.md"
    run_deploy; [ "$status" -eq 0 ]
    grep -q 'my own gemini rules' "$FAKEHOME/.gemini/GEMINI.md"
    local before
    before="$(md5sum < "$FAKEHOME/.gemini/GEMINI.md")"
    run_deploy; [ "$status" -eq 0 ]
    [ "$(md5sum < "$FAKEHOME/.gemini/GEMINI.md")" = "$before" ]
    [ "$(grep -c 'BEGIN HARNESS GENERATED' "$FAKEHOME/.gemini/GEMINI.md")" -eq 1 ]
    grep -q 'my own gemini rules' "$FAKEHOME/.gemini/GEMINI.md"
}

@test "HARNESS-054: a file over the platform's documented cap warns" {
    seed_doctrine_fixture
    run_refresh; [ "$status" -eq 0 ]
    mkdir -p "$FAKEHOME/.gemini"
    head -c 12500 /dev/zero | tr '\0' 'x' > "$FAKEHOME/.gemini/GEMINI.md"
    run_deploy; [ "$status" -eq 0 ]
    [[ "$output" == *"over the 12000"* ]]
}

@test "HARNESS-054: a shadow file that wins at read time warns" {
    seed_doctrine_fixture
    run_refresh; [ "$status" -eq 0 ]
    mkdir -p "$FAKEHOME/.codex"
    printf 'override wins\n' > "$FAKEHOME/.codex/AGENTS.override.md"
    run_deploy; [ "$status" -eq 0 ]
    [[ "$output" == *"shadows"* ]]
    [[ "$output" == *"never read"* ]]
}

@test "HARNESS-054: every declared agent surface carries a generated region" {
    seed_doctrine_fixture
    for f in ".claude/CLAUDE.md" ".config/opencode/AGENTS.md" ".pi/agent/AGENTS.md" ".copilot/copilot-instructions.md"; do
        seed_instructions_file "$FAKEHOME/$f"
    done
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    # presence surfaces + doctrine surfaces, together, with nothing declared and unserved
    local f
    while read -r f; do
        [ -f "$FAKEHOME/$f" ] || { echo "declared surface never created: $f"; return 1; }
        grep -q 'HARNESS' "$FAKEHOME/$f" || { echo "declared surface carries no region: $f"; return 1; }
    done < <(jq -r '(.agents.presence[]?.file), (.doctrine.deploy[]?.file)' "$REPO/harness/manifest.json")
}

@test "agents: --refresh stamps the committed AGENT.md record with its own provenance (HARNESS-069), no \$HOME" {
    seed_agents_fixture
    run_refresh
    [ "$status" -eq 0 ]
    REC="$REPO/harness/agents/curator/AGENT.md"
    [ -f "$REC" ]
    grep -q '^name: curator' "$REC"
    grep -q '^generated: true' "$REC"
    grep -q '^generated_from: 00_meta/agents/definitions/curator/AGENT.md' "$REC"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$REC"
    [ ! -d "$FAKEHOME/.claude/agents" ]
}

@test "agents: --deploy renders agent-md (name+description+provenance; neutral keys dropped)" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    F="$FAKEHOME/.claude/agents/curator.md"
    [ -f "$F" ]
    grep -q '^name: curator' "$F"
    grep -q '^description: ' "$F"
    grep -qE '^generated_sha: [0-9a-f]{16}' "$F"
    grep -q '^generated_from: 00_meta/agents/definitions/curator/AGENT.md' "$F"
    grep -qF 'Body line one.' "$F"
    # neutral-only / deferred keys must NOT leak into the native agent frontmatter.
    # `model` is NO LONGER one of them: it is resolved, not dropped (see the tier
    # tests below), so removing it from this list is the behaviour change.
    ! grep -qE '^(kind|capabilities|skills|targets):' "$F"
    # the record's own provenance (HARNESS-069, added at --refresh) must not
    # survive alongside deploy's own — render_agent's name/description-only
    # passthrough already drops it, but pin that behavior explicitly
    [ "$(grep -c '^generated:' "$F")" -eq 1 ]
    [ "$(grep -c '^generated_from:' "$F")" -eq 1 ]
    [ "$(grep -c '^generated_sha:' "$F")" -eq 1 ]
}

@test "agents: --deploy resolves the neutral model tier into the rendered frontmatter" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    F="$FAKEHOME/.claude/agents/curator.md"
    # the record declares `model: top`; the deployed file must name the harness's
    # model id, never the neutral tier — a harness reading `model: top` would ask
    # its provider for a model called "top"
    grep -q '^model: opus' "$F"
    ! grep -q '^model: top' "$F"
    # exactly one model line: the resolved one replaces the record's, never joins it
    [ "$(grep -c '^model:' "$F")" -eq 1 ]
}

@test "agents: --deploy passes the record's own tier through to the resolver" {
    seed_agents_fixture
    sed -i 's/^model: top$/model: low/' "$VAULT/00_meta/agents/definitions/curator/AGENT.md"
    run_refresh; [ "$status" -eq 0 ]
    STUB_TIER_MODEL=haiku run_deploy; [ "$status" -eq 0 ]
    grep -q '^model: haiku' "$FAKEHOME/.claude/agents/curator.md"
}

@test "agents: the resolver's own diagnosis reaches the deploy output, not just a generic tier error" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    # An unreadable MAP and an undeclared TIER both fail the render, and only the
    # resolver can tell them apart. Swallowing its stderr made a schema-invalid
    # map read as "tier top does not resolve", blaming the record for a defect in
    # the registry. The stub stands in for the resolver naming a real cause.
    cat > "$STUB_BIN/dotf" <<'DIAG'
#!/usr/bin/env bash
if [ "$1 $2" = "harness --help" ]; then
    printf 'Available Commands:\n  resolve-tier  Resolve a neutral model tier\n'
    exit 0
fi
printf 'chains.top[0] names "ghost" - the pools block does not declare it\n' >&2
exit 1
DIAG
    chmod +x "$STUB_BIN/dotf"
    run_deploy
    [ "$status" -ne 0 ]
    # the CAUSE survives, not only the generic wrapper
    [[ "$output" == *ghost* ]]
    # and the wrapper must not assert a cause it cannot know
    [[ "$output" != *"does not resolve for harness"* ]]
}

@test "agents: the rendered file keeps umask permissions, not a temp file's 0600" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    F="$FAKEHOME/.claude/agents/curator.md"
    # The render now writes through a temp file. Staging it in $TMPDIR with
    # mktemp would deploy 0600, differing from every other deployed artifact;
    # a sibling temp created by the same redirect keeps the umask result.
    # Compared against a file the same deploy wrote by plain redirect, so this
    # pins "same as everything else" rather than a hardcoded mode.
    printf 'probe\n' > "$FAKEHOME/.probe-umask"
    [ "$(stat -c '%a' "$F")" = "$(stat -c '%a' "$FAKEHOME/.probe-umask")" ]
}

@test "agents: a failed render leaves no temp file beside the target" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    STUB_TIER_MODEL="" run_deploy
    [ "$status" -ne 0 ]
    # the sibling temp lives in the deploy dir, so a leak would be visible to
    # whatever reads that directory looking for agent definitions
    run bash -c "ls '$FAKEHOME/.claude/agents/' | grep -c '\.tmp\.'"
    [ "$output" = "0" ]
}

@test "agents: a record declaring no tier renders without a model line" {
    seed_agents_fixture
    sed -i '/^model: /d' "$VAULT/00_meta/agents/definitions/curator/AGENT.md"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    F="$FAKEHOME/.claude/agents/curator.md"
    [ -f "$F" ]
    grep -q '^name: curator' "$F"
    # not declaring a tier is not an error — it renders as it always did
    ! grep -q '^model:' "$F"
}

@test "agents: an unresolvable tier fails the deploy instead of rendering a model-less definition" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    STUB_TIER_MODEL="" run_deploy
    [ "$status" -ne 0 ]
    # the operator must learn WHICH tier and WHICH harness could not be resolved
    [[ "$output" == *top* ]]
    [[ "$output" == *claude* ]]
    # and the render must not leave a truncated definition behind: a file naming
    # no model is the exact degrade the resolution was added to prevent
    [ ! -f "$FAKEHOME/.claude/agents/curator.md" ]
}

@test "agents: an unresolvable tier leaves the PREVIOUS agent definition intact" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    F="$FAKEHOME/.claude/agents/curator.md"
    grep -q '^model: opus' "$F"
    # a later deploy whose resolution fails must not truncate what is already there
    STUB_TIER_MODEL="" run_deploy
    [ "$status" -ne 0 ]
    [ -s "$F" ]
    grep -q '^model: opus' "$F"
}

@test "agents: an absent dotf warns and renders without a model line, it does not fail the deploy" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    # Genuinely absent, which means the developer's own ~/.local/bin must be off
    # PATH too: leaving it on would find the REAL dotf and test something else
    # entirely, passing or failing for reasons unrelated to the branch under
    # test. System dirs only, which is also what CI has.
    rm -f "$STUB_BIN/dotf"
    DEPLOY_PATH="$STUB_BIN:/usr/local/bin:/usr/bin:/bin" run_deploy
    # NOT fatal: setup-linux.sh installs dotf best-effort, so a missing resolver
    # must not take the whole harness deploy down with it. C15 governs a map that
    # cannot be READ; an absent binary is a bootstrap state.
    [ "$status" -eq 0 ]
    [[ "$output" == *dotf* ]]
    F="$FAKEHOME/.claude/agents/curator.md"
    [ -f "$F" ]
    grep -q '^name: curator' "$F"
    # degraded exactly to the pre-change behaviour — never worse than status quo
    ! grep -q '^model:' "$F"
}

@test "agents: a dotf too old to know resolve-tier warns rather than embedding its help output" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    # Cobra runs a parent command's own RunE for an unrecognised subcommand, so
    # `dotf harness resolve-tier` on a binary predating it prints the harness help
    # and exits 0. A zero status alone would make that help text the model id.
    # This is #1158's class: the deployed dotf routinely predates the tree.
    cat > "$STUB_BIN/dotf" <<'STALE'
#!/usr/bin/env bash
# A binary that knows `harness` but not `resolve-tier`. Its help omits the
# subcommand, and — measured against the real stale binary on 2026-08-21 — it
# rejects the unknown flag with exit 1, the SAME status a genuine routing refusal
# returns. Only the capability probe separates them.
if [ "$1 $2" = "harness --help" ]; then
    printf 'Available Commands:\n  suggest   x\n  triggers  x\n'
    exit 0
fi
printf 'Error: unknown flag: --harness\n' >&2
exit 1
STALE
    chmod +x "$STUB_BIN/dotf"
    run_deploy
    [ "$status" -eq 0 ]
    [[ "$output" == *"predates"* ]]
    F="$FAKEHOME/.claude/agents/curator.md"
    [ -f "$F" ]
    # the help screen must not have become the model id
    ! grep -q '^model:' "$F"
    ! grep -q 'Usage:' "$F"
}

@test "agents: skill deploy survives a failed agent render" {
    seed_agents_fixture
    # give the fixture a skill too, so both halves of --deploy have work to do
    mkdir -p "$VAULT/00_meta/skills/demo-skill"
    cat > "$VAULT/00_meta/skills/demo-skill/SKILL.md" <<'EOF'
---
name: demo-skill
description: A demo skill.
render: skill
---

Skill body.
EOF
    local tmp
    tmp="$(mktemp)"
    jq '.skills = { "vault_subpath": "00_meta/skills", "record_dir": "harness/skills",
                    "schema": "harness/skill-frontmatter.schema.json",
                    "deploy": [ { "agent": "claude", "dir": ".claude/skills" } ] }' \
        "$REPO/harness/manifest.json" > "$tmp" && mv "$tmp" "$REPO/harness/manifest.json"
    cat > "$REPO/harness/skill-frontmatter.schema.json" <<'EOF'
{ "required": ["name", "description"] }
EOF
    run_refresh; [ "$status" -eq 0 ]
    STUB_TIER_MODEL="" run_deploy
    [ "$status" -ne 0 ]
    # deploy_skills runs BEFORE deploy_agents, so a failed agent render must not
    # cost the skills their deploy — the ordering in cmd_deploy is load-bearing
    [ -f "$FAKEHOME/.claude/skills/demo-skill/SKILL.md" ]
}

@test "agents: --deploy resolves neutral capabilities into the rendered frontmatter" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    F="$FAKEHOME/.claude/agents/curator.md"
    # the record declares `capabilities: [read, search, edit]`; the deployed file
    # must carry the harness's NATIVE tool names, never the neutral verbs
    grep -q '^tools: Read, Glob, Bash' "$F"
    ! grep -qE '^capabilities:' "$F"
    [ "$(grep -c '^tools:' "$F")" -eq 1 ]
}

@test "agents: a record declaring no capabilities renders without the field" {
    seed_agents_fixture
    sed -i '/^capabilities: /d' "$VAULT/00_meta/agents/definitions/curator/AGENT.md"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    F="$FAKEHOME/.claude/agents/curator.md"
    [ -f "$F" ]
    ! grep -q '^tools:' "$F"
    # the model line is independent and must still be there
    grep -q '^model: opus' "$F"
}

@test "agents: an unmappable capability fails the deploy without truncating the definition" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    F="$FAKEHOME/.claude/agents/curator.md"
    STUB_CAP_LINE="" run_deploy
    [ "$status" -ne 0 ]
    [[ "$output" == *capabilit* ]]
    # the previous definition survives, same guarantee as the tier path
    [ -s "$F" ]
    grep -q '^tools: Read, Glob, Bash' "$F"
}

@test "agents: a dotf that knows resolve-tier but not resolve-capabilities warns for that field only" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    # The realistic staleness shape once two subcommands exist: a binary new
    # enough for one and not the other. Each field probes independently, so the
    # model line must still resolve.
    cat > "$STUB_BIN/dotf" <<'HALF'
#!/usr/bin/env bash
if [ "$1 $2" = "harness --help" ]; then
    printf 'Available Commands:\n  resolve-tier  Resolve a neutral model tier\n'
    exit 0
fi
[ "$1 $2" = "harness resolve-tier" ] || { printf 'Error: unknown flag\n' >&2; exit 1; }
printf 'opus\n'
HALF
    chmod +x "$STUB_BIN/dotf"
    run_deploy
    [ "$status" -eq 0 ]
    [[ "$output" == *"predates the resolve-capabilities"* ]]
    F="$FAKEHOME/.claude/agents/curator.md"
    grep -q '^model: opus' "$F"
    ! grep -q '^tools:' "$F"
}

@test "BUG-1168: a whitespace-bearing model id blames the map, not a stale dotf" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    # The resolver ran and answered; the answer is just not a model id. Reporting
    # "dotf is too old" sent the operator to rebuild a binary that was fine.
    STUB_TIER_MODEL="opus 4" run_deploy
    [ "$status" -ne 0 ]
    [[ "$output" == *"model-map.json"* ]]
    [[ "$output" == *"opus 4"* ]]
    [[ "$output" != *"predates the resolve-tier"* ]]
}

@test "DESIGN-1169: a bad record does not block the agents behind it" {
    seed_agents_fixture
    # a second persona on a DIFFERENT tier, so the stub can refuse exactly one
    mkdir -p "$VAULT/00_meta/agents/definitions/scribe"
    printf -- '---\nname: scribe\ndescription: second persona.\nkind: invocable\nmodel: mid\n---\n\n# Scribe\n' \
        > "$VAULT/00_meta/agents/definitions/scribe/AGENT.md"
    run_refresh; [ "$status" -eq 0 ]
    cat > "$STUB_BIN/dotf" <<'ONEBAD'
#!/usr/bin/env bash
if [ "$1 $2" = "harness --help" ]; then
    printf 'Available Commands:\n  resolve-tier          x\n  resolve-capabilities  x\n'
    exit 0
fi
if [ "$1 $2" = "harness resolve-capabilities" ]; then printf 'tools: Read\n'; exit 0; fi
if [ "$1 $2" = "harness resolve-tier" ]; then
    # refuse `mid` only: scribe fails, curator (top) resolves
    [ "$3" = "mid" ] && { printf 'tier "mid" declares no model for harness "claude"\n' >&2; exit 1; }
    printf 'opus\n'; exit 0
fi
exit 127
ONEBAD
    chmod +x "$STUB_BIN/dotf"
    run_deploy
    # non-zero, because one record genuinely failed
    [ "$status" -ne 0 ]
    # ...and the GOOD record still deployed. Before #1169 this depended on
    # directory iteration order: `curator` sorts before `scribe`, so aborting on
    # the first failure would have left curator deployed and any record after
    # scribe silently skipped.
    [ -f "$FAKEHOME/.claude/agents/curator.md" ]
    grep -q '^model: opus' "$FAKEHOME/.claude/agents/curator.md"
    # the failing one wrote nothing, not even a truncated file
    [ ! -f "$FAKEHOME/.claude/agents/scribe.md" ]
    [[ "$output" == *scribe* ]]
}

@test "DESIGN-1169: every failing record is named in one run, not just the first" {
    seed_agents_fixture
    mkdir -p "$VAULT/00_meta/agents/definitions/scribe"
    printf -- '---\nname: scribe\ndescription: second persona.\nkind: invocable\nmodel: top\n---\n\n# Scribe\n' \
        > "$VAULT/00_meta/agents/definitions/scribe/AGENT.md"
    run_refresh; [ "$status" -eq 0 ]
    STUB_TIER_MODEL="" run_deploy
    [ "$status" -ne 0 ]
    # BOTH records appear, so one deploy tells the operator the whole story
    [[ "$output" == *curator* ]]
    [[ "$output" == *scribe* ]]
    # ...and when NONE survived, the summary says so. Claiming "others deployed"
    # here would be false, and the operator decides whether to roll back on
    # exactly this sentence.
    [[ "$output" == *"NO agent deployed"* ]]
}

@test "agents: the capability list parses identically under bash and zsh" {
    # `${caps#[}` unescaped is a bracket expression that never closes: bash
    # strips the literal, zsh aborts with `bad pattern: [` at RUN time. `zsh -n`
    # does not catch it, because it is a pattern error rather than a syntax one —
    # which is why this is a behavioural test and not another lint.
    local expr='caps="[read, search, edit]"; caps="${caps#\[}"; caps="${caps%\]}"; caps="${caps// /}"; printf "%s" "$caps"'
    local from_bash from_zsh
    from_bash="$(bash -c "$expr")"
    [ "$from_bash" = "read,search,edit" ]
    if command -v zsh >/dev/null 2>&1; then
        from_zsh="$(zsh -c "$expr")"
        [ "$from_zsh" = "$from_bash" ]
    fi
}

@test "agents: --deploy injects a presence region (forced skills) into every harness instructions file" {
    seed_agents_fixture
    seed_instructions_file "$FAKEHOME/.claude/CLAUDE.md"
    seed_instructions_file "$FAKEHOME/.config/opencode/AGENTS.md"
    seed_instructions_file "$FAKEHOME/.pi/agent/AGENTS.md"
    seed_instructions_file "$FAKEHOME/.copilot/copilot-instructions.md"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    for f in "$FAKEHOME/.claude/CLAUDE.md" "$FAKEHOME/.config/opencode/AGENTS.md" \
             "$FAKEHOME/.pi/agent/AGENTS.md" "$FAKEHOME/.copilot/copilot-instructions.md"; do
        # presence region present, naming the persona + its forced skills
        grep -q 'BEGIN HARNESS AGENT-PRESENCE' "$f"
        grep -q 'END HARNESS AGENT-PRESENCE' "$f"
        grep -q 'curator' "$f"
        grep -q 'vault-doctor' "$f"
        # the pre-existing patterns region + user content survive untouched
        grep -q 'patterns content' "$f"
        grep -q 'user intro' "$f"
        grep -q 'user outro' "$f"
    done
}

@test "agents: presence injection is idempotent and leaves the patterns region intact" {
    seed_agents_fixture
    F="$FAKEHOME/.config/opencode/AGENTS.md"
    seed_instructions_file "$F"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    # exactly one presence region after two deploys (no accumulation)
    [ "$(grep -c 'BEGIN HARNESS AGENT-PRESENCE' "$F")" -eq 1 ]
    [ "$(grep -c 'END HARNESS AGENT-PRESENCE' "$F")" -eq 1 ]
    # the patterns region is still single and intact
    [ "$(grep -c 'BEGIN HARNESS GENERATED' "$F")" -eq 1 ]
    grep -q 'patterns content' "$F"
}

@test "agents: --check validates the record renders offline (no vault)" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    run env VAULT_PATH="$TMP/nonexistent" "$SCRIPT" --check
    [ "$status" -eq 0 ]
    [[ "$output" == *"no harness drift"* ]]
}

@test "agents: --check fails when a record is missing a required key (kind)" {
    seed_agents_fixture
    run_refresh; [ "$status" -eq 0 ]
    printf -- '---\nname: curator\ndescription: no kind on purpose.\n---\n\n# x\n' > "$REPO/harness/agents/curator/AGENT.md"
    run "$SCRIPT" --check
    [ "$status" -ne 0 ]
    [[ "$output" == *"kind"* ]]
}

@test "agents: per-agent targets[] excludes a non-targeted harness from deploy" {
    seed_agents_fixture
    mkdir -p "$VAULT/00_meta/agents/definitions/scribe"
    printf -- '---\nname: scribe\ndescription: opencode only.\nkind: invocable\ntargets: [opencode]\n---\n\n# Scribe\n' > "$VAULT/00_meta/agents/definitions/scribe/AGENT.md"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    [ -f "$FAKEHOME/.claude/agents/curator.md" ]
    [ ! -f "$FAKEHOME/.claude/agents/scribe.md" ]
}

@test "agents: presence appends a fresh region when the file has no presence markers" {
    seed_agents_fixture
    F="$FAKEHOME/.pi/agent/AGENTS.md"
    mkdir -p "$(dirname "$F")"
    printf 'just user content, no markers\n' > "$F"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    grep -q 'just user content' "$F"
    grep -q 'BEGIN HARNESS AGENT-PRESENCE' "$F"
    grep -q 'vault-doctor' "$F"
}

@test "agents: presence respects per-agent targets[] (a persona only appears for harnesses it targets)" {
    seed_agents_fixture
    mkdir -p "$VAULT/00_meta/agents/definitions/scribe"
    printf -- '---\nname: scribe\ndescription: opencode only.\nkind: invocable\nskills: [docs-skill]\ntargets: [opencode]\n---\n\n# Scribe\n' > "$VAULT/00_meta/agents/definitions/scribe/AGENT.md"
    seed_instructions_file "$FAKEHOME/.claude/CLAUDE.md"
    seed_instructions_file "$FAKEHOME/.config/opencode/AGENTS.md"
    run_refresh; [ "$status" -eq 0 ]
    run_deploy; [ "$status" -eq 0 ]
    # claude: curator targets it, scribe does not
    grep -q 'curator' "$FAKEHOME/.claude/CLAUDE.md"
    ! grep -q 'scribe' "$FAKEHOME/.claude/CLAUDE.md"
    # opencode: both personas target it
    grep -q 'curator' "$FAKEHOME/.config/opencode/AGENTS.md"
    grep -q 'scribe' "$FAKEHOME/.config/opencode/AGENTS.md"
}
