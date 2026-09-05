---
spec: "BUG-093-gate-f-fail-closed"
verdict: "FAIL"
reviewed_sha: "dab7b6e0576cdedef251be383e78c56a0eb78890"
reviewer: "nan/qwen3.8-flash"
date: "2026-09-05"
---
## Adversarial review

**Scope**: BUG-093-gate-f-fail-closed (round 3)
**Sources**: `specs/BUG-093-gate-f-fail-closed/{proposal,tasks,verification,review}.md` +
`features.json`; `git log 9e6ee7d^..HEAD`; the working tree at `1de7910` and `dab7b6e`.

**The sha I signed, and why the gate will refuse it.** The review was launched against
`1de7910` (`review-request.json`). While it ran, the implementing session committed
`dab7b6e` — *test-only* (`cli/internal/worktree/sweep_test.go`), contract files unchanged.
`reviewed_sha` records the tip I finished on, per the launcher's instruction. Consequence,
verified by running `checkReviewProvenance` against both shas: it reports
`review.md claims reviewed_sha dab7b6e0576c but the review was launched against 1de79102790c`.
That is the guard doing its job, not a malformed artifact, and I did not sign the older sha to
sail past it — recording the older sha would have made the gate blind to the commit that moved.
Remedy is in the next-steps list; note the verdict is FAIL, so nothing is unblocked either way.

### Spec and task alignment

- AC1, AC3, AC4, AC5, AC6, AC7, AC8 are met as written. AC2 is met for the three failure classes
  it now enumerates (Abs, EvalSymlinks, ReadDir) and the three-valued per-process read is
  genuinely implemented. I re-ran all seven `features.json` verifications at the tip: **all rc=0**.
- `go build ./...`, `go test ./...`, `go test -race ./internal/worktree/`, `golangci-lint run ./...`
  (2.12.2, the `versions.conf` pin) → `0 issues`; `GOOS=windows go vet ./...` and
  `GOOS=darwin go vet ./internal/worktree/` → clean. Reproduced independently, not trusted.
- Claimed mutations re-run and confirmed: caller ignoring Gate f **CAUGHT**; `EvalSymlinks`
  resolution removed **CAUGHT** (`TestGateFSeesThroughASymlinkedWorktreePath`); vanished-PID
  counted as unreadable **CAUGHT**; `SweepWithRunner` ignoring the candidate **CAUGHT**. Round 1's
  fixes are real, and the AC2 amendment was *declared* rather than quietly applied — the strongest
  part of this artifact set.
- What the round-2 work did **not** do is close the fail-open class it was inside. Four of the
  findings below are paths where Gate f still answers "nobody is inside" without having earned it,
  or where the test suite cannot tell whether it did.
- Out of scope, and I agreed after trying: a missing/prunable worktree directory cannot be
  stranded by the new `EvalSymlinks`-answers-`true` branch, because `Classify` requires on-disk
  metadata with `reap_ok` (`list.go:224`, `worktree.go:162`) and never reaches `gateF` for it. The
  kernel's `" (deleted)"` cwd suffix mostly self-cancels, but not entirely — I probed three shapes.
  Deleted worktree root: the target's own `EvalSymlinks` fails first and answers `true` (masked).
  Deleted *subdirectory*: still matches the separator-anchored prefix, `Inside=true` (correct).
  Deleted-**then-recreated** root: `readlink` yields `"…/repo-wt-feat (deleted)"`, neither the
  equality nor the prefix test matches, so the answer is `Inside=false` — a live process counted as
  outside. That one is not a data-loss path, because the process holds the *old* inode and the
  recreated directory is a different one; refusing there would protect nothing. It is the same
  string-comparison weakness as finding 1, and the same `dev`/`ino` fix retires it.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | path comparison | Gate f still fails open for a live process reached through a **mount namespace or bind mount** — the same door AC2c closed for symlinks. `/proc/<pid>/cwd` is resolved in the *process's* namespace, so a container whose `-v` mounts the worktree reports `/wt-mount-target`, not `/home/…/repo-wt-feat`; the strings differ, so the answer is `outside` and the caller deletes. | Reproduced with `docker run -u 1000:1000 -v /tmp/wtmount/repo-wt-feat:/wt-mount-target -w /wt-mount-target busybox sleep`: pid 2191811, host-side `readlink` → `/wt-mount-target`, `inspectProcessCwd(pid, "/tmp/wtmount/repo-wt-feat")` → `verdict=0` (`cwdOutside`), `isHostProcessInside(…).Inside=false`. Same-directory proof: `dev/ino` of `/proc/<pid>/cwd` and of the host path were **both 50/951431**, i.e. `os.SameFile` sees what the string comparison does not. Second variant, worse: with the container's default root-owned process the readlink is EACCES, so it is folded into `Uninspectable` — see the next row, where that count is meaningless. | UNTESTED (`TestGateFSeesThroughASymlinkedWorktreePath` covers symlinks only) | code (`os.SameFile` / dev+ino walk instead of string prefix) + tests + spec (proposal AC2c's rationale claims resolution removes the different-door fail-open; it removes one of them) |
| Major | REAL | observability | The `Uninspectable` count — the sole mitigation for the EACCES decision, and therefore the answer to round 1's Major — is **permanently maxed out**, so it reports nothing. It is not an anomaly signal; it is the normal state of every Linux machine. Because `gateF` only scans `StateReapable` worktrees, the note is silent on a machine with nothing to consider — it fires on exactly the runs where a human is about to trust a refusal-free answer. | Measured through the production path (`SweepWithRunner`, real gate, one genuinely reapable fixture worktree): `UninspectableProcesses=367`, `ProcessDiscovery=true` → the user reads `note: 367 process(es) could not be inspected, so this scan was partial.` Direct count: 363 of 600 PID dirs unreadable as uid 1000. The document's own argument against counting vanished PIDs is "the signal fire constantly and mean nothing" — 367 EACCES processes do exactly that, on every run, and the note is `else if`, so the constant note is also what a reader learns to skip. | `TestUninspectableProcessesDoNotBlockButAreCarried` asserts propagation of a fabricated 7; no test bounds or sanity-checks the produced number | code (bound it, e.g. count only processes that share neither mount- nor user-namespace traits we can rule out — or report "N processes belong to other users" as a standing caveat, not a warning) or spec (state that the note is expected on every run) |
| Major | REAL | tests | A test whose **name states the property it does not check**: `TestGateFMatchesASubdirectoryButNotASiblingWithASharedPrefix` never asserts the "but not a sibling" half. `otherRoot := filepath.Join(root, "repo-wt-feat")` is the *same path* as `target`, already asserted `Inside=true`, so the block is a tautology, and its failure branch calls `t.Skip` rather than reporting. | Mutation: `strings.HasPrefix(absDest, absTarget+sep)` → `strings.HasPrefix(absDest, absTarget)` → whole suite **rc=0, NOT CAUGHT**. `verification.md` claims the six real-`/proc` tests cover "a sibling sharing a name prefix"; they do not cover it in any direction. | UNTESTED (named test exists but is vacuous) | tests (spawn a child in the sibling only, assert `isHostProcessInside(target).Inside == false`; drop the `t.Skip`-as-assertion) + spec (`verification.md` claim) |
| Major | REAL | tests | The counting half of the AC2 amendment is unproduced-by-test: no test drives `reading.Uninspectable++` in `sweep_proc_linux.go`. | Mutation: replace `reading.Uninspectable++` with a no-op → whole suite **rc=0, NOT CAUGHT**. So the count can silently pin at 0 forever, which is precisely "a partial scan reporting nothing is indistinguishable from a complete one" — the failure the test comment itself names. | `TestUninspectableProcessesDoNotBlockButAreCarried` injects the value through the seam and cannot see the producer | tests (assert the real scan yields a non-zero count as a non-root uid, or inject the link read) |
| Major | REAL | tests | The gate check that actually guards the deletion is unprotected, and after `dab7b6e` **no test drives Gate f's refusal through `SweepWithRunner` at all**. `dab7b6e` stubs the seam to `false` in all five sweep integration tests ("these exercise sweep's own gates"), so the refusal→skip path now exists only in `sweep_proc_test.go` against `isCandidateForReap` — which has no production caller (`sweep.go:127`; `SweepWithRunner` calls `gateF` directly). | Mutations: drop the re-check in `reapSingleWorktree` (`absWT == absCwd \|\| false`) → **rc=0, NOT CAUGHT**, at both `1de7910` and the tip; `grep -c "withHostProcessInside(t, true)" sweep_test.go` → 0. AC5 as written pins a function production does not call. | UNTESTED for the delete path (`TestGateFRefusesWhenProcessDiscoveryCannotAnswer` pins the dead wrapper) | tests (drive `SweepWithRunner` with the seam answering `true`, assert `SkippedCount`/no `WorktreeRemove`) + code (delete `isCandidateForReap` or use it) |
| Major | REAL | spec-vs-code | `proposal.md` says the `processDiscoverySupported` drift risk is "**Pinned by a test**". On Linux it is pinned by a test that cannot fail: `TestProcessDiscoveryIsReportedForThisPlatform` skips when the constant is false, and its only assertion — `hostProcessInside == nil` — is unreachable, since the var is initialised to a function at package load. | Mutation: `const processDiscoverySupported = false` in `sweep_proc_linux.go` → **rc=0, NOT CAUGHT**. The real consequence is a lying message: Linux would print "no process-liveness check on this platform" next to a working one. The `!linux` direction *is* pinned — I verified `Inside:false` in `sweep_proc_other.go` fails `TestUnsupportedPlatformAnswersTrueForEveryPath`. | `TestUnsupportedPlatformDoesNotAdvertiseDiscovery` (non-Linux leg only); `TestProcessDiscoveryIsReportedForThisPlatform` vacuous | tests + spec (soften the sentence or pin both directions) |
| Minor | REAL | tests | AC2's fail-closed branches for `filepath.Abs` and `filepath.EvalSymlinks` errors are unmutatable-and-untested. `verification.md` declares one mutation gap (unreadable `/proc`) and I confirmed it honestly survives; it does not say the other two are also uncovered. Worse, the `EvalSymlinks` branch is *not* blocked on the stated reason ("reaching it needs … injecting the directory read") — it needs only a nonexistent path. | Mutations: answer `Inside:false` on the Abs error → **rc=0**; on the EvalSymlinks error → **rc=0**; unreadable `/proc` → **rc=0** (declared). Counter-evidence to the "cannot be tested" framing: `isHostProcessInside("/tmp/definitely-not-here-9f3a2b/nope")` → `Inside=true`, measured. | UNTESTED | tests (the EvalSymlinks case is one `if`); spec (state the Abs branch is unreachable-by-input, if that is the intent) |
| Minor | REAL | tests | AC6's ordering is asserted by grep, not behavior: `features.json` f5 checks two string literals exist in `worktree.go`. | Mutation: delete the entire note block from `newWorktreeSweepCmd` → **rc=0, NOT CAUGHT**; and move it *after* the counts → **rc=0**. `grep -rn "Sweep complete\|no process-liveness" internal/cmd/*_test.go` → no matches, i.e. no CLI-level test of sweep output exists. | UNTESTED | tests (a cmd-level output test) or accept the grep knowingly |
| Major | REAL | CI (introduced at `1de7910`, fixed at `dab7b6e`) | Round 2 shipped a **red required check on the platform this spec exists for**. With Gate f answering `true` unconditionally, `sweep` reaps nothing, so the pre-existing sweep tests that assert a reap failed on `test (windows-latest)` (`cli.yml:47` runs `go test ./...` on `[ubuntu-latest, windows-latest]`). | Proxy at `1de7910` (build tags flipped so `sweep_proc_other.go` is the implementation and the `//go:build linux` test file is excluded): `--- FAIL: TestSweepFailClosed`, `--- FAIL: TestSweepLogsSHA`, `--- FAIL: TestSweepPreservesGitignoredLocalFiles`, package rc=1. Same proxy at `dab7b6e`: `ok`. `GOOS=windows go vet ./...` — the evidence AC7 names — compiles and never runs, which is why it stayed green. | `TestSweepFailClosed` et al. (would have caught it on the Windows leg; they do not catch it locally, because only `go vet` runs cross-compiled here) | tests + spec (AC7 must name the Windows **test** leg, not `go vet`; `verification.md`'s "the pre-existing worktree tests are untouched and still pass" is false at the tip — `dab7b6e` touched them) |
| Minor | REAL | handoff accuracy | Two verification claims are stale at the tip: "Gate f subset … → 3 PASS" (measured **7** PASS for that exact command), and the promotion candidate "Not filed as a ticket yet" — **GUARD-010 #1470 is already OPEN** covering the `features.json` state vocabulary. Related: this spec's own `features.json` uses `state: "verified"`, one of the seven ungoverned spellings that issue exists to end. | `go test ./internal/worktree/ -run 'GateF\|ProcessDiscovery' -v` → 7; `gh issue view 1470` → OPEN, title names the same defect. | n/a | spec (verification.md), and #1470 already owns the vocabulary fix |
| Minor | SPECULATIVE | maintainability | Small stdlib/duplication smells in code this change moved: `isCandidateForReap` is now production-dead; `itoa` in `sweep_proc_linux_test.go` reinvents `strconv.Itoa`; `GateFReading` carries no zero value doc for `cwdVerdict`'s `iota`-zero = `cwdOutside`, so a future `switch` author must read the const block to learn that the zero value means "safe to delete". Also `SweepWithRunner` + `reapSingleWorktree` walk all of `/proc` twice per worktree (~600 readlinks × 2 × W here). | Code read; no failure demonstrated. | UNTESTED | code (optional) |
| Question | THEORETICAL | UX / advice | The Windows note tells the user to "remove one deliberately with `dotf worktree done`". I confirmed that works — `Done` (`done.go`) checks dirty + unpushed and never consults Gate f — so the advice is not a lie. But it also means the recommended escape hatch carries **zero** liveness protection on the platform where the note is shown, and `--force` bypasses even the dirty check. Is that the intended posture, or should the note mention `--dry-run` / `-v`? | `cli/internal/worktree/done.go` read; no Gate f call on any path. | n/a | spec (proposal Risks), or code (note wording) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All 8 ACs are met as literally written and all 7 verifications reproduce rc=0, but the fail-open class the spec is about survives through mount namespaces, and AC2c's stated rationale overclaims what `EvalSymlinks` buys. |
| Verification       | C | Commands are reproducible and round 1's fixes are mutation-proven, yet four properties named in `tasks.md`/`verification.md` (sibling prefix, uninspectable counting, the delete-path refusal, the AC4 pin) survive mutations, and two evidence lines are stale at the tip. |
| Scope              | B | Diff matches the proposal; `dab7b6e`'s test edits are in-family but undeclared in `tasks.md`. |
| Reliability        | C | Conservative refusals throughout and no crash path found (`-race` clean, idempotent under lock), but the check that guards the actual delete is unpinned and the partial-scan signal fires always, so it cannot warn. |
| Maintainability    | B | Functions short, `golangci-lint` 0 issues, platform boundary clean; docked for a dead production wrapper, a vacuous test with `t.Skip` in place of an assertion, and reinvented `Itoa`. |
| Handoff-readiness  | C | The AC2 amendment is declared in the open — exemplary — while `verification.md` carries a false no-regressions claim and a "not filed yet" note for an already-open ticket. |

### Verdict
FAIL

Mechanically: six REAL Majors (any Major ⇒ FAIL under this skill's aggregation), and rubric Cs in
Correctness/Verification/Reliability/Handoff. The platform fix — the thing BUG-093 was opened for —
is **correct and I could not break it**: non-Linux refuses, `dab7b6e` restores the Windows leg, and
round 1's three findings are genuinely applied rather than argued away. What keeps this from
PASS-WITH-GAPS is that the change's central claim is "the gate no longer guesses", and I reproduced
a case where it guesses wrong while a process is inside.

### Recommended next steps (before archive)

1. **Disposition the namespace fail-open.** Either close it — compare `os.Stat("/proc/<pid>/cwd")`
   against the target by `dev`+`ino` (`os.SameFile`), walking parents for the subdirectory case;
   I measured identical `50/951431` on both sides, so it detects what the string does not — or add
   it to `proposal.md` **Out of scope** with the reproduction, the way the Windows gap is declared.
   Silence is the only unacceptable option: AC2c's comment currently says resolution removes
   "the same fail-open … arriving through a different door", and one door is still open.
2. **Fix the vacuous test, don't delete the claim.** Give
   `TestGateFMatchesASubdirectoryButNotASiblingWithASharedPrefix` its negative assertion, and make
   the naive-prefix mutation fail. Then the sibling case in `verification.md` is true.
3. **Pin the three unpinned properties** with tests that fail when broken: the `Uninspectable++`
   producer, the `reapSingleWorktree` re-check driven through `SweepWithRunner` with the seam
   answering `true`, and the Linux side of `processDiscoverySupported`.
4. **Decide what `Uninspectable` is for.** ~363 unreadable processes per scan on an ordinary desktop
   is not an anomaly signal. Either report it as a standing caveat, bound it, or state in the
   proposal that the note is expected on every Linux run — and if the number stays, use it to
   detect the container case in finding 1 rather than merely counting it.
5. **Amend AC7 and `verification.md` honestly.** AC7 must name the Windows **test** leg, not
   `GOOS=windows go vet`, which cannot see this defect class and did not; `verification.md` should
   record that `1de7910` reddened that leg, `dab7b6e` fixed it, and the "untouched / 3 PASS" lines
   were wrong; `tasks.md` should carry the round-3 test change.
6. **Refresh the launch pin, then re-review.** `review-request.json` still says `1de7910`, so
   `dotf spec archive` will refuse this review on provenance before it ever reads the verdict.
   Re-run `dotf spec review BUG-093-gate-f-fail-closed` after steps 1–5 so the sidecar and the
   frontmatter name the same head; `git diff 1de7910 dab7b6e -- specs/` is empty, so nothing else
   in this review goes stale when you do.
7. Housekeeping, cheap and in-session: correct the "not filed as a ticket yet" line to point at
   #1470 (it is OPEN), drop `isCandidateForReap` or use it, and replace `itoa` with `strconv.Itoa`.

`dotf spec archive` is **not advisable**: the verdict blocks, and independently the provenance check
refuses because the recorded head and the launch pin differ. Minimum set to flip to PASS: findings
1 (closed or declared), 3, 4, 5 — plus steps 2 and 6, which are what make the remaining Majors
disappear rather than merely be argued down.
