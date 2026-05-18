---
tags: [spec, tasks]
created: "2026-05-18"
---

# Tasks - SDD-001-discipline-gate

> TDD order. Scope of THIS PR = Tier 1 (AGENTS.md rule) + Tier 2 (hook nudge). Tiers 3-5 ship in sibling specs SDD-002 / SDD-003. One spec equals one atomic PR per `pattern-spec-driven-development.md`.

## Setup

- [x] Vault entry exists in `10_projects/dotfiles/11-tasks.md` (added 2026-05-18 same session; pending commit by user in vault repo)
- [x] Branch created from main: `feat/SDD-001-discipline-gate`
- [x] Spec scaffolded via `scripts/init-spec.ps1 SDD-001-discipline-gate` (vault gate passed)
- [x] `proposal.md` complete; acceptance criteria are testable
- [ ] Resolve anchor question in `proposal.md` "Risks / open questions" before locking the hook reminder text (decision: include `#sdd-discipline-gate-non-negotiable` anchor)

## Implementation (TDD)

### AGENTS.md rule (Tier 1)

- [ ] Write failing bats test `tests/agents-md.bats` (new file) asserting: AGENTS.md exists, contains H2 `## SDD Discipline Gate (NON-NEGOTIABLE)`, enumerates trigger criteria + skip criteria + banned-phrases block
- [ ] Edit `AGENTS.md` to add the section. Keep ≤30 lines. Link to `[[pattern-spec-driven-development]]` for full rationale, do not duplicate the pattern body
- [ ] Verify bats green (manual grep simulation while bats not local)

### Hook reminder Windows (Tier 2 .ps1)

- [ ] Add failing bats assert in `tests/hooks.bats` (new file): `.ps1` contains literal SDD reminder text + `[sdd]` prefix marker
- [ ] Edit `scripts/claude-session-start.ps1` to inject the reminder at the START of `$ContextLines` (right after line 37 `$ContextLines = ''`, before claude-mem heal block at line 42). Non-conditional — runs every session regardless of CWD / vault / repo state. Use `[sdd]` prefix to match `[claude-mem]` / `[doctor]` / `[specs]` convention
- [ ] Local smoke: `echo '{"cwd":"C:\\test","session_id":"test"}' | pwsh -NoProfile -File scripts/claude-session-start.ps1` -- output JSON's `hookSpecificOutput.additionalContext` STARTS with the SDD reminder
- [ ] PSScriptAnalyzer clean on the modified `.ps1`

### Hook reminder Linux (Tier 2 .sh)

- [ ] Add parity assert in `tests/hooks.bats`: `.sh` contains same SDD reminder text, same `[sdd]` prefix
- [ ] Edit `scripts/claude-session-start.sh` to inject reminder in matching position with identical text content (cross-OS parity)
- [ ] Local smoke: `echo '{"cwd":"/test","session_id":"test"}' | bash scripts/claude-session-start.sh` -- output JSON's `additionalContext` includes the SDD reminder
- [ ] `bash -n` syntax clean; `shellcheck` clean (severity error+warning)

### Cross-OS parity locks

- [ ] Bats assert: SDD reminder text is byte-identical between `.ps1` and `.sh` (no drift class like the verify-string bugs we just fixed)
- [ ] Bats assert: prefix `[sdd]` present in both
- [ ] Existing hook diagnostics (`Test-RepoSpecs`, `find_hive_project`/`Find-HiveProject`, claude-mem heal, doctor drift, vault integrity, knowledge health) still fire when their respective conditions match -- no regressions

## Closing

- [ ] Every acceptance criterion from `proposal.md` covered by ≥1 test
- [ ] PSScriptAnalyzer (Error+Warning) clean
- [ ] `bash -n` + `shellcheck` (severity error) clean
- [ ] No unrelated changes in the diff (no scope creep into Tier 3/4/5)
- [ ] `verification.md` filled with: smoke command outputs (both .ps1 and .sh), bats simulation results, PSScriptAnalyzer + shellcheck reports, before/after snippets
- [ ] PR opened referencing this spec folder; PR body notes scope split (Tier 1+2 here; Tier 3 in SDD-002; Tier 4+5 in SDD-003)
- [ ] Spec status moved `draft` → `implementing` when first code commit lands; → `verifying` when smoke green; → `archived` (move folder to `specs/archive/`) only after PR merge per archive policy in `pattern-spec-driven-development.md`

## Machine-readable features

`features.json` not required for this PR (no harness verification automation yet in this repo). Future SDD specs may opt in once the harness pattern matures. Acceptance criteria are verified by bats + manual smoke output captured in `verification.md`.
