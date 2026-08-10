#!/usr/bin/env bats
# Tests for AGENTS.md SDD Discipline Gate enforcement (SDD-001)

setup() {
    export DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    export AGENTS_MD="$DOTFILES_DIR/AGENTS.md"
}

@test "AGENTS.md exists" {
    [[ -f "$AGENTS_MD" ]]
}

@test "AGENTS.md states the memory single-sink rule (GUARD-001, AC6)" {
    grep -qF 'MEMORY SINGLE-SINK' "$AGENTS_MD"
    grep -qF 'only' "$AGENTS_MD"
    grep -qF 'Hive is the memory API' "$AGENTS_MD"
    grep -qF 'core.hooksPath' "$AGENTS_MD"
    grep -qF 'MEMORY.md' "$AGENTS_MD"
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
    grep -qF 'dotf spec init' "$AGENTS_MD"
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

# --- CLI-004: Go/shell language boundary (ADR-020, #338) ---

@test "AGENTS.md declares the Language Boundary section" {
    grep -qE '^## Language Boundary \(this repo\)$' "$AGENTS_MD"
}

@test "AGENTS.md language boundary names the two layers and excludes Python" {
    grep -qF 'Python is not a layer here' "$AGENTS_MD"
    grep -qF 'adr-020-tooling-cli-go-convergence' "$AGENTS_MD"
}

@test "AGENTS.md language boundary mandates strangler-fig porting on contact" {
    grep -qF 'Strangler-fig on contact' "$AGENTS_MD"
    grep -qF 'never a new `.sh`/`.ps1` twin' "$AGENTS_MD"
}

@test "AGENTS.md does NOT reintroduce the retired work/personal routing axis" {
    # Negative guard: these markers encode the old axis that mis-routed build/operate
    # artifacts to the vault. Their presence is a regression.
    ! grep -qF 'for work projects' "$AGENTS_MD"
    ! grep -qF '30-architecture/adr' "$AGENTS_MD"
}

# --- HARNESS-064: adversarial-review trigger (#879) ---
#
# CLI-034 bound the review ARTIFACT to `dotf spec archive`; nothing bound the
# MOMENT. These pin the trigger that puts it in the verification window, where a
# reviewer's finding is still cheap to act on. A rule nothing checks is a rule
# that does not fire — which is the defect this whole spec exists to remove, so
# not pinning it would repeat that defect at one remove.

@test "AGENTS.md carries the adversarial-review trigger with its evidence" {
    grep -qF '/adversarial-review' "$AGENTS_MD"
    grep -qF 'dotf spec archive' "$AGENTS_MD"
    grep -qF 'review.md' "$AGENTS_MD"
}

@test "AGENTS.md names the verification window, not just 'before archiving'" {
    grep -qF 'verification window' "$AGENTS_MD"
}

@test "AGENTS.md trigger forbids the implementer supplying their own review" {
    grep -qF 'cannot be the reviewer' "$AGENTS_MD"
}
