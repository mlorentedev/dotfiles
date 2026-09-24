---
spec: "SDD-042-content-bound-review"
verdict: "PASS"
reviewed_sha: "12d3755888b392d07a73f1632fdc0d17be266b25"
reviewer: "nan/glm5.3-flash"
date: "2026-09-23"
---

## Adversarial review

**Scope**: SDD-042-content-bound-review (mlorentedev/dotfiles#1566, #970, #998 part 2, W1.4 precondition)
**Sources**: `specs/SDD-042-content-bound-review/{proposal,tasks,verification}.md` + `git diff 8a68a49c01b38bac6694eda4eb87aeb15b63e67a...HEAD` (6 commits, 19 files, +1204/−72)

### Spec and task alignment

- The diff implements exactly the proposal's five "What" items: `contract_digests` recorded by the launcher over normalised content (`contract_digest.go`, `review_request.go`), content-decided freshness with no git consultation (`reviewStale`/`changedContracts` in `review.go`), a legacy path whose absent-object refusal names the cause and still refuses, a bypass that requires `--reason`, still runs the checks it overrides, and records `review_bypass:` frontmatter (`archive.go`), refusal texts rewritten to name recovery paths only (AC5, source-walk test), and `--help` + both skill renders updated (`compile-harness.sh --check`: "OK: no harness drift", re-verified this session).
- Task boxes match diff evidence. The two unticked boxes are correct at review time: AC6 ("launch this spec's own review; its archive runs through the digest path") is completed by the very artifact being read — the launcher wrote `contract_digests` into `review-request.json` (verified live: recorded digests equal `ContractDigests()` recomputed from disk, key for key), so this spec's archive will be decided by content, not by `reviewed_sha` surviving the squash.
- `features.json` has 4 features, all `state: pending` with non-empty executable verification commands that encode the §5 "at least one test ran" guard (`grep -c '^=== RUN'` ≥ 1). No `passing` entries with empty evidence.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any spec file.

### Verification performed by this review (not trusted from verification.md)

- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean. `go test ./... -count=1`: all packages ok. `golangci-lint run ./...` at the pinned 2.12.2: 0 issues. `bats` on the four named suites: 7+23+4+18 = 52/52.
- `go test ./internal/spec/... -run 'Stale|Squash|Rebase' -count=1 -v`: 17 tests ran, all pass — matches the claimed `tests_run=17`, including the real-git fixtures that assert the reviewed commit object is actually pruned before asserting acceptance (`TestStaleSquashLandingIsAccepted`, `TestStaleRebaseLandingIsAccepted`, `TestStaleMergeLandingIsAccepted`) and the refusal naming `proposal.md` (`TestStaleContractEditRefused`).
- All named AC2/AC3/AC4/AC5 tests re-run individually: pass.
- **Mutation spot-check, 4 compiling mutants applied and reverted in the real worktree**: CRLF fold turned into an identity (killed by `TestContractDigestIgnoresCRLF`); digest path never taken, `len(req.ContractDigests) > 0` → `< 0` (killed by the squash-landing and contract-edit fixtures); bypass never recorded, `if bypass` → `if false && bypass` (killed by `TestArchiveBypassRecordedWhatTheGateWouldHaveRefused`); `--reason` never required (killed by `TestArchiveBypassRecordedRequiresReason`). This corroborates the claimed 12/12 battery, including its lesson-284 discipline (the non-compiling first attempt was discarded and re-run as a compiling mutant).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | normalisation width | Ordered-list task checkboxes are not folded: `listCheckbox` matches only `-`/`*`/`+` bullets, so ticking `1. [ ]` → `1. [x]` reads as contract drift and refuses the archive. Fail-closed annoyance, not a bypass — the dangerous direction (over-folding hides a real change) is pinned by tests | Probed this session: digests of `- [ ] a\n1. [ ] b\n- [X] c` vs the fully ticked file differ; regex at `contract_digest.go:18`. No measured incident — this repo's specs use `- [ ]` bullets exclusively | UNTESTED (`TestContractDigestIgnoresCheckboxTicks` covers bullets and nesting only) | code + tests in a follow-up (widen to `[-*+]|\d+\.`), or ticket and document the width |
| Minor | THEORETICAL | forward compatibility | Widening `contractFiles` (W3.6, #1153) will make every digest-carrying review stale: a newly digested file has no recorded entry (`""`), `changedContracts` reports it as changed, and the refusal text ("changed since the review was launched (its content digest differs)") misdescribes the cause. The proposal claims the mechanism is "built so W3.6 can widen its file set", but widening as written refuses in-flight archives | Read of `changedContracts` (`review.go:394-404`): comparison is `current[name] != recorded[name]` over the *current* `contractFiles` | UNTESTED (future change) | W3.6 spec — decide the missing-key policy (fail closed is defensible; the message must then say "no recorded digest for this file") |
| Minor | THEORETICAL | refusal hygiene | The FAIL refusal asserts an absolute that the flag disproves: it says a FAIL "is resolved by its findings, not overridden", yet `--force-without-review --reason` still archives over a FAIL (deliberately — `runPreflights` captures the gate refusal, and the override is recorded) | `review.go` FAIL message vs `TestArchiveForceWithoutReviewOverridesFail`, which pins that the override works and the record names the FAIL | `TestArchiveForceWithoutReviewOverridesFail` proves the override; no test pins the message's accuracy | code (message wording: "not overridden" → "resolved by its findings — a bypass is recorded if used") or disposition as deliberate in `verification.md` |
| Minor | REAL | docs | `unquoteScalar` comment cites a curly quote (`”`) as the single-quote escape instead of `''` | `grep -n '”' cli/internal/spec/review.go` → line 85 | UNTESTED (comment-only) | code (comment, trivial) |
| Minor | REAL | provenance | This round's `specs/SDD-042-content-bound-review/review-request.json` is untracked; sibling specs commit it. Until it is committed alongside `review.md`, the GUARD-005 provenance record exists only in this working tree | `git status --short`: the only non-review change in the tree | n/a (process step) | process — commit it with `review.md` |
| Question | SPECULATIVE | CLI ergonomics | `--reason` without any `--force-*` flag is accepted silently and records nothing | flag wiring in `cmd/spec.go` (no mutual validation) | UNTESTED | surface only — no gate impact |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All ACs verified including real-git squash/rebase/merge fixtures and negative paths; residual gaps are the ordered-list fold width and the FAIL-message contradiction |
| Verification       | A | Reproducible commands with red-before-fix narratives and a 12-mutant battery under the compiling-mutant rule; this session independently reproduced 4 killed mutants and the 17-run guard count |
| Scope              | A | Diff matches the proposal exactly; skill renders drift-checked clean; the one out-of-scope defect found (gofmt) was ticketed (#1154), not fixed |
| Reliability        | A | Every error path fails closed: tag-scan errors refuse even under force, malformed `features.json` digests by its bytes, absent objects refuse, a bypass needs a proposal to record into |
| Maintainability    | A | Functions small and single-purpose, CC low, comments explain why (fold width pinned in both directions by kill tests); one comment typo |
| Handoff-readiness  | A | `verification.md` records decisions (including the #1630 merge-order warning), promotions explicitly dispositioned; AC6 pending by design, completed by this review + archive |

### Verdict

PASS — no Blockers, no REAL Majors; all rubric dimensions B or above; six Minor/question findings listed above, each with a reality tag and named test or UNTESTED mark.

### Recommended next steps

For the implementer to disposition in `verification.md` (applied / ticketed / declined with reason) — none of these require touching the contract set, which is closed by this verdict:

- Commit `review.md` together with `review-request.json` (finding 5), tick the AC6 boxes (checkbox ticks are fold-safe — that is the mechanism this spec built), and run `dotf spec archive`: freshness will be decided by the digests, which this review verified still match disk.
- Finding 1 (ordered-list checkbox fold): ticket on the bitácora with the probe above, or widen the regex + extend `TestContractDigestIgnoresCheckboxTicks` in a follow-up.
- Finding 2 (missing-key policy on widening): carry verbatim into the W3.6 spec — it is a design decision for that spec, not a patch here.
- Finding 3 (FAIL-message wording): apply the wording tweak or record the deliberate decision with a reason.
- Finding 4 (comment typo): one-line code fix, fold into the next touched PR.

`dotf spec archive` / `/spec archive` is **advisable** in the current state once `review.md` and `review-request.json` are committed and AC6 is ticked.
