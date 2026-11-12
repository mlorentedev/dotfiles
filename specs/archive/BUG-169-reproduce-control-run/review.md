---
spec: "BUG-169-reproduce-control-run"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "5292c30c7512911cea26a2f8c01acac9b8949831"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: `BUG-169-reproduce-control-run` — `5292c30c7512 fix(reproduce): run a probed reproduction's check again on the clean head`, base `4718855ac149e9414cd5196b74dd123598524e59` (`git diff 4718855...HEAD`: 8 files, +241/−2; the only branch commit in the range).
**Sources**: `specs/BUG-169-reproduce-control-run/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`; `cli/internal/reproduce/reproduce.go`, `cli/internal/cmd/pr_reproduce.go`, the two new tests; `cli/internal/prtriage/reproduction.go` (the row's consumer); `harness/skills/pr-review-triage/SKILL.md`.

### Spec and task alignment

- Every `tasks.md` implementation box has diff evidence: `TestRunWithAProbeNeedsTheCleanHeadAsItsControl` (4 cases) plus `controlled()` for AC1/AC2, `TestRunWithAProbeRunsNoControlForARefutation` for the no-control path, `TestReproductionRowNamesTheControl` plus the help-text paragraph for AC3. The one unticked closing box ("`verification.md` filled in; independent review; archive") is accurate at review time — the review and archive halves are genuinely open.
- `features.json` carries `state: "pending"` / `evidence: ""` for all three features. That is the harness-owned pair the contract digest normalises out (`cli/internal/spec/contract_digest.go:52-55`), not an unimplemented feature, so it is not a finding.
- `contract_digests` in `review-request.json` equal `dotf spec digest BUG-169-reproduce-control-run` exactly:
  `features.json ee1c30b0…`, `proposal.md a58d89f5…`, `tasks.md fd58a0d7…`, `spec.md ""`.
- AC1/AC2/AC3 were verified by running, not by reading assertions: targeted tests, the full suite, and four end-to-end smoke runs of the built binary against a scratch git repository.
- The row's new `control <state> (exit N) on the clean head` clause does not disturb its only consumer: `cli/internal/prtriage/reproduction.go:20-21` matches the `Reproduction: reproduced` / `…did-not-reproduce` prefixes with `strings.Contains` (`judge`, line 92-101), and the whole `prtriage` package passes. The triage skill's table (`harness/skills/pr-review-triage/SKILL.md:162-163`) also matches by prefix with `…`, so it stays correct; the skill's silence on the control is a declared non-goal in the proposal.

### Evidence run in this session

| Command | Result |
|---|---|
| `cd cli && go build ./...` | exit 0 |
| `go test ./internal/reproduce -run 'TestRunWithAProbe' -count=1 -v` | PASS (all cases, incl. the 4 new sub-cases) |
| `go test ./internal/cmd -run 'TestReproductionRow' -count=1 -v` | PASS |
| `go test ./... -count=1 -timeout 5m` | exit 0, 59 packages `ok`, no `FAIL`/`panic` line |
| `go vet ./internal/reproduce/... ./internal/cmd/...` and with `GOOS=windows` | both exit 0 |
| `golangci-lint run ./internal/reproduce/... ./internal/cmd/...` | `0 issues` |
| **Mutation (red-green)**: removed `if res.State == Reproduced { res = controlled(ctx, o, res) }`, re-ran the new test | 4/4 sub-cases FAIL with `State, Control = "reproduced", ""`; file reverted, `git status` clean. The new tests bind the behaviour, they are not vacuous. |
| Smoke A — plain check that cannot run (`git grep -q nosuch`, both runs exit 1) | `could-not-run (exit 1, control reproduced (exit 1) on the clean head)`, stderr names the clean-head reading, exit 1, tree clean |
| Smoke B — plain check failing only under the probe (`git grep -q two`) | `reproduced (exit 1, control did-not-reproduce (exit 0) on the clean head)`, exit 0 |
| Smoke C — inverted check reading the same both ways (`git status`, `--defect-when-passes`) | `could-not-run (exit 0, defect when the check passes, control reproduced (exit 0) on the clean head)`, exit 1 |
| Smoke D — inverted check that **cannot run** (`git grep -q nosuch`, `--defect-when-passes`) | `did-not-reproduce (exit 1, defect when the check passes)`, **no control clause**, exit 0 (finding 1) |
| Smoke E — #3325 replay from `cli/` (`go test -C cli …`, probe on a comment) | `could-not-run (exit 1, control reproduced (exit 1) …)`, output contains `go: chdir cli: no such file or directory`, exit 1, tree clean — the `verification.md` smoke claim reproduces exactly |

### Findings

| # | Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|---|
| 1 | Major | THEORETICAL | inverted semantics | With `--defect-when-passes`, the guard misses the mirror image of #3325. A check that *cannot run* fails on both runs, so the probe run classifies `did-not-reproduce` and `probed()` never calls `controlled()` (`res.State == Reproduced` is false). The record therefore reads **refuted**, and `prtriage.judge` accepts exactly a `did-not-reproduce` row to justify the `refuted` disposition — so a REAL finding can be dismissed by a row produced by a check that never executed. AC1 deliberately scopes inverted coverage to the *passes*-both-ways (false-confirmation) case, so this is a declared boundary, not a contract violation; the cost is that the change's own help text ("one that reads the same both ways … is could-not-run") is false in this direction. | Smoke D: `Reproduction: did-not-reproduce (exit 1, defect when the check passes)` with no control clause, exit 0. `classify(1,false,true) == DidNotReproduce`; `cli/internal/cmd/pr_reproduce.go:48-51` states the broader claim. | `TestRunWithAProbeNeedsTheCleanHeadAsItsControl` covers only the inverted passes-both-ways case; the inverted fails-both-ways path is **UNTESTED** | code + tests (run the control on an inverted refutation too, or correct the help text) — or a follow-up ticket if the owner keeps option A as scoped |
| 2 | Minor | REAL | diagnostics wording | In the inverted case that *is* caught (Smoke C), the record says "the check read `reproduced` on the clean head too (exit 0), so the probe is not what made it fail". For an inverted check `reproduced` means the check **passed** (exit 0), so the sentence contradicts the state it just printed. The state and the reading are correct; only the causal clause is direction-blind. | Smoke C output, reproduced this session. | `TestRunWithAProbeNeedsTheCleanHeadAsItsControl` asserts only the substring `on the clean head`, never the clause → the wording is **UNTESTED** | code (`reproduce.go:160`, make the clause depend on `DefectWhenPasses`) + tests |
| 3 | Minor | THEORETICAL | record completeness | `controlled()` overwrites `res.Output` with the control run's output, so the probe run's failing output — the evidence that the probe did trigger something — is dropped from the record; only `res.Exit` survives for it. This matches the proposal's wording ("shows the control's output") and is therefore spec-compliant; the observation is that the could-not-run record is less diagnosable than the reproduced one was. | `reproduce.go:151-161`; Smoke A prints only the clean-head output (`git grep -q` output is empty). | `TestRunWithAProbeNeedsTheCleanHeadAsItsControl` asserts only the presence of the clean-head phrase → **UNTESTED** | code (append both outputs) — declining is defensible against the spec text |
| 4 | Minor | THEORETICAL | false-negative cost | The control compares *states*, not *failures*, so a check coarser than the defect now loses a genuine reproduction to `could-not-run` with no override: e.g. `-- go test ./pkg` where the clean head already has an unrelated failing test. `proposal.md`'s risk list names the opposite direction only (a flaky check passing the control). Consequence is a conservative no-verdict, never a false confirmation, so it is the safe direction to err; the mitigation is "narrow the check to `-run`". | Reading `controlled()` (any control state other than `DidNotReproduce` → `could-not-run`) and `classify`; no reproduction of a real-world case was built. | **UNTESTED** (no case exercises a coarser check) | code (document "narrow the check", or a `--no-control` escape) or docs (`verification.md` note) or follow-up ticket |

Non-findings checked and dismissed, so they are not silently dropped: the control runs **after** `restore()` and after the fingerprint equality check, so it cannot see a probed tree (correct by construction); a control that cannot start itself (exit −1 → `could-not-run`) degrades to `could-not-run`, not to a confirmation; `--timeout` is per run so a probe plus control can take `2 × timeout`, which the proposal's "accept the extra run" resolution covers; no new dependency; no injection/secret/auth surface added (the only new strings are a `fmt.Sprintf` of a `State` and an exit int); the code-level checklist turns up nothing else — `controlled()` is 11 lines, cyclomatic complexity ≤ 3.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1–AC3 verified by named tests and end-to-end smoke; negative-path gap in the inverted direction (finding 1) and a direction-blind message (finding 2) |
| Verification       | A | Every claim re-ran green this session: full suite exit 0/59 ok, vet native + `GOOS=windows`, lint 0 issues, red-green mutation, and the #3325 replay reproduced verbatim |
| Scope              | A | Diff is `cli/` + the spec folder only; matches AC1–AC3 with no creep — help text and row are the only non-test surface |
| Reliability        | B | Revert-and-verify happens before the control, tree left clean in every smoke case, control failures degrade to `could-not-run`; the inverted path is left unguarded |
| Maintainability    | A | 11-line function, ≤3 branches, WHY-comments naming the incident, state sentinel (`Control == ""`) documented, table-driven test |
| Handoff-readiness  | A | `verification.md` with reproducible commands and reproduced evidence, promotion candidates all answered `no: <reason>`, help text updated; the inverted gap needs a recorded disposition |

### Verdict

**PASS WITH GAPS**

One Major, **THEORETICAL** (finding 1): it does not force FAIL under `severity × reality`, and it is outside AC1's deliberate scope. Findings 2–4 are Minors. Rubric floor is B, no C and no D, so the rubric path also yields PASS WITH GAPS. `dotf spec archive` is **advisable** once the review round completes: no Blocker, no REAL Major, no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags, contract digests match, and the archive's third pre-flight (promotion candidates) is already answered.

### Recommended next steps

All four items land in **code/tests** or `verification.md`, **not** in the contract set — the verdict stands and `proposal.md` / `tasks.md` / `features.json` must not be edited now.

- **Finding 1** (Major/THEORETICAL) — disposition either way, recorded in `verification.md`: (a) apply — have `probed()` run the control whenever the *exit says nothing* as well as when it reproduced, i.e. cover the inverted refutation, with a `TestRunWithAProbeNeedsTheCleanHeadForAnInvertedRefutation` case; or (b) decline with the AC1 scoping as the reason **and** narrow the help text at `cli/internal/cmd/pr_reproduce.go:48-51` so it does not claim blanket "reads the same both ways → could-not-run". Do not leave both the broad claim and the unguarded path.
- **Finding 2** (Minor/REAL) — make the clause direction-aware (`reproduce.go:160`: "the probe is not what made it fail" vs. "…what made it pass") and assert the full message in the test, not just `on the clean head`.
- **Finding 3** (Minor) — either keep both outputs (probe run, then control run) or record in `verification.md` that dropping the probe output was chosen because the proposal says the message shows the control's output.
- **Finding 4** (Minor) — one line in the help text ("narrow the check to the test that names the defect; a coarser check that fails on the clean head is `could-not-run`") or a follow-up ticket, since it is an accepted cost of option A and not reproducible as a defect today.
