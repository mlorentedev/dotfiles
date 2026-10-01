---
tags: [spec, verification, templates]
created: "2026-08-27"
---

# Verification - HARNESS-088-handoff-threads

## Evidence

| AC | Proof |
|---|---|
| AC1 | `TestWriteThreadLeavesEveryOtherThreadByteIdentical` — a fixture shaped like the actual clobber (session 4's block over session 3's); plus a live run against a copy of the real `MEMORY.md` |
| AC2 | `TestWriteThreadIsIdempotent` |
| AC3 | `TestWriteThreadAppendsANewThreadWithoutReordering` |
| AC4 | `TestWrittenThreadsStayInsideTheArchivedBlock` — drives `extractHandoffBlock` over the written file |
| AC5 | `TestWriteThreadRefusesWhenTheSectionIsAbsent`, `TestWriteThreadRejectsAnEmptyKey` |
| AC6 | `TestThreadKeyIsTheBranchSoItTravelsBetweenMachines` and `TestRepoIdentityResolvesTheProjectFromAnywhereInTheTree` (renamed from `TestThreadKeyDerivesFromTheWorktree` in #1280; see Archive below) |
| AC7 | `TestJournalNameIsDerivableAndDistinctPerWorktree` |
| AC8 | **Declined at archive time, tracked as #1881.** Never built, and not claimed |

## Command output

```
$ go test ./internal/mem/ -run 'TestWriteThread|TestThreadKey|TestJournalName|TestWrittenThreads' -v
--- PASS: TestWriteThreadLeavesEveryOtherThreadByteIdentical
--- PASS: TestWriteThreadIsIdempotent
--- PASS: TestWriteThreadAppendsANewThreadWithoutReordering
--- PASS: TestWriteThreadMatchesAThreadWhoseBranchChanged
--- PASS: TestWriteThreadRefusesWhenTheSectionIsAbsent
--- PASS: TestWriteThreadRejectsAnEmptyKey
--- PASS: TestWrittenThreadsStayInsideTheArchivedBlock
--- PASS: TestThreadKeyDerivesFromTheWorktree
--- PASS: TestJournalNameIsDerivableAndDistinctPerWorktree
ok      .../internal/mem

$ go test ./...
19/19 packages ok

$ go build ./... && go vet ./... && GOOS=windows go vet ./...
(clean, both platforms)

$ golangci-lint run          # pinned 2.12.2
0 issues.

$ bats tests/compile-harness.bats
exit 0
```

### Against the real file, run as a binary

```
$ dotf mem thread --date 2026-08-27 --project dotfiles --agent claude
thread   wt-pi-harness
journal  sessions/2026-08-27-dotfiles-claude-wt-pi-harness.md

$ printf '...' | dotf mem handoff-write --memory <copy of the real MEMORY.md>
wrote      thread "wt-pi-harness"

session 4's block survived : True
my thread written          : 1 heading
sections intact            : Index, Findings, CI Pipeline, User Preferences, Session Handoff
```

## Decisions made during implementation

- **The writer shares `extractHandoffBlock`'s boundary.** Checked before writing
  a line, because a second boundary rule would have been the fifth
  silently-divergent parser found here in a week. It stops at the next `## `, so
  `###` sub-blocks fall inside and archival is unchanged — asserted by AC4 rather
  than assumed.
- **A new thread is appended, never inserted.** Reordering a foreign entry is a
  diff its author did not make and would have to review.
- **Empty stdin is refused.** Blanking a thread is the clobber this exists to
  prevent, wearing a different shape.
- **The file is replaced through a temp file in the same directory.** A
  half-written `MEMORY.md` is the one outcome worse than a clobbered one, and it
  is read at the start of every session.

## The defect this found in itself

`ThreadKey` first read only `filepath.Base(cwd)`. A session working in any
subdirectory — `cli/`, where most work in this repository happens — resolved to
`main`. **Every such session would have shared one thread key and clobbered the
others exactly as before**, while the tests passed, because every case they
supplied was a worktree root.

Found by running `dotf mem thread` from `cli/` and reading the output. The fix
walks up; three subdirectory cases are now in the table.

That is the third time this session that a defect survived unit tests and died on
the first real invocation, and the pattern is consistent: the tests supplied the
input the author was imagining.

## Promotion candidates

- **Strong, and now on its second instance in this repository**: *a config
  surface written by several agents is merged by marker, never by position or by
  discipline.* First instance #1272 (hook emission alongside Orca), second this
  one (the handoff block). A third from outside this repo would make it a
  pattern; two is already enough to stop writing the rule as prose.

## Second sitting — the debt, and cross-machine identity

| AC | Proof |
|---|---|
| The project resolves from anywhere in the tree | `TestRepoIdentityResolvesTheProjectFromAnywhereInTheTree` — worktree root, `cli/`, `cli/internal/mem/`, and a main-checkout subdirectory |
| A thread is the branch and travels | `TestThreadKeyIsTheBranchSoItTravelsBetweenMachines` — the same branch in two different worktrees is ONE thread, which is the cross-machine property the design turns on |
| The host qualifies only where it disambiguates | `TestThreadKeyQualifiesOnlyTheDefaultBranchWithTheHost` — `main@host`, `master@host`, and a feature branch carrying no `@` |
| A detached HEAD says so | `TestThreadKeyNamesADetachedHeadRatherThanGuessing` |
| git knowing nothing is reported, not guessed | `TestRepoIdentityReportsWhenGitKnowsNothing` |

Live, from `cli/` inside a worktree:

```
$ dotf mem thread --date 2026-08-27 --project dotfiles --agent claude
thread   feat-harness-088-handoff-threads
journal  sessions/2026-08-27-dotfiles-claude-feat-harness-088-handoff-threads.md

$ cd ~/Projects/dotfiles && dotf mem thread
thread   main@msi
```

### A mutation test of mine gave a FALSE NEGATIVE

Worth recording because it is the failure mode this repository keeps meeting from
a new angle. The first mutation replaced `Project:  filepath.Base(...)` written
with **one** space where `gofmt` had written **two**, so nothing was mutated — and
the passing test read as *"the test does not catch this"*. Re-run against the
real target, it failed exactly as it should.

**A mutation test that reports "no change" has not proven the test is weak; it
has proven nothing at all.** Assert the mutation applied before believing its
result.

## Archive (2026-09-30)

Closed under the #770 sweep. AC1-AC7 shipped in #1279 (`71a4c39f`) and #1280
(`4397ba3e`) and still pass on `main`:

```
$ cd cli && go test ./internal/mem/ -run 'TestWriteThread|TestWrittenThreads|TestThreadKey|TestRepoIdentity|TestJournalName' -v
(every test listed in the tables above: PASS)
ok      github.com/mlorentedev/dotfiles/cli/internal/mem
```

- **AC8 declined, #1881.** The doctor check was never built. It is a separate,
  read-only feature and does not block what this spec delivered.
- **Gate session-state key, #1882.** The last unticked audit item now has a ticket.
- **`features.json` f5 pointed at a test that no longer existed.** #1280 renamed
  `TestThreadKeyDerivesFromTheWorktree`. `go test -run` with a name that matches
  nothing still exits 0, so f5 had been passing without running anything. It now
  names `TestThreadKeyIsTheBranchSoItTravelsBetweenMachines`.
- `tasks.md`'s Implementation section still held the template placeholders. It
  now lists the tasks as they were done.

## Round-1 review fixes (#1884, 2026-10-01)

Round 1 (`nan/deepseek-v4-flash`, FAIL) found the lost update, measured on the
built binary. Its record is kept as `review-round-1.md`.

- **Lost update, AC9.** The owner reversed the lock-file non-goal and picked a
  lock the kernel releases over O_EXCL: a writer killed while holding an O_EXCL
  file leaves it behind, and every later handoff waits on it. `cli/internal/filelock`
  is extracted from the pattern `worktree` and `agent` already use (`flock`; an
  unshared `CreateFile` on Windows). The lock file lives under
  `XDG_RUNTIME_DIR` or the user cache dir, outside the vault, and its name hashes
  the canonical path. Moving the two older copies onto the package is a separate
  ticket.
- **The test fails without the fix.** `TestConcurrentWritesDoNotLoseAThread`
  starts 8 writers together, 5 rounds. Before the lock, every run lost threads:
  `thread feat-w0 is missing after 8 concurrent writes`, and more. After it, the
  test passes under `-race`, `-count=3`.
- **Two paths, one lock.** With `EvalSymlinks` removed,
  `TestTwoPathsToOneMemoryShareTheLock` fails (mutation applied and observed).
  Restored, it passes. Whether `EvalSymlinks` resolves a Windows junction is
  not measured; the test skips on Windows and says so.
- **`journalWriter`, AC10.** The SPECULATIVE finding is fixed, not just stated.
  The scan skips the journal name's first word, which is always the project's.
  Residual: a project with an agent word after its first segment (`my-pi-app`).
  None exists.

```
$ cd cli && go build ./... && go vet ./... && go test ./... && GOOS=windows go vet ./... && golangci-lint run
ok (every package); 0 issues.
```

