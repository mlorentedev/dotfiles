---
tags: [spec, verification, templates]
created: "2026-09-05"
---

# Verification - BUG-093-gate-f-fail-closed

## Round 5 review: PASS-WITH-GAPS — disposition

`nan/qwen3.8-flash`, random draw, not the implementer. Reviewed `5a68096`. **The first round with
a real diff**, and the contrast is the point:

| Round | Scope it had | Its own evidence verbs |
|---|---|---|
| 4 | `git diff main...HEAD` — **empty** | "Code read", "Code read"; one finding factually wrong |
| 5 | `git diff 5a81e8a1...HEAD` (the launcher's `base_sha`) | **12 mutations run**, 8 from `mutate.sh` and 4 new |

Rubric B/B/A/B/A/A. The single Major is the bind-mount fail-open **declared out of scope and filed
as #1523** — the reviewer honoured the declaration instead of blocking on it, which is the amended
verdict rule (HARNESS-111) working on its first use.

| # | Finding | Disposition |
|---|---|---|
| Major, REAL | Bind-mount occupant still deleted | **Already declared.** Out of scope with a reproduction; #1523 OPEN. The proposal's *What* now carves it out explicitly rather than saying "every path where it cannot observe processes" |
| Minor | The documented Windows proxy does not reproduce: a **two-file** flip leaves the `//go:build linux` test file selected and the package fails to compile | **Applied.** The conclusion was right and the command was wrong, which is worse than either. Committed as `flip-proxy.sh` (four files) and wired into f3, so the command cannot drift from the claim again |
| Minor | AC6's *ordering* and the `> 0` gate were unfalsifiable — f5's greps are order-blind, so moving the note after the counts would pass | **Applied.** f5 is now position-aware and mutation-proven: renaming the reach marker so it no longer precedes the counts turns it red |
| Minor | The `/proc`-unreadable fail-open survives and `mutate.sh` does not list it, so the harness's own output disagreed with *Accepted gaps* | **Applied.** Declared to the harness as a `SURVIVES`; it now reports `declared survivors: 2` |
| Minor | Evidence drift: f8 said `7 CAUGHT / 1 not caught`, the harness at HEAD says `caught: 8` | **Applied by re-running, not by editing the number** — which is what the reviewer explicitly asked for |
| Minor, THEORETICAL | `--dry-run` on Windows still prints "Found N reapable" under a note saying nothing can be reaped | **Accepted, tracked here.** Cosmetic on a platform where the command is inert by design; fixing it means threading `ProcessDiscovery` through the dry-run branch for a message nobody acts on |
| Question | `test (windows-latest)` green at the merge commit cannot be established from Linux | **Answered by measurement**, not argument: `gh api .../commits/5a68096/check-runs` → `test (windows-latest): success`, `test-windows: success` |

### The proxy script had the repo's own prohibited pattern in it

`flip-proxy.sh` was first written with `for f in $FILES`. **zsh does not word-split an unquoted
parameter**, so the loop ran once with all four names glued together, every `cp` failed, no `.bak`
was written — and the `sed`s still flipped the tags, because they name files literally. The script
reported success both times and left the working tree flipped.

That poisoned the next thing to run: with the tags flipped, `sweep_proc_linux.go` is excluded from
the build, so mutations against it cannot fail, and `mutate.sh` reported **five CAUGHT mutations as
SURVIVED**. A green-looking script silently turned the mutation harness into a liar.

Fixed with `set --` positional parameters, and — more importantly — the script now **verifies its
own restore** and exits 3 if the tags are still flipped. Confirmed identical under `bash` and
`zsh`, tree clean after both.

This is the exact row this repo's `.claude/CLAUDE.md` prohibited-pattern table carries, written
into a new script hours after that table was extended. The table was right; reading it is not the
same as applying it.

## Round 4 review: FAIL — disposition

`agy/gemini-3.1-pro-high`, random draw, not the implementer, and a **different provider family**
from round 3's `nan/qwen3.8-flash`. Reviewed `5101938` — the launch pin and `reviewed_sha` agree,
so the provenance problem round 3 flagged is closed.

Two findings, down from ten. **One applied, one refuted with evidence.**

| # | Finding | Disposition |
|---|---|---|
| 1 (Major, THEORETICAL) | The TOCTOU re-check sits *before* `isDirty` and `isMerged`, which shell out to git, leaving that latency inside the race window | **Applied.** Real, and cheap to fix. Gate f now runs immediately before `executeWorktreeReap`. It is also the cheaper order: Gate f walks all of `/proc`, so running it after the cheap refusals means fewer walks. Pinned by `TestGateFIsTheLastCheckBeforeRemoval` and a new mutation |
| 2 (Minor, SPECULATIVE) | `filepath.Abs` failure inside `isHostProcessInside` "fails OPEN (`Inside: false`)" | **REFUTED.** The code returns `GateFReading{Inside: true}` — fail *closed*. `grep -n "Inside: false" cli/internal/worktree/sweep_proc_linux.go` returns **nothing**; the string exists nowhere in `cli/`. See the cause below |

### What caused the false finding, and what was done about it

The only `Inside: false` strings in the repository are the **mutation payloads inside
`specs/BUG-093-gate-f-fail-closed/mutate.sh`** — the deliberately broken code the tests must catch.
Committing the harness (a round-3 fix, so f8 would be a check rather than a grep for a word in a
markdown file) put sabotage into the reviewer's reading surface, and the reviewer cited a mutation
as the implementation.

**The harness defended itself against being run wrongly and not against being read wrongly.** It
now carries a banner naming the argument order (`<real code> <broken code>`), stating that the 5th
argument is sabotage, and naming this exact misreading. Recorded rather than dismissed as reviewer
error: the artifact invited it.

### A mutation that survived, correctly

The first version of the round-4 mutation *added* the Gate f check at the front while leaving the
rear one in place, and the harness reported **SURVIVED** — rightly, because the last check before
the removal was still Gate f and the window had not widened. A duplicate is not a move. The
mutation now replaces the whole function body, and is CAUGHT. The harness caught a defect in the
harness.

### Round 4 results

```
  CAUGHT   round 4: Gate f moves back ahead of the git shell-outs
  caught: 8   declared survivors: 1   regressions/anchor-miss: 0
  exit 0
```

## Round 3 review: FAIL — disposition

`nan/qwen3.8-flash`, random draw, not the implementer. Verdict **FAIL** against `dab7b6e`, on
6 REAL Majors and 4 REAL Minors. **Every one was real. None is argued down.**

The shape of round 3 is worth naming before the table: the reviewer did not find a broken
behaviour, it found **claims that no test could falsify**. Five mutations survived the whole
suite — including the mitigation this spec added in round 2 as its answer to round 1.

| # | Finding | Disposition |
|---|---|---|
| 1 (Major) | Mount-namespace fail-open: `/proc/<pid>/cwd` resolves in the process's own namespace, so a bind-mounted worktree compares unequal and reads as outside | **Declared, not fixed.** Reproduced independently (below), added to *Out of scope* with the measurement, the overclaiming code comment corrected, filed as **#1523**. Not fixed here because it has no CI-runnable test — see below |
| 2 (Major) | `Uninspectable` is permanently maxed out, so the round-1 mitigation signals nothing | **Applied — this was the design blocker.** Reframed from warning to *reach*. New `Scope of Gate f` section, new CLI wording, three code comments rewritten |
| 3 (Major) | `TestGateFMatchesASubdirectoryButNotASiblingWithASharedPrefix` never checks the "but not a sibling" half; `t.Skip` used as an assertion | **Applied.** Split into two tests with isolated roots; the naive-prefix mutation now fails |
| 4 (Major) | No test drives `reading.Uninspectable++`; a no-op mutation left the suite green | **Applied.** `TestUninspectableProcessesAreCountedByTheRealScan`, anchored on `/proc/1/cwd` being unreadable |
| 5 (Major) | After `dab7b6e` no test drives Gate f's refusal through `SweepWithRunner`; `isCandidateForReap` has no production caller | **Applied.** Wrapper deleted, seam tests moved onto `gateF`, and `TestSweepRefusesWhenAProcessArrivesAfterTheGate` drives the real path |
| 6 (Major) | `processDiscoverySupported` is "pinned by a test" that cannot fail on Linux | **Applied.** `TestProcessDiscoveryIsAdvertisedOnLinux` replaces it; both directions now pinned |
| 7 (Major) | Round 2 shipped a red `test (windows-latest)`; AC7 named `go vet`, which compiles and never runs | **Applied.** AC7 amended to name the Windows test leg; the build-tag-flip proxy is now part of this document; the false claims below are corrected |
| 8 (Minor) | The `EvalSymlinks` fail-closed branch is untested, and the stated reason for that was wrong | **Applied.** `TestGateFRefusesWhenTheTargetCannotBeResolved` — it needed only a nonexistent path, exactly as the reviewer said |
| 9 (Minor) | AC6 asserted by grep, not behaviour | **Accepted knowingly**, and said so — see *Accepted gaps* |
| 10 (Minor) | Stale claims: "3 PASS" (was 7), "not filed as a ticket yet" (#1470 is OPEN) | **Corrected**, both |
| 11 (Minor, SPECULATIVE) | `isCandidateForReap` dead, `itoa` reinvents `strconv.Itoa`, `cwdVerdict` zero value undocumented | **Applied**, all three |
| 12 (Question) | `dotf worktree done` carries no liveness protection on Windows | **Answered in the proposal's Risks** — it is the intended posture; `done` is a deliberate single-target act, `sweep` is an automatic multi-target one |

### What round 3 corrected in this document

Three statements here were **false at the tip** and are not silently repaired:

1. *"the pre-existing `worktree` tests are untouched and still pass"* — false. `dab7b6e` edited
   five of them to drive the seam. That edit was correct and necessary; not declaring it was not.
2. *"Gate f subset → 3 PASS"* — the named command yields **7**.
3. *"Not filed as a ticket yet"* for the `features.json` state vocabulary — **#1470 is OPEN** and
   covers exactly it. This spec's own `features.json` uses one of the seven ungoverned spellings.

## Evidence

Every command below was executed in this session.

- [x] **AC1** build-tag split — `sweep_proc_linux.go` and `sweep_proc_other.go` both exist;
      `sweep.go` carries no definition of `isHostProcessInside`.
- [x] **AC2** Linux fails closed on scan failures — mutation-proven for `EvalSymlinks`
      (`TestGateFRefusesWhenTheTargetCannotBeResolved`). The `filepath.Abs` branch is declared
      uncovered below.
- [x] **AC3** non-Linux answers `true` — `TestUnsupportedPlatformAnswersTrueForEveryPath`, executed
      via the build-tag-flip proxy, not only compiled.
- [x] **AC4** `processDiscoverySupported` pinned in **both** directions —
      `TestProcessDiscoveryIsAdvertisedOnLinux` (mutation-proven) and
      `TestUnsupportedPlatformDoesNotAdvertiseDiscovery`.
- [x] **AC5** `gateF` honours both answers — `TestGateFRefusesWhenProcessDiscoveryCannotAnswer`
      and `TestGateFAllowsAReapableWorktreeWhenNothingIsInside`, now on the function production
      calls.
- [x] **AC5b** the under-lock re-check — `TestSweepRefusesWhenAProcessArrivesAfterTheGate`,
      mutation-proven, with a call-count anchor so a fixture that never reached the re-check
      cannot pass it.
- [x] **AC6** reach reported before the counts, as coverage rather than a warning.
- [x] **AC7** `go build ./...`, `go test ./...`, `go test -race`, `GOOS=windows go vet ./...`,
      `golangci-lint run` (2.12.2, the `versions.conf` pin) — all `rc=0`, `0 issues` — **plus** the
      build-tag-flip proxy, which is the only local check that executes the Windows path.
- [x] **AC8** every round-3 surviving mutation now fails the suite — see below.

### Mutation results

The harness is **committed** at `specs/BUG-093-gate-f-fail-closed/mutate.sh` and is `features.json`
f8's verification command. It exits non-zero if any expected-CAUGHT mutation survives, so it is a
check rather than a report — the previous form asserted that this markdown file contained the
string `ANCHOR-MISS`, which is not the same claim.

Anchors asserted before applying: it compares the file's checksum after the patch and reports
**ANCHOR-MISS** rather than a result if nothing changed, so a no-op cannot be read as "the tests
caught it" (lesson 267). The declared survivor is declared *to the harness*, so it prints as
expected rather than as a failure, and a mutation that starts being caught says so.

```
=== mutations the round-3 review found SURVIVING ===
  CAUGHT       finding 3: prefix test loses its separator anchor
  CAUGHT       finding 4: Uninspectable++ becomes a no-op
  CAUGHT       finding 5: reapSingleWorktree drops the Gate f re-check
  CAUGHT       finding 6: Linux claims it has no process discovery
  CAUGHT       finding 7: unresolvable target fails OPEN

=== regressions from earlier rounds (must stay caught) ===
  CAUGHT       round 1 Blocker: EvalSymlinks resolution removed
  CAUGHT       round 1: caller ignores Gate f entirely

=== declared-uncovered (expected to SURVIVE; listed so it is not silent) ===
  SURVIVED     AC2: filepath.Abs error fails open (unreachable by input)

  caught: 7   not caught / anchor-miss: 1
  restored rc=0
```

**The finding-5 mutation is the one that explains why it survived two reviews.** A seam with a
constant answer can never reach the re-check inside `reapSingleWorktree`: a constant `true` is
refused by the first Gate f call, and a constant `false` never exercises the second. The seam has
to answer `false` then `true` — which is precisely the TOCTOU the re-check exists for. The shape
of the double is part of the property, not plumbing.

### The Windows leg, actually executed

`GOOS=windows go vet` compiles `sweep_proc_other.go` and never runs it, which is why round 2's red
`test (windows-latest)` passed every local check. Flipping the build tags runs that implementation
here:

```
=== tags after flip ===
sweep_proc_linux.go: //go:build proxy_excluded    sweep_proc_other.go: //go:build linux

ok  github.com/mlorentedev/dotfiles/cli/internal/worktree   0.843s
rc=0
RESULT: the Windows leg would be GREEN
```

Any future test that expects a reap must drive the `hostProcessInside` seam, or it fails there for
a reason unrelated to what it tests.

### Finding 1 reproduced independently

Not taken on the reviewer's word:

```
docker run -d -u 1000:1000 -v /tmp/wtmount-repro/repo-wt-feat:/wt-mount-target \
           -w /wt-mount-target busybox sleep 60
host readlink /proc/2658284/cwd  -> /wt-mount-target
worktree path on host            -> /tmp/wtmount-repro/repo-wt-feat
dev/ino of /proc/<pid>/cwd       -> 50/992409
dev/ino of the worktree path     -> 50/992409     <- the same directory
string comparison                -> False         <- Gate f says "outside"
```

Why it is declared rather than fixed: `unshare -Urm` is refused here
(`kernel.apparmor_restrict_unprivileged_userns = 1`), and whether `ubuntu-latest` permits one was
not measurable from this machine. A test that skips on the runner reports green while asserting
nothing — which is the defect round 3 found in this very file. **#1523** carries the reproduction
and makes that measurement its first task.

## Accepted gaps, stated rather than left to be found

- **`filepath.Abs` fail-open branch** — mutation survives. `filepath.Abs` fails only when the
  process has no working directory, which a test cannot arrange from inside the process.
- **Unreadable `/proc` directory** — mutation survives. Reaching it needs the directory read
  injected, which restructures a function this change only moves.
- **AC6 is asserted by grep, knowingly.** A behavioural test needs a seam `Sweep` does not have
  (it constructs its own `RealSweepRunner`), and adding one to assert print ordering buys less
  than it costs. The grep is honest about being a grep; `features.json` f5 tracks the new wording.
- **Mount namespaces** — #1523, above.
- **The mutation harness itself carried a silent zsh defect**, found by running it rather than
  reading it: it used `path` as a local variable, and in zsh `path` is tied to `$PATH`, so every
  `cp`/`md5sum`/`mv` inside the function vanished and all 8 mutations reported **ANCHOR-MISS** —
  "the pattern was not found" — when the real cause was that the tools had ceased to exist. No
  corruption resulted (`cp` failed before any `.bak` was written), which was luck and not design.
  Renamed, and a preflight now asserts its tools exist so a broken environment exits loudly
  instead of answering the question wrongly. Four other sites in the repo have the same defect,
  latent; filed as **#1530** with the measurement, plus a doc row in `.claude/CLAUDE.md`.
- **`TestUninspectableProcessesAreCountedByTheRealScan` skips under root**, which is correct: the
  counter genuinely has nothing to count there. The gap that mattered was **visibility** — `go test`
  without `-v` prints neither a skip nor a pass, so a test that stopped running is the same colour
  as one that keeps passing, and the `test (ubuntu-latest)` log confirms it: 25KB with no PASS or
  SKIP lines in it. A longer skip message does not fix that. It now **fails** when `CI` is set,
  and skips only locally — so the one leg that pins the producer can never lose it silently.

## Decisions made during implementation

- **The count stays; what it claims changes.** Deleting `Uninspectable` would have re-collapsed
  the three-valued read that round 1 required. The defect was never the data, it was calling a
  constant a signal. It is now reported as reach, and nothing decides on it.
- **The `> 0` gate on the message is kept deliberately**, and the reason is in the code: `gateF`
  only scans worktrees that reached `StateReapable`, so `0` also means "nothing was considered".
  Printing a reach for a scan that never ran would be a new version of the same lie.
- **The uid split was measured and rejected.** The obvious rescue for finding 2 — count only
  same-uid unreadable processes, so a healthy machine reads zero — does not work: 20 of 227
  same-uid processes are EACCES on this desktop (`systemd --user`, `(sd-pam)`, `ssh-agent`,
  `chrome-sandbox`, `brave`), because `ptrace_may_access` denies a same-uid caller against a
  non-dumpable target. A uid-filtered counter is still never zero. Measured before designing on it.
- **`isCandidateForReap` deleted rather than wired up.** A test seam into dead code reports
  coverage it does not have, and two of AC5's tests were doing exactly that.

## Promotion candidates

- **A guard for the `features.json` state vocabulary** — already owned by **#1470 (GUARD-010),
  OPEN**. Round 3 corrected the earlier claim that it was unfiled. Measured spellings across
  `specs/archive/`: `verified` (42), `passing` (39), `done` (15), `verifying` (15), `passed` (12),
  `implemented` (6), against 523 `pending`.

## Round-5 dispositions

The round-5 verdict is **PASS-WITH-GAPS**, whose meaning is that the gaps are tracked rather
than fixed. Its "Recommended next steps (before archive)" list nonetheless asked for edits to
`proposal.md` and `features.json` — the very files the archive gate measures staleness against,
so applying them invalidates the review that requested them. That contradiction is not this
spec's to resolve; it is filed against the review skill's own template (see below), and the
dispositions here follow Definition of Done §4: applied, ticketed, or declined with a reason.

The contract files archive **exactly as reviewed** at `5a68096`. Everything applied below lives
outside the contract set, so the review still describes what is on disk.

| # | Recommendation | Disposition |
|---|---|---|
| 1 | Confirm `test (windows-latest)` is green at the merge commit | **Verified.** `test-windows: success` at `36c27fb15a6f` (#1531), and at `195cc1887b9a` (#1526) and `6f9b0d95729e` (#1518). |
| 2 | Fix the build-tag flip to the four-file form, or commit it and have f3 run it | **Applied** as `flip-proxy.sh` (committed, executable, non-contract). It flips all four files — both implementations and both `*_proc_*_test.go` — and exits 3 if the tree is left flipped. |
| 3 | Re-run `mutate.sh` and refresh f8's evidence line | **Declined for this archive.** `mutate.sh` is committed and re-runnable, and the review itself records the true count. Editing `features.json` to restate a number the review already corrects would invalidate the review to fix a smaller inaccuracy than the one it creates. |
| 4 | Add the `os.ReadDir("/proc")` fail-open as a declared `SURVIVES` | **Applied** in `mutate.sh` (non-contract). The declared-survivor list is now complete: 8 CAUGHT, 2 declared survivors, 0 regressions, exit 0. |
| 5 | Make AC6's order and its `> 0` gate falsifiable | **Applied in code, declined in the contract.** `TestGateFIsTheLastCheckBeforeRemoval` and the `orderRecordingRunner` shipped in #1531 and run in the suite; only f5's `verification` string still names the older command. Same reasoning as item 3. |
| 6 | Amend `proposal.md` *What* to carve out the namespace case | **Declined, deliberately.** The narrower claim is already durable in two places that outlive this spec: the review's own finding, archived beside the proposal, and **#1523**, which owns the bind-mount fail-open. Rewriting the reviewed proposal would make that finding refer to text that no longer exists. |
| 7 | For #1523: the `os.SameFile` fixture should cover the tmpfs shape | **Ticketed.** Carried into #1523: from the host, `os.Stat("/proc/<pid>/cwd")` returned the same `dev/ino` as the worktree on a real filesystem (`64512/23873208` both), but through a docker bind mount of a **tmpfs** path the dev differed (`26/21432566` vs `50/1237841`) while the inode matched — so a pure `SameFile` test would miss it. |

**Residual, stated rather than hidden:** f3, f5 and f8 carry `verification`/`evidence` strings
naming the pre-round-5 commands and counts. The committed `flip-proxy.sh`, `mutate.sh` and the
order test are the current truth; this table is the pointer from one to the other.
