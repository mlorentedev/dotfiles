---
spec: "GUARD-003-review-loop-determinism"
verdict: "FAIL"
reviewed_sha: "8be1f21a26922f66edf2ea6bd0aabc73cf868fd3"
reviewer: "nan/deepseek-v4-flash"
date: "2026-10-21"
---

## Adversarial review

**Scope**: `GUARD-003-review-loop-determinism` — the review-loop wake-up: the `workflow_run` trigger on
`review-attestation.yml`, the `/review` (issue_comment) resolution path, and `dotf pr triage-queue`
(`cli/internal/prtriage`). Reviewed at HEAD `8be1f21a`.

The stated scope, `e608989e5437f6e465279c83ee09aa34b9aba8e5...HEAD`, is **3034 files / 262,196
insertions**: that base is the parent of this spec's first commit (`1a5f3cf4`, 2026-08-17) and roughly
2000 unrelated merges have landed between it and HEAD. The spec's own surface is 8 commits touching
~15 files. I reviewed that surface — every file each GUARD-003 commit touches, at its HEAD state — and
loaded the HEAD state of the two workflows and the registry rather than only the diff, because later
work (e.g. #2209's exemption, the `pull_request` blanket check-run exemption in #1185) changes what the
early commits mean. Recorded as F3: the base is not a reviewable unit for this spec, and anyone reading
a PASS/FAIL here should know what was actually read.

**Sources**: `specs/GUARD-003-review-loop-determinism/{proposal,tasks,verification}.md`; commits
`1a5f3cf4` (spec + trigger), `1a29340f` (triage queue), `4b76defa` (check-run amendment), `0ad7ac21`
(`/review` path, #2848), `af2d8dc5`/`17b59ddb`/`8be1f21a` (spec records);
`.github/workflows/review-attestation.yml`, `.github/workflows/pr-agent.yml`,
`scripts/check-review-attestation.sh`, `harness/review-attestation.json`,
`cli/internal/prtriage/{prtriage,fetch}.go`, `cli/internal/cmd/pr_land.go`,
`tests/review-attestation-workflow.bats`, `tests/check-review-attestation.bats`.

### Spec and task alignment

- **T1/T2 (AC5, AC1, AC4)** — `workflow_run: workflows: [pr-agent], types: [completed]` is present, the
  job `if:` admits it, and the resolve step recovers the PR by `workflow_run.head_sha` for the
  `pull_request` path and by the `reviewed-pr` artifact for the `issue_comment` path. All three are
  guarded by name (`tests/review-attestation-workflow.bats` cases 2, 3, 4, 16, 20).
- **T3 (AC2)** — the verdict is published to the `review-attestation` **commit status**, and the step
  that would freeze the check-run is exempted on the `pull_request` event only. Confirmed against the
  live API, not the prose.
- **T4 (AC3)** — non-`pull_request` paths still fail; named cases 8 (`every non-pull_request path still
  fails [#1041]`) and 7 exist and pass.
- **T5/T12 (AC1 live)** — reproduced from the actual run log: run `39621973877` is
  `event=workflow_run`, `conclusion=success`, and prints `re-evaluating PR #2855 because the reviewer
  finished` followed by `[OK] attested — reviewed by: coderabbitai, github-actions`.
- **T6 (AC6)** — the adoption order is in the workflow header, and the required check is real: branch
  protection's contexts are `["lint","lint-powershell","test","test-windows","review-attestation",
  "cli-gate","spec-gate","changes","knowledge-gate","from-zero-gate"]`. The name required is the
  **status** context; the check-run is named `attestation`, so the two cannot be confused in practice.
- **T7/T8 (AC8, AC7, AC9)** — the marker is declared once, in `harness/review-attestation.json`
  (`triage.marker = "## Review triage"`); the queue reads it and refuses an empty one; the queue lists
  and never applies. The read side is genuinely single-sourced. The write side is not (F2), and the
  record's *author* is not considered at all (F1).
- **T10/T11** — delivered, with their own issues (#1196, #1128); both were outside this review's
  adversarial focus except where they interact with the trigger's grouping, which they do correctly
  (a `/review` run keys its group on the run id, so two such runs cannot cancel each other — case 18).
- One task box is deliberately open (`Adversarial review passes before archive`) — correct as of this
  review. All nine `proposal.md` acceptance criteria remain **unticked** (F4).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | trust boundary — triage record authorship | `lastTriage` accepts **any** comment whose body has a line starting with the marker, from **any** author (`Author` is never inspected; the REST fetch stores it and nothing reads it). That record clears the PR from `dotf pr triage-queue`, and `pr land` treats "the triage queue does not list it" as one of its landing conditions (`pr_land.go` `Untriaged:` → `prtriage.FetchWithRegistry`). On this **public** repository any account can therefore clear a merge precondition by posting `## Review triage`, and the reviewer findings the queue exists to surface go unread. Same shape the gate already rejects on its own side: `pr-agent.yml` gates `issue_comment` on `author_association ∈ {OWNER, MEMBER, COLLABORATOR}`, and the gate counts reviewer output only from declared logins — the triage reader does neither. Consequence is bounded (the required `review-attestation` status still refuses an *unreviewed* PR), but the DoD "Review" item this spec set out to make checkable becomes forgeable by a stranger, which is the property the spec claims to have replaced. | Reproduced this session: a temporary `Evaluate` test with reviewer output at `t00` and a triage comment at `t10` authored `drive-by-stranger` returned `Pending=false`, `Reason="triaged"` (probe file removed after the run). Code: `cli/internal/prtriage/prtriage.go:210-233` (`lastTriage`), `cli/internal/prtriage/fetch.go:320-347` (author stored, never filtered), `cli/internal/cmd/pr_land.go:36-49`. | UNTESTED — every existing table case uses an owner author (`cli/internal/prtriage/prtriage_test.go:49,59,75,88,100`); no case asserts that an undeclared author does *not* clear the queue | **code + tests** (`Evaluate`/`lastTriage` gains an authorship predicate; add `TestEvaluate_AnUndeclaredAuthorDoesNotClearTheQueue`, red before the fix) |
| Minor | THEORETICAL | SSOT drift (AC9) | AC9 holds for *readers* only. The **writer** — `harness/skills/pr-review-triage/SKILL.md:62-63` — hardcodes `## Review triage`, as do `docs/runbooks/release-dotf.md:79` and `docs/runbooks/guide-pr-agent-reviewer.md:333,338`. Nothing ties the prose to `triage.marker`: changing the registry marker failed **no** test. The failure direction is loud, not green (the queue would list every PR as never triaged and `pr land` would refuse), so this is drift, not a bypass. | `go test -count=1 ./internal/prtriage/` → `ok` with `triage.marker` mutated to `## Review dispositions` (mutation confirmed by re-reading the file, then reverted) | UNTESTED — no guard compares the skill's heading with the registry | **tests** (a bats/golden guard asserting the skill's marker equals `triage.marker`, the same two-consumer guard AC9 asked for) |
| Minor | REAL | review scope / base resolution | The launcher-resolved base predates ~2000 merges, so `base...HEAD` is not this spec's change (3034 files vs the spec's ~15) and cannot be reviewed as one unit. Not a defect in the change; recorded so the archive decision is made on a stated scope. `proposal.md` also still reads `status: draft`. | `git diff --stat e608989...HEAD` → `3034 files changed, 262196 insertions(+), 18993 deletions(-)`; `git log -1 --format=%p 1a5f3cf4` → `e608989e` | n/a (measurement) | **spec/vault** — out of scope here (the archive gate's implementation lives outside this repo); the launcher's base rule is the owner's call |
| Minor | REAL | spec artifacts | All nine acceptance criteria in `proposal.md` are still `- [ ]` although `verification.md` maps each to evidence. A reader who trusts the checkboxes reads the spec as unimplemented. | `grep -n '^- \[ \] AC' specs/GUARD-003-review-loop-determinism/proposal.md` → AC1–AC9 unticked | n/a | **spec** (contract set — legitimate to edit on a FAIL round; it re-stales this review, and the next round re-measures) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | AC1/AC2/AC4/AC5/AC6 reproduced from live logs, statuses, branch protection and mutation runs; AC7–AC9 hold on the read side, but the AC8 mechanism has one unhandled negative case — a triage record from an undeclared author (F1), reproduced. |
| Verification       | A | Every claim I could reach reproduced exactly, including the numbers: 83/83 bats, `shellcheck`/`bash -n`/`actionlint` exit 0, the AC3 mutation producing precisely the claimed 21 failures, and the 2026-10-21 run log and status timestamps. Mutation landing is checked by checksum, which is the discipline this repo has been burned without. |
| Scope              | B | Diff matches the proposal plus three documented follow-on tasks, each with its own issue; no creep inside the reviewed surface. The outsized base diff is the launcher's, not the author's (F3). |
| Reliability        | B | Fail-closed on every path it does handle — unknown artifact, non-numeric number, closed PR, skipped reviewer run, unreadable registry, truncated comment page. The one gap is F1's trust boundary, counted under Correctness so it is not double-charged. |
| Maintainability    | B | Small functions, low nesting, the whole rationale carried in comments at the point it constrains; the reader-side single-sourcing (`ParseRegistry` refusing an empty marker) is the right seam. Writer-side duplication is F2. |
| Handoff-readiness  | B | `docs/lessons/lesson-502-a-workflow-run-from-a-comment-triggered-run-has-lost-its-pull-request.md` is present and the three promotion candidates are answered with reasons; the unticked ACs and `status: draft` (F4) are the gap. |

### Verdict
FAIL

One **REAL Major** (F1). Its fix is small and its regression test is nameable, and the spec is not yet
archived, so this is the cheapest round in which to close it. The three Minors are tracked, not
blocking on their own; F1 alone decides this verdict.

### Recommended next steps

Contract set is **open** on a FAIL, so edits there are legitimate this round — and each one re-stales
this review, which the next round re-measures.

- **Fix F1 (code + tests).** Require the triage record's author to be someone with standing to write
  it — the `author_association` precedent is already in this repo (`pr-agent.yml:147`: `OWNER`,
  `MEMBER`, `COLLABORATOR`). Add the named regression test
  `TestEvaluate_AnUndeclaredAuthorDoesNotClearTheQueue`, observe it red before the fix, and record the
  checksum-verified mutation in `verification.md`.
- **Amend AC8 in `proposal.md`** to say *who* may write the record. Without it the next reader will
  re-derive the same gap, and "a record exists" is not the same claim as "someone accountable made it".
- **F2 (tests).** Guard the writer side: assert the `## Review triage` heading in
  `harness/skills/pr-review-triage/SKILL.md` equals `harness/review-attestation.json → triage.marker`,
  so the one-string-two-consumers claim is enforced in the direction that is currently a copy.
- **F4 (spec).** Tick the AC boxes once the fix lands, alongside the closing checks.
- **Do not archive yet.** `dotf spec archive` should be run only after a re-review of the fix passes;
  the verification window is between the fix and the merge, and the reviewer for that round cannot be
  this session.

### Evidence reproduced in this session

```
bats tests/review-attestation-workflow.bats            -> 21/21 ok (cases 1-21)
bats tests/check-review-attestation.bats \
     tests/review-attestation-workflow.bats            -> 83 ok
shellcheck scripts/check-review-attestation.sh        -> exit 0
bash -n scripts/check-review-attestation.sh           -> exit 0
actionlint review-attestation.yml pr-agent.yml        -> exit 0
go build ./... ; go test ./internal/prtriage/         -> ok
(cd cli && go test -count=1 ./internal/prtriage/)     -> ok
```

Mutations, each with the landing confirmed by `shasum` on the file before and after, then reverted
(`git status --porcelain` clean apart from this review's own untracked files):

```
workflow_run trigger deleted from review-attestation.yml
  -> not ok 2 "re-evaluates when the pr-agent reviewer finishes [AC1]"
     # no workflow_run trigger: our own reviewer finishing wakes nothing
  -> restored, no diff

artifact name reviewed-pr -> reviewed-pr-x in pr-agent.yml
  -> not ok 20 "pr-agent: the /review path publishes the PR number for the gate [#1128]"
  -> restored, no diff

classifier's final exit 1 (line 437) -> exit 0
  -> bats tests/check-review-attestation.bats: 21 "not ok"  (verification.md claims 21)
  -> restored, no diff

triage.marker "## Review triage" -> "## Review dispositions" in the registry
  -> go test -count=1 ./internal/prtriage/ still ok, no bats failure  (F2)
  -> restored, no diff
```

Live, read-only (no writes to GitHub):

```
gh run view 39621973877 --json event,conclusion,headSha
  -> {"event":"workflow_run","conclusion":"success","headSha":"4c92cfc4..."}
   log -> "re-evaluating PR #2855 because the reviewer finished"
          "[OK] attested — reviewed by: coderabbitai, github-actions"
gh api repos/mlorentedev/dotfiles/commits/9be46c48/statuses
  -> 03:46:44 failure review-attestation [FAIL] pending — coderabbitai could not review (advisory)
     03:49:37 success review-attestation [OK] attested ...
     03:49:53 success review-attestation [OK] attested ...       (AC2's "latest verdict", not the first)
gh api repos/mlorentedev/dotfiles/branches/main/protection --jq .required_status_checks.contexts
  -> [..., "review-attestation", ...]                            (AC6)
```

F1's reproduction (temporary file `cli/internal/prtriage/zz_adv_test.go`, removed after the run):

```
--- FAIL: TestAdv_UndeclaredAuthorClearsTheQueue (0.00s)
    REPRODUCED: an undeclared author's triage cleared the queue (triaged)
```
