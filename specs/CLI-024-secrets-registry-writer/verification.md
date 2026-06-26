---
tags: [spec, verification, secrets, cli, go]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-registry-writer

## Evidence

- [x] **AC1** (flip + byte-for-byte preservation) — PASS:
  `TestSetBackendBW_FlipsOnlyTargetBlock` (aligned trailing comment, blank line, and
  the whole sibling block survive verbatim) and `TestSetBackendBW_RealRegistry_OnlyTargetChanges`
  (golden against the real `registry.yaml`: net-zero lines ⇒ every line outside the
  `github-bitacora-pat` block is byte-identical; the block flips correctly).
- [x] **AC2** (idempotent) — PASS: `TestSetBackendBW_Idempotent`.
- [x] **AC3** (guards + re-validation) — PASS: `TestSetBackendBW_Guards`
  (unknown id / multi-var / file / empty item / empty field), and `ParseRegistry`
  re-validation inside `SetBackendBW`.
- [x] **AC4** — see below.

## Test status

- `go test ./internal/secrets/ -count=1` → **ok**. Targeted `-run SetBackendBW`: 4
  tests / 5 guard subtests PASS.
- `go vet ./internal/secrets/` → clean. `go build ./...` → clean. `gofmt -l` on staged
  (LF) blobs → clean.
- Additive only: a new file `registry_write.go` + its test; nothing else changed, no
  command wired (the primitive is consumed by `migrate` later).

## Decisions made during implementation

- **yaml.v3 Node round-trip rejected by a probe.** Re-encoding the parsed
  `registry.yaml` collapsed 24 blank lines (217→193) and aligned trailing comments
  (`id: x            # …` → `id: x # …`) and flow spacing (`{ env: … }` → `{env:…}`).
  So the writer is line surgery, not a marshal — chosen *because* the data said so.
- **Net-zero-lines transformation** (backend value in place, −age, +bw) makes the
  golden assertion exact: line indices outside the block are stable, so a direct
  index-by-index comparison proves nothing else moved.
- **Refuse any shape but single scalar env** before editing — a multi-var secret given
  one top-level `bw.field` would *validate* but silently collapse all vars onto one
  field; fail-fast avoids that class entirely. Multi-field is the deferred path.
- **Re-validate via `ParseRegistry`** after the edit so a malformed result can never be
  written.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **Maybe** — "probe round-trip fidelity before
  choosing a Node vs text rewriter for a human-maintained config" is a reusable
  gotcha; capture if it recurs. Not promoting now.
- [ ] ADR-worthy? **no** — implements #612 / ADR-028.
- [ ] New cross-project pattern? **no** — repo-specific for now.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-024-secrets-registry-writer/`
- [ ] #612 Phase C item C2 (registry mutation half) checked off; PR linked
- [ ] Promotions above executed (if any)
