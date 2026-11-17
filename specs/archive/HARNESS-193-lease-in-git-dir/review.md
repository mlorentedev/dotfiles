---
spec: "HARNESS-193-lease-in-git-dir"
verdict: "PASS"
reviewed_sha: "05a35fcde496379c58ffc4e871de941d7317dd6d"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: HARNESS-193-lease-in-git-dir (issue mlorentedev/dotfiles#3307, PR #3421)
**Sources**: `specs/HARNESS-193-lease-in-git-dir/{proposal,tasks,verification,features}.md`,
diff `git diff 42ba1f13926dfa49145bed9252d20851661f1cf6...HEAD` (43 files, +1271/−100; the
range holds exactly the 12 spec commits — no merges, no foreign work), PR #3421 checks.

### Spec and task alignment

- Contract set verified: `dotf spec digest HARNESS-193-lease-in-git-dir` reproduces
  `review-request.json`'s `contract_digests` byte for byte (features.json 46561fa…, proposal.md
  01ac6e86…, tasks.md 244a5268…). The contract files were not edited during this review.
- All 6 acceptance criteria map to named tests, all re-run fresh and green at HEAD:
  - AC1 → `TestActiveLaneSurvivesGitClean` (`cli/internal/lane/clean_survival_test.go`) — runs the
    actual `git clean -fdx` end to end and asserts status, list and a commit through the installed
    guard; it also asserts the mirror is *gone* after the clean, so it cannot silently stop
    exercising the defect.
  - AC2 → `TestAdminDir`, `TestLeaseLivesInTheWorktreeGitDir` (`worktree/lease_location_test.go`).
  - AC3 → `TestLaneInitMigratesInTreeLeases` (`lane/lanes_test.go`), including the second run
    `changed=0` with the mirror present and removed.
  - AC4 → `TestPreCommitLaneGuard` (`hooks/lane_guard_test.go`): admin-only PARKED blocks, admin-only
    ACTIVE passes rule 6 detached test, ACTIVE admin outranks a stale PARKED in-tree copy, and the
    pre-existing rule-5 case (linked worktree, in-tree-only PARKED) pins the fallback. Every other
    case in the file passes unchanged.
  - AC5 → `TestIsWorktreeParkedReadsTheAdminDirLease` (`mem/session_end_lease_test.go`): root,
    subdirectory, precedence, and legacy fallback.
  - AC6 → `TestListNamesALegacyLocationLease` (`worktree/lease_location_test.go`); the four-package
    suites pass (`f6` command, verbatim, exit 0).
- All six `features.json` verification commands run verbatim at HEAD: exit 0 (states remain
  `pending` — see finding Q1).
- Task-box claims reproduced: `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` → 0;
  `golangci-lint run ./...` (pinned 2.12.2 per `versions.conf`, fresh cache) → "0 issues" on darwin
  and `GOOS=windows`; `sh -n` and `shellcheck -s sh git-hooks/lib/lane-guard.sh` clean;
  `go test ./... -count=1` → all packages ok, exit 0.
- Test-retention bar: zero test functions deleted, zero assertions removed across the diff (the 33
  removed test-file lines are fixture setup only). Sampled rewrites *strengthen*: the "no lease"
  audit case now removes both copies and gains a "mirror gone ≠ missing lease" case; the two
  "lease cannot be written" fixtures now break the admin-dir copy (the copy every reader prefers).
- Docs promised exist and match: ADR-062 (answers Lesson 272 by name), lesson-595 (594 taken by
  #3442, documented in tasks), runbook updated, follow-up #3443 (HARNESS-194) open, #3327 open.
  No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags anywhere in the spec folder.
- Mutation probes (each run, then reverted; tree clean at end, `git status` shows only the review
  files):
  - **A** — neutered the guard's `$git_dir` admin-dir read → `TestActiveLaneSurvivesGitClean` and
    `TestPreCommitLaneGuard` both FAIL; the AC1 failure reproduces the original #3307 symptom
    verbatim ("missing its lease metadata"), confirming the test pins the defect.
  - **B** — swapped `LoadMetadata` precedence (in-tree first) →
    `TestLeaseLivesInTheWorktreeGitDir` FAIL ("want the admin-dir lease").
  - **C** — made `migrateLegacyLease` a no-op → `TestLaneInitMigratesInTreeLeases` FAIL
    ("want 1", admin file missing).
  - **D** — removed the `list` legacy note → `TestListNamesALegacyLocationLease` FAIL.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | test-traceability | AC2 names three Save refusals (main checkout, plain directory, submodule); `SaveMetadata` refusal is exercised end-to-end only for a plain directory — main checkout and submodule are asserted at `AdminDir` level only | `lease_location_test.go`: Save-refusal block covers `other := t.TempDir()`; `TestAdminDir` covers the other two refusals with path-naming errors; `Save`'s only gate is its first statement (`AdminDir`), so the behavior itself holds by construction and by code read | `TestAdminDir` (AdminDir level); Save-level main/submodule: UNTESTED | tests |
| Minor | THEORETICAL | maintainability / SSOT | The lease filename now lives in four places — `worktree.LeaseFileName`, `worktree.MetadataFileName`, string literals in `mem/session_end.go`, and literals in `git-hooks/lib/lane-guard.sh` — with no consistency guard. A future rename would silently strand session-end and the guard on the old name, re-creating the two-location drift this spec removes. The proposal's justification for the duplication is real: `worktree/done.go` imports `mem`, so `mem` cannot import `worktree` (cycle confirmed by grep) | grep of `dotf-worktree.json` literals across `cli/` and `git-hooks/`; no test pins them together | UNTESTED | tests (source-inspection pin: literals in `mem/session_end.go` and `lane-guard.sh` equal `worktree.LeaseFileName`/`MetadataFileName` — cheapest independent guard per the retention bar) |
| Minor | THEORETICAL | spec risk enumeration | The "old binary writes while a new one reads" analysis in `proposal.md` enumerates guard/status/sweep consequences of a stale admin copy but not `mem session-end`: after an old binary's `lane park` (mirror-only write), the new `isWorktreeParked` prefers a stale ACTIVE admin copy and writes the detached `<lane>@<host>` journal for a parked lane | code read of `session_end.go` precedence + the proposal's own stale-copy tracing; same window, noise-class consequence only | `TestIsWorktreeParkedReadsTheAdminDirLease` covers the precedence, not this cross-version state | verification.md disposition line (contract set is closed under this PASS — do not edit `proposal.md`) |
| Minor | SPECULATIVE | quality | Duplicate `// 6c.` case label: the new admin-dir block and the pre-existing main-checkout-detached block in `lane_guard_test.go` are both numbered 6c | comment text in `lane_guard_test.go` | n/a (comment) | tests (comment) |
| Question | — | process | All six `features.json` entries remain `"state": "pending"` with non-empty evidence; pass-state gating says only the harness may write `passing`. Confirm the harness run that flips terminal state is expected to happen around archive, or whether these are meant to be `passing` before this review's archive | `features.json` states + tasks.md gating rule | f1–f6 commands (verbatim) exit 0 in this session | spec-adjacent (harness), confirm only — not blocking |

No Blocker. No REAL Major. The two risk classes the proposal itself raises — the one-release mirror
overlap and the stale-admin window under an old binary — are owner-decided, mechanism-traced in the
spec, and each outcome is a refusal or a conservative block, never a lease loss; I re-derived the
guard/Go precedence consistency (both prefer the admin dir, pinned in both directions by tests) and
did not find a divergence outside that accepted window.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 6 ACs verified by named tests re-run green at HEAD; four mutation probes go red on their property; no observed defects in code read of every production reader/writer |
| Verification       | A | Every `verification.md` claim and every `features.json` command reproduced fresh here (build, vet ×2, lint ×2 pinned, shellcheck, full suite); probes prove tests bite, not just pass |
| Scope              | A | Diff is exactly the proposal's 43 files; range contains only the 12 spec commits; zero test functions/assertions deleted; no side work |
| Reliability        | A | Fail-closed on corrupt leases (list→nil, audit→finding, Save refuses before any write), admin-last write ordering keeps the preferred copy truthful on partial failure, migration idempotent (`changed=0` pinned); Windows legs green in CI (`test-windows`, `windows-latest` checks pass on PR #3421) |
| Maintainability    | A | `golangci-lint` 0 issues both platforms; lease helpers small and commented with the why; only smell is the documented literal duplication (finding 2) |
| Handoff-readiness  | A | ADR-062 (answers Lesson 272 by name), lesson-595, runbook, follow-up #3443 filed and open; tasks/verification filled with reproducible evidence; PR `knowledge-gate` now pass (an earlier failure on a stale lesson-594 line was fixed by a PR-body edit) |

### Verdict

**PASS**

Severity × reality: all findings are Minor and THEORETICAL/SPECULATIVE; rubric is all A. No
Blocker, no REAL Major. The contract set is closed by this verdict — the recommendations below are
for disposition in `verification.md` or a follow-up ticket, not for contract edits.

### Recommended next steps

- Disposition finding 3 (session-end stale-copy note) in `verification.md`, or fold it into #3443's
  preconditions — `verification.md` only, never `proposal.md`.
- Land the literal-consistency source-inspection test (finding 2) and the Save-level main/submodule
  refusal cases (finding 1) as ordinary test commits; neither blocks archive.
- Confirm the `features.json` state flip with the harness before running `dotf spec archive`
  (question Q1).
- PR #3421: CI is green except one pending job (`from-zero (macos-latest)`) at time of writing; no
  human review yet (`reviewDecision` empty, CodeRabbit skipped) — merge remains a supervised action.
- **`dotf spec archive` is advisable once the PR merges, the board ticket closes, and the pending
  macOS job and any late CI output are dispositioned.** The archive gate's own implementation is
  outside this repo and was not exercised (UNVERIFIED beyond `dotf spec digest`, which parses and
  matches); this review's frontmatter follows the required format and `reviewed_sha` is the commit
  examined.

### Definition of Done (this review)

1. **Debt** — five findings recorded above with fix locations; none left as chat-only mentions.
2. **Knowledge** — this `review.md` is the artifact for this round; no cross-project insight
   surfaced that outranks it (reviewer session, no code owned).
3. **Board** — issue #3307 verified OPEN with its real title, matching `status: implementing`;
   #3443 and #3327 verified OPEN (live queries, this session).
4. **Review** — PR #3421 checks read and dispositioned in Spec and task alignment (green; one
   pending job; knowledge-gate history noted); no reviewer comments to triage (`reviewDecision`
   empty at time of writing).
5. **Evidence** — every pass claim above was produced by a command run in this session at
   `05a35fcde496379c58ffc4e871de941d7317dd6d`; probes were reverted and `git status` shows only
   the two spec-folder review files.
