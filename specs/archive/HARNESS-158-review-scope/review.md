---
spec: "HARNESS-158-review-scope"
verdict: "PASS"
reviewed_sha: "150a02b03bed19cc0d425b20a0fae6babf68fbff"
reviewer: "nan/qwen3.8-flash"
date: "2026-11-09"
---

## Adversarial review

**Scope**: HARNESS-158-review-scope / branch commit `150a02b0` (1 of 1 commits in range; base `8b2d0902` resolved by launcher)
**Sources**: `specs/HARNESS-158-review-scope/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`; diff `git diff 8b2d0902...HEAD`; live runs of build, vet, full `go test ./...`, `golangci-lint` (incl. `GOOS=windows` go vet), `dotf spec digest`, and the branch binary's `spec review --dry-run`.

### Spec and task alignment

- Contract digests in `review-request.json` match `dotf spec digest HARNESS-158-review-scope` exactly (features.json, proposal.md, tasks.md; spec.md empty) — this review is for the current contract.
- All five acceptance criteria map to named tests present in the diff, and all five test groups were re-run fresh in this session and PASS (not skip), including `TestOwnCommitsRecoversHarness142` against the real pinned history `adf49c93..c4d0e6c8` → exactly `(#3178) (#3195) (#3196) (#3205) (#3213)` of 14 — AC5 holds verbatim.
- AC4 verified live, not only by unit test: a binary built from this branch prints `Scope: git diff 8b2d0902...HEAD (1 of 1 commits are the spec's own)` on `--dry-run`, and the generated prompt carries the own-commit list section naming `150a02b03bed` with the rule text "(they touch the spec folder under any of its names, or reference #1727)" — byte-for-byte what this reviewer's own launcher produced. An `--initial` scope prints no count and no list (tests + code path: `reviewRange` returns before `ownCommitsOf` when `initial`).
- `tasks.md`: Implementation all `[x]`, each backed by a diff-visible test; Closing correctly still `[ ]` (PR not merged yet) — ticked boxes carry evidence, no orphan `[x]` found.
- Out-of-scope claims hold: `ReviewRequest` struct unchanged (`review-request.json` carries no own-commit field); `--initial` empty-tree scope untouched; `review: waived` not declared, no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags anywhere in the spec folder (grep returned nothing).
- Mutation probes (each reverted; working tree confirmed clean afterwards):
  - removing the digit-boundary `([^0-9]|$)` from the issue grep → `TestOwnCommitsHoldsOnlyTheSlicesOfTheSpec` FAILS (the `#17270` decoy leaks in): AC2 is genuinely pinned;
  - disabling the origin-equality guard in `specIssueNumber` → `TestSpecIssueNumberFallsBackToTheFolderAndSaysWhy` FAILS (cross-repo issue leaks in): the origin gate is genuinely pinned;
  - removing `--no-merges` from the range-count query → RangeSize assertion FAILS (7→8): merge exclusion from the count is genuinely pinned;
  - a raw-git probe independently confirmed the regex semantics: `#1727` mid-body at a line end matches (newline is `[^0-9]`, and `$` under REG_NEWLINE or end-of-message all behave), `#17270` alone never matches — the boundary rule is robust to message position.
- Full suite green, run fresh: `go build ./...`, `go vet ./...`, `go test ./... -count=1` every package ok, `golangci-lint run ./...` → 0 issues, `GOOS=windows go vet ./...` clean, `gofmt -l` clean on every touched file. The verification.md claims check out against re-execution, not reading.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | `OwnCommits` / `review_scope.go` | `--full-history` is load-bearing for no test and its stated rationale is inaccurate. Removing the flag keeps the whole `TestOwnCommits*` suite green (mutation probe). Raw-git probes: a side-branch commit that touches the folder and survives the `--no-ff` merge is listed by DEFAULT simplification too (code comment says `--full-history` is needed for exactly this case — it is not); the flag only changes results when the merge DISCARDS the side change (`-s ours`, overridden content), and no fixture covers either the flag's necessity or the discarding case. The verification.md AC1 line "a side branch merged `--no-ff`" proves only the exclusion side (side work untouched the folder; the merge itself stays out). Behaviour with the flag is the documented rule (touch the folder = own) and harmless advice, so severity stays Minor. | mutation probe: suite ok without the flag; scratch-repo git probes (default vs `--full-history` under `--no-ff` and `-s ours`) | UNTESTED — no named test depends on the flag; add a side-branch-slice case (folder file committed on a branch merged `--no-ff`) and an `-s ours` discarded-slice case to `review_scope_test.go` | tests + comment |
| Minor | THEORETICAL | `specIssueNumber` in `cli/internal/cmd/spec.go` | A hand-written frontmatter `issue: "#0"` parses cleanly (`ResolveIssueLink` returns Number 0, no error), so `specIssueNumber` returns `(0, "")`: the issue rule is silently disabled and — unlike every other fallback — `OwnFolderOnly` stays empty, so the launch line prints the count with no "folder rule only: …" reason, weakening the stated design promise that a disabled issue rule is never invisible. Reachable only via a malformed proposal (`spec init` writes a fully-qualified ref and the work gate requires an open issue); no legitimate in-repo spec hits it. | code read of `issuelink.go` `parseOneRef` (no `>0` check) + `specIssueNumber` returning `(ref.Number, "")`; grep shows no positive-number guard | UNTESTED — a table row `{"#0", ...}` in `TestSpecIssueNumberFallsBackToTheFolderAndSaysWhy` would pin it | code + tests |
| Question | — | prompt wording | `N of M`: M counts NON-merge commits, while the diff range also carries whatever merges brought in. The prompt mitigates this explicitly ("the range stays the whole change", first-parent hint), and the proposal's format string says "commits", so this is consistent — flagged only so the human is aware the count is deliberately not the `git log base..HEAD` total. | `OwnCommitSet.RangeSize` doc, fixture assertion RangeSize=7 with the merge excluded | `TestOwnCommitsHoldsOnlyTheSlicesOfTheSpec` | — (no change asked) |

No Blocker or Major found. The three genuinely risky properties — decoy-number exclusion, cross-repo issue exclusion, and refuse-on-git-failure (an empty list must not read as "no commits of its own") — are each pinned by a named test AND broken-and-observed-red by a mutation probe this session.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All five ACs verified by fresh test runs plus live `--dry-run` on a branch-built binary; negative paths (git failure, cross-repo issue, decoy, no-issue) covered and mutation-pinned; no behavioral defect observed. |
| Verification | A | Every verification.md claim reproduced verbatim in this session, including the f5 `--- PASS`-grepping command and the AC5 real-history result; smoke-test claim re-observed as "1 of 1" post-squash. |
| Scope | A | Diff matches the proposal exactly; the `review_request.go` refactor is the single-trace seam the proposal requires; no contract or gate field changed; nothing unrelated in the one-commit range. |
| Reliability | B | All failure paths refuse or fall back with a stated reason, except the degenerate `#0` case (Minor above), which disables the issue rule without its promised disclosure. |
| Maintainability | B | Functions short and CC well under 10, explicit seams; one comment states a git-history-simplification rationale that probes show is inaccurate, behind a flag no test needs. |
| Handoff-readiness | B | Spec triad complete and current; promotion-candidate questions and Closing boxes still for the implementer's closing pass — normal at pre-merge review. |

### Verdict

PASS

Two Minor findings (one REAL, one THEORETICAL) are tracked above; neither gates the archive per the severity×reality rule. This review's own scope is a working demonstration of the change: the launcher's prompt to this reviewer carried the derived commit list exactly as specified.

### Recommended next steps

These are dispositions for the implementer to record in `verification.md` (applied, ticketed, or declined with a reason) — none touches `proposal.md`, `tasks.md` or `features.json`, and none is required before archive:

- **F1 (tests + comment)** — either add a `review_scope_test.go` fixture whose side-branch commit touches the spec folder (covering both the preserved `--no-ff` case and, if the discarded-slice inclusion is wanted, an `-s ours` case), or drop `--full-history` and its inaccurate comment; current tests pass either way, so say which behaviour the repo wants and pin it.
- **F2 (code + tests)** — in `specIssueNumber`, return a reason (e.g. "proposal.md declares a non-positive issue number") when `ref.Number <= 0`, so the folder-only fallback keeps its disclosure promise; one table row in `TestSpecIssueNumberFallsBackToTheFolderAndSaysWhy` pins it.
- Carry into the spec's Closing tasks: tick the closing checklist only with the command outputs (this session's fresh runs all passed: `go test ./...` ok, lint 0 issues, `GOOS=windows` vet clean).
- Not verified by this round (UNVERIFIED, environment not reachable): the archive gate's own refusal logic (lives outside this repo; per run constraints). What this round could check — digest freshness, reviewer spelling, `reviewed_sha`, frontmatter shape — is consistent.
