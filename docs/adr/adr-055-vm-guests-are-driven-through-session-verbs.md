---
id: "ADR-055-vm-guests-are-driven-through-session-verbs"
type: adr
status: proposed
owner: manu
date: "2026-10-21"
supersedes: []
extends: [adr-053-the-windows-test-guest-runs-on-qemu-driven-by-dotf, adr-045-one-entrypoint-converges-every-os, adr-052-one-forge-write-surface-in-dotf]
issue: mlorentedev/dotfiles#2842
tags: [architecture, decision, virtualization, windows, linux, testing, cli]
created: "2026-10-21"
---

# ADR-055: The VM guests are driven through session verbs

## Context

`dotf vm linux` and `dotf vm windows` are each one fixed pipeline: boot, copy a source in, converge twice, judge `last.json`, tear down (`vm/vm.go` `Verify`, `vm/run.go` `Run`). Nothing in the middle is reachable. #2763 shows the cost: proving the `sshd -T` effective config on Windows needs the guest to run a command nobody wrote a step for, as an administrator, and bring its output back. Three obstacles stand between the Windows guest and that proof (the scheduled task has no `/rl highest`, `clean-base` has no OpenSSH Server, there is no way to give the role a mesh address), and all three come down to one missing thing: a way to hold a guest, run something in it and read the result.

The owner wants that interface usable by other projects and products, not only by the converge verification. The two guests share nothing today:

- **Addressing.** Lima instances are named (`dotf-linux-<unix time>`, several may coexist). The Windows guest is one fixed definition under `StateDir/vm/windows`, run as a throwaway overlay of the sealed `clean-base`, and a second run is refused while one is up.
- **Transport.** Lima is `limactl shell` and `limactl copy`. Windows has no SSH (ADR-053 decision 3): only the QEMU guest agent, which runs as LocalSystem, has no stdin, and reaches a user's session through a scheduled task.
- **Credentials.** Linux receives `GITHUB_TOKEN` on stdin only; Windows receives none (ADR-053 decision 5).

Constraints that hold whatever the design (all measured or pinned by tests): no host directory is shared with either guest (`mounts: []`, `Isolated()`, `Machine.Args()`), so `cp` is a transfer through the channel, never a mount; no credential enters an image, a snapshot or `clean-base`; `clean-base` is read-only; the Windows guest builds on macOS Arm64 only; Go owns the logic with no per-OS twin (ADR-020 C1).

## Decision

1. **A guest is driven by four verbs, the same on both guests**: `dotf vm <os> up`, `exec`, `cp`, `down`, plus `status` to list sessions. `<os>` stays a subcommand, as today.
2. **A session has a name, and the name is the only address.** `up --session <name>` creates it (default: `<os>-<unix time>`) and prints it; the other verbs take `--session`. Linux allows many sessions. Windows allows one, so its only valid name is the one it holds: a second `up` is refused with exit 2 and status `busy`. The limit is the guest's (one QGA socket, one overlay, one set of ports), not the contract's: lifting it later changes no verb.
3. **`up` always keeps.** A session is something the caller asked to hold, so it persists until `down`. `--keep` survives only on `run`, where the guest is otherwise discarded. `down` deletes the Lima instance or the overlay and its state; `clean-base` is never touched.
4. **`exec` takes its payload from outside the command line.** The program and its arguments follow `--` and are executed directly, with no shell; a script is `--script <host file>`, copied in and run by the guest's native interpreter (`sh`, PowerShell). Prose never travels in argv (ADR-052: the deny glob `Bash(*dotf secrets show*)` matches anywhere in a command line, the BUG-152 failure). `exec` forwards its stdin on Linux; on Windows the guest agent has none, so `exec` refuses it with `unsupported` rather than dropping it.
5. **`cp` is a transfer through the channel**: `limactl copy` on Linux, `QGA.WriteFile`/`ReadFile` (48 KiB chunks) on Windows. `cp <host path> <session>:<guest path>` and the reverse. No mount, no share device.
6. **Elevation is per `exec`: `--elevated`.** On Windows it runs the scheduled task with `/rl highest`; on Linux it prefixes `sudo`. It is a flag on the call, not a property of the session, so the common case never needs it and a test names where it uses it.
7. **`--json` writes one record per call to stdout, whatever the outcome**, the shape of `agent.go` `emitRecord`:
   `{"version":1,"session":"…","os":"…","status":"ok|failed|busy|unsupported|no-session|timeout","exit":<guest exit code or null>,"stdout":"…","stderr":"…","truncated":false,"duration_ms":0}`.
   A consumer reads `status` and `exit`, never stderr and never the process code. `up` and `down` carry `session`, `status` and the paths the caller needs (`state_dir`). Empty lists are `[]`. `version` changes only when a field is removed or changes meaning.
8. **Process exit codes keep their meaning** (`exit.go`): 0 success; 1 any failure, including a non-zero guest exit (the guest's own code is in `exit`, because the process code space is spoken for); 2 a refusal before any work (bad flag, `busy`, `unsupported`, host not supported); 3 no session or guest could serve the call. A timeout is 1, not 3, so a composer does not retry it elsewhere.
9. **Progress goes to stderr.** The `==> ` lines move off stdout, so with `--json` stdout is exactly the record, and without it stdout is exactly the guest's output.
10. **`run` becomes a composition of the verbs**: `up`, `cp` the source in, `exec` the converge twice, judge `last.json` (`Converged`, `Idempotent`), `down`. Its behaviour and flags do not change; the first PR proves it with the existing tests green and no edits to them.
11. **`clean-base` keeps no `sshd` (ADR-053 decision 3 is unchanged).** The first use of the guest is still proving that `dotf` installs `sshd` from nothing (#2672). A test that needs `sshd` in a session installs it with `exec --elevated` inside that session, which the overlay discards; it does not belong in the image. That leg tests the effective-config proof (#2763), not the from-zero install, and the PR says which one it covers.
12. **Credentials keep their rule.** A token reaches a Linux session on `exec` stdin only, never in argv or on disk (ADR-028, lesson 144); a Windows session receives none. `down` leaves no credentialed disk because none was written.

## Consequences

- A session is the unit another project drives: it can `up` a guest, `cp` its own build in, `exec` its own checks and read a stable record, without `dotf` knowing what it verifies. `dotf`'s converge check is the first consumer, not the only one.
- `--elevated` inherits ADR-053's weakness: UAC consents without a prompt in the guest (`ConsentPromptBehaviorAdmin=0`). An elevated command therefore passes in the guest even where a real machine would stop at a prompt, and that difference is not verified here. The verb proves what an administrator can do, not that a prompt would appear.
- Windows `exec` runs through a scheduled task, polls `<step>.exit` and `<step>.log`, and is slower than Linux `limactl shell`. The record's `duration_ms` makes that visible; it is not hidden behind a different contract.
- Windows `exec` has no stdin. A caller that needs to pass data writes it with `cp` first. The refusal is `unsupported` with exit 2, never a silent drop.
- A new agent-facing verb needs an allow rule in each harness dialect, landed by the owner (ADR-048 clause 6). The PR that adds the verbs names the rules and does not widen the permission baseline itself.
- Every refusal test sets `XDG_STATE_HOME` to an empty temp dir (`noGuest(t)`, lesson 489), so a regression cannot boot a real guest on a machine that holds a `clean-base`.
- The Windows guest still builds only on macOS Arm64. The contract is the same on a Linux host, and the Linux host is still the follow-up ADR-053 names.
- Three PRs follow, each independent of the next: the verbs with `run` recomposed (no behaviour change), `--elevated` with a stand-in mesh address for the role (#2763 obstacles 1 and 3), and the Windows `sshd -T` fixture with the guest acceptance that closes #2763.

## Alternatives rejected

- **A second pipeline for each new need** (an `--exec` flag on `run`, a `--elevated` flag on `windows`): every consumer that wants a different middle forks the pipeline again, and the Windows and Linux ones drift.
- **SSH for exec and cp on Windows**: vetoed by ADR-053 decision 3, and it would need an `sshd` that `clean-base` must not have.
- **A host mount for `cp`**: breaks the isolation both guests pin by test.
- **The command as a single string argument** (`exec "<command>"`): puts prose in argv, which the deny globs match anywhere, and needs shell quoting that differs between `sh` and PowerShell.
- **One guest address for all, a UUID per session**: hides the fact that Windows holds one guest and makes `status` the only way to find it. A name the caller chose is findable and stays readable in logs.
- **Elevation as a session property**: every command would run elevated to serve the few that need it.
