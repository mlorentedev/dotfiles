---
id: lesson-187-cobra-s-print-family-writes-to-stderr-and-a-setout
type: lesson
status: active
created: "2026-08-10"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 187: Cobra's `Print` family writes to stderr, and a `SetOut` test cannot tell you otherwise

**Context**: #915. `dotf env path VAULT_PATH` is the ADR-025 seam — `AGENTS.md` tells every agent to resolve `$VAULT_PATH` through it, and `setup-linux.sh` provisions the hive daemon from it. Captured in a `$(...)`, it returned an empty string. It had done so since the subcommand shipped.

**Problem**: `cobra.Command.Print`, `Printf` and `Println` write to `OutOrStderr()`, not stdout. Four subcommands whose output is read by a caller used them — including `env generate --stdout`, a flag named for the stream it was not using. The consequence was not a visible error anywhere: `setup-linux.sh:1088` captured nothing and fell through to a hardcoded `$HOME/Projects/knowledge` literal, which is correct on this machine and wrong on any box whose vault sits elsewhere. ADR-025 exists to delete exactly that literal, and the seam meant to replace it was quietly returning nothing.

Two layers of test blindness kept it alive for months. `tests/verify-setup.bats` asserted the resolved path with `run`, which merges stdout and stderr into `$output` — it passed on the value while every real caller got an empty string. And no Go test could have caught it either: `cmd.SetOut(buf)` makes `OutOrStderr()` return that same buffer, so the shared `execute()` helper reports `Print*` output as "stdout" and passes identically before and after a fix. The bug had also been *found* once already, by whoever wrote `install-dotf.{sh,ps1}` — the comment there reads "root cause: dotf writes version to stderr" — and was worked around with a `2>&1` merge in two files rather than fixed or ticketed.

**Solution**: `fmt.Fprintln(cmd.OutOrStdout(), …)` at the four sites, which is the idiom `dotf secrets show` and `dotf mem project-key` already used correctly. The guard swaps `os.Stdout`/`os.Stderr` at the process level and installs no Cobra writers, so it observes what a shell `$(...)` observes; it was run against the unfixed binary first and reported the value sitting on stderr.

**Rule**: In Cobra, `cmd.Print*` is for status messages; anything a caller parses needs `fmt.Fprint(cmd.OutOrStdout(), …)`. More generally: **a test that mocks two streams into one object cannot test which stream anything went to** — the mock erases the property under test. When the bug is about a channel (stream, exit code, file descriptor, header), the guard has to exercise the real channel, which usually means executing the real binary. And when a workaround comment states a root cause, that is a defect report written in the wrong place: a `2>&1` added "just in case" is floating debt, and the next caller will not know to add it.
