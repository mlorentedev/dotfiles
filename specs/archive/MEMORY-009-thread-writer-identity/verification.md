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
- [x] **AC5** -> slice 2, after 0.59.0 (which carries `--agent`) was installed on 2026-09-25. Vault `27a2c9c8`; the `harness/skills/handoff` record was refreshed (f5).
  - The skill's `handoff-write` command passes `--agent <agent>`, with the agent's own harness name, and says what a fork means.
  - The same edit fixes the skill's other agent-specific line: `dotf mem thread … --agent claude` became `--agent <agent>`. Before it, a pi, agy, Copilot, opencode or Codex session following the skill named its journal as claude's, and its `Journal:` line then attributed its block to claude, which is the attribution this spec reads.

## Review scope

This spec's change is two pieces:

- slice 1, merged to main as #1711 (`6436c71`);
- slice 2, this branch's commits after `280af58`.

The commits between them on main belong to other specs (#1714, #1715, #1716, #1721, #1724 and the 0.59.0 release). The review base the launcher resolves spans them too, so judge only the files this spec names: `cli/internal/mem/handoff.go`, `cli/internal/cmd/mem_handoff.go`, their tests, `harness/skills/handoff/SKILL.md` and this folder.

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

## Independent review

Verdict **PASS-WITH-GAPS**, from nan/glm5.3-flash on 2026-09-25. The review ran from 17:17 to 17:27 under HARNESS-152's deadline, with a 30m target. It found no Blockers and no Majors, and judged the archive advisable once the findings below have a disposition.

| # | Finding | Disposition |
|---|---|---|
| 1 | Minor, THEORETICAL. No named test covers the legacy migration and a fork happening in one write. The reviewer's probe found it correct. | **Applied.** `TestAnAgentWriteMigratesTheLegacyBlockAndForksFromAnotherAgent` uses the reviewer's seed case. |
| 2 | Minor, THEORETICAL. The skill's agent list omitted `antigravity` and `gemini`, both of which `journalAgents` knows. | **Applied** (vault `38fec1aa`, record refreshed). The skill now names every agent the writer attributes. |
| 3 | Minor, SPECULATIVE. A hand-edited stamp such as `(writer: Claude)` reads as a different writer, so `claude` forks from it. | **Declined.** The write path cannot produce it, since `writerName` refuses capitals, and the cost is one extra thread, never a loss. |
| 4 | Minor, SPECULATIVE. The journal heuristic can misattribute when a project word equals an agent name. | **Declined.** This is the risk the proposal already accepts: a wrong attribution costs one extra thread, and no vault project name contains an agent's. |
| 5 | Question. The live dry run could not be reproduced. | **Recorded.** The reviewer looked for `~/Projects/knowledge/memory/MEMORY.md`; the file is at `~/Projects/knowledge/10_projects/knowledge/memory/MEMORY.md`. The named unit tests carry the mechanism either way. |

## Decisions made during implementation

- `WriteThreadAs` is a new entry point, and `WriteThread` is unchanged in signature and output. So AC4 holds by construction and is also pinned by a test.
- The fork notice prints whenever the write lands in the fork, even when the fork's content is unchanged. It is the one outcome where the handoff is not where its writer asked.
- `--agent` accepts one lower-case word. `Claude` and `claude` would stamp as two writers and fork from each other, so the command refuses the first instead of guessing.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? no. The measured overwrites and the options are in the vault research note, and the spec carries the decision.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no. This extends HARNESS-088's thread model without changing it.
- [ ] New pattern candidate for `00_meta/patterns/`? no. "Identity is declared, never inferred" is already the stated rule behind the thread marker.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/MEMORY-009-thread-writer-identity/` -> `specs/archive/MEMORY-009-thread-writer-identity/`
- [x] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018): the archive PR carries `Closes #1690`
- [x] Promotions above executed (if any): none, since all three were decided no
