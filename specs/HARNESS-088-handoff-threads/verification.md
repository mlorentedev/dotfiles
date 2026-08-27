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
| AC6 | `TestThreadKeyDerivesFromTheWorktree`, including three subdirectory cases |
| AC7 | `TestJournalNameIsDerivableAndDistinctPerWorktree` |
| AC8 | **Not built.** Recorded in the proposal rather than claimed |

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
