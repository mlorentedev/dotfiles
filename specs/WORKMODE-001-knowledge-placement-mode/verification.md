---
id: "WORKMODE-001-knowledge-placement-mode"
type: verification
status: draft
created: "2026-06-02"
template_version: "1.0"
---

# WORKMODE-001 — Verification

| AC | How verified | Evidence |
|---|---|---|
| AC1 | Read SO#2/#3/#7 post-edit; grep AGENTS.md for residual "personal projects → the vault" routing of build/operate classes → none. | manual + grep |
| AC2 | "Document Dynamic" + "Lessons" route ADR/runbook/lesson/troubleshooting → repo `docs/`. | manual read |
| AC3 | `grep -n "for work projects" AGENTS.md` → 0 hits; replaced by "placement-model repo". | grep |
| AC4 | `## Knowledge Placement` block present with defaulted `brain`/`tasks`; removing it changes nothing for dotfiles (defaults == current behavior). | manual |
| AC5 | `bats tests/agents-placement.bats` — RED on `git show HEAD:AGENTS.md` (pre-change), GREEN on working tree. Proves the guard bites. | bats run, both states |
| AC6 | `scripts/compile-harness.sh --check` exit 0; `AGENTS.md` within caps; `<!-- BEGIN HARNESS GENERATED -->` markers intact. | command exit |
| AC7 | Vault `pattern-platform-governance.md` diff shows personal column deferring to placement; linked vault `master` commit SHA. | vault git log |
| AC8 | Epic #162 consumer-4 box `[x]`; #159 + #197 show "Closed" with PR link. | gh issue view |

## Manual smoke (decide-vs-operate routing sanity)
After reconcile, an agent prompt "where do I record a new ADR / a deploy runbook / a cross-project pattern?" must resolve to: ADR → repo `docs/adr/`; runbook → repo `docs/runbooks/`; pattern → vault `00_meta/`. No "depends on personal vs work" hedging for the first two.

## Anti-regression
The kubelab failure mode (personal repo on placement model → lesson routed to vault) must be impossible to derive from the reconciled SO#3. AC5's bats guard pins this.

## Out-of-scope confirmation
No repo artifact migrated; no compile-harness logic added beyond `--check` passing; Windows parity check deferred to the batched Windows session (tracked, not silently dropped).
