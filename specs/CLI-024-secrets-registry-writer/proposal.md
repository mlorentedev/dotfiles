---
id: "CLI-024-secrets-registry-writer"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#612"
tags: [spec, proposal, secrets, cli, go]
template_version: "1.0"
---

# CLI-024-secrets-registry-writer

## Why

The age→bw migration (#612) must be **idempotent and CLI-driven** — no hand-edited
YAML. The one registry mutation it needs is "flip secret X to the bw backend": set
`backend: bw`, add `bw: { item, field }`, drop `age:`. Today that is a manual edit of
`registry.yaml`, which is error-prone and irreproducible. This is the programmatic
mutation primitive `dotf secrets migrate` (#612 C4) will call. It must preserve the
file's comments, blank-line structure, and alignment exactly — `registry.yaml` is a
human-maintained SSOT with section headers and aligned trailing comments.

## What

`SetBackendBW(data []byte, id string) ([]byte, error)` **activates** secret `id`'s
pre-declared Bitwarden backend and returns the new bytes:

- flips its `backend:` value to `bw`,
- drops its now-dead `age:` source line,
- **keeps the already-declared `bw: { item, field }` block byte-for-byte in place** —
  it neither computes nor relocates the target,
- **touches only those two lines** — every comment, blank line, alignment space, and
  other secret is byte-for-byte identical,
- is **idempotent** (already-bw, no age line → input unchanged),
- and **re-validates** the result through `ParseRegistry` before returning.

**Why no `item`/`field` parameters (model shift since the #617 draft):** the
env-var-per-entry registry (ADR-028 §2 addendum) pre-declares `bw:` on every migratable
entry, dormant while `backend: age`. The declared block is therefore the SINGLE source
for both `migrate`'s parity write and the post-flip resolution — passing item/field to
the writer would be a second, divergeable copy of that SSOT. So the writer only toggles
the backend and drops age; it reads the target it needs from the entry itself. This pins
the value parity verified and the value resolved after the cutover to the same
item/field, with no chance of drift.

It is scoped to the single, scalar env-var shape (`expose: { env: VAR }`) that the
bulk of the registry uses, in block form. The entry MUST carry a `bw:` target to
activate (else fail-loud — nothing to flip to). Multi-var / per-var (dockerhub,
x-twitter) and file secrets are rejected with a clear error (the multi-field path is
#612 M3/M6).

**Approach decision (empirically driven):** a yaml.v3 Node round-trip was probed
against the real `registry.yaml` and **rejected** — re-encoding the parsed document
collapsed 24 blank lines and the alignment padding of trailing comments. So the
implementation is deliberate line surgery, guarded by `ParseRegistry` re-validation.

## Out of scope

- The `BWWriter` (writing the value into Bitwarden) — the sibling PR (#612 C2 part 2).
- The `migrate` command that composes read-age + write-bw + this mutation — #612 C4.
- Multi-var / per-var / file migration, the github 1→2 split — #612 M3/M6.
- `UpsertEntry`/`RemoveEntry` (add/remove a whole secret) — only `SetBackendBW` is
  needed for the migration; add the others when a command needs them.

## Risks / open questions

- **Fidelity.** The golden test flips a real-registry secret and asserts every line
  *outside* the target block is byte-identical (the transformation drops exactly one
  line — the dead `age:` — so the tail shifts up by one; the test asserts content
  preservation before/after the block, not stable indices) — the strongest guarantee
  that nothing else drifted.
- **Block detection.** Targets block form (`- id: x` then indented keys); an
  inline-mapping secret is reported as "not found in block form" (the real registry is
  all block form). Documented.
- **Wrong-shape safety.** A multi-var secret flipped with one top-level `bw.field`
  would validate but silently point every var at one field — so the writer refuses any
  shape other than single scalar env, fail-fast, before editing.

## Acceptance criteria

- [ ] **AC1** — `SetBackendBW` flips backend→bw, drops age, keeps the pre-declared
  `bw:` block in place, leaving comments/blanks/other-secrets byte-identical (one line
  shorter overall). *Verify:* unit test + golden against the real `registry.yaml`.
- [ ] **AC2** — idempotent (second apply == first; already-bw → unchanged).
  *Verify:* Go test.
- [ ] **AC3** — guarded: unknown id, multi-var, file secret, and **no declared bw:
  target** each error fail-fast; the result re-validates via `ParseRegistry`.
  *Verify:* Go test.
- [ ] **AC4** — `go test ./internal/secrets && go vet && gofmt && go build` clean; no
  command wired (primitive only). *Verify:* CI.

## References

- Issue / backlog: `mlorentedev/dotfiles#612` (Phase C, C2 — registry mutation half).
- Reuse: `cli/internal/secrets/registry.go` (`ParseRegistry`, `Lookup`, the schema).
- Next: `BWWriter` (write the value), then `migrate` (#612 C4) composes both.
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`.
