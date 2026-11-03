---
tags: [spec, verification, templates]
created: "2026-10-22"
---

# Verification - PLAT-005-vm-session-verbs

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

## Test status

- Test suite: `go -C cli test ./... -count=1` -> exit 0, 50 packages ok (Linux exec and cp, 2026-10-30).
- Manual smoke test, Linux exec and cp, Lima 2.2.1 on macOS, session `plat005-probe`:
  - AC2: `printf` with `a b`, `$HOME`, `;id` came back literal, `ok`, `exit: 0`; `exit 3` gave `ok`, `exit: 3`, process code 1.
  - AC2: `--script` ran from a temporary guest path and left none behind.
  - AC3: piped stdin reached the guest, on argv and on `--script`.
  - AC4: a file went host to guest to host byte-identical through `/tmp/a b`; `findmnt` in the guest listed no share.
  - AC7: the wire protocol ran on the guest with synthetic values only. The masking is proved by unit tests, never against a live store (lesson 261).
  - AC11: `--timeout 2s` on `sleep 20` gave `timeout`, process code 1, in 2068 ms, with no `sleep` left in the guest; `no-session` exit 3; a stopped guest `stale` exit 2.
- Manual smoke test, Windows session verbs on the real guest, macOS Arm64, session `plat005-win`:
  - AC1: `up` booted an overlay in 36 s and returned; QEMU outlived dotf, and `status` answered `ok`. A second `up` under another name was `busy`, exit 2, naming the holder.
  - AC1: with the guest stopped behind the record, `status` answered `stale` and `up` was `stale`, exit 2, naming `down`.
  - AC2: `exec -- whoami` gave `dotf-win\\dotf`, `ok`, `exit: 0`, in 6.5 s; a program exiting 3 gave `ok`, `exit: 3`, process code 1; a missing one `ok`, `exit: 127`; a UTF-8 `--script` printed its text whole and exited 4.
  - AC3: piped stdin was `unsupported`, exit 2.
  - AC4: 200 kB of random bytes went host to guest to host byte-identical through `C:\Users\Public\a b.bin`; a directory and a missing guest file both failed out loud.
  - AC4: unit tests pin that a failed `cp proof:C:\nothing <target>` leaves `<target>` unchanged with no temporary file beside it; the real-guest copy above ran on the streaming read.
  - AC8: on the real guest `exec --elevated -- whoami /groups` showed `High Mandatory Level` with `BUILTIN\Administrators` enabled; without the flag it showed `Medium Mandatory Level` and `net session` was denied, and with it `net session` exited 0. On a real Lima guest `exec --elevated -- id -u` gave 0 and a plain call 501; `--elevated --timeout 2s` on a program ignoring TERM was `timeout`, with no `sleep` left.
  - AC10: `exec --elevated` ran `Add-WindowsCapability` inside a session in 10.6 min, giving `OpenSSH_for_Windows_9.5p2`; `clean-base` is unchanged. With the NIC already static, it failed with `0x80240438` (no network).
  - AC11: `--timeout 8s` on PowerShell running `ping -n 60` gave `timeout`, process code 1, with the output so far; no `ping` was left in the guest, and the call left no task or directory.
  - AC11: `down` on an unknown name was `no-session`, exit 3. `down` cleared the stale session, `status` went back to `[]`, and `clean-base` stayed.
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

-
-

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-005-vm-session-verbs/` -> `specs/archive/PLAT-005-vm-session-verbs/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)

## `run` composed below the verbs (AC5, AC6)

`dotf secrets run -- dotf vm linux` from `33ce746c`, on a real Lima guest: first converge, second converge, "converged, and the second run changed nothing", exit 0, guest deleted. The token was set, and the run's log carries 28 `[REDACTED:` markers. `vm_test.go`'s assertions are unchanged; `TestVerify_MasksTheTokenInAConvergesOutput` and `TestVerify_RefusesATokenItCouldNotMask` are new. The Windows `run` is untouched.
