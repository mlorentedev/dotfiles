---
spec: "CLI-080-secrets-reconcile"
verdict: "PASS"
reviewed_sha: "5fd4de6cc452d67a1c7348abcfc7920bd48494f2"
reviewer: "nan/glm5.3-flash"
date: "2026-09-22"
---

## Adversarial review

**Scope**: CLI-080-secrets-reconcile
**Sources**: `specs/CLI-080-secrets-reconcile/{proposal,tasks,verification,features.json}`, git diff `33c3aa7bf94b07a173632059371d2e1331b4c007...5fd4de6cc452d67a1c7348abcfc7920bd48494f2` (whole change, round 2 — round 1 was `agy/gemini-3.1-pro-high` on `f16fb2e`, verdict FAIL). The base spans seven commits, of which five are already-merged work this branch sits on (#1600/CLI-078, #1612, #1614, #1616, #1619); this review grades CLI-080's footprint and the interactions with those.

### Spec and task alignment
- All ten tasks under Implementation/Verification carry `[x]` with diff evidence found; the one deliberately-unticked box is the review gate itself (correct — ticking it would self-certify).
- Round-1 dispositions are real, not cosmetic: `maxApplyPasses = 2` in `applyUntilConverged` with the `onlyRetires` guard (`cli/internal/cmd/secrets_reconcile.go`), and `syncAfterWrite` after folder creation (`bwserve_writer.go`, `ResolveFolder`). AC10 was amended to match.
- AC1–AC8 and AC10 are pinned by named tests; AC9 is live evidence, which this review **independently corroborated read-only** against the unlocked vault: `dotf secrets drift` → 1 finding (zoho, dormant), 187 items, 164 unmanaged; `dotf secrets reconcile` → `Plan: 0 to apply, 0 blocked, 2 deferred`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | docs | The runbook's CONVERGE step 6 says retire is "**Never in the same run as the copy**", but the round-1 fix this same diff ships makes copy+retire converge in ONE `--apply` (two passes) — and the spec's own first use declared `retire: true` up front. The sentence describes behavior the tool no longer has; an operator reading it will mis-predict what `--apply` does with a `retire:` declared from the start. | code read of `applyUntilConverged` vs `docs/runbooks/guide-secrets-governance.md` step 6, both in this diff; `TestReconcileApplyCopiesAndRetiresInOneRun` proves one-run convergence | UNTESTED (prose) | docs (repo runbook; outside the contract set) |
| Minor | REAL | evidence | `features.json` f1–f10 evidence cites runs "on top of c74675a" — that commit is not reachable in this repo (`git log --all` finds nothing), so the recorded provenance is unverifiable as written. The substance is not: this review re-executed f1–f8 and f10 against `5fd4de6` (all pass) and reproduced f9's live claims. | this review's runs, this session | the f1–f10 commands themselves (re-ran, exit 0) | contract set (`features.json`) — **track, do not edit**: an edit would invalidate this review; record the disposition in `verification.md` |
| Minor | THEORETICAL | planner/apply | A retire whose compared values differ leaves the plan permanently non-appliable: every `--apply` re-refuses ("settle it, then re-run"), but nothing tells the operator *which* side `set`/`rotate` must settle — the tool correctly cannot know. Deliberate fail-closed; needs a human decision, so it is surfaced as an assumption, not a defect. | code read of `sameValue`; refusal proven by `TestApplyRetireRefusesWhenTheValuesDiffer` | `TestApplyRetireRefusesWhenTheValuesDiffer` | code (error message) or docs |
| Minor | SPECULATIVE | scope | `sync ci [SECRET_NAME...]` accepts duplicate names (`scopeUpload` appends per name) → the same GitHub secret uploaded twice; noisy, not wrong. | code read of `scopeUpload` | UNTESTED | — (surface only) |
| Minor | SPECULATIVE | lifecycle | An item whose every declared field gets retired remains as an empty husk; cleanup is out of scope per proposal. | none | UNTESTED | — (surface only) |
| Question | SPECULATIVE | concurrency | Plan→write TOCTOU: "never overwrite" is judged against a synced snapshot; a field created between plan and `--apply` would be overwritten by `SetField`. Accepted in proposal (concurrent writers, OPS-028, single operator). | none | UNTESTED | — (accepted risk) |

**Mutation spot-check (this review's own, all reverted):** `maxApplyPasses 2→1` → `TestReconcileApplyCopiesAndRetiresInOneRun` FAILS (round-1 Blocker regression is pinned); `onlyRetires` always-true → `TestReconcileSecondPassIsOnlyForRetires` FAILS (6 writes > 4); delete `syncAfterWrite` in `ResolveFolder` → `TestBWServeWriter_ResolveFolder` FAILS (duplicate `Dotfiles/infra`); delete the equality gate in `sameValue` → `TestApplyRetireRefusesWhenTheValuesDiffer` FAILS. 4/4 killed — the round-1 fixes are not just present, they are regression-locked.

**Commands run this session**: `go build`, `go vet`, `GOOS=windows go vet` (exit 0), `go test ./...` (ok), `golangci-lint run` at the pinned v2.12.2 (0 issues), features.json f1–f8+f10 (pass), live `drift` + read-only `reconcile` plan (AC9 corroborated), 4 hand-applied mutations each reverted. Working tree left clean apart from this review and the launcher's own `review-request.json`.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All ACs verified by named tests plus live corroboration; negative paths (blocked, empty source, ambiguous, differing values, non-convergence) each have a test. |
| Verification       | A | Reproducible feature commands (re-executed by this review), honest mutation table (4 spot-checked, all killed), live AC9 evidence reproduced read-only. |
| Scope              | B | CLI-080's footprint matches the proposal with no creep, but the launcher's base bundles five already-merged changes, so the reviewed diff is wider than the spec. |
| Reliability        | A | Fail-closed everywhere: two-phase apply, bounded convergence loop (lesson 286), sync before every plan and after every write, whole-plan blocking. |
| Maintainability    | A | Small functions, shared pure cores (`setItemField`/`setItemFolder`/`removeItemField`) across both backends, lint clean at the pinned version. |
| Handoff-readiness  | A | CONVERGE runbook, inventory and escrow refreshed, lessons 284–286 committed, decisions recorded in verification.md — minus the one stale runbook sentence above. |

### Verdict
PASS

### Recommended next steps
Route via `verification.md` dispositions (applied / ticketed / declined) — the contract set (`proposal.md`, `tasks.md`, `features.json`) is closed by this verdict; none of these require editing it:

- Fix the runbook CONVERGE step 6 sentence to state the real invariant — *a retire never rides in the pass that creates its copy; one `--apply` may run a second pass for it* (docs, code-repo).
- Record a disposition for the `c74675a` provenance blemish in `features.json` evidence: note that round 2 re-executed f1–f10 against `5fd4de6` (this review) rather than editing the contract file.
- Decide the "settle it" ergonomics of a refused retire — at minimum name in the error which two operations (`set` on one side, or editing `bw.from`) are the escape, or ticket the UX.
- Follow-up tickets, no gate: duplicate names in `sync ci` scoping; empty-husk item cleanup after full retire.
