---
tags: [spec, verification, templates]
created: "2026-09-05"
---

# Verification - BUG-093-gate-f-fail-closed

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

Anchors asserted before applying: the harness compares the file's checksum after the patch and
reports **ANCHOR-MISS** rather than a result if nothing changed, so a no-op cannot be read as
"the tests caught it" (lesson 267).

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
