---
id: "CLI-024-secrets-set"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#612"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, cli, go, bitwarden]
template_version: "1.0"
---

# CLI-024-secrets-set

## Why

<!-- from issue #612: secrets lifecycle: fail-loud hardening + reproducible CLI write/migrate/rotate (audit backlog) -->

The age→bw migration (#612) must be fully CLI-driven — no Bitwarden GUI clicks. The
read seam (`BWReader`/`BWGet`), the write seam (`BWWriter`/`BWPut`, #620), and the
registry mutation (`SetBackendBW`, #617) all exist, but nothing wires them into a
command an operator (or `migrate`) can call to **put a value into a secret**. C3 is
that command: `dotf secrets set <id>` — the idempotent write primitive that C4
`migrate`, C7 `rotate`, and manual provisioning all compose.

## What

A new `dotf secrets set <id> [var]` subcommand that writes a value into the Bitwarden
item/field a registry secret maps to:

- **Value never on the command line.** Read from stdin when piped (`printf %s "$v" |
  dotf secrets set id`), else a hidden TTY prompt — so the value never lands in shell
  history or `ps`.
- **Idempotent.** It reads the current field value first; if it already equals the new
  value, it writes nothing and exits 0 (`unchanged`). C4's parity gate and re-runs
  depend on this.
- **All three shapes, one field per invocation.** A single-field secret (token, or a
  file/notes secret) needs no `[var]`; a multi-field secret requires `[var]` to pick
  which field. env targets store the value trailing-newline-trimmed (single-line
  token); file/notes targets store the bytes exactly (multi-line text — SSH/kubeconfig).
- **Create-absent, gated.** `BWPut.SetField` deliberately refuses to create (a locked
  vault must never be mistaken for a missing item and spawn a duplicate). `set` adds the
  create path: it fires **only** when bw reports the item is specifically *not found*
  (a locked/unauthenticated vault produces a different error and is surfaced fail-loud,
  never treated as absent), and then only after an interactive confirm or `--yes`.
- **`--dry-run`** reports the intended action (update / create / unchanged) without
  writing. **`--yes`** skips the create confirmation for non-interactive callers
  (`migrate`).

## Out of scope

- `dotf secrets migrate` (C4), `rotate` (C7) — they *compose* `set`; not built here.
- `--item`/`--field` overrides to target a bw location before the registry has a `bw:`
  block — deferred to C4 if `migrate`'s ordering needs it. C3 resolves item+field from
  the secret's existing `bw:` block.
- Writing the age backend, deleting/clearing a field, multi-value-in-one-shot.
- A live write against the real vault — that is the canary smoke (#612 C8), run with
  the operator's `bw unlock`, not in CI.

## Risks / open questions

- **Create-vs-locked discrimination.** Relies on matching bw's "Not found." message
  (`ErrBWItemNotFound`) — a locked/unauthenticated vault yields a different message and
  falls through to fail-loud. Documented as fragile-by-CLI; `bw serve` would give proper
  status codes behind the same seam later. Belt-and-braces: create also needs confirm
  or `--yes`.
- **stdin is consumed by the value** in the piped case, so an interactive create-confirm
  is impossible there → non-interactive create requires `--yes` (error otherwise). The
  TTY path reads value (hidden) then confirm as separate reads.
- **New dependency `golang.org/x/term`** for cross-platform hidden input (Windows
  included). The battle-tested choice over hand-rolled termios/Console syscalls
  (Decision Hierarchy #3); one golang.org/x package.
- **Empty value refused** (consistent with `EnvFor`/`verify` fail-loud #612 A1) — a
  blank secret is a bug, not a clear.
- **Two extra `bw get item` shell-outs** on the update path (idempotency read + the RMW
  inside `SetField`); acceptable for a non-hot command, collapsible under `bw serve`.

## Acceptance criteria

- [ ] **AC1 — Idempotent no-op.** When the stored field already equals the new value,
  `set` calls no writer method and exits 0 reporting `unchanged`. *Verify:* Go test,
  fake reader returns the same value → fake writer records zero writes.
- [ ] **AC2 — Write on change, per-shape normalization.** A differing value triggers
  exactly one `SetField(item, field, value)`; env targets are stored
  trailing-newline-trimmed, file targets byte-exact. *Verify:* Go test asserting the
  recorded value for an env and a file secret.
- [ ] **AC3 — Field disambiguation.** A multi-var secret with no `[var]` errors listing
  its vars; with `[var]` it writes the matching field. A single-var secret needs no arg.
  *Verify:* Go test.
- [ ] **AC4 — Create-absent gated.** `ErrBWItemNotFound` + `--yes` → exactly one
  `CreateItem(item, field, value)`; not-found without `--yes` (non-interactive) errors
  and creates nothing; a non-not-found read error (locked) is returned and creates
  nothing. *Verify:* Go test with the sentinels.
- [ ] **AC5 — Empty refused + dry-run inert.** An empty resolved value errors with no
  write; `--dry-run` performs no write on update/create/unchanged and exits 0.
  *Verify:* Go test.
- [ ] **AC6 — Clean + additive.** `go test ./internal/... && go vet ./... && go build
  ./... && gofmt -l` clean; `registry.yaml` and existing command behavior untouched.
  *Verify:* commands.

## References

- Bitácora board: `mlorentedev/dotfiles#612` (Phase C, C3).
- Composes / mirrors: `cli/internal/secrets/bw.go` (`BWPut`/`BWGet`/`setItemField`),
  `cli/internal/secrets/registry_write.go` (`SetBackendBW`, #617),
  `cli/internal/cmd/secrets.go` (`verify` command + the `bwReader` injection pattern).
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`.
