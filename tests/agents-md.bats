#!/usr/bin/env bats
# Tests for AGENTS.md SDD Discipline Gate enforcement (SDD-001)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export AGENTS_MD="$DOTFILES_DIR/AGENTS.md"
}

@test "AGENTS.md exists" {
    [[ -f "$AGENTS_MD" ]]
}

@test "AGENTS.md has Spec-Driven Development H2 section (pre-existing, regression guard)" {
    grep -qE '^## Spec-Driven Development$' "$AGENTS_MD"
}

# --- SDD-001: Discipline Gate enforcement ---

@test "AGENTS.md has Discipline Gate H3 subsection" {
    grep -qE '^### Discipline Gate \(NON-NEGOTIABLE\)$' "$AGENTS_MD"
}

@test "AGENTS.md Discipline Gate enumerates all 5 trigger criteria" {
    grep -qF '50' "$AGENTS_MD"
    grep -qF '300 LOC' "$AGENTS_MD"
    grep -qF 'public contract' "$AGENTS_MD"
    grep -qF 'dependency' "$AGENTS_MD"
    grep -qF 'multi-PR sequence' "$AGENTS_MD"
    grep -qF 'Socratic Guardrail pause' "$AGENTS_MD"
}

@test "AGENTS.md Discipline Gate documents the mandatory ordered process" {
    grep -qF '11-tasks.md' "$AGENTS_MD"
    grep -qF 'init-spec' "$AGENTS_MD"
    grep -qF 'proposal.md' "$AGENTS_MD"
    grep -qF 'tasks.md' "$AGENTS_MD"
    grep -qF 'verification.md' "$AGENTS_MD"
}

@test "AGENTS.md Discipline Gate has banned-phrases list for knowledge hygiene" {
    grep -qF 'Banned phrases' "$AGENTS_MD"
    grep -qF "knowledge hygiene later" "$AGENTS_MD"
    grep -qF "spec entry after merge" "$AGENTS_MD"
    grep -qF "commit first and document later" "$AGENTS_MD"
}

@test "AGENTS.md Discipline Gate references Standing Order 3 (in-session, not later)" {
    grep -qF "in-session, not 'later'" "$AGENTS_MD"
    grep -qF 'Standing Order' "$AGENTS_MD"
}

# --- WORKMODE-001: decide-vs-operate knowledge placement (incident->guard, #197/#159) ---
# The kubelab regression (a personal+placement repo whose lesson got routed back to
# the vault by the retired work/personal axis) is the incident; these are its guards.

@test "AGENTS.md SO#2 states decide-vs-operate as the placement discriminator" {
    grep -qF 'Knowledge placement is by layer' "$AGENTS_MD"
}

@test "AGENTS.md has the per-repo Knowledge Placement declaration (brain/tasks)" {
    grep -qE '^## Knowledge Placement' "$AGENTS_MD"
    grep -qF 'brain:' "$AGENTS_MD"
    grep -qF 'tasks:' "$AGENTS_MD"
}

@test "AGENTS.md routes build/operate artifacts to the repo, not the vault" {
    grep -qF 'docs/troubleshooting/' "$AGENTS_MD"
    grep -qF 'docs/adr/' "$AGENTS_MD"
    grep -qF 'docs/lessons.md' "$AGENTS_MD"
}

@test "AGENTS.md does NOT reintroduce the retired work/personal routing axis" {
    # Negative guard: these markers encode the old axis that mis-routed build/operate
    # artifacts to the vault. Their presence is a regression.
    ! grep -qF 'for work projects' "$AGENTS_MD"
    ! grep -qF '30-architecture/adr' "$AGENTS_MD"
}
