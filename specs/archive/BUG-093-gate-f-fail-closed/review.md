---
spec: "BUG-093-gate-f-fail-closed"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "5a68096fb19d9d6fb64b2abca033d8f37ce579ab"
reviewer: "nan/qwen3.8-flash"
date: "2026-09-05"
---
## Adversarial review

**Scope**: BUG-093-gate-f-fail-closed (round 5)
**Sources**: `specs/BUG-093-gate-f-fail-closed/{proposal,tasks,verification,features}.md`,
`git diff 5a81e8a118d129adfc47d29915accf60dbabb28e...HEAD` (the launcher's `base_sha`), the built
`dotf` binary at HEAD, and 12 mutations run in this session (8 from `mutate.sh`, 4 new).

The range also carries three unrelated merged specs (HARNESS-112 review-base/#1535, gate
allow-reason/#1534, persona work/#1527/#1528/#1524). Their suites pass at HEAD; they are not
attributed to this spec, and this review's verdict is about BUG-093's acceptance criteria.

### What was re-executed (not read about)

| Check | Result |
|---|---|
| `go build ./...`, `go test ./...`, `go test -race` (worktree/spec/harness/cmd), `-count=3` on worktree | all rc=0, no flake |
| `GOOS=windows go vet ./...`, `GOOS=darwin go vet ./internal/worktree/` | rc=0 |
| `golangci-lint run ./...` (2.12.2, the `versions.conf` pin) | `0 issues` |
| `features.json` f1/f2/f3/f5/f7 | PASS, PASS, clean, PASS, PASS; counts **4 / 4 / 10** as claimed |
| `mutate.sh` (f8) | exit 0 — **8** CAUGHT, 1 declared survivor, 0 anchor-miss |
| e2e: `dotf worktree sweep` against a real git repo, a REAPABLE worktree, and a live process inside it | refused while occupied, reaped after the occupant exited, reach line printed on every scan — AC5/AC5b/AC6 hold end-to-end, not only via seams |

Round 3's and round 4's Majors are all genuinely applied: the re-check ordering mutation
(`round 4: Gate f moves back ahead of the git shell-outs`) is CAUGHT, `TestVanishedProcessCountsAsOutsideNotUnreadable`
catches the vanished→unreadable flip I re-ran independently, and both `processDiscoverySupported`
directions are pinned.

### Spec and task alignment
- **AC1-AC5b**: met, and the met-ness is falsifiable — every claim I re-mutated either failed the
  suite or is declared.
- **AC6**: behaviourally correct, contractually under-pinned (finding 3).
- **AC7**: every local gate is green at HEAD; the *Windows* half is asserted by a procedure that no
  longer reproduces (finding 2), and CI cannot be read from this session (finding 7).
- **AC8**: holds for the round-3 list, but the harness under-declares a second surviving gap
  (finding 4) and f8's quoted numbers are stale (finding 5).
- `tasks.md` boxes match the diff; no `[x]` without a corresponding artefact.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` in any contract file (only inside the review
  transcripts, which `isReviewOutput` deliberately skips).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | REAL | deletion safety (pre-existing, declared out of scope) | Gate f still deletes a worktree with a live occupant when that occupant reaches it through a bind mount in another mount namespace. | **Reproduced end-to-end at HEAD, not only at the link-comparison level**: `docker run -d -u 1000:1000 -v <repo-wt-m1>:/wt-mount-target -w /wt-mount-target busybox sleep 120`, then the built binary: `dotf worktree sweep` → `Deleting branch feat/m1 … reaped 1 worktree(s)` while the process was inside. | UNTESTED (no CI-runnable fixture; #1523 AC1) | spec + #1523 (code). **Disposition: does not block this archive** — it is not introduced or worsened by this diff, it is declared in *Out of scope*, and it is tracked by an OPEN ticket whose ACs cover it. It **does** block any future claim that Gate f protects containerised occupants; `proposal.md` *What* still says Gate f answers true "on every path where it cannot actually observe processes", and this path observes a process and misreads it. |
| Minor | REAL | verification integrity | The `verification.md` Windows proxy ("Flipping the build tags runs that implementation here") does not reproduce as documented: only the two implementation files are flipped, so the still-`//go:build linux` `sweep_proc_linux_test.go` is selected and the package fails to compile (`undefined: isNumericPID`, `cwdOutside`, `inspectProcessCwd`). The claimed `rc=0` is only reachable with a four-file flip that also retags `sweep_proc_linux_test.go` → `proxy_excluded` and `sweep_proc_other_test.go` → `linux`. | Reproduced both ways in this session: 2-file flip → `[build failed]`; 4-file flip → `ok … 0.950s`. So the *conclusion* (the Windows leg would be green) survived; the *command* is wrong. | UNTESTED — and AC3's own mutation (`sweep_proc_other.go` → `GateFReading{}`) SURVIVES every Linux-side check, so the only guard is `TestUnsupportedPlatformAnswersTrueForEveryPath` on the Windows leg | spec (`verification.md`), plus commit the flip as a script so the claim is re-runnable |
| Minor | REAL | test traceability | Two clauses of AC6 are unfalsifiable by anything in the Go layer: the note's *ordering* ("before the counts") and the `report.UninspectableProcesses > 0` gate that keeps the reach line from describing a scan that never ran. | `run_mutation`-style edits reproduced: deleting the whole note block → `go test ./...` green; changing `} else if report.UninspectableProcesses > 0 {` to `} else {` → green. f5's greps assert string presence and are order-blind, so even reordering the note *after* the counts would pass them. | UNTESTED (`newWorktreeSweepCmd` has no test at all) | tests + spec. Round 3 declared "grep not behaviour" knowingly, but never declared that the order and the always-print case would pass; the honest fix is one command-level assertion or one order-aware grep. |
| Minor | REAL | harness completeness | The `/proc`-unreadable fail-closed branch is a surviving mutation that `mutate.sh` does not list. | Reproduced: `entries, err := os.ReadDir("/proc"); if err != nil { return GateFReading{} }` → whole `go test ./...` green. `verification.md` *Accepted gaps* names it, but the harness's declared-survivor section carries only the `filepath.Abs` branch, so its own output reads "declared survivors: 1" when there are two. | UNTESTED (needs the `/proc` read injected — declared out of scope) | tests/harness (add the mutation as `SURVIVES`) |
| Minor | REAL | evidence drift | f8's `evidence` and the mutation block report `7 CAUGHT / 1 not caught`; the harness at HEAD reports `caught: 8 / declared survivors: 1 / regressions: 0`. | Re-run in this session; the extra CAUGHT is round 4's ordering mutation, added after the quote was taken. | f8 itself | spec (`verification.md` / `features.json` evidence, by re-running f8 — not by hand-editing the number) |
| Minor | THEORETICAL | messaging | The Windows note says "so nothing can be reaped" while `sweep --dry-run` on the same platform still prints `[DRY-RUN] Found N reapable worktree(s)`; and *What* promises `SweepReport.ProcessDiscovery` "so a JSON consumer can tell the two apart", while `sweep` has no `--json` flag. | `dotf worktree sweep --dry-run --json` → `Error: unknown flag: --json`; the dry-run line is unconditional in `cli/internal/cmd/worktree.go`. | UNTESTED | spec/code (cosmetic; the fields are still on the struct for any future JSON surface) |
| Question | — | AC7 | `test (windows-latest)` green at the merge commit cannot be established from a Linux worktree: `go vet` compiles and never runs, and the documented proxy is broken (finding 2). My corrected four-file flip is the strongest local signal and it is green. | — | `TestUnsupportedPlatformAnswersTrueForEveryPath` | human: confirm the check on the merge commit before `dotf spec archive` |

### Mutations run this round beyond `mutate.sh`

| Mutation | Result |
|---|---|
| note block deleted from `newWorktreeSweepCmd` | SURVIVED |
| reach line always printed (`> 0` → unconditional) | SURVIVED |
| Linux `os.ReadDir("/proc")` error → `GateFReading{}` | SURVIVED |
| non-Linux answer → `GateFReading{}` | SURVIVED on Linux (caught only by the `!linux` test on the Windows leg) |
| `filepath.Abs` failure → fail open | SURVIVED — matches the declared survivor |
| vanished process → `cwdUnreadable` | **CAUGHT** |
| `gateF` drops the `StateReapable` guard | **CAUGHT** |
| `gateF` drops the `absWT == absCwd` self-check | SURVIVED (still covered by the real `/proc` scan; pre-existing line, not this diff's) |

Working tree verified clean afterwards apart from the launcher's own
`review-request.json`; every mutation was reverted and the package re-tested green.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1-AC5b hold under independent mutation and end-to-end probing; the one demonstrated wrong behaviour is pre-existing, declared out of scope, and tracked by OPEN #1523. |
| Verification       | B | Everything I could re-run reproduces (build/test/race/vet×2/lint/f1-f8/12 mutations), but one documented check is broken-as-written and two AC6 clauses are unfalsifiable. |
| Scope              | A | BUG-093's commits touch exactly what the proposal declares; the extra commits in the range belong to other merged specs. |
| Reliability        | B | Every failure path in the reaper refuses (dirty/merged/lease/Abs/EvalSymlinks/ReadDir all fail closed by measurement); the residual is the declared container gap. |
| Maintainability    | A | Functions well under 40 lines, no dead code after `isCandidateForReap`'s removal, comments state why rather than what. |
| Handoff-readiness  | A | Rounds 1-4 each dispositioned line by line, #1470/#1523/#1530 verified OPEN live, lessons 273/274 captured. |

### Verdict
**PASS WITH GAPS**

No Blocker and no *untracked* REAL Major is attributable to this change: the one reproduced
fail-open (mount namespaces) predates it, is named in the proposal's *Out of scope*, and is
carried by OPEN #1523 — so per the repo's track-or-fix doctrine it is tracked rather than
blocking, and per the skill's verdict rule only findings attributable to the diff can move it.
Mechanically: rubric has no C/D, so the severity path governs, and every open Major is
declared-and-tracked → PASS WITH GAPS.

**`dotf spec archive` is advisable** at this sha once (a) `test (windows-latest)` is confirmed green
on the merge commit and (b) f8 is re-run so its quoted numbers match the harness. Findings 2-5 are
evidence-hygiene fixes that should ride along or be filed; none of them weakens the shipped
behaviour.

### Recommended next steps (before archive)
1. Confirm the `test (windows-latest)` check is green at the merge commit (AC7's only unverifiable half).
2. Fix `verification.md`'s build-tag-flip to the four-file form (impl **and** both `*_proc_*_test.go`), or commit it as `flip-proxy.sh` and have f3 run it.
3. Re-run `bash specs/BUG-093-gate-f-fail-closed/mutate.sh` and refresh f8's evidence line (`caught: 8`, 1 declared survivor).
4. Add the `os.ReadDir("/proc")` fail-open to `mutate.sh` as a declared `SURVIVES`, so AC8's "declared" list is complete.
5. Make AC6's order and its `> 0` gate falsifiable — one order-aware assertion in f5 or a command-level test with a `Sweep` seam.
6. Amend `proposal.md` *What* so "every path where it cannot actually observe processes" carves out the namespace case, and do not describe Gate f as protecting containerised occupants until #1523 lands.
7. For #1523 (not this diff): the proposed `os.SameFile` mechanism is now partly validated from this session — from the host, `os.Stat("/proc/<pid>/cwd")` returned the *same* `dev/ino` as the worktree on a real filesystem (both `64512/23873208`), but through a docker bind mount of a **tmpfs** path the dev differed (`26/21432566` vs `50/1237841`) while the inode matched. A pure `SameFile` test would still miss the tmpfs shape, so #1523's AC1 fixture should cover both.
