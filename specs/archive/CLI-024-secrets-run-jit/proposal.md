---
id: "CLI-024-secrets-run-jit"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-25"
issue: "dotfiles#493"
tags: [spec, proposal, secrets, age, jit, dotf]
template_version: "1.0"
---

# CLI-024-secrets-run-jit

> Phase 1a of ADR-028: the JIT secrets primitive. Adds `dotf secrets run -- <cmd>` over the existing age store, with no migration. Pure addition — does NOT yet remove the ambient export (that is 1b, blocked on consumer migration; see `## Sequencing`).

## Why

ADR-028's primary objective is **"not always exposed"**: secrets must be available on demand to the process that needs them, not exported into every interactive shell at login. Today `load-secrets.{sh,ps1}` is sourced by `.bashrc`/`.zshrc`/`profile.ps1` and exports ~30 decrypted secrets into the ambient environment of every shell — the exposure this redesign exists to remove.

The on-demand replacement is a single primitive: decrypt the mapped secrets and inject them into **one child process only**, then exit. Nothing lands in the parent shell.

## What

`dotf secrets run [--only VAR[,VAR...]] -- <cmd> [args...]`:

1. Reads `sensitive/env-mapping.conf` (the existing consumer map — over age **as-is**, no migration).
2. Decrypts each selected secret by shelling out to `age --decrypt --identity <key> <file>` (the same tool + key load-secrets uses — boring, already provisioned by Phase 0; no new Go crypto dependency).
3. Builds the child environment = the parent env **plus** the decrypted `KEY=VALUE` pairs; file secrets (`@VAR=file>dest`) are materialized to their `dest` (`0600`) and `VAR` is set to that path — parity with load-secrets' file-secret behaviour.
4. `exec`s the child with that environment and inherited stdio, and propagates its exit code.

`--only` scopes the injection to named vars (smaller blast radius; the default is the full mapped set, i.e. parity with what load-secrets exports today).

The secrets live only in the child's `cmd.Env` (process memory), never in the parent shell — the security property the ambient export violated.

## Out of scope

- **Removing the ambient `load-secrets` sourcing** — Phase 1b (`## Sequencing`). This PR is purely additive and changes no startup behaviour.
- **The registry / `bw` backend / `--only <id>`** — Phase 2 (#378). This primitive resolves by env-var name from env-mapping.conf, not the future registry id.
- **Deleting the `load-secrets.{sh,ps1}` twins** — the later full-convergence step of #493.
- **`sync` (materialize for headless consumers)** — Phase 2.

## Sequencing — the consumer blast radius (why 1a/1b split)

`powershell/profile.ps1` documents that the ambient export is **"mandatory for opencode (reads `{env:NAN_API_KEY}` from opencode.jsonc) and agy (reads `ANTHROPIC_API_KEY` etc. from environment)."** opencode and agy read their keys from the ambient env at their *own* startup, not via `dotf secrets run`. So removing the sourcing (1b) must first migrate those consumers (launch them via `dotf secrets run`, or feed their keys another way) — consumer-mapped, never big-bang (ADR-028). 1a ships the primitive that 1b will route those consumers through.

## Risks / open questions

- **R1 — secret values must never leak to logs/argv.** *Mitigation:* values go only into `cmd.Env`; they are never printed, never passed as argv, and decryption pipes `age` stdout straight into memory (no temp plaintext for env secrets). File secrets reuse load-secrets' decrypt-to-`0600` path.
- **R2 — child exit-code fidelity.** A wrapper that swallows the child's exit code breaks `&&`/CI. *Mitigation:* propagate `exec.ExitError.ExitCode()`; tested.
- **R3 — missing age key / `age` absent.** *Mitigation:* a clear error (Phase 0's `dotf doctor` check surfaces the root cause); `run` fails fast rather than launching the child with missing secrets — *unless* `--only` selected none, in which case it just runs the child with the parent env.
- **R4 — Windows `exec` semantics.** Go has no `execve`; the child is a subprocess whose exit code is propagated. Acceptable (parity with a wrapper).

## Acceptance criteria

- [ ] **AC1** — `dotf secrets run -- printenv FOO` (env secret FOO mapped) prints the decrypted value; the value is NOT present in the parent shell after the command returns. *Verify:* unit test on the env-builder + a smoke.
- [ ] **AC2** — `--only A,B` injects exactly A and B; unmapped/unselected vars are absent. *Verify:* table-driven `go test` with a fake decryptor.
- [ ] **AC3** — file secrets (`@VAR=file>dest`) are written to `dest` (`0600`) and `VAR` points at the path. *Verify:* `go test` materializing to a temp dest.
- [ ] **AC4** — the child's exit code is propagated (non-zero stays non-zero). *Verify:* `go test` running a child that exits 3.
- [ ] **AC5** — mapping parser handles `VAR=file`, `@VAR=file>dest`, comments and blanks; bad lines are skipped, not fatal. *Verify:* `go test`.
- [ ] **AC6** — no secret value is ever written to stdout/stderr by `dotf secrets run` itself. *Verify:* code review + a test asserting the command's own output is empty on success.

## References

- Issue: dotfiles#493 (CLI-024; reconciled to ADR-028 Phase 1a)
- ADR: adr-028 (two-tier model), adr-002 (age)
- Code: `scripts/load-secrets.sh` (the behaviour being ported), `sensitive/env-mapping.conf` (the map)
- Related: #578 / OPS-017 (Phase 0 — provisions age/bw + doctor check), #378 (registry/bw backend, Phase 2)
