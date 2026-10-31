---
spec: "TOOL-017-deterministic-review-triage"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "a8fd02dea5b7aa927180ba4ce68d1a3b6f91a299"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-30"
---

## Adversarial review

**Scope**: TOOL-017-deterministic-review-triage — the whole accumulated change, per
the launcher-resolved base `bb2fa2383a2990c741a1a0148790f4973f40da8f` (the parent of
the spec scaffold `b25751ec`, 2026-08-17) through
`a8fd02dea5b7aa927180ba4ce68d1a3b6f91a299`.

**Sources**: `specs/TOOL-017-deterministic-review-triage/{proposal.md,tasks.md,verification.md,features.json,review-request.json}`;
the eight TOOL-017 commits in range (`b25751ec`, `50e538db`, `97724428`, `ac5d6fb0`,
`b2523828`, `eb272d10`, `37ecb8db`, `a8fd02de`); code in `cli/internal/{prtriage,reproduce,prland}`,
`cli/internal/cmd/{pr_reproduce,pr_land,pr_triage*}`, `harness/review-attestation.json`,
`.pr_agent.toml`, `tests/pr-agent-config.bats`, `cli/internal/spec/{promotion,archive}.go`
(the archive gate, in this repo).

### Spec and task alignment

- **Base note (scope, not a defect):** the stated range spans 1107 commits / 3166
  files — the repo's whole evolution since the spec scaffold, because TOOL-017's
  slices interleaved with other work over ten weeks. Within it, TOOL-017's footprint
  is the eight commits above plus the spec folder, the prtriage/reproduce/prland
  gates, the registry's `classification` block, and the `.pr_agent.toml` ask. That
  footprint — early slices and late fixes together — is what was reviewed; commits
  of other issues in the range were attributed to their own specs, not to this change.
- **Contract integrity:** `dotf spec digest TOOL-017-deterministic-review-triage`
  output matches `review-request.json`'s `contract_digests` exactly for all three
  contract files. Contract files were not modified by this review.
- **Evidence freshness:** the only commit after the evidence sha `37ecb8db` is the
  docs-only `a8fd02de`; `git diff --stat 37ecb8db..HEAD -- cli/ harness/ .pr_agent.toml
  scripts/ tests/` is empty, so `features.json`'s evidence applies at `reviewed_sha`.
- **AC1** (superseded trigger, owner-accepted): pull-based queue — `TestEvaluate`,
  `TestFetchReviewTriagedAfterwardsIsClear` ran ok in-session (f1).
- **AC2** (registry single-source): `TestTheRealRegistryServesBothConsumers`,
  `TestEvaluate_AnUndeclaredAuthorDoesNotClearTheQueue` ran ok (f2); no second copy
  of the review-vs-notice distinction found in the diff.
- **AC3** (declared marker, no prose parse): `Classify` requires a whole line equal
  to `prefix + class`; registry declares `classification.reviewers: ["github-actions"]`;
  `.pr_agent.toml` asks for exactly `Class: REAL|THEORETICAL|SPECULATIVE`. f3 (5/5)
  and the named bats contract test `pr-agent: the reviewer is asked for every class
  marker the registry declares` both passed in-session.
- **AC4/AC6** (REAL needs reproduction; refuted needs its row): `UnreproducedFindings`
  + `withReproductions` hold the PR; `dotf pr land` refuses through `Untriaged`
  (`internal/cmd/pr_land.go`, `prland.go:513`). f4 (5/5), f6 (5/5), f7 (3/3) ran ok.
- **AC5** (four states, did-not-land ≠ did-not-reproduce): f5 (8/8 named tests)
  ran ok; `classify` keeps 126/127/signal/timeout as `could-not-run` under inversion too.
- **AC7** (owner option B, PR-level gate): `TestLand_RefusesAnUntriagedPR`,
  `TestLand_AnUnanswerableTriageQueueIsARefusalNotAPass` ran ok.
- **AC8** (red-direction): the PR-recorded mutation passes were not re-run in full
  (UNVERIFIED); this review ran **four independent mutations**, each confirmed landed
  by diff and each killed by a named test, then reverted: (M1) exact-line match
  weakened to a prose `strings.Contains` → `TestClassify/{header_suffix_only,
  first_word_of_the_body_only,marker_inside_prose}` red; (M2) 127 removed from
  `could-not-run` → `TestClassify/not_found_by_a_shell` and
  `TestClassifyAnInvertedCheck/not_found_by_a_shell` red; (M3) the
  `Reproduction: reproduced` row check neutered → `TestUnreproducedFindings`
  (+3 subtests) and `TestFetchClearsATriagedPRWhoseRealFindingWasReproduced` red;
  (M4) the `Pending = true` on unreproduced findings removed →
  `TestFetchHoldsATriagedPRWithAnUnreproducedRealFinding` red.
- **Fresh command evidence (this session):** `go build ./...` exit 0; `go vet ./...`
  exit 0; `GOOS=windows go vet ./...` exit 0; full `go test -count=1 ./...` exit 0
  (50 packages ok, zero FAIL); all eight `features.json` verification commands exit 0
  with their named tests actually running (e.g. 22 PASS lines for f4); `golangci-lint
  run` on the three packages → 0 issues; the named bats contract test → ok; working
  tree clean apart from this review and the launcher's `review-request.json`.
- `tasks.md` boxes all `[x]` and consistent with the diff; `verification.md` has one
  unticked evidence box (Criterion 3) — finding below. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]`
  tags anywhere in the spec folder.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | REAL | archive gate / spec | `dotf spec archive`'s third pre-flight (promotion answers) will refuse this spec as `verification.md` stands: the lesson line answers `yes, written: lesson 512 …` with no file path, and the pattern line answers `candidate, not yet: …`, which is neither `yes` nor `no` | ran the gate's own `CheckPromotions` (`cli/internal/spec/promotion.go`) against this spec dir in-session → 2 problems: `"Lesson for the repo's docs/lessons.md? "yes" names no file"`; `"New pattern candidate … unanswered: write \"yes: <path>\" or \"no: <reason>\""` | UNTESTED (the parser is tested in `cli/internal/spec`; no test covers these two answers) | spec (`verification.md` — outside the contract set; answer before archive) |
| Minor | REAL | spec / verification | Criterion 3's evidence line is the only unticked box in `verification.md`'s Evidence list, although f3 passes and its named tests exist — and it sits alongside promotion answers that are ticked `[x]` yet malformed (row 1), so the file contradicts itself in two places the different archive pre-flights will not reconcile for a reader | `verification.md` `- [ ] Criterion 3` vs `features.json` f3 (`state: passing`); f3 rerun in-session, 5/5 ok | f3 command (`go test -run '^(TestClassify\|…)$'`, ran) | spec (`verification.md`) |
| Minor | THEORETICAL | prtriage / AC4-AC6 | `rowFor` identifies a finding's triage row by **base name** + line only; two findings on same-basename files at the same line (e.g. `a/util.go:4`, `b/util.go:4`) read each other's row, so a REAL finding is judged against another row — worst direction is wrongly *cleared* when the look-alike row carries `apply` + a reproduction | code read of `reproduction.go` `rowFor`/`judge` (regex uses `path.Base`); mechanism is in the code, never observed (needs that table shape) | UNTESTED (no test for basename collision) | code + tests |
| Minor | THEORETICAL | prtriage / AC3 | findings from reviewers **not** listed in `classification.reviewers` get neither the unclassified escalation nor any gate: `Classify` is author-blind, so a marked `Class: REAL` from anyone is judged, but an *unmarked* finding from an unlisted reviewer (CodeRabbit) passes through the whole mechanism untouched — heavier than a reporting gap, and narrower than proposal AC3's text ("a finding without [a marker] escalates to a human"). The narrowing is deliberate and documented (registry `why`; `verification.md` decision log: CodeRabbit's severity format would flood the report) | code read of `UnclassifiedFindings`' `asked[]` filter vs proposal AC3 | `TestFetchReportsUnclassifiedInlineFindings` (pins the current scoping) | spec — owner disposition on the AC3 wording (contract set: do not edit post-verdict; record the disposition in `verification.md` or on #1062) |
| Minor | SPECULATIVE | prtriage / AC4 | the gate skips a finding newer than the triage record on the premise "it already makes the PR pending"; that holds only when the reviewer's *counted* channel (marker comment or formal review) also moved — an inline comment that arrives or is edited after triage without a new counted review is neither pending nor judged | code read: `UnreproducedFindings` (`f.Spoken().After(record.CreatedAt)` → continue) + `reviewOutput`'s channels; ordering window argued, never reproduced | UNTESTED | code + tests (or an owner note that the premise is accepted) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All eight acceptance criteria verified fresh with named tests; fail-closed design throughout; residual negative-path gaps are rows 3–5. |
| Verification       | B | Every `features.json` command, the full suite, lint, windows vet and the bats contract all reproduced in-session; held back by the unticked AC3 box, the unanswered promotion lines, and PR-recorded mutation passes not re-run (4/4 spot-kills instead). |
| Scope              | A | The spec's footprint matches the proposal's four pieces exactly (trigger superseded by owner decision, registry consumer, marker, gate) with no creep; the range's other ~1100 commits belong to other issues by construction of the base. |
| Reliability        | B | Fail-closed on every unread/refuse path (dirty tree, could-not-run, did-not-land, unread findings, unanswerable queue), bounded fan-out and WaitDelay; one speculative ordering window (row 5). |
| Maintainability    | B | `golangci-lint` 0 issues, functions within the repo's limits, tests ≈ half the footprint, rationale comments; comment volume is high but purposeful. |
| Handoff-readiness  | B | Proposal/tasks/verification current, lessons 512 and 518 written, decisions logged; outstanding: two promotion answers, the AC3 evidence box, the pattern-candidate answer. |

Aggregation: no D, no C → no rubric escalation; severity path: no Blocker, no Major.

### Verdict

**PASS WITH GAPS** — zero Blockers, zero Majors; five Minors (2 REAL, 3 THEORETICAL/SPECULATIVE),
each tracked above with a disposition line. The two REAL Minors are closing paperwork in
`verification.md` (outside the contract set), not defects in the shipped gates; the code
gates were verified by fresh test runs and by four independent, landed-and-killed mutations.

**Second opinion** (advisory, per the standing order; it does not sign this review): a
non-flash model re-read the skill and this review, spot-checked the `rowFor` base-name
claim and the `CheckPromotions` refusal against the code (both confirmed), searched for a
missed Blocker/Major and found none, and agreed with PASS-WITH-GAPS. It differed only in
tone — pressing that row 4's consequence is heavier than a reporting gap and that row 2's
file is self-contradictory; both refinements are folded into the table above. (An earlier
advisory pick, `nan/mimo-v2.5`, was refused by the API with 401 and was not used.)

### Recommended next steps

(For the implementer to disposition in `verification.md` — applied, ticketed, or declined
with a reason. None of these touch the contract set; editing `proposal.md`, `tasks.md` or
`features.json` would stale this verdict.)

1. **Before `dotf spec archive`** (this will otherwise refuse at pre-flight 3): rewrite the
   lesson promotion answer as `yes: docs/lessons/lesson-512-a-landing-check-must-read-paths-in-the-frame-the-patch-wrote-them.md`
   (confirm the exact path) and answer the pattern line as `no: <reason>` or
   `yes: 00_meta/patterns/<name>.md` — proven by running `CheckPromotions` in-session (2 problems).
2. Tick (or explicitly explain) Criterion 3's evidence box in `verification.md`; its tests
   were rerun in this session.
3. Disposition rows 3–5: a follow-up ticket for the `rowFor` basename-collision guard test
   is the concrete candidate; rows 4–5 are owner dispositions on recorded design decisions.
4. Optional: re-run the PR-recorded mutation passes (#2845/#2863/#2903/#2924/#2935) via
   `dotf pr reproduce` if a fresh record is wanted for the archive — this review could not
   fetch those PR logs (**UNVERIFIED**), and the full `bats tests/*.bats` suite was not
   re-run (only the named TOOL-017 contract test).

**Archive advisable?** Yes — once item 1 is written; the review itself will pass the
freshness pre-flight (digests match `review-request.json`, `reviewed_sha` is HEAD).
