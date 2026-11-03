---
tags: [spec, verification, templates]
created: "2026-11-01"
---

# Verification - HARNESS-185-handoff-loss-check

## Evidence

All tests are in `cli/internal/doctor/checks_handoff_threads_test.go`; the check is `cli/internal/doctor/checks_handoff_threads.go`.

- [x] AC1 -> `TestHandoffThreads_WarnsOnALostEntry`
- [x] AC2 -> `TestHandoffThreads_EntryOrForkPasses`
- [x] AC3 -> `TestHandoffThreads_ArchivedThreadIsQuiet`
- [x] AC4 -> `TestHandoffThreads_JournalAloneIsNotProof`
- [x] AC5 -> `TestHandoffThreads_Skips` (no vault, no vault project, no MEMORY.md, no threads/)
- [x] AC6 -> `TestHandoffThreads_NeverWritesNeverFails` (vault tree byte-identical, zero failures); and a real `dotf doctor --fix` against the real vault left `10_projects/dotfiles/memory` byte-identical (sha256 of every file before and after).
- [x] AC7 -> `TestHandoffThreads_KeysLikeTheWriter` (a `fix/deep/x` branch and a detached worktree resolve to whatever `mem.ThreadKey` returns; the file name comes from `mem.ThreadFileName`)
- [x] AC8 -> the real vault run below.

Each test was mutation-checked: an always-found entry fails AC1 and AC7; dropping the fork match fails AC2; keying by directory name fails AC1 and AC7; reporting a worktree without a thread file fails AC2, AC3 and AC4.

## Test status

- `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./...` -> clean.
- `golangci-lint run ./internal/doctor/... ./internal/mem/... ./internal/memlink/...` (v2.12.2, the pin) -> 0 issues.
- `go test ./... -count=1` -> every package ok except `internal/initrepo` and `internal/vault`, both `TestEmbeddedTemplatesMatchVault`: the vault's `agents-spec-section.md` and `work-sdk-memory.md` changed after the last vendoring. This is unrelated to the change and already ticketed as #3062.
- Real vault (`dotf doctor` on 2026-11-02, this Mac): the "Handoff threads" section passes. Four `threads/*.md` files have no entry: `chore-ci-workflows-protection`, `ci-workflows-scaffold`, `feat-dotf-pr-gate`, `feat-dotf-pr-guard`. None belongs to a live worktree (`git worktree list` names none of them), so they are finished threads whose file was left behind, the case the second remedy covers. On 2026-11-01 the count was seven, one of them live (`feat-worktree-one-root`, whose worktree still exists); its entry is back in `MEMORY.md` now (`### thread: feat-worktree-one-root`), so it no longer warns. That is the case the check exists for.

## Decisions made during implementation

- The check covers every non-bare worktree including the main checkout, not only linked ones: a lost entry for `main@<host>` is the same loss, and the key comes from the same function.
- The project's vault directory is resolved from the main checkout's directory name (first entry of `git worktree list`), not from `start`: a linked worktree is named after its branch and would resolve to no project. It reuses `memlink`'s resolver through a new exported `memlink.VaultMemory`; `mem.ThreadsDir` and `mem.ThreadFileName` are exported so the file name cannot drift from the writer's.
- A fork entry is any heading starting `<key>+`, which is how `handoff-write` names another agent's block.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: nothing surprised; the design decisions are in this file and the spec.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the check follows ADR-056's store and adds no new decision.
- [x] New pattern candidate for `00_meta/patterns/`? no: used in one project only.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-185-handoff-loss-check/` -> `specs/archive/HARNESS-185-handoff-loss-check/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
