---
spec: "AI-057-gemini-4-bindings"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "f887cd7d5293ff98305284b0785dbea92d4a1e32"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-13"
---

## Adversarial review

**Scope**: `AI-057-gemini-4-bindings` (fast-track spec, one `spec.md`). The spec's own
commit in `be0c69419fe783a57d67bd9564a1e1b87b9e372e...HEAD` is `8da54c87` (#3478). The other
commit in the range, `f887cd7d` (#3488, `mem_handoff --finished`), is unrelated work: it
touches `cli/internal/mem/`, `cli/internal/cmd/mem_handoff*` and an archived `MEMORY-019`
spec, shares no file with this spec, and is not reviewed here beyond that check.

**Sources**: `specs/AI-057-gemini-4-bindings/spec.md`; `git diff
be0c69419fe783a57d67bd9564a1e1b87b9e372e...HEAD`; commits `8da54c87`, `f2e75ba6` (#3482,
already merged into the branch base line); `harness/model-map.json`,
`harness/model-pins.json`, `harness/reviewer-pool.json`, `ai/agy/settings.json`,
`ai/agy/AGY.md`, `cli/internal/harness/{model_map_test.go,model_pins.go,model_pins_test.go}`,
`docs/lessons/lesson-599-*.md`; issues #3471, #3475, #3482; live `agy models`.

### Spec and task alignment

Contract digests verified fresh: `dotf spec digest AI-057-gemini-4-bindings` returns
`spec.md: 9a61776c382b2b70...`, identical to `review-request.json`. No `[AGENT-DRAFT]` or
`[AGENT-SUGGESTION]` tags in the spec folder. Spec id matches the repo grammar and
`tests/guard-spec-ids-unique.bats` passes (4/4).

| Checklist item | Verdict | Evidence produced in this review |
|---|---|---|
| `model-map.json` top/mid ids and comments | **Verified** | `tiers.top.gemini = gemini-4-argon-high`, `tiers.mid.gemini = gemini-4-hydrogen-high`; live `agy models` (1.5.9 here; spec measured 1.5.8 on 2026-11-14) serves both ids and their sibling effort levels. |
| `model-pins.json` agy pin `why` + NOT LISTED note | **Verified with a defect** | Pin now names `Gemini 4 Hydrogen (High)` → `gemini-4-hydrogen-high`, the exact label/id pair the live catalog prints. The NOT LISTED prose kept a stale claim (finding 2). |
| `ai/agy/settings.json` `model` and `defaultModel` | **Verified** | `model` = `Gemini 4 Hydrogen (High)`; `defaultModel` = `gemini-4-hydrogen-high`. `defaultModel` is a recorded `excluded` entry (agy does not read it), so editing it is consistency, not routing. |
| `model_map_test.go` asserts top is Argon, mid differs from top | **Verified** | `go test ./internal/harness/...` ok; the new assertion sits in `TestModelMapConsumerClasses`. |
| `model_pins_test.go` Gemini 4 rows; `model_pins.go` comment | **Verified** | All seven Gemini 4 rows match the live catalog, including `Gemini 4 Hydrogen (XHigh)` → `gemini-4-hydrogen-xhigh`; `Orchestrator (Argon)` → `gemini-pro-orchestrator-g1-pro` is correctly annotated as a row the rule does not cover. |
| Lesson 599 | **Verified** | Present; `go run ./cmd/dotf lessons fmt --check` → `[OK] lessons are in their canonical form`. Its probe is #3475, confirmed OPEN. |
| Pool swap removed from this branch | **Verified** | `git diff be0c694...HEAD -- harness/reviewer-pool.json cli/internal/prreview/reviewer-pool.json cli/internal/cmd/spec.go` is empty; the pool already carries `agy/gemini-4-hydrogen-high` from `f2e75ba6` (#3482, MERGED). |

Gates re-run in this worktree, not taken from `spec.md`: `go build ./...` (ok),
`gofmt -l ./internal/harness/` (clean), `go vet ./internal/harness/` (ok),
`go test ./internal/harness/...` (ok), `go test ./internal/deploy/` (ok),
`go run ./cmd/dotf harness check` → `[check] OK: no harness drift`,
`go run ./cmd/dotf doctor --scope machine` → `[Model pin drift] (1 checks, all ok)`,
`bats tests/reviewer-pool.bats` (4/4), `bats tests/antigravity.bats` (15/15),
`bats tests/guard-spec-ids-unique.bats` (4/4).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | test coverage | The seed label's **tier** has no detector. The agy pin proves `ai/agy/settings.json` `model` resolves to *some* model the map declares in pool `gemini` — not that it is `tiers.mid.gemini`, which `ai/agy/AGY.md:30` states as the contract. Before this change the claim was vacuous (top and mid were one id); the change makes the two tiers distinct, so it creates the invariant without a guard. | Mutation, reverted: `model` set to `"Gemini 4 Argon (High)"` (the top tier) → `gofmt`-clean, `go test ./internal/harness -count=1` **ok**. The current value is correct (verified); the risk is a future edit that no check would price. | `TestEveryRepoRoutingPinResolvesInTheMap` covers membership only; the tier property is **UNTESTED** | tests (`model_pins_test.go`: assert `Normalize(agy-model, settings.model) == tiers.mid.gemini`, or extend the pin to declare the tier it must match) |
| Minor | REAL | documentation | `harness/model-pins.json` NOT LISTED prose says the pool's `agy/gemini-4-hydrogen-high` entry "is unroutable today — a question about the map's completeness, filed as #1253". It is not unroutable: the map declares pool `gemini` (`pools.gemini`, `harnesses.agy.pools: ["gemini"]`) and `DeclaredModels` contains `gemini:gemini-4-hydrogen-high` (probe test, this run). The claim was already stale at the base commit for the previous id; the diff rewrites that line's id and carries the stale assertion forward. | A probe test inside `cli/internal/harness` reported `qualified["gemini:gemini-4-hydrogen-high"]=true`. #1253 is about map completeness, not unroutability. | UNTESTED (prose) | code (`harness/model-pins.json` prose), or the real remedy: declare `harness/reviewer-pool.json` as a pin site so the gate arm is checked against the map (ticket #1253) |
| Minor | REAL | documentation | The map's top-tier comment says Argon "passed the reviewer pool's admission bar in 50 s (#3471)", while the pool's own entry records admission on **two** bars and states "Argon is not admitted". A reader ranging over both files meets the same class of contradiction the branch's second commit fixed once for the stall note. | `harness/model-map.json:147` vs `harness/reviewer-pool.json:83`; #3471's measured table does show 50 s for bar 1 only. | UNTESTED (prose) | code (comment wording: "cleared bar 1 of the pool's admission, 50 s") |
| Minor | REAL | test comment | `TestDisplayNameSpellingMatchesTheAgyCatalog`'s comment calls its rows "the gemini rows of `agy models`", but the live catalog no longer serves `gemini-3.7-flash-*`, `gemini-3.6-flash-high` or `gemini-3.1-pro-*` (8 of 16 rows are historical). The rule is a pure string transform so the guard is sound; only the comment misleads. | Live `agy models` output captured this run lists 3.8 and 4.x rows only. | `TestDisplayNameSpellingMatchesTheAgyCatalog` (passes; the rows are historical, not wrong) | tests (comment: "historical rows, no longer served") |

No injection, secret, auth, performance, concurrency or resilience surface exists in this
diff: it is JSON config, one Go comment, one test table and one new lesson markdown.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | Every checklist item verified against the live agy catalog and the real repo; the seed label is mutation-proven to resolve into the map. |
| Verification       | B | The spec's evidence block is reproducible and reproduced here, but no gate states the seed's *tier* — the reproduction in finding 1 survives the whole suite. |
| Scope              | A | Diff matches the declared scope exactly; the removed pool swap is documented in both the commit and the spec's Out of scope. |
| Reliability        | B | Config-only, no runtime path changed; `defaultModel` is edited despite being unread by agy (recorded `excluded`), which is harmless but adds churn to a dead field. |
| Maintainability    | B | Comments carry dates, versions and ticket numbers; one of them now asserts something the sibling file contradicts (finding 3). |
| Handoff-readiness  | A | Lesson 599 captured, #3475 filed as the probe, #3482 owns the pool decision, per-seat deploy named in Next. |

### Verdict

**PASS WITH GAPS** — one Major, THEORETICAL (a test gap the change creates but does not fill;
no defect in the shipped values) plus three REAL Minors in documentation. Rubric: no C, no D.

### Recommended next steps

Routed by set: `spec.md` is a **contract** file here, so nothing below is an edit to it — an
edit to `spec.md` would invalidate this review; disposition each item in the archive note or a
follow-up ticket.

- **(code)** Fix the two stale prose claims while the context is fresh: `harness/model-pins.json`
  NOT LISTED ("unroutable today", finding 2) and `harness/model-map.json:147`'s
  "passed the reviewer pool's admission bar" (finding 3). Both are comments, outside the
  contract set; neither changes behaviour.
- **(tests)** Add the tier assertion from finding 1 to `model_pins_test.go`, so the guard fails
  if the agy seed leaves `tiers.mid.gemini` — the property `ai/agy/AGY.md:30` already claims.
  Prefer this over a ticket: it is a few lines and it closes the gap the change created.
- **(ticket)** #1253's real remedy is not prose: declaring `harness/reviewer-pool.json` as a
  pin site would have caught the pool's old `gemini-3.1-pro-high` drift mechanically. Keep that
  on the board rather than in a comment.
- **(deploy)** After merge, the spec's Next stands: `dotf deploy agy-settings` per seat and
  confirm agy's start log resolves `Gemini 4 Hydrogen (High)`; this worktree cannot observe a
  deployed `$HOME/.gemini/antigravity-cli/settings.json` — UNVERIFIED here by design (recorded
  `excluded` in `model-pins.json`).
- `dotf spec archive AI-057-gemini-4-bindings` is **advisable** in this state: the contract is
  unchanged since the digests in `review-request.json` were taken, and the open items above all
  sit in code, tests or tickets rather than in the accepted change.
