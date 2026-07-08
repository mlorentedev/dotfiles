---
id: "CLI-024-secrets-migrate"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#612"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, cli, go, bitwarden, migration]
template_version: "1.0"
---

# CLI-024-secrets-migrate

## Why

<!-- from issue #612: secrets lifecycle: fail-loud hardening + reproducible CLI write/migrate/rotate (audit backlog) -->

The ~20 dev secrets must move from the age store to Bitwarden (ADR-028 Phase 3 / #585)
**reproducibly, idempotently, and fully CLI-driven** — no manual copy-paste between
`age -d` and the Bitwarden GUI, and no chance of a botched write silently replacing a
working secret. `migrate` is the orchestrator that composes the primitives already
built: `set` (C3, the bw write), `SetBackendBW` (#617, the registry flip), and the
resolver/`verify` (#616). It is the step that actually performs the age→bw cutover for
one secret, behind a parity gate.

## What

A new `dotf secrets migrate <id> [--yes] [--dry-run]` that cuts a single secret over
from age to bw (the bw target is read from the entry's pre-declared `bw:` block, not
passed as flags):

1. **Resolve + scope-guard.** Look up the secret (by its env-var `id`). Under the
   env-var-per-entry model every entry is single-var, so `SetBackendBW` accepts them all
   — the former multi-field deferral (M3/M6) dissolves. `migrate` proceeds only when ALL
   hold, else a clear, specific error and no writes:
   - `backend` is `age`/`age-offline` (already `bw` → the idempotent short-circuit, §2);
   - the expose is a single **env** var, not a **file** — file secrets are deferred until
     `SetBackendBW` grows `expose.file` support (a follow-up; it rejects files today);
   - a `bw:` block resolves an item+field (else "declare the bw: target first");
   - the `age:` source is **not shared** with another entry — a shared source signals a
     per-purpose split (github-cli-pat / github-release-pat both read `github.token`):
     migrating it plainly would copy the SAME token to both, so refuse and defer to C9
     `migrate --split` (issues distinct tokens);
   - no `ci:*` consumer — `ls --pairs` excludes bw, so migrating a CI secret would
     silently drop it from `github-secrets-manager.sh`; refuse until C5 `sync` lands.
2. **Idempotent short-circuit.** If the secret is already `backend: bw`, re-verify it
   (resolves to a non-empty value) and exit 0 (`already bw (verified)`) — no writes.
3. **Read age** (the current source of truth). Absent age file → error (nothing to
   migrate). This is the value the cutover must preserve.
4. **Write bw.** Reuse C3's write core to put the value into the Bitwarden item/field
   the registry entry **declares** in its `bw:` block — the SSOT for the mapping. The
   A1 rewrite (DECIDED 2026-06-26) populates `bw: {item, field}` on every entry while it
   is still `backend: age`, so `migrate` reads the target and never guesses; a missing
   `bw:` block is an error. `--item`/`--field` exist only as a one-off override.
   Create-absent allowed with `--yes`, idempotent.
5. **Parity gate (the safety interlock).** Re-read the value back from bw and compare to
   the age value **under the run-time transform** (env: newline-stripped). On mismatch:
   **abort before touching the registry**, reporting a non-leaking diff (e.g. lengths).
6. **Flip the registry — last.** Only after parity passes, `FlipRegistryToBW(registry,
   id)` rewrites `secrets/registry.yaml` atomically (`backend: bw`, declared `bw:` kept,
   age source dropped — comment-preserving, #617).
7. **Final verify.** Confirm the secret now resolves via bw (#616). Report success. The
   `.secret.age` file is **kept** (retiring it is C6 `retire`, deliberately separate).

`--dry-run` does steps 3–4-preview only: reports the intended write + flip, performs no
bw write and no registry change.

## Out of scope (all enforced by the §1 guards)

- **File secrets** (`expose.file`: KUBECONFIG, the backup-code files) — `SetBackendBW`
  rewrites env exposes only; a file variant is a follow-up.
- **`ci:*`-consumer secrets** — gated on C5 `sync` (#612), which must make `ls --pairs`
  backend-agnostic before any CI secret leaves age.
- **Shared-age-source / per-purpose split** (`github.token` → two PATs) — C9
  `migrate --split` issues distinct tokens; plain `migrate` refuses it.
- `retire` (removing the `.secret.age` after escrow) — #612 C6.
- Bulk `migrate --all` — one id per invocation here; a loop wrapper is a follow-up.
- (No multi-var path: the env-var-per-entry model made every entry single-var.)

## Risks / open questions

- **Ordering is the correctness crux.** Write+parity happen *before* the registry flip;
  the flip is the last mutation. Until it lands the registry still points at age, so any
  earlier failure leaves the secret fully working on age (safe rollback by construction).
  The age file is retained, so even post-flip a revert is one registry edit. This is what
  makes `migrate` safe to run and re-run.
- **bw target naming (DECIDED 2026-06-26 — model A1).** item = service/account, field =
  named purpose (`api-key`, `bearer-token`; `notes` for multi-line) — the 1Password
  "API Credential" / Vault grouped-KV model. The registry's `bw:` block is the **SSOT**
  for the mapping; `migrate`/`set` read it, never derive a convention from the id. The
  registry rewrite declares `bw:` on every entry up front (even age-backed ones), so
  `SetBackendBW` at flip time just keeps the already-declared target in place + drops the
  age source. Per-purpose split (A2) is applied **selectively** where a credential rotates or
  revokes independently (the github 8-in-1 split, C9). Convention to be encoded in the
  rewrite and surfaced in ADR-028 / `docs/secrets-inventory.md`.
- **Parity is functional, not raw-byte.** Defined as equality of the *resolved* value
  (env: `stripNewlines`; file: exact) — i.e. "the consumer process sees the same value
  after the cutover", which is the property that actually matters. A trailing-newline
  difference between the age plaintext and the bw store is therefore not a failure.
- **bw must be unlocked.** Locked / unauthenticated → fail loud early (the existing
  reader/writer errors); no partial state. (Ergonomics tracked by #622, `bw serve`.)
- **Depends on C3** (`set`'s write core, #621 — merged) and the registry `bw:` targets
  (#624). The migrate code only needs C3 (on main); the live cutover needs #624's targets.

## Acceptance criteria

- [ ] **AC1 — age secret migrates end-to-end.** Given an age single-scalar-env secret,
  `migrate` writes the value to bw, parity passes, and the registry entry is rewritten to
  `backend: bw` with the right `item`/`field`; the `.secret.age` file is untouched.
  *Verify:* Go test with a fake age resolver + fake bw writer/reader; assert the bw write,
  the post-edit registry bytes (via `ParseRegistry` → `backend == bw`, item/field set),
  and that no age file was removed.
- [ ] **AC2 — parity mismatch aborts before the registry changes.** If the value read
  back from bw differs from the age value, `migrate` errors and the registry is left
  unchanged (still `backend: age`). *Verify:* Go test (fake bw returns a different value).
- [ ] **AC3 — idempotent.** A secret already `backend: bw` → re-verify, exit 0, zero
  writes, registry byte-identical. *Verify:* Go test.
- [ ] **AC4 — scope guards.** Each of: a file secret, a `ci:*`-consumer secret, an entry
  whose `age:` source is shared with another (split-pending), and a missing `bw:` block →
  a clear, specific error with no bw write and no registry change. *Verify:* Go test per
  guard.
- [ ] **AC5 — `--dry-run` inert.** Reports the intended write + flip; performs no bw
  write and no registry change. *Verify:* Go test.
- [ ] **AC6 — clean + additive.** `go test ./... && go vet && golangci-lint (v2)` clean;
  only the new command + a write-core refactor; existing behaviour unchanged.

## References

- Bitácora board: `mlorentedev/dotfiles#612` (Phase C, C4).
- Composes: C3 `set` (#621, the bw write core), `SetBackendBW` (#617, registry flip),
  `Loader.Verify` / age resolver (#616).
- Sequencing: C5 `sync` (#612) must precede any `ci:*` migration; C6 `retire` removes the
  age file after escrow. Unlock ergonomics: #622 (`bw serve`).
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`.
