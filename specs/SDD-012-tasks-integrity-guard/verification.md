---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - SDD-012-tasks-integrity-guard

## Evidence

- [x] **AC1 — duplicate IDs flagged** → `tests/check-backlog-integrity.bats` test "AC1 duplicate" exits 1 + greps the ID. Also empirically: the real `11-tasks.md` reports 30 `DUPLICATE:` lines.
- [x] **AC2 — status contradiction flagged** → bats test "AC2 contradiction" exits 1 + greps `contradict`. Real file flags `IDEAS-007`, `SDD-004`, `BUG-022` as `CONTRADICTION` (the legitimately-partial tickets — correctly distinguished from plain duplicates).
- [x] **AC3 — clean passes + sub-ids safe** → bats "AC3 clean" with `WIN-002` + `WIN-002a` distinct → exit 0.
- [x] **AC4 — wired + discoverable** → bats "AC4 dispatcher" (`vault check-tasks` exit passthrough) green; `vault-health.sh` section "7/7 Backlog Integrity" greps present (scans `10_projects/*/11-tasks.md`, GUI-independent).
- [ ] **AC5 — real backlog green** → DEFERRED follow-up (vault busy). Today the real file exits 1 (33 issues) by design — that IS the guard surfacing the drift to be consolidated.

## Test status

- `~/.local/bin/bats tests/check-backlog-integrity.bats` → `1..6` all `ok`.
- `~/.local/bin/shellcheck scripts/check-backlog-integrity.sh scripts/vault.sh scripts/vault-health.sh` → clean.
- `bash -n` → clean on all 3.
- Real-data smoke: `check-backlog-integrity.sh <real 11-tasks.md>` → exit 1, 30 duplicates + 3 contradictions. Proves the guard works on production data, not just fixtures.
- No regressions: vault-health.sh section renumber 1/6..6/6 → 1/7..7/7; new section is additive + GUI-independent.

## Decisions made during implementation

- **Portability over gawk.** Extraction uses `sed -nE 's/.../\1<TAB>\2/p'` + plain awk (assoc arrays, field access), avoiding gawk-only `match(s,re,arr)`/`gensub` so the guard runs under BSD tooling too.
- **Greedy `[a-z]?` in the ID regex** keeps `WIN-002a` whole while leaving `WIN-002` intact before a `-slug` — the sub-id false-positive the naive approach would hit.
- **DUPLICATE vs CONTRADICTION split.** Contradiction (same ID `[ ]` and `[x]`) is reported distinctly from plain duplication — it marks the tickets that need a human decision (partial-done) vs mechanical merge.
- **Local-only guard, fixture-tested.** Vault is private (no CI access); the guard lives in `vault-health`/`vault check-tasks` (SessionStart, local) per ADR-012/ENGINE-001, while bats fixtures verify the logic in CI.
- **Vault gate deferred, not skipped.** The vault checkout was busy with a parallel session, so the SDD-012 backlog entry + consolidation are an explicit tracked follow-up (AC5), not a silent omission.

## Promotion candidates

- [ ] Lesson? **yes** — "Layered, hand-maintained views of the same list drift; enforce one-entry-per-ID with a guard (incident→guard), don't rely on discipline." (Capture at archive.)
- [ ] ADR-worthy? **no** — extends the existing incident→guard pattern (SDD-006 sibling).
- [ ] Pattern candidate? **no** — folds into the incident→guard guardrail already in the vault.

## Archive checklist

- [ ] `proposal.md` → `status: archived`
- [ ] Folder moved to `specs/archive/SDD-012-tasks-integrity-guard/`
- [ ] Vault `11-tasks.md` SDD-012 entry added + ticked with PR link (part of the AC5 consolidation pass)
- [ ] Lesson promoted
