---
spec: "HARNESS-136-model-limit-drift"
verdict: "FAIL"
reviewed_sha: "c7a8adca25b9e151340d68b15d56502378c3a7bd"
reviewer: "nan/qwen3.8-flash"
date: "2026-09-22"
---

## Adversarial review

**Scope**: HARNESS-136-model-limit-drift (issue #1594, PR #1595 — merged as `c52e637`)
**Sources**: `specs/HARNESS-136-model-limit-drift/{proposal,tasks,verification,features}.md`;
diff `3f98705...c7a8adc` as resolved by the launcher. That range also contains three
commits from other specs (`799ca66` = CLI-078, which carries its own spec folder; `5780215`
and `c7a8adc`, dependabot bumps) — an artifact of the base being resolved before those
merged, not scope creep of this change. The review judged `c52e637`'s files as this spec's
diff: `ai/pi/models.json`, `cli/internal/doctor/checks_model_limits.go` (+_test),
`doctor.go` registration, lesson-283, and the spec artifacts.

Everything below was verified by running, not by reading claims: `go build ./...`,
`go vet ./...`, `GOOS=windows go vet ./...` clean; `go test -count=1 ./...` green;
all 8 `features.json` commands re-run verbatim from the repo root (f1–f8 all exit 0);
`golangci-lint` at the pinned v2.12.2 → 0 issues; `bats tests/pi-config.bats` 18/18.
The proof-by-consequence claim reproduces exactly: the built binary reports
`[ OK ] 7 models match the provider catalog` against this tree and **9 findings (2 FAIL,
7 WARN)** against the pre-fix `ai/pi/models.json` from `3f98705`. The seven corrected
values were independently re-derived from `~/.cache/opencode/models.json` (parsed with
python, not with the check's own parser) and cross-checked against
<https://nan.builders/docs/models> (qwen3.8-flash 262K ctx/131K max, mimo-v2.5 1M/131K,
contexts 1M/262K for the rest) — **AC1 is genuinely met**. The four mutations claimed by
`verification.md` were re-run and each is killed by exactly the named test: drop provider
scoping → `TestModelLimitsResolvesTheSameIdPerProvider` (over 25 runs), swap severities →
both severity tests, drop the `actual == 0` guard → `TestModelLimitsIgnoresAnUnpublishedLimit`,
absent catalog PASSes → `TestModelLimitsSkipsWhenTheCatalogIsNotCached`. AC8 holds:
`checkModelLimits` takes no fix flag (registration passes only `sys, cfg, rep`), and a
full read of the 214-line file finds no write syscall — only `os.ReadFile`.
No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in any contract file.

What below is not claimed by the author was found by reproduction with temporary probes
and mutations (all reverted; tree verified byte-identical afterwards).

### Spec and task alignment

- AC1 [x] — met, verified independently above.
- AC2/AC3 [x] — met; both directions tested, neither leaks into the other, both numbers
  named. Mutation-killed.
- AC4 [x] — mechanism met and test-pinned, but the motivating fact in `proposal.md` is
  inaccurate against the catalog today (finding F3).
- AC5/AC6 [x] — met for the paths they test; AC6's "never a vacuous PASS" is violated by
  one untested corner (finding F2).
- AC7 [x] — **only half implemented.** The AC names "an unreadable **or** unparseable
  declaration"; the code FAILs on unparseable and WARNs on unreadable, and no test covers
  the unreadable half (finding F1). `tasks.md`'s "Every acceptance criterion is covered by
  at least one test" is therefore also overstated.
- AC8 [x] — met.
- `tasks.md` final box ("PR opened referencing this spec folder") is unchecked while #1595
  merged 2026-09-22T04:09:41Z — stale bookkeeping (F7).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | AC7 / `checks_model_limits.go:117-124` | AC7 requires an **unreadable** declaration to FAIL; the code emits WARN on the non-IsNotExist `ReadFile` error branch (only unparseable FAILs). A WARN does not move doctor's exit code (0 unless ≥1 FAIL, `doctor.go:24`), so a consumer gating on the exit status treats a declaration it never read as acceptable. The AC is ticked `[x]`, but its unreadable half is neither implemented nor tested. | Reproduced this session with a temporary probe: declaration chmod 000 → `FAIL=0 WARN=1`; `grep -ci 'Chmod\|permission'` over `checks_model_limits_test.go` → 0 | UNTESTED (no `TestModelLimitsFailsOnAnUnreadableDeclaration` exists) | code + tests — change the branch to Fail and name a test for it; *or* if WARN was deliberate (per-machine vs repo-content asymmetry), the contract must say so — either route ends in a re-review |
| Major | THEORETICAL | AC6 / `compared` counter | `compared++` fires on any catalog **hit**, even a row publishing no limits at all; both fields then skip (`d.actual == 0`), findings stay 0, and the check prints `1 models match the provider catalog` — "nothing to compared" reading as agreement, the exact confusion the spec's own design rule forbids and AC6's clause names. The input shape is not exotic: 121 of 7,556 rows in today's live cache have both limits zero/absent (none under `nan`/`openrouter` yet — which is why this is THEORETICAL, not REAL). | Reproduced with a temp probe (`{"nan":{"models":{"brand-new":{}}}}` → PASS printed); survival-proved: mutating `compared++` to count only actually-comparable fields passes the entire 7-test suite (mutation M5 survives), so nothing pins the semantics either way | UNTESTED — the vacuous-PASS corner has no named test; M5 survival demonstrates the suite's blindness | code + tests |
| Minor | REAL | AC4 motivation / spec accuracy | `proposal.md` ("Resolved — provider scoping") says `qwen3.8-flash` "is published by both `nan` … and `openrouter` (1000000)". In today's cache openrouter keys it `qwen/qwen3.8-flash` — no bare row. The real same-id collision is with 10 *other* providers (alibaba 1000000, requesty 1048576, hyper, llmgateway, vancine, opencode-go…). The mechanism AC4 protects is more load-bearing than stated (e.g. `deepseek/deepseek-chat` exists under 6 providers with limits from 128000 to 1000000), but the named example is verifiably wrong. | Cache scan this session (python, independent of the check) | n/a (documentation) | spec artifacts (contract set — this FAIL round is when to fix it); optionally the code doc-comment's "one model name … two providers" sentence |
| Minor | THEORETICAL | test gap / "What" promise | The "not inside a checkout → SKIP, never PASS" behaviour named in the What section has no test: replacing that `rep.Skip` with a vacuous `rep.Pass` keeps all 7 tests green (mutation survived this session). Trivial branch today; unpinned regression surface for the check's core invariant. | Mutation run this session | UNTESTED | tests (one fixture with `RepoDir: ""`) |
| Minor | REAL | Maintainability / repo rules | `checkModelLimits` is 84 non-comment lines, CC≈17 — against AGENTS.md thresholds (<40 lines, CC<10) and the rubric's B bar (CC≤15). golangci-lint's "0 issues" does not contradict this: funlen/gocyclo are not in the enabled set, so the repo rule is unenforced here, and the sibling `checkModelPins` has the same shape (101 lines, CC≈20). Following the neighbour is defensible; the mechanical rubric still lands on C. | Measured this session (comment/string-stripped CC count; `golangci-lint` v2.12.2 0 issues) | n/a (structure) | code (extract the two read-parse blocks), or a package-wide decision recorded once as a follow-up ticket — not a blocker for this PR alone |
| Minor | SPECULATIVE | path edge / whole package | `sys.home()` can return `""` (no HOME and no USERPROFILE), making the catalog path cwd-relative — a stray `.cache/opencode/models.json` under the cwd could then satisfy the comparison. No sibling doctor check guards this either (checked `checks_deploy.go`, `checks_agentconfig.go`), so it is package-wide convention, not introduced here. Surfaced, not gated. | Code read; not reproduced (would require an environment no real shell provides) | UNTESTED | — (surface only; do not gate) |
| Minor | REAL | Handoff / record | `features.json` still carries `state: "pending"` and `evidence: ""` for all 8 entries although `tasks.md` documents the harness filling them (`verification.md` instead asserts the runs narratively); and `tasks.md`'s closing box for "PR opened" is unchecked while PR #1595 is merged. Repo convention is mixed — 4 of 8 archived specs sampled also remain pending — so this is hygiene, not a gate. | `gh pr view 1595` + archive folder scan this session | n/a | spec artifacts (contract set) — record while the contract is open for this re-review round |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | AC7's unreadable half unimplemented and untested (F1, reproduced); AC6's never-vacuous-PASS rule violated by the zero-limit-row corner (F2); AC1–AC5, AC8 verified met. |
| Verification       | B | Every claim I could re-run I re-ran and it held (8/8 feature commands, build/vet/tests/lint/bats, both-direction consequence proof, 4/4 claimed mutations killed, AC1 values independently re-derived from two sources); deductions for features.json never recording evidence and for an `[x]` AC whose second half no test covers. |
| Scope              | B | The spec's own diff matches the proposal and the declared out-of-scope list exactly (no writes, no extra fields, no CI snapshot); the review range's three other-PR commits are a base-resolution artifact, judged as such. |
| Reliability        | B | Loud-failure discipline on parse, deterministic provider ordering, SKIP-not-PASS on absent cache; costs are the WARN-instead-of-FAIL and vacuous-PASS corners above. |
| Maintainability    | C | New code at CC≈17 / 84 lines breaches the repo's <10 / <40 rules and the rubric's ≤15 bar (mechanically C); comments explain WHY unusually well, and the shape follows the established sibling. |
| Handoff-readiness  | B | Spec triad complete, two resolved risks recorded with their measurements, lesson-283 captured and indexed with the right generalisation; stale PR box and unfilled features.json evidence. |

### Verdict

**FAIL** — not on rubric (no D) but on severity × reality: F1 is a **REAL Major** (an
acceptance criterion marked `[x]` is demonstrably only half-implemented, reproduced this
session, with the missing half UNTESTED), and the rules are explicit that a REAL Major
forces FAIL until addressed. F2 is a Major of its own kind but labelled THEORETICAL, and
does not by itself carry the verdict — it should ride along with the fix, since both live
in the same file the re-review will re-read.

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is open for edits in this
FAIL round — a passing verdict re-reviews whatever it changes, so fix both sides here
rather than routing around them:

1. **code + tests (F1)**: make the declaration read-error branch `rep.Fail` (matching AC7's
   letter and the check's own broken-path doctrine), and add a named
   `TestModelLimitsFailsOnAnUnreadableDeclaration` (chmod-based fixture, cleanup restoring
   the mode). If instead WARN was the deliberate choice for read errors, say so in AC7's
   text — but either way the next round must be able to read one story from code and spec.
2. **code + tests (F2)**: count `compared` only when at least one limit was actually
   comparable (the M5 shape — proven invisible to the current suite), and add the
   catalog-hit-with-zero-limits fixture asserting SKIP-not-PASS when nothing was compared.
3. **spec (F3)**: correct the AC4 motivating example — the collision partners are
   `alibaba`/`requesty`/`hyper`/`llmgateway`/`vancine`/`opencode-go` (10 providers, bare id),
   not `openrouter` (which only publishes `qwen/qwen3.8-flash`); optionally refresh the
   code doc-comment's matching sentence. The mechanism itself needs no change — today's
   cache shows it is even more needed than argued.
4. **tests (F4)**: one fixture with `RepoDir: ""` pinning the checkout SKIP; one test for
   the zero-limit row covers F2's same "never vacuous PASS" rule, so 2+4 together close the
   two untested SKIP corners.
5. **handoff (F7 + F5)**: re-run the `features.json` harness so state/evidence are recorded
   before archive; tick or restate the merged PR's box; and either extract the
   read/parse helpers to bring `checkModelLimits` inside the repo's CC/length bar or file
   the package-wide (checkModelPins shares the shape) ticket instead — do not silently
   leave the rule unenforced *and* unacknowledged.
6. Re-run `dotf spec review HARNESS-136-model-limit-drift` for round 2. `dotf spec
   archive` remains correctly refused until a fresh passing review exists on the edited
   contract files.

**Advisability**: `dotf spec archive` is **not advisable** in this state — the gate itself
will refuse (FAIL verdict + contract files will have changed). Minimum flip-to-PASS set:
items 1 (either side of the reconcile), 2, 3, 4 — then a clean round-2 review.
