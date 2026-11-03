---
id: "HARNESS-185-handoff-loss-check"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-01"
issue: "mlorentedev/dotfiles#1881"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-185-handoff-loss-check

## Why

<!-- from issue #1881: doctor: report a handoff section that lost a thread whose worktree still exists -->

`dotf mem handoff-write` stops one session overwriting another's thread, but a hand edit, a wrongly resolved merge conflict or an older agent can still delete a live thread's entry from `MEMORY.md`, and nothing reports it: the owner finds out when the session that owned the thread next looks (HARNESS-088 AC8, declined at archive time). This is not hypothetical: on 2026-11-01 the dotfiles vault held seven `threads/<key>.md` files with no entry in `MEMORY.md`, one of them for a worktree that still existed. #1881 was parked unbuilt because its shape ("a journal for that thread exists under `sessions/`") rests on a file name that no longer identifies the thread, and the store has two layouts. This spec fixes the proof and the layouts first, so the check is not a guess.

## What

`dotf doctor` gains a read-only check, "Handoff threads", that WARNs for each linked worktree of the repository whose thread has **proof of a handoff** but **no entry** in the project's `MEMORY.md`.

- **The worktree's key** is `mem.ThreadKey(<worktree path>)`, the same function `dotf mem thread` and `handoff-write` use, so the check cannot disagree with the writer about what a thread is.
- **Proof a handoff was written** is the per-thread file `threads/<key>.md` beside `MEMORY.md` (MEMORY-008, ADR-056). Not a journal under `sessions/`: `JournalName` appends the key to `<date>-<project>-<agent>`, the agent set is open and a key may itself contain `-`, so a name cannot be split back into a key without guessing. The file is exact, written by the same call that writes the entry, and the sanctioned archive step moves it to `archive/` together with deleting the entry, so an archived thread leaves no file behind and never trips the check.
- **"Has an entry"** means `MEMORY.md` holds a `### thread: <key>` heading, or one for a fork `<key>+<agent>` (`handoff-write` keeps another agent's block by writing to `<key>+<agent>`).
- **Supported layouts:** the per-thread store only. A thread kept as an inline block, with no `threads/<key>.md`, has no proof, so it yields no finding; this is stated in the check's SKIP line when the project has no `threads/` directory at all, so a vault still on the inline layout reads "not covered", never "all fine".
- **Output:** one WARN naming each lost key and the file that still holds its body, with the remedy: re-run `dotf mem handoff-write --thread <key>` with that body, or, if the thread is finished, move the file to `archive/` (the documented archive step). Never FAIL, never `--fix`: restoring an entry needs a human to say whether the thread is live.
- **Skips** (SKIP, with the reason): no vault resolves on this machine, the repository has no vault project, or `MEMORY.md` is absent.

## Out of scope

- Repairing the entry. The check reports; `handoff-write` and the owner decide.
- Threads with no worktree (finished or on another machine). Seven such files exist today; whether they should be archived is a vault hygiene question for `/crystallize`, not a doctor warning.
- The inline layout. Detecting loss there needs a proof the store does not keep; vaults still on it are told they are not covered.
- A check on `threads/<key>.md` files whose entry points to a missing file (the reverse direction). `handoff-write` already refuses that write (`CheckThreadFile`).
- Cross-machine keys. A default-branch key carries the host (`main@macmini`); the check compares the key as this machine derives it and does not look for another machine's.

## Risks / open questions

- **A worktree whose thread was never written** (a session that did no handoff) has no file, so it is not reported. That is the right reading of "lost": there was nothing to lose. It means the check does not catch "this worktree never handed off"; that is a different gap, and it is not filed here.
- **Archived by hand, file left behind.** Someone who deletes an entry and leaves `threads/<key>.md` looks identical to a loss. The warning text must therefore offer the archive step as the second remedy, which is what the seven files today most likely are.
- **Detached worktrees** key as `<worktree-name>@<host>`; the check uses the same function, so a detached worktree is covered when its thread was written, and no new rule is needed.
- **Cost.** One `git worktree list` and one directory read per doctor run. If the vault is on a slow mount the check should not stat anything but `threads/`.

## Acceptance criteria

- [x] AC1: a fixture with a linked worktree, `threads/<key>.md` present and no `### thread: <key>` entry makes the check WARN, naming the key and the file.
- [x] AC2: the same fixture with the entry present, or with a `### thread: <key>+<agent>` fork entry, passes.
- [x] AC3: a thread whose file was moved to `archive/` (the sanctioned archive step) does not warn.
- [x] AC4: a worktree with no `threads/<key>.md` does not warn, even when a journal for it exists under `sessions/`.
- [x] AC5: with no vault, no vault project, no `MEMORY.md`, or no `threads/` directory the check SKIPs and says why; the no-`threads/` SKIP says the inline layout is not covered.
- [x] AC6: the check never writes to the vault and never FAILs; `dotf doctor --fix` leaves the vault byte-identical.
- [x] AC7: the worktree key is computed with `mem.ThreadKey`, and a test fails if the check derives it any other way (a detached worktree and a branch with `/` both resolve as `handoff-write` would key them).
- [x] AC8: run against the real vault once, the result is recorded in `verification.md` (which of today's seven entry-less files, if any, belong to a live worktree).

## References

- Bitácora board: mlorentedev/dotfiles#1881 (parked with options A and B; the owner chose A on 2026-11-01)
- Related: HARNESS-088 (#1278, per-thread blocks; AC8 declined), MEMORY-008 and ADR-056 (the per-thread store), #1477 (the budget and the archive step)
- Code: `cli/internal/mem/handoff.go` (`ThreadKey`, `JournalName`), `cli/internal/mem/thread_store.go` (`threadsDir`, the archive wording)
