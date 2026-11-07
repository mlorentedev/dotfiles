---
id: "ADR-059-operator-controlled-userspace-bridge-lifecycle"
type: adr
status: accepted
owner: manu
date: "2026-11-06"
issue: "mlorentedev/dotfiles#3244"
extends: [adr-020-tooling-cli-go-convergence, adr-058-role-gated-userspace-bridge-client]
tags: [architecture, decision, ts-bridge, lifecycle, process, cross-platform]
created: "2026-11-06"
---

# ADR-059: Operator-controlled lifecycle for the userspace bridge

## Status

Accepted 2026-11-06.

## Context

ADR-058 installs the pinned ts-bridge client, owns a Gitea-only Git drop-in and
diagnoses binary, listener and destination health. It deliberately leaves the
process lifecycle operator-controlled.

That boundary is incomplete for daily use. A detached bridge ran successfully
for several hours, then its SSH bootstrap exited with status 255. The SOCKS
listener disappeared, Copilot App retained the proxy configuration and private
origin operations failed. Restoring the listener fixed Git, but a project added
while the origin was unreachable remained cached as a GitHub repository until
it was removed and added again.

The operator can reconstruct the full `ts-bridge connect` command, but every
other session cannot. Dotfiles needs a safe lifecycle contract without taking
ownership of profiles, credentials, tsnet state, system DNS or browser proxy
settings.

## Reference audit

The detailed matrix lives in the project prestudy as
`tool035-reference-audit-matrix.md`.

- `BWServeState` proves that a detached process needs durable PID and bounded
  log evidence.
- `selfDeploy` proves role-first inspect/reconcile/probe semantics, while also
  showing why an unattended scheduler is a separate decision.
- Hive doctor proves that health should consume the owning tool's stable status
  contract rather than duplicate supervisor logic.
- ts-bridge exposes profile locking, stable `READY` / `ERROR reason=` signals
  and health endpoints, but its current status exit semantics do not provide
  dotf's required ownership or destination verdict.

## Constraints

- Lifecycle is opt-in through `ts-bridge-client`.
- Start/stop/status are explicit operator commands; no login task or service is
  installed.
- Profiles, auth material and tsnet state remain ts-bridge-owned.
- PID existence alone is not ownership; stale/reused PID state fails closed.
- A second start is a no-op when healthy, and stop is a no-op when absent.
- Success is bounded and proves listener plus TLS-valid destination health.
- State and logs live in the per-user dotf state directory, never in the repo.
- Errors use `ERROR` / `WHY` / `FIX`.
- Implementation is cross-platform Go with OS-specific process seams.
- Git URL routing stays host-specific; system/browser proxy is out of scope.

## Options considered

### A. `dotf bridge start|stop|status` with owned state

Dotf records a non-secret launch descriptor, exact PID/process identity and a
bounded log per profile. It consumes ts-bridge's stable signals and the shared
health probe. Selected.

### B. Install an always-on service or Scheduled Task

Rejected. It changes login/startup, credential availability and crash-restart
policy. The owner selected operator-controlled lifecycle.

### C. Add PowerShell and shell wrappers

Rejected by ADR-020. Two process implementations would drift on detach,
signalling and PID reuse.

### D. Wait for ts-bridge to own service control

Rejected for the current incident. Reconsider if ts-bridge publishes stable
cross-platform lifecycle semantics with exact process identity and destination
health.

## Decision

### 1. Public command

Add:

```text
dotf bridge start --profile <name> [--config <path>] [--socks5 <addr>]
dotf bridge status [--profile <name>]
dotf bridge stop [--profile <name>]
```

The first successful start stores the safe launch descriptor. Later starts may
reuse it. When more than one profile is stored, omitting `--profile` fails and
names the available choices.

### 2. State and ownership

Each profile owns a directory below:

```text
<dotf state>/bridge/<profile>/
```

It contains:

- `launch.json`: profile name, config path, SOCKS address, executable path and
  argument fingerprint; never credential values;
- `process.json`: PID, process start identity and launch fingerprint;
- `bridge.log` plus one bounded rotation.

Start writes state atomically only after the child starts. Stop validates PID,
executable and process-start identity before signalling. Any mismatch is
`unowned`, never a process to kill.

### 3. Start contract

Start:

1. requires the machine role and a pinned ts-bridge on PATH;
2. resolves or records the safe launch descriptor;
3. reports healthy existing state as an idempotent no-op;
4. detaches the child with the existing cross-platform daemon pattern;
5. captures stdout/stderr to the bounded log;
6. waits for `READY`, `ERROR`, child exit or timeout;
7. proves the configured listener and TLS-valid Gitea destination before
   returning success.

### 4. Status contract

Status distinguishes:

- `stopped`: no owned live process;
- `starting`: owned process, listener not ready;
- `destination-failed`: listener ready, destination failed;
- `healthy`: process owned, listener ready, destination valid;
- `unowned`: PID/process identity mismatch or foreign listener.

Human output uses `ERROR` / `WHY` / `FIX` for non-healthy states. A JSON mode
uses the same fields plus structured context. Healthy exits 0; every other
state exits non-zero.

### 5. Stop contract

Stop is idempotent when absent. For an owned process it requests graceful
termination, waits a bounded interval, then uses the platform hard-stop only
for the same validated identity. It never kills by process name.

### 6. Rollout

- Land the shared health and strict state model first.
- Add exact Windows, Linux and macOS process identity as an isolated safety boundary.
- Add read-only `bridge status` only after ownership can be proven.
- Add detached `bridge start`, owned `bridge stop` and bounded logs last.

## Consequences

### Positive

- Any session can restore private-forge access without reconstructing a command.
- A crash leaves actionable PID/log evidence.
- PID reuse and foreign listeners fail closed.
- Doctor and lifecycle share one health interpretation.

### Negative

- Cross-platform process identity requires OS-specific code and tests.
- First use still needs a profile/config reference because dotfiles does not own
  profile provisioning.
- An operator must invoke start after login until a separate always-on decision.

### Neutral

- Browser access remains direct and may still fail by design.
- Copilot App origin-cache recovery remains remove/re-add; dotfiles never edits
  its internal database.

## References

- #3244
- [ADR-020](adr-020-tooling-cli-go-convergence.md)
- [ADR-058](adr-058-role-gated-userspace-bridge-client.md)
- Vault pattern `pattern-setup-script-idempotence`
- Vault pattern `pattern-agent-oriented-errors`
