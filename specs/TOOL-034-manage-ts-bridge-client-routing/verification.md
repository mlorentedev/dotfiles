---
tags: [spec, verification, templates]
created: "2026-11-05"
id: "TOOL-034-manage-ts-bridge-client-routing"
type: spec-verification
status: active
owner: manu
---

# Verification - TOOL-034-manage-ts-bridge-client-routing

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criteria 1-3 -> PR 1 generic archive source and rollout-safety tests
- [x] Criteria 4-6 -> PR 2 role/catalog/drop-in tests
- [ ] Criteria 7-8 -> PR 3 doctor and live consequence evidence

## Test status

- PR 1: focused archive/catalog tests -> PASS; full `go test ./internal/tools` -> PASS; `go vet ./internal/tools` -> PASS; changed-file `gofmt -l` -> clean.
- Independent code review: all three Medium findings applied.
  - Final placement now writes/syncs/chmods a sibling temp file and renames it over the destination.
  - Unsupported archive extensions are refused by both Plan and Install before download/extraction.
  - Catalog load requires asset/member resolution parity for every supported GOOS/GOARCH pair.
- PR 2: full `go test ./internal/env ./internal/tools ./internal/converge` -> PASS; command tool selectors -> PASS; f3/f4 exact feature commands -> PASS; focused vet -> PASS.
- PR 3 tests/live smoke: pending

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- ADR-058 selects a generic archive source rather than a ts-bridge-specific downloader.
- Tool role filtering is rollout-safe for ts-bridge because the same entry uses a source type older dotf binaries explicitly skip.
- Git routing uses an entirely dotf-owned include file, not writes into the user/gh-owned `~/.gitconfig`.
- PR 1 streams only the exact member into a temp file after the release checksum passes; no archive path is expanded onto disk.
- PR 2 filters role-bound tools for no-argument install/converge while an explicit named install remains an operator override.
- The proxy is a complete dotf-owned drop-in. A foreign file at that path is refused, not overwritten or removed.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? pending until implementation exposes a non-obvious lesson
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: `docs/adr/adr-058-role-gated-userspace-bridge-client.md`
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: existing setup-idempotence and per-machine-override patterns cover the reusable method.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-034-manage-ts-bridge-client-routing/` -> `specs/archive/TOOL-034-manage-ts-bridge-client-routing/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
