---
spec: "HARNESS-187-lane-identity-git-dir"
verdict: "FAIL"
reviewed_sha: "594d1764c6fab7c0c9bfa883a2d5549ebc492f51"
reviewer: "nan/glm5.3-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: HARNESS-187-lane-identity-git-dir (#3261)
**Sources**: `specs/HARNESS-187-lane-identity-git-dir/{proposal,tasks,verification}.md`, `features.json`; `git diff 3ef5dfa6b7386ee2a39cb25e129caf915b3a119e...HEAD` (branch commits 98210cea→594d1764 plus `5b1df35f`, an envelope-hygiene deletion unrelated to the feature).

Contract digests in `review-request.json` match `dotf spec digest HARNESS-187-lane-identity-git-dir` exactly. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec folder.

### Spec and task alignment

- All five ACs map to named tests in `features.json`; AC1–AC3 and AC5 verified by running them fresh in this session (`go test ./internal/lane/ ./internal/hooks/ -count=1` ok; every named test PASS).
- AC5 verified beyond the two packages: full `go test ./... -count=1` on HEAD is green (all 56 packages).
- Guard test case 10 was rewritten exactly as the proposal's Risks section declares; no other guard case changed (`git diff` shows only case 10).
- The central design claim was independently reproduced on a scratch repo: a record written into a worktree's git admin dir survives `git clean -fdx`, `git gc` and `git worktree move`.
- Mutation red-proof: reverting rule 5b of `lane-guard.sh` to name-based matching (semantics, not syntax) turns `TestPreCommitLaneGuard` red, so the test actually pins the new contract — the old defect cannot come back silently.
- Tasks/implementation drift: `tasks.md` names `--absolute-git-dir` for the guard; the implementation uses the script's existing `case` absolutisation idiom instead. Documented with rationale in `verification.md` ("Decisions made during implementation"); behavior proven by tests. Tracked below as a Minor, no contract action.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | AC4 / lane.Identity | `Identity()` on a **main checkout** returns an **error** instead of `("", false, nil)` whenever `<common-git-dir>/dotf-lane` exists — which it does in every repository that has run the handoff prompt feature: `cli/internal/cmd/lane.go:256` sets `PromptDir: filepath.Join(common, "dotf-lane")`, and `lane.Handoff` writes prompts there (`handoff.go:350`). The operator's own checkout carries `.git/dotf-lane/` with prompt files dated Oct 15 (verified by `ls`). Reproduced in this session with a temporary probe test on a fixture repo with that directory: `Identity(repo)` → `err=... read <git>/dotf-lane: is a directory`. `os.ReadFile` on a directory returns EISDIR, which is not `os.ErrNotExist`, so `identity.go` returns the error. The AC4 test (`TestIdentity_ReadsTheRecordFromTheWorktreeGitDir`) passes only because its fixture never creates the prompt directory — the negative path is verified against a fixture shape that no real repository matches. Root cause: the identity record's filename collides with a pre-existing, unrelated directory name in the shared git-dir namespace. Blast radius today is zero (grep confirms no production caller of `lane.Identity`; `provisionLane` uses `writeIdentity`, which refuses main checkouts before reading), but the function exists for #3262 (doctor) and #3293 (worktree list/sweep), and errors on its documented "main checkout" input the moment those call it. | Reproduced probe test output (run and reverted this session); `ls .git/dotf-lane` on the real repo; `identity.go:25-38`; `lane.go:256` | UNTESTED — no named test covers a main checkout carrying the prompt dir | **code + tests**: in `identity.go`, answer `("", false, nil)` when the record path is a directory (or short-circuit `gitDir == commonDir` to `("", false, nil)` before reading), and extend `TestIdentity_ReadsTheRecordFromTheWorktreeGitDir` with a main checkout that carries `.git/dotf-lane/`. Fix is outside the contract set; no contract edit needed. |
| Minor | THEORETICAL | guard rule 5b | A record file that is empty or whitespace-only makes the guard print `lane '' at ...` and still block, while Go's `Identity` answers not-a-lane for the same content — the sh and Go halves disagree on what the record means. `writeIdentity` never produces this content; it requires manual corruption. Cosmetic (the commit is still blocked). | Code read of `lane-guard.sh:107-111` vs `identity.go:29-36`; no repro | UNTESTED | code (read with the same trim-and-empty rule, or accept the asymmetry with a comment) |
| Minor | SPECULATIVE | robustness | `samePath` falls back to `filepath.Clean` per-side when `EvalSymlinks` fails, so one side resolved and the other not could compare a physical against a logical path. At every current call site both paths exist (worktreeGitDirs succeeded first), so it cannot mis-answer today. | Code read of `named.go` `samePath`; no repro | UNTESTED | — (surface only; do not gate) |
| Minor | THEORETICAL | naming | Two unrelated artifacts share the name `dotf-lane` in the git-dir namespace: the handoff prompt **directory** in the common dir and the identity record **file** in per-worktree admin dirs. Finding 1 is the symptom; renaming one (e.g. prompt dir → `dotf-lane-prompts`) removes the trap for future readers and callers. | `lane.go:256` vs `identity.go:18` | UNTESTED | code (follow-up ticket; the prompt-dir rename touches the handoff path, out of this spec's scope) |
| Minor | THEORETICAL | spec drift | `tasks.md`'s AC2 green step says the guard resolves `git_dir` with `rev-parse --absolute-git-dir`; the implementation keeps the file's existing `case` absolutisation idiom. Documented in `verification.md` with a git-version rationale; tests prove the behavior. Contract set is closed for this review; recorded here so the deviation is visible, no action. | `tasks.md` AC2 green line vs `lane-guard.sh:34-37`; verification.md "Decisions" | `TestPreCommitLaneGuard` (behavior proven) | none (documentation already carries it) |
| Question | — | portability | Windows runtime remains UNVERIFIED (declared in proposal Risks: authoring box is macOS; guard test runs on Linux/macOS legs, Go tests compile for Windows via `GOOS=windows go vet`, clean). Accepted as a declared risk; CI holds the authoritative leg. | proposal.md Risks; verification.md | — | none (already declared) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | AC1/2/3/5 verified with negative paths and a mutation red-proof, but AC4's main-checkout contract is violated on production-shaped repositories (demonstrated, not argued). |
| Verification       | B | Reproducible commands, named tests, per-AC mapping, full suite green; the one miss is a fixture blind spot, not missing evidence discipline. |
| Scope              | B | Diff matches the proposal; the only extra commit (`5b1df35f`) deletes a run-envelope file, documented in its message. |
| Reliability        | B | Refusals happen before any write (marker never armed by a refused init), a failed record write self-heals on the next init, and a hand-deleted record is restored; idempotency tested. |
| Maintainability    | B | Small functions, one absolutisation idiom, sh/Go drift guarded by the shared `lane.LaneIdentityFile` constant in the test; the `dotf-lane` name collision is the smell. |
| Handoff-readiness  | A | ADR-061, lesson 579, runbook sections, features.json complete, follow-up #3318 filed, promotion lines all answered. |

### Verdict

FAIL — one **REAL Major** (AC4 violated by `Identity()` on a main checkout carrying the handoff prompt dir, reproduced in this session and UNTESTED). Per the reality rule a REAL Major cannot stand under a passing verdict; the rubric aggregation (C, no D) would alone cap this at PASS WITH GAPS, but the severity axis escalates.

### Recommended next steps

To flip to PASS (fix is in code + tests, outside the contract set, so this review's digests stay valid and no re-scaffold is needed):

- Fix `cli/internal/lane/identity.go` so a main checkout (or a directory at the record path) answers `("", false, nil)` with no error.
- Add the named regression test: main checkout with `.git/dotf-lane/` present (the prompt-dir shape), asserting `("", false, nil)`.
- Re-run `go test ./internal/lane/ -count=1` and obtain a fresh review for the new head.

For the implementer to disposition in `verification.md` (applied / ticketed / declined), not contract edits:

- Ticket the `dotf-lane` namespace collision (finding 4) so the prompt-dir rename is decided deliberately.
- Findings 2 and 3 can stand as tracked minors; no gate.

### Completion notes (Definition of Done, reviewer side)

- Debt: findings above are the debt; each is classified with a fix location. Findings 2/3/4 belong in a ticket or a decline; finding 1 blocks archive until fixed.
- Knowledge: none new for the repo beyond what ADR-061/lesson 579 already record; the review itself is the record of the collision.
- Board: #3261 status untouched by this review (reviewer writes only `review.md`).
- Review: this is the review; no PR exists yet (`tasks.md` closing item honestly unticked).
- Evidence: all claims above carry fresh command output from this session (test runs, `dotf spec digest`, `ls`, mutation round-trip, scratch-repo lifecycle probe). Working tree left clean apart from `review.md` (and the launcher's `review-request.json`).
