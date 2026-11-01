---
spec: "CLI-132-root-commit-review"
verdict: "PASS"
reviewed_sha: "8bc1913b4c14f8f86900d480519720ed0e144ac6"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-01"
---

## Adversarial review

**Scope**: CLI-132-root-commit-review, diff `055467ab8395be40f473cfce68d437a495a272dc...HEAD` (launcher-resolved base = the parent of the spec's first commit, `77ebe09e^`)
**Sources**: `specs/CLI-132-root-commit-review/{proposal.md,tasks.md,verification.md,features.json}`, `review-request.json`, PR #2977

### Spec and task alignment

- **Range composition (stated up front, because it shapes every reading of this
  diff):** the stated range is 163 files, +6326/−4260. The branch's own work is
  `055467ab..fea6939c` = **14 files, +1037/−58** — exactly the spec folder plus
  `cli/internal/spec/{review,review_launch,review_request}.go`,
  `cli/internal/cmd/spec*.go` and their tests, matching the proposal's
  `## References → Code` list with no creep. Merge `d313672f` (origin/main,
  #2976–#3015) and tip `8bc1913` (a `tasks.md` tick) supply the rest:
  `git diff fea6939c 8bc1913 -- cli/internal/spec cli/internal/cmd/spec*` is
  empty, so the merge touched no spec code. `AGENTS.md`, `harness/reviewer-pool.json`
  and the pr-agent/vm/secrets changes in the range came from main, not from this
  spec — see finding 1 for what that does to the reviewer prompt.
- **AC1–AC5**: every criterion maps to named tests, all run fresh this session
  (below). `[x]` in `tasks.md` is backed by diff evidence in each case.
- **Contract freshness**: `dotf spec digest CLI-132-root-commit-review` matches
  `review-request.json.contract_digests` exactly on all three files
  (`features.json f617e9bf…`, `proposal.md 2afc60cc…`, `tasks.md 9111b2aa…`).
  The request carries `base_sha: 055467ab…` and no `scope` — correct for an
  ordinary launch (`scope` is written only for `--initial`, and AC5 keeps
  pre-scope requests passing).
- **No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags** in any spec file.
- **`features.json`**: all five features `pending` with empty `evidence` —
  harness-owned by design ("Only the verification harness may move a feature"),
  not a defect.

**Fresh verification run this session** (from `cli/`):

- `go test ./internal/spec/ ./internal/cmd/ -count=1 -p 1 -timeout 40m` →
  `ok internal/spec 17.348s`, `ok internal/cmd 22.242s`.
- All five `features.json` verification commands → `ok` (f1–f5).
- `go vet ./internal/spec/ ./internal/cmd/` clean; `gofmt -l` empty;
  `go build ./cmd/dotf` exit 0.
- golangci-lint not run locally (endpoint-security doctrine: CI authoritative);
  PR #2977 `cli-lint` / `lint` checks are SUCCESS.

**Mutation testing** (each edit reverted; tree verified clean after):

| Mutation | Caught by | Result |
|---|---|---|
| Remove the `BaseResolved` refusal in `initialScope` (flag would accept an ordinary spec) | `TestSpecReviewInitialRefusesEveryOtherBaseState` / `TestSpecReviewInitialLaunchesForARootCommitSpec` | FAIL (red) |
| `checkReviewScope` `default:` → `return nil` (unknown scope accepted by archive) | `TestProvenanceRefusesAnUnknownReviewScope` (`review_request_test.go:462`) | FAIL (red) |
| Prompt two-dot `git diff %s HEAD` → triple-dot `git diff %s...HEAD` | `TestReviewPromptStatesTheInitialScope` | FAIL (red) |

**Live end-to-end smokes** (binary built from this HEAD, `/tmp/dotf-cli132`,
dry-run only):

- Ordinary spec (this one) + `--initial`: exit 1, refusal names the resolved
  base `055467ab…` and says to run without the flag. `review-request.json`
  sha256 `63229bcf…` identical before and after both dry-runs; nothing written.
- Ordinary spec without `--initial`: exit 0, `Scope: git diff
  055467ab…HEAD`, prompt bytes match `writeReviewScope`'s ordinary branch.
- Temp repository whose spec was added in its **root commit** (plus a later
  correction commit): without `--initial` → exit 1, refusal names the exact
  root (`d018028f…`) and the `--initial` command; with `--initial --dry-run`
  → exit 0, `Scope: initial review, git diff 4b825dc642cb6eb9a060e54bf8d69288fbee4904
  HEAD (spec first added in root commit d018028f…)`, prompt states the two-dot
  diff, forbids `...`, offers `git log <root>..HEAD`, and adds the
  "judge what the spec names first" guidance. Spec folder still contains only
  `proposal.md`; `git status --porcelain` clean.
- **Archive gate observed end-to-end** (throwaway local clone, so nothing in
  this repo could move): with this `review.md` and `review-request.json` copied
  in, `dotf spec archive CLI-132-root-commit-review` passed the draft and the
  review pre-flights — frontmatter parsed, verdict PASS accepted, reviewer
  `nan/mimo-v2.6-flash` matched the pool, contract digests fresh — and refused
  only at the third pre-flight (unanswered promotion candidates). With those
  three lines answered in the clone (`no: <reason>`), the archive completed:
  `[OK] Archived: specs/CLI-132-root-commit-review -> specs/archive/...`,
  `status: archived`. This reproduces finding 4 live and validates this review
  artifact against the real gate.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | REAL | prompt / ordinary scope | The ordinary-scope prompt tells the reviewer the range "is the WHOLE change" of the spec's work, but when origin/main is merged into the branch after the base, the range also holds other PRs' work — observed in this very round: 148 of 163 files and ~5.3k of 6.3k added lines arrived via merge `d313672f` (#2976–#3015), while the authored diff is 14 files. The initial-scope prompt guards the analogous case ("may hold files the root commit added for other work: judge what the spec names first"); the ordinary prompt has no such line | `git diff --stat 055467ab...HEAD` vs `git log --first-parent`; my own prompt text (written by `writeReviewScope`, ordinary branch) | UNTESTED — `TestReviewPromptKeepsTheTripleDotScopeForAnOrdinaryBase` pins the current bytes, so it would go red if guidance were added | follow-up ticket → code + tests (out of CLI-132's declared scope: the proposal leaves ordinary behavior unchanged) |
| Minor | REAL | CI / workflow | PR #2977 `gate` and `spec-gate` are FAILURE: `[FAIL] SDD archive-on-merge violation: This PR closes an issue whose spec is still active: CLI-132-root-commit-review (#2566)` — expected while the spec is unarchived; it clears only when `dotf spec archive` lands in this same PR, which this review enables. All other checks green (cli-lint, lint, tests on 3 OSes, knowledge-gate, cli-gate, review) | GitHub Actions run 41326679066 log, fetched this session | n/a (workflow state, not code) | process — no code change; disposition: expected, not a merge blocker once archived |
| Minor | THEORETICAL | tests | The combinations `--initial --second` and `--initial --fallback-reason` are untested: the second/fallback slot's request write shares `WriteReviewRequestScoped`, and `checkReviewScope` is slot-generic, but no test drives an initial scope into `SecondSigner`'s request | code read of `recordReviewRequest` / `WriteReviewRequestScoped`; no failure observed | UNTESTED | tests (outside contract set; follow-up or this PR) |
| Minor | REAL | verification.md | The three `Promotion candidates` lines are still the `<yes: path / no: reason>` placeholders; `dotf spec archive`'s third pre-flight refuses until each line reads `yes: <path>` (with the file existing) or `no: <reason>` | `dotf spec archive --help` + live refusal reproduced in a clone this session (`promotion candidates in verification.md are not all answered`) | UNTESTED | verification.md (outside the contract set — free to edit alongside this verdict) |

No Blocker and no Major was found. Security/Performance note: the new paths
spawn only `git` with argument arrays (`rev-list --parents`, `hash-object -t
tree --stdin`, `diff`) on launcher-derived values — no shell, no unbounded I/O,
no secrets; git failures are read as untraceable rather than as a root commit,
so a broken repository fails closed instead of widening the review to the whole
tree (`parentOf`).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All five ACs verified by named tests run fresh, three mutation reds, and two live end-to-end smokes; no defect found in the spec's code |
| Verification       | A | `verification.md` gives per-AC named tests plus a red-before-green table; every claim independently reproduced in this session |
| Scope              | A | Authored range is exactly the proposal's file list (14 files, no creep); the extra range content is merged main — surface as finding 1, not attributed to the change |
| Reliability        | A | Every non-root state refused fail-closed (git error → untraceable, never root); refusal and dry-run paths write nothing (evidence-unchanged tests + live sha256) |
| Maintainability    | A | gofmt/vet clean, new functions well under the 40-line/CC-10 bars, behavior-named tests; CI `cli-lint` green (local lint blocked by endpoint-security doctrine) |
| Handoff-readiness  | B | Proposal/tasks/verification complete, but promotion candidates unanswered and no lesson captured — the repo's own checklist is still open |

Aggregation: all B or above → PASS; no severity escalation applies.

### Verdict
PASS

### Recommended next steps

Routed by set — the contract set (`proposal.md`, `tasks.md`, `features.json`)
is **closed** by this verdict; none of the steps below edit it.

1. **verification.md** (free to edit): answer the three `Promotion candidates`
   lines (`yes: <path>` with the file, or `no: <reason>`) — the archive's third
   pre-flight refuses until then. Disposition the four findings above there, per
   the Definition of Done (applied / ticketed / declined with reason).
2. **Process**: commit `review.md` + `review-request.json`, then run
   `dotf spec archive CLI-132-root-commit-review` in this PR; that is what turns
   the red `spec-gate`/`gate` checks green (finding 2).
3. **Follow-up ticket (bitácora)** for finding 1: give the ordinary-scope prompt
   the same "judge what the spec names first" guidance the initial-scope prompt
   already carries, when `base..HEAD` includes merged-in work; its covering test
   (`TestReviewPromptKeepsTheTripleDotScopeForAnOrdinaryBase`) pins the current
   bytes and must move with it.
4. Optional: a test for finding 3 (`--initial` into a second/fallback slot).

**Verdict: PASS (adversarial).** `dotf spec archive` is **advisable** once
step 1 and step 2 are done; nothing in this verdict blocks it.
