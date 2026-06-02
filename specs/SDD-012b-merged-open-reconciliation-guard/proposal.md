---
id: "SDD-012b-merged-open-reconciliation-guard"
type: spec
status: active
created: "2026-06-01"
tags: [spec, proposal, vault-health, incident-to-guard, backlog-hygiene, sdd-012-followup]
template_version: "1.0"
---

# SDD-012b-merged-open-reconciliation-guard

> **Naming**: file lives at `<repo>/specs/SDD-012b-merged-open-reconciliation-guard/proposal.md`.

## Why

<!-- from 11-tasks.md: implement SDD-012's deferred "merged-but-open" cross-reference -->

SDD-012 (`check-backlog-integrity.sh`) guards **structural** backlog drift — one id on two lines, an id marked both `[ ]` and `[x]`. It explicitly deferred the **semantic** class as out of scope: a `- [ ]` entry whose work has *already shipped*. A stale-merged tick is perfectly well-formed text, so the structural guard cannot see it. On 2026-06-01 a single dotfiles reconciliation sweep found **four** such entries — `BUG-022`, `BUG-023`, `BUG-024`, `SDD-009` — all merged weeks earlier, all still `[ ]`, all passing `check-backlog-integrity.sh`. `session_briefing` reads the vault, so these surfaced as phantom open work and nearly caused duplicate implementation. This is the exact failure mode of [[pattern-verify-against-source-of-truth]]: the doc drifts from the repo, and "reconcile manually" rots because nothing detects recurrence. The principle is canonized; this ships its mechanical enforcement.

## What

Concrete, observable behavior after this PR:

1. **A standalone guard `scripts/check-backlog-merged.sh`** (sibling of `check-backlog-integrity.sh`) takes a `<tasks-file>` and flags each open `- [ ]`/`[~]`/`[-]` entry whose **FULL ticket id** (the bold token between `** **`) has an archived spec at `<repo>/specs/archive/<id>/`. Exit `0` clean / `1` candidates found / `2` usage. The done-signal is the SDD lifecycle's own archival marker (per [[pattern-three-layer-proposal-lifecycle]]) — local, network-free, CI/fixture-testable.
2. **Full-id keying** (not the bare number): `SDD-009-opencode-deploy-time-secrets` matches its archive; `SDD-009-some-other-ticket` does not. This is what dodges the **number-reuse** trap (dotfiles has had two `BUG-024`s and two `HERMES-002`s). Sub-ids (`WIN-002a`) resolve distinctly.
3. **Repo inference**: `<vault>/10_projects/<proj>/11-tasks.md` → `$HOME/Projects/<proj>`, overridable with `--repo`. Generalizes to every project, not just dotfiles.
4. **Wired into `vault.sh`** as `vault check-merged <file>`, standalone-runnable like its siblings.
5. **A GUI-independent advisory line in `vault-health.sh`** runs the guard over every `10_projects/*/11-tasks.md` and reports findings via `warn` (yellow, not a failure) — so stale-merged ticks auto-surface at SessionStart for **whatever agent opens the next session**, turning the "verify against source of truth" discipline into a backstop no one has to remember.

## Out of scope

- **Merged-PR / commit cross-referencing** (matching a `[ ]` id against `gh`/`git log`). PR titles cite the *number*, not the slug, so the match is heuristic and number-reuse makes it noisy. Stays a manual/advisory step (documented in the pattern). The archived-spec signal ships the reliable, zero-false-positive half — the same scoping discipline SDD-012 used.
- **Auto-ticking** the stale entries. The guard flags; a human/agent verifies against git and ticks with the PR link. Closing the loop is a deliberate action, not an automated mutation of the SSOT.
- **Fixing `init-spec.sh`'s id regex** so it accepts sub-ids like `SDD-012b` (it currently rejects them — a sibling defect to BUG-024, found while scaffolding this very spec). Separate concern, separate PR.
- **Multi-id lines** (`BUG-008 / BUG-009 / BUG-010` on one line): the guard reads the leading id per entry line — a documented limitation inherited from SDD-012, not a target.

## Risks / open questions

- **Archived spec ≠ absolute proof of merge** (a spec could be archived prematurely). **Mitigated:** the guard is advisory (`warn`/exit-1-as-warn), output says "verify + tick", and the human confirms against git. False positives cost one `git log`, not a wrong mutation.
- **Bug/direct-fix tickets have no spec** (BUG-022/023/024 were not caught automatically — only SDD-009 was). **Accepted:** the spec'd class is the recurring, highest-volume drift; the bug class is covered by the pattern's manual procedure + the structural guard. Honest partial coverage beats a brittle full-coverage heuristic.
- **Guard is local-only** (vault is private; dotfiles CI has no vault access). **Accepted:** consistent with ADR-012 / SDD-012 / ENGINE-001 — drift guards live in `vault-health` (local, SessionStart), not CI. The bats tests use fixtures, so CI still verifies the *logic*.

## Acceptance criteria

- [ ] **AC1 — stale-open flagged**: an open entry whose full id has `specs/archive/<id>/` → exit 1, names the id. **Verify:** bats fixture.
- [ ] **AC2 — ticked + no-spec pass**: `[x]` entries and `[ ]` entries without an archive → exit 0. **Verify:** bats fixtures.
- [ ] **AC3 — full-id keyed**: a different ticket sharing a number is NOT flagged; sub-ids (`WIN-002a`) resolve distinctly. **Verify:** bats fixtures.
- [ ] **AC4 — repo inference + skip**: path→repo inference works; missing `specs/archive` → advisory skip exit 0; missing tasks file → exit 2. **Verify:** bats fixtures.
- [ ] **AC5 — wired + discoverable**: `vault check-merged <file>` dispatches; `vault-health.sh` carries a GUI-independent advisory section invoking it over `10_projects/*/11-tasks.md` as `warn`. **Verify:** dispatcher run + grep of vault-health section.
- [ ] **AC6 — live backlog green**: after the 2026-06-01 reconciliation, `check-backlog-merged.sh` on the real `10_projects/dotfiles/11-tasks.md` exits 0. **Verify:** run on the live file (done at build time: exit 0).

## References

- Sibling precedent: `specs/archive/SDD-012-tasks-integrity-guard/` + `scripts/check-backlog-integrity.sh` (the structural half this completes)
- Canonized in: `pattern-verify-against-source-of-truth.md` §"Automating the Reconciliation"; merge-time step of `pattern-three-layer-proposal-lifecycle`
- Dispatcher: `scripts/vault.sh` (REFACTOR-005), `scripts/vault-health.sh` section 7
- Incident: the 2026-06-01 sweep (BUG-022/023/024 + SDD-009 stale-merged); AGENTS.md "incident → guard"
