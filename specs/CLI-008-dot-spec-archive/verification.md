---
tags: [spec, verification, templates]
created: "2026-06-13"
---

# Verification - CLI-008-dot-spec-archive

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (test name + the end-to-end parity smoke).

- [x] Move → `specs/archive/ID/`, status `archived` → `TestArchiveMovesAndSetsStatus`, `TestSpecArchiveHappyPath`
- [x] `--abandoned` → `specs/archive/_abandoned/ID/`, status `abandoned` → `TestArchiveAbandonedRoute`, `TestSpecArchiveAbandonedAndPR`
- [x] Unresolved `[AGENT-*]` tags block (no move, lists file:line); `--force-with-drafts` overrides → `TestFindUnresolvedTags`, `TestArchiveBlocksOnDrafts`, `TestArchiveForceWithDrafts`, `TestSpecArchiveBlocksOnDrafts`
- [x] Missing `specs/ID/` → error, no move → `TestArchiveMissingSpecFails`, `TestSpecArchiveMissingSpecFails`
- [x] Target already in archive → no-clobber, source left in place → `TestArchiveNoClobber`
- [x] `--pr <url>` appends provenance comment → `TestArchiveRecordsPRURL`, `TestSpecArchiveAbandonedAndPR`
- [x] `status:` rewrite scoped to first frontmatter block; body decoy + meaningful comment preserved → `TestSetStatus`, `TestSetStatusOnlyFirstBlock`
- [x] `gofmt -l` empty, `go vet`, `go test`, `go build` all green → see Test status below

## Test status

- Test suite: `cd cli && go test ./...` → `ok cmd/dot`, `ok internal/cmd`, `ok internal/spec` (18 new tests, all PASS); `gofmt -l .` empty; `go vet ./...` clean; `go build ./...` ok.
- **End-to-end parity smoke vs `archive-spec.sh`:** built `dot`, archived an identical throwaway spec with both the Go binary and the shell. Output structure identical; the **only** diff is the deliberate improvement — the Go twin preserves the trailing frontmatter comment (`status: archived # draft | ...`), the shell awk drops it. The body decoy `status: pending …` was left untouched by both; the `<!-- archived <date> — PR: <url> -->` comment was appended correctly.
- No regressions: the full pre-existing suite (init twin, review, root) stays green.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Frontmatter rewrite = union of both shells, not a faithful copy of either.** Repo evidence (`specs/archive/_abandoned/DX-002-.../proposal.md` carries `status: abandoned # superseded by ADR-020 …`) proved the `.sh` awk's whole-line replace would destroy meaningful comments. Ported the `.sh` *scoping* (first frontmatter block only) + the `.ps1` *value-only* replace (comment preserved). This is the lone intentional behaviour difference from `archive-spec.sh`.
- **Feature-id validation (reversed after review).** Initially `archive` skipped grammar validation "for faithfulness to the `.sh`". CodeRabbit (#362) flagged that the raw id flows into `filepath.Join`, so `../../foo` could move a dir outside `specs/`. Reversed: `Archive` now calls `ValidateID(id)` first — the grammar bars `/`, `\`, `..`, so it doubles as the traversal guard. New `TestArchiveRejectsTraversalID` locks the bug-class shut (incident → guard).
- **Exit-code fidelity dropped on purpose.** Both shells `exit 4` on the tag block; no caller branches on it. `dot spec archive` returns a plain Cobra error (exit 1) like `dot spec init`, keeping the Go twin family consistent. Trivially reversible.
- **`os.Rename` (not copy+remove).** `specs/<id>` → `specs/archive/<id>` is always same-tree, so rename is atomic and correct; no cross-filesystem fallback needed.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-008-dot-spec-archive/` -> `specs/archive/CLI-008-dot-spec-archive/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
