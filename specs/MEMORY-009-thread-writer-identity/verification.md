---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - MEMORY-009-thread-writer-identity

## Evidence

- [x] **AC1** -> commit `f6dc6f6`, tests `TestAnotherAgentsStampedBlockIsForkedNotReplaced` and `TestAnUnstampedBlockIsAttributedByItsJournalLine` (feature f1). Journal forms covered: the project's own `sessions/`, another project's tree, a thread-suffixed name, a markdown link.
- [x] **AC2** -> commit `f6dc6f6`, test `TestMemHandoffWriteAnnouncesAFork` (f2): stderr names `"master@msi"`, pi, claude and `"master@msi+claude"`; stdout names the thread actually written.
- [x] **AC3** -> commit `f6dc6f6`, tests `TestTheSameAgentRewritesItsOwnBlockInPlace` (its own stamp, and an unstamped block whose journal names the writer) and `TestABlockWithNoKnownWriterIsReplacedAndStamped` (f3).
- [x] **AC4** -> commit `f6dc6f6`, test `TestAWriteWithNoAgentIsUnchanged` (f4): four documents and three keys, compared with `WriteThread`; `TestMemHandoffWriteWithoutAgentSaysWhatItAlwaysSaid` pins the command's words.
- [ ] **AC5** -> slice 2, after a release carrying `--agent` is installed (see proposal, Risks).

## Test status

- `go build ./... && go vet ./... && go test ./... -count=1` in `cli/`: every package ok.
- `golangci-lint run ./internal/mem/... ./internal/cmd/...`: 0 issues.
- **Mutation checks**, each reverted:

  | Mutation | Tests that went red |
  |---|---|
  | never fork | the two AC1 tests |
  | no journal attribution | `TestAnUnstampedBlockIsAttributedByItsJournalLine` |
  | no stamp | four tests |
  | no fork notice | `TestMemHandoffWriteAnnouncesAFork` |

- **Live dry run against the vault**, the failure the issue measured:
  - Setup: from the knowledge checkout, `printf 'x\n' | dotf mem handoff-write --memory 10_projects/knowledge/memory/MEMORY.md --dry-run`, using a binary built at `f6dc6f6`.
  - With `--agent claude`, stderr reads `forked thread "master@msi" is antigravity's, so this claude handoff went to "master@msi+claude"`. antigravity's block is byte-identical, compared with `diff`.
  - Without `--agent`, the block is gone, as before this change (0 occurrences of its text).
- No regressions in the existing suite: yes.

## Decisions made during implementation

- `WriteThreadAs` is a new entry point, and `WriteThread` is unchanged in signature and output. So AC4 holds by construction and is also pinned by a test.
- The fork notice prints whenever the write lands in the fork, even when the fork's content is unchanged. It is the one outcome where the handoff is not where its writer asked.
- `--agent` accepts one lower-case word. `Claude` and `claude` would stamp as two writers and fork from each other, so the command refuses the first instead of guessing.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? no. The measured overwrites and the options are in the vault research note, and the spec carries the decision.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no. This extends HARNESS-088's thread model without changing it.
- [ ] New pattern candidate for `00_meta/patterns/`? no. "Identity is declared, never inferred" is already the stated rule behind the thread marker.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/MEMORY-009-thread-writer-identity/` -> `specs/archive/MEMORY-009-thread-writer-identity/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
