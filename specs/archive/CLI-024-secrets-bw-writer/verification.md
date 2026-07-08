---
tags: [spec, verification, secrets, cli, go, bitwarden]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-bw-writer

## Evidence

- [x] **AC1** (`setItemField` sets each shape, preserves siblings + other keys) — PASS:
  `TestSetItemField_TypedLogin_PreservesSiblings` (password set; username/notes/api-key/
  client-id untouched; id/name/type preserved), `_CreatesLoginIfAbsent`, `_Notes`,
  `_CustomFieldUpdate`, `_CustomFieldAppend`.
- [x] **AC2** (bad JSON errors; mock round-trips) — PASS: `TestSetItemField_BadJSON`,
  `TestItemID`, `TestBWWriter_MockRoundTrip`.
- [x] **AC3** (`BWPut` wired, update path, absent→clear error) — compiles + `go build`;
  shell orchestration (`bw get`→setItemField→base64→`bw edit`) verified by live smoke
  with the operator's unlocked vault (deferred, documented — same as `BWGet`).
- [x] **AC4** — see below.

## Test status

- `go test ./internal/secrets/ -count=1` → **ok**. Targeted `setItemField`/`ItemID`/
  `BWWriter`: 8 tests PASS.
- `go vet ./...` → clean. `go build ./...` → clean. `gofmt -l` on staged (LF) blobs →
  clean.
- Additive only: extended `bw.go` (the read seam's file) with the write seam + a new
  test file; nothing else changed; no command wired; `registry.yaml` untouched.

## Decisions made during implementation

- **`setItemField` works on `map[string]any`, not the narrow `bwItem` struct** — so the
  item's other keys (id, type, name, folderId, reprompt, …) and sibling fields survive
  the read-modify-write. A struct round-trip would silently drop unknown keys; tested
  that `id`/`name`/`type` + every sibling field are unchanged.
- **Update-only, no accidental create.** `SetField` edits an existing item and errors
  clearly when it is absent — distinguishing "not found" from "vault locked" needs care,
  and creating on the wrong signal would spawn a duplicate. The create path lands with
  `dotf secrets set` (C3), which can prompt/confirm.
- **No plaintext to disk.** The modified item reaches `bw` as base64 over **stdin**,
  never a temp file.
- **base64 in Go, not `bw encode`** — one fewer shell hop; the payload is just base64.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **no** (the read-modify-write / preserve-unknown-keys
  point is a standard JSON-patch practice).
- [ ] ADR-worthy? **no** — implements #612 / ADR-028.
- [ ] New cross-project pattern? **no**.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-024-secrets-bw-writer/`
- [ ] #612 Phase C item C2 (write half) checked off; PR linked
- [ ] Promotions above executed (if any)
