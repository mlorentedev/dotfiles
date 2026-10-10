---
spec: "HARNESS-063-spec-gate-adjacency"
verdict: "FAIL"
reviewed_sha: "dbed0e3cfc6677e62cd928dbed61354c2c178286"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-10"
---

## Adversarial review

**Scope**: `HARNESS-063-spec-gate-adjacency` — advisory issue-adjacency report in `spec-gate`
**Sources**: `specs/HARNESS-063-spec-gate-adjacency/{proposal,tasks,verification}.md`, `features.json`,
`review-request.json`; `scripts/check-spec-gate.sh` (lines 12, 37-80, 427-538), `scripts/spec-gate-pr.sh`,
`.github/workflows/spec-gate.yml`, `tests/spec-gate-adjacency.bats`, `tests/spec-gate-pr.bats`;
change commits `1913c59c` (#860, the feature), `327feac2` (#885, the CI wiring + the workflow pin),
`dbed0e3c` (#2301, features + closing status). Live CI runs `38082579092`, `38082350747`.

**Review scope note (finding 8).** The stated base `bed3f1f7` is 769 commits and 2,790 files / 222k
insertions behind `HEAD` — the spec's work landed on `main` in August, so `base...HEAD` is the whole
repository, not this change. I reviewed the change set identified by the commits that touch the files
this spec names (above), which is the only reading under which the diff is reviewable; the range itself
was not usable. Every command below was run against `HEAD` = `dbed0e3c` (= `origin/main`, 0 ahead).

### Spec and task alignment

- **What shipped matches the proposal's split.** Network in the workflow (`Collect open issues for the
  adjacency report`, one `gh issue list`, `permissions: issues: read`); matching in the script behind
  `--adjacency-issues <file>`; `_closing_issue_numbers`/`_repo_slug` contain no network call, so the
  offline pre-push path really is offline. Confirmed at `HEAD`, not from the claim.
- **AC1 holds in production, verified independently.** I did not take `verification.md`'s local replay as
  proof of CI emission: 5 of the 12 most recent `spec-gate` runs printed the report, e.g. run
  `38082579092` — `#2278 names versions.conf`, `#2273 names versions.conf`, `#2136 …`. The wiring line
  (`spec-gate.yml:91`) is present and `spec-gate-pr.sh` forwards unknown args verbatim (`FORWARD[@]` →
  `exec`), which `tests/spec-gate-pr.bats:77-79` pins with a stub gate.
- **AC2's polarity pin is real.** I reproduced the mutation `verification.md` claims: injecting
  `haystack=$(_strip_markdown_code "$haystack")` into `_adjacent_open_issues` turns case 2 red and leaves
  case 1 green (because #849's *title* carries the bare basename). The stated asymmetry is exact.
- **AC4's byte-identical pin has moved** (finding 10): the baseline it diffs against is
  `origin/main:scripts/check-spec-gate.sh`, which at this HEAD *contains the feature* — `cmp` against the
  worktree copy returns identical. The test therefore cannot fail on `main`; it is a branch-vs-main drift
  guard, not a pin on "this feature changed no offline output".
- **Records contradict the shipped state** (finding 2). `tasks.md` carries 3 ticked boxes at both `1913c59c`
  and `HEAD` — all of them Setup. Every Implementation and Closing box, including "shellcheck passes",
  "`bats tests/*.bats` passes" and "`verification.md` filled in", is unchecked while all three are true as
  measured by me. `proposal.md` shows all 4 acceptance criteria unchecked and `status: draft` in the same
  folder where `features.json` records f1-f4 as `passing`. `cli/internal/spec/archive.go` reads no task
  ticks (it auto-ticks only the archive checklist), so nothing corrects this: the folder would archive as
  a permanent record saying the work was not done.
- **Board state is correct.** `#858` is OPEN, labelled `chore`/`debt`, assigned; this spec carries
  `Refs #858`, and `verification.md`'s 2026-10-10 closing status records that the fixture-shape inventory
  did not ship and stays tracked there. Consistent with the proposal's out-of-scope section. No open PR
  exists for the archive branch yet (`HEAD` == `origin/main`).

### Verification performed (this session, fresh)

```console
$ bats tests/spec-gate-adjacency.bats                  → 1..9, all ok, exit 0
$ bats tests/spec-gate-{adjacency,pr,prepush,archive}.bats tests/check-spec-gate.bats
                                                       → 102 ok, 0 not ok, 0 skip, EXIT=0
$ shellcheck scripts/check-spec-gate.sh                → clean
$ python3 <features.json verification commands f1..f4> → exit 0, exit 0, exit 0, exit 0
$ cmp <(git show origin/main:scripts/check-spec-gate.sh) scripts/check-spec-gate.sh
                                                       → identical (finding 10)
$ gh issue list --state open --limit 500 --json number --jq length
                                                       → 420 open issues today (finding 3)
$ gh run view 38082579092 --log | grep 'Adjacent open issues' → report emitted in CI (AC1)
$ gh show 1913c59c -- .github/workflows/spec-gate.yml  → collected the feed, never consumed it (finding 1)
$ git show <1913c59c|dbed0e3c>:…/tasks.md | grep -c '^- \[x\]' → 3 / 3 (finding 2)
```

Mutation probes, all reverted (scratch copy under `/tmp`; `git status` clean apart from this file and the
launcher's `review-request.json`): stripping the haystack (case 2 → red), neutralising
`ADJACENCY_GENERIC_BASENAMES` (all 9 green), deleting both `break` statements (all 9 green), deleting the
workflow wiring line (the pin stays green), plus two behaviour fixtures in a temp repo (`ssh/config` vs an
issue saying "configuration"; a renamed script vs an issue naming its old path).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| **Major** | **REAL** | CI wiring / test strength | The pin that exists to stop the adjacency report being wired-off again cannot fail: `grep -qF -- '--adjacency-issues' "$WORKFLOW"` is satisfied by the **comment** at `spec-gate.yml:60`, which mentions the flag while the gate ignores it. Deleting the wiring at line 91 keeps the suite green. | Past incident: `1913c59c` (#860) shipped the collection step and no consumer — the workflow comment records "had never once run in CI". Reproduction: the same guard's assertion run against a copy with line 91 removed → passes. The replacement recipe below is proven both ways. | `@test "spec-gate workflow: the adjacency feed it collects is actually consumed"` (`tests/spec-gate-pr.bats:205`) — exists, is the only pin for this half, and is vacuous → **UNTESTED in effect** | **tests** |
| **Major** | **REAL** | spec records (contract set) | `tasks.md` and `proposal.md` still record the change as unimplemented (0/7 implementation boxes, 0/6 closing boxes, 0/4 acceptance criteria ticked, `status: draft`) while the code is merged, `features.json` says f1-f4 `passing`, and CI emits the report. `dotf spec archive` never checks ticks, so archiving would freeze a false record beside its own evidence. | `git show 1913c59c:…/tasks.md`/`dbed0e3c:…/tasks.md` → `3` ticked both times (all Setup); `grep -c '^- \[ \]' proposal.md` → 4; vs. the CI runs above and my own green re-runs | UNTESTED (no gate reads task state; `cli/internal/spec/archive.go` has no such check) | **spec** (`tasks.md` ticks, `proposal.md` AC boxes + `status:`) |
| **Major** | **THEORETICAL** | feed completeness | `gh issue list --limit 500` truncates the feed silently. The moment open issues exceed 500 the report drops part of the backlog while still claiming to name "every other open issue", with no annotation and no test — silence indistinguishable from "nothing adjacent". | Live count today: **420**. The proposal measured **159** on 2026-08-09 → +261 in two months (~2.6×), so the cap is reachable within a few months, not hypothetically. Nothing in the workflow detects `rows == limit` | UNTESTED | code (workflow: annotate/`--search`-paginate when the count equals the limit) + tests |
| Minor | REAL | matching noise | Basename matching is a raw substring test with no word boundary, so a short basename matches ordinary prose. Mitigated by advisory-only + the report naming the token, but the deny-list only blocks 8 literal names. | Fixture: changed file `ssh/config`, issue body "the agent **config**uration drifts" → report printed `#424 names config` | UNTESTED | code (min basename length or word-boundary match) + tests (a `spec-gate-adjacency.bats` case) |
| Minor | REAL | untested branch | The generic-basename deny-list — a decision `verification.md` records as deliberate ("short explicit deny-list … rather than a length/shape heuristic") — has no test: neutralising it changes nothing. Case 5 pins `_excluded`/`_is_active_spec_path` precision, not the deny-list. | Mutation: delete `case " $ADJACENCY_GENERIC_BASENAMES " … continue` and empty the variable → 9/9 still green | UNTESTED | tests |
| Minor | REAL | AC4 pin moved | The characterization baseline for "byte-identical to the previous version" is now `origin/main`, which already contains the feature, so at `HEAD` the test compares a script with itself and can never fail on `main`. It guards branch-vs-main drift (useful) but no longer pins the criterion it cites. | `cmp <(git show origin/main:scripts/check-spec-gate.sh) scripts/check-spec-gate.sh` → identical; case 8 passes trivially | `@test "adjacency: with no flag the output is byte-identical to the previous version"` — passes against itself | tests (pin against the feature's merge-base or a stored golden offline output) |
| Minor | THEORETICAL | false negative | A renamed production file is matched only by its new path, so an open issue describing the old path — the shape of "we moved it and left the defect" — is not surfaced. `_excluded`/rename normalisation deliberately keeps the destination (`#397`), so this is a policy gap, not a bug. | Fixture: `git mv scripts/gamma.sh scripts/delta.sh`, feed naming `scripts/gamma.sh` → no report row; `git diff --name-only` prints only the new path | UNTESTED | code + tests, or record as accepted in `verification.md` |
| Minor | SPECULATIVE | report shape | Step-summary rows interpolate the raw issue text after `%.88s` with no escaping: a `\|` in a body breaks the markdown row, and `%` in a matched path is unescaped in the `::warning::` workflow command. Cosmetic on an advisory artifact. | Code read of `spec-gate` … `_report_adjacent_issues` printf block; not reproduced | UNTESTED | — (surface only; do not gate) |
| Minor | REAL | review protocol | The launcher's base for this review is 769 commits behind `HEAD`, so the "whole change" range is the whole repository. A reviewer following it literally either rubber-stamps 2,790 files or reports nothing. | `git rev-list --count bed3f1f7..HEAD` → 769; `git diff --stat` → 2790 files / 222,375 insertions | UNTESTED (`dotf spec review` base resolution has no "base must be the spec's own first commit" check) | code (`review_launch.go` base resolution) + a spec-worktree test |
| Question | — | AC1 wording | AC1 says the report fires "on a PR that closes an issue"; the implementation reports on any PR with a readable feed (the run cited above belongs to a `release-please` PR closing nothing). Broader than the criterion and arguably better — confirm it is intended, since the criterion text is what a future reviewer will test against. | Run `38082579092` log + `_report_adjacent_issues` (no dependency on `_closing_issue_numbers` being non-empty) | n/a | spec (author's call) |

**Strengths that mitigate documented risk** (not filler): the unstripped-matching policy is pinned by a
mutation-proven test — the exact failure class this spec was written for cannot silently re-enter the
*script* half; every failure path of the report returns 0 and the advisory property is pinned by a
same-status test (case 6), so a broken feed can never cost a merge; and the offline guarantee AC4 protects
is genuinely kept (no `gh` call reachable from the script's adjacency path).

### Fix for finding 1, proven both directions

The current assertion is a file-level grep. A step-scoped check cannot be satisfied by the comment:

```bash
run awk '
    /- name: Run SDD spec-gate/  { step = 1; next }
    step && /- name: /           { step = 0 }
    step && /--adjacency-issues/ { found = 1 }
    END { exit !found }' "$WORKFLOW"
[ "$status" -eq 0 ]
```

Verified in this session: green against `.github/workflows/spec-gate.yml` as it stands, red against the
copy with line 91 deleted — the mutation the present guard survives.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| **Correctness** | B | All four ACs hold and AC1 is verified live in CI; negative paths (precision, self-exclusion, missing feed) are tested; real noise/rename false paths remain. |
| **Verification** | B | Reproducible commands, and the claimed mutation reproduces exactly; but CI emission — the criterion's actual venue — had to be verified by the reviewer, and the "159 open issues" measurement is now 420. |
| **Scope** | B | Diff matches the proposal with the deliberate out-of-scope list honoured; the workflow half landed in another spec's PR (#885), which the spec does not mention. |
| **Reliability** | C | Three separate ways this feature goes silent with no signal: fetch failure → no file; >500 open issues → truncated feed; wiring deleted → suite green. Advisory-by-design is right, but a visibility check that cannot report its own absence repeats the class it exists to catch. |
| **Maintainability** | C | `_report_adjacent_issues` is 39 lines (1 under the cap) with ~15-16 decision points, and carries three untested branches (deny-list, dedupe `break`, summary-table shape); the refactor task covered the matcher only. |
| **Handoff-readiness** | C | Lesson promotion and closing status recorded; but both contract files still say the work is undone, and `status: draft` on a merged, CI-live change. |

Aggregation: no D (which alone would not save it), three C → PASS WITH GAPS at best; two REAL Majors
(findings 1 and 2) → FAIL. The more severe path governs.

### Verdict

**FAIL**

Not because the change misbehaves — on the evidence it works, in CI, and the polarity pin is genuinely
strong. It fails because the one test whose whole job is to keep the feature wired is satisfiable by a
comment, which is the same no-op that already happened here once; and because the contract files in the
folder about to be archived record the change as not done.

### Recommended next steps

**Contract set (this round is the mechanism; a re-review follows):**

1. Tick `tasks.md`'s seven implementation boxes and six closing boxes, and `proposal.md`'s four acceptance
   criteria — or record in each place why it stays open. Then set `status:` to match reality
   (`verifying`/ready-to-archive). Do not archive while the record and `features.json` disagree.
2. If the AC1 wording (Question row) was intended as written, narrow the criterion or the trigger in
   `proposal.md` in the same edit, so the next review round reads one contract.

**Outside the contract set — applicable without invalidating any verdict:**

3. Replace the workflow pin with the step-scoped awk above (proven) and keep the comment.
4. Add the missing named cases: generic-basename deny-list, word-boundary/short-basename noise, and a
   rename case if the old path is to be matched.
5. Make truncation observable in the workflow: if `wc -l < "$RUNNER_TEMP/adjacency.tsv"` equals the limit,
   emit `::warning::` naming the cap (the feed is at 420/500 today).
6. Re-pin AC4 against the feature's merge-base or a stored golden offline output instead of `origin/main`.
7. Record the report-shape items (pipe/`%` escaping, duplicate-row `break`) as `verification.md`
   dispositions — applied, ticketed, or declined with a reason — rather than leaving them only here.
8. Split `_report_adjacent_issues` (match / print / CI artifacts) so it is not one line under the 40-line
   cap at CC ~16.
9. For the review protocol (finding 8): `dotf spec review` should fail or warn when its resolved base is
   not an ancestor close to the spec's own first commit; until then, prompt the reviewer with the spec's
   own commit range. Filed as a recommendation, not a gate on this change.

**Minimum set that flips this to PASS:** items 1-3 (strengthened workflow pin, green suite after it, and
contract files that state what shipped), then a re-review round, since 1 edits the contract set.
`dotf spec archive HARNESS-063-spec-gate-adjacency` is **not advisable** in the current state.
