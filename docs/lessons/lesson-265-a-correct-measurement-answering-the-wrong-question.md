# 265 - A correct measurement answering the wrong question, three times in one session

**Date:** 2026-09-03
**Area:** verification, CI, cross-session coordination

## What happened

Three separate near-misses in a single session on CLI-072's archive. Every
measurement taken was accurate. Every conclusion drawn from it was wrong,
because the question being answered was not the question that mattered. None of
the three announced itself as an error — each produced a plausible, well-formed
result that nothing downstream could distinguish from a correct one.

They are recorded together because they are one failure, not three.

### 1. The exit status of the wrong process

Verifying the Go suite before committing:

```bash
go test ./... 2>&1 | grep -v "^ok\|no test files" | tail -10; echo "GO_SUITE_EXIT=$?"
```

`GO_SUITE_EXIT=0` — and it reports **`tail`'s** status, not `go test`'s. `tail`
succeeds whether the suite passed or every package failed.

This is the exact defect `.claude/CLAUDE.md` had gained a prohibited-patterns row
for **that same morning** (#1465, the `${PIPESTATUS[0]}` entry: *"a verification
that reads a pipeline's exit status is exactly where this bites: `cmd | tail`
reports `tail`'s status, so a failing `cmd` passes"*). Reading the rule did not
prevent writing the bug. The honest form redirects to a file and reads `$?`
directly:

```bash
go test ./... > /tmp/gotest.log 2>&1; echo "GO_TEST_EXIT=$?"
```

### 2. The green check that was a different job

`test-windows` was cancelled at its 30-minute ceiling. A peer session offered a
control: *"my `test (windows-latest)` went green at about the same time on a
larger Go diff, so the runner is not slow tonight — it looks specific to your
run."* That was relayed onward as evidence before anyone checked it.

`test (windows-latest)` is the **Go test matrix** leg and runs in ~1m25s.
`test-windows` is a **different job** that runs `setup-windows.ps1` end-to-end
and normally takes 9-10m. The green one controlled for nothing. The peer's own
`test-windows`, on a different branch and a different diff, was cancelled at the
same ceiling — which is the opposite conclusion.

Two names differing by punctuation, in the same check list, for jobs with
nothing in common.

### 3. The rogue writer that was the owner

`ai/pi/models.json` — a tracked file — appeared modified in a worktree nobody
claimed. Two agents investigated it properly and refuted each other's mechanisms
with real evidence: the deploy target is a regular file, not a symlink, so
nothing writes through from `~/.pi/agent/`; `git show HEAD:` matched the deployed
copy exactly; the writer was narrowed to a single worktree's cwd; a timing
argument clearing `pi` was itself corrected, because the file kept growing after
the window that argument covered.

Every measurement was right. The file was being edited, by hand, by the one
human on the machine, who was still typing while it was being measured — and who
had already told the other session so.

## The shape

All three, and the three of the same family recorded in lesson 260's session:
**the verification succeeds.** That is the genus. A crash, a timeout, a
suspicious empty result — those announce themselves. These return a plausible
value that is correct as an answer to a question nobody asked:

| Measured | Believed to mean | Actually meant |
|---|---|---|
| `tail` exited 0 | the suite passed | `tail` ran |
| `test (windows-latest)` green | Windows CI is healthy | one 1m25s Go job passed |
| a tracked file changed unbidden | a tool is writing it | a person is writing it |

## What to do differently

**Name the question before trusting the number.** In all three cases the gap
between "what I measured" and "what I need to know" was one sentence long and
was never written down. Writing it is the whole defence.

**When a result is surprising, check the cheapest explanation first, and check
it before building a mechanism.** "A tracked file changed and I did not change
it" has a far more likely cause on a single-user machine than a rogue tool. Two
agents spent half an hour on hypotheses because neither tested the framing, only
the mechanisms inside it.

**A peer's measurement is evidence, not a conclusion — verify the premise, not
the description.** The wrong-job control was accepted and relayed onward because
it was plausible and offered in good faith. Checking it cost one `gh pr checks`
call. Throughout the rest of the session the opposite habit — reading
`.pre-commit-config.yaml` rather than accepting a description of it, reading
`setup-windows.ps1:1305` rather than accepting an analysis of it — corrected a
peer's claim once and one of our own proposals once.

**Never read a pipeline's exit status.** Redirect and read `$?`. The rule was in
the repo's own instructions and was still violated within hours; a rule that is
read is not a rule that is followed, which is why the prohibited-patterns table
exists and why this belongs beside it.

## See also

- Lesson 260 — the same genus at a different altitude: `go test -run '<no-match>'`
  exits 0, so a command naming a deleted test passes forever.
- `.claude/CLAUDE.md`, prohibited patterns — the four silent rows.
- #1472 — the CI defect these near-misses surrounded: a required check that is
  `skipped` on `main` accumulates no evidence about itself in the one place
  people look.
