---
id: "PLAT-005-vm-session-verbs"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-22"
issue: "mlorentedev/dotfiles#2842"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "owner decision 2026-10-22: PLAT-005 unblocks the Windows leg of #2763; the cleanup sessions have not archived any yet (12 active against 10) (11 active, limit 10, 2026-10-22)"
---

# PLAT-005-vm-session-verbs

> The contract is ADR-055 (accepted). This spec orders the work into three PRs; it does not reopen a decision.

## Why

<!-- from issue #2842: PLAT-005: dotf vm session verbs (up, exec, cp, down) so other projects can use the guest -->

`dotf vm linux` and `dotf vm windows` are fixed pipelines (boot, copy, converge twice, judge, tear down); nothing in the middle is reachable. The Windows leg of #2763 (`sshd -T` fixture and acceptance on the guest) needs to run one command in the guest as an administrator and read its output, and the owner wants that interface usable by other projects, not only by dotf's converge check. Without it the Windows proof stays unmeasured and each new need forks the pipeline again.

## What

After the three PRs, `dotf vm <os> up | exec | cp | down | status` hold and drive a guest through one contract on Linux (Lima) and Windows (QEMU, QGA), per ADR-055:

- a named session (Linux many, Windows one; a second Windows `up` is `busy`, exit 2);
- `exec` runs argv with no shell, or `--script <file>`, with `--elevated` per call, and returns one JSON record (`version`, `session`, `os`, `status`, `exit`, `stdout`, `stderr`, `truncated`, `duration_ms`) whatever the outcome; `status` is `ok`, `failed`, `busy`, `unsupported`, `no-session`, `timeout` or `stale`;
- `exec --secret <registry-id>` resolves the value through the catalog as `dotf secrets run` does and writes it on the guest's stdin the way `withToken` does today, which gives the redaction something to key on;
- `cp` moves files through the channel in both directions, with no host share;
- progress goes to stderr, so with `--json` stdout is exactly the record;
- the guest's output is redacted for every value `--secret` forwarded; plain piped stdin is the caller's and is not redacted (ADR-055 decision 13);
- `run` is the composition of the verbs, with its flags, lifecycle, two converges and judgment unchanged; it passes `--secret GITHUB_TOKEN` where it forwards the token today.

## Out of scope

- Multiple concurrent Windows sessions (a limit of the guest, not of the contract).
- A Linux host for the Windows guest (the follow-up ADR-053 names).
- Changing `clean-base`: it keeps no `sshd` (ADR-053 decision 3); a test installs it inside its own session.
- The allow rules for the new verbs in each harness (#2859, owner-landed, ADR-048).
- Any other harness permission change.

## Risks / open questions

- UAC consents without a prompt in the guest, so `--elevated` proves what an administrator can do, not that a prompt would appear. Documented in ADR-055; the verb's help repeats it.
- Windows `exec` goes through a scheduled task and polling, so it is slower than Linux; `duration_ms` shows it.
- The scheduled task needs `/rl highest` for `--elevated`; the stand-in mesh address for the `ssh-target` role has no injection point yet (#2763 obstacles 1 and 3). Resolve how it is injected in PR 2 before writing PR 3.
- Resolved: the hypervisor is the source of truth and a state file is only a name, never a liveness claim. ADR-055 decision 2's `busy` is a liveness question, and a leftover that outlived a crashed guest would make `up` say `busy` forever. `status` lists names with what the hypervisor answers (`limactl list` on Linux; on Windows the `running` check `vm/run.go` already uses to refuse a second run, shared by `up` and `status`, not written twice). `up` on a name whose guest is gone refuses with `stale`, exit 2, and names `down`; `down` on a stale name cleans the leftovers. Refuse and name, never silent cleanup: a crash that vanishes quietly is the absence this repo wants red. `stale` is an additive value in decision 7's status list, which by that decision's own rule does not bump `version`; the PR 1c description says so, and the ADR is not reopened.
- Resolved: AC7 needs something to key on. A raw stdin pipe gives dotf no value to redact, so `exec --secret <registry-id>` is the forwarding path (see What), and `run` uses it for `GITHUB_TOKEN`.
- Resolved (owner, 2026-10-30): `up` on a held, live Linux name stays `busy`, exit 2. It is not idempotent, so two callers that pick one name do not share a guest without knowing.
- Resolved: `--timeout` acts on both ends. Killing `limactl shell` leaves the guest command running and its ssh holding the pipes (measured on Lima 2.2.1, lesson 524), so the guest runs under `timeout -k 5 <limit>`, the host deadline starts first and decides the status, and `execLimactl` sets `WaitDelay`. Windows needs its own answer in PR 1c.
- Resolved: `--secret` takes one value per id (`ShowEntry`), so a file or multi-var secret is refused before anything is decrypted, and it is repeatable. A value shorter than redaction's 6-byte floor, or multi-line, is refused, because forwarding it would break AC7 in silence.
- Known residual: `limactl shell` returns 255 both for a command killed by a signal and for ssh losing the guest, so that case reads `ok, exit: 255`. The check before the shell rules out a stopped guest; the verb's help states the rest.
- Resolved: `cp` runs `limactl copy --backend=scp`. Lima's default rsync passes a guest path through a remote shell, which split `/tmp/a b` and wrote `/tmp/a`, and it copies a directory with no `-r` to nowhere with exit 0 (both measured on Lima 2.2.1). Over scp a path is literal and a directory is a loud failure, which is also the file-only contract Windows has.
- No open question blocks `tasks.md`.

## Acceptance criteria

- [ ] AC1: `up --session <name>` creates a session on Linux; on Windows a second `up` is refused with status `busy` and exit 2; `up` on a name whose guest is gone is refused with `stale`, exit 2, naming `down`; `status` reports what the hypervisor answers, never a state file alone.
- [ ] AC2: `exec -- <argv>` and `exec --script <file>` run in the guest without a shell and return one `--json` record with `version` 1, whatever the outcome; a guest command that exits 1 is `status: ok, exit: 1` and the process exits 1.
- [ ] AC3: `exec` forwards stdin on Linux and refuses it on Windows with `unsupported` and exit 2.
- [ ] AC4: `cp` moves a file host to guest and back on both guests through the channel; the isolation checks (no share device, no mount) stay green.
- [ ] AC5: With `--json`, stdout is exactly one record and the `==> ` progress lines are on stderr, for the verbs and for `run`.
- [ ] AC6: `run` is composed of the verbs; the assertions of the tests that pin its flags, lifecycle, two converges and judgment are unchanged (a test that passes `Out: &bytes.Buffer{}` may take a mechanical edit for the split writers).
- [ ] AC7: A guest command that prints a value `exec --secret <registry-id>` forwarded yields a masked record and masked plain stdout; piped stdin without `--secret` is not redacted.
- [ ] AC8: `exec --elevated` runs in a high-integrity token on the Windows guest and under `sudo` on Linux.
- [ ] AC9: Every refusal test sets `XDG_STATE_HOME` to an empty temp dir, so none reaches the hypervisor.
- [ ] AC11: `down` leaves no instance, overlay or state and never touches `clean-base`; `exec`, `cp` and `down` on an unknown name answer `no-session` with exit 3; a `timeout` exits 1.
- [ ] AC10: The Windows guest, with OpenSSH Server and the `ssh-target` role installed inside a session, produces an `sshd -T` dump that becomes a fixture, and the converge proof passes with a second run at `changed` 0 (closes #2763).

## References

- Bitácora board: mlorentedev/dotfiles#2842 (the `issue:` frontmatter field); #2763 is the consumer.
- Related ADR: `docs/adr/adr-055-vm-guests-are-driven-through-session-verbs.md`, extending ADR-053 and ADR-045; ADR-052 (no prose in argv), ADR-048 (allow rules), ADR-028 (redaction).
- Lessons: 473, 484, 489 (guest bootstrap, QGA, refusal tests that must not reach the hypervisor), 496.
