---
tags: [spec, verification, templates]
created: "2026-09-26"
---

# Verification - AI-044-move-off-qwen38-quota

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 -> commit `0b18c9a` / test `opencode.jsonc default model is nan/glm5.3-flash, and titles use unmetered qwen3.6 (AI-044)`
- [x] AC2 -> commit `0b18c9a` / test `opencode.jsonc and ai/pi/models.json declare the same context window for every NaN model both carry` (fails on main for four models)
- [x] AC3 -> commit `d4b78ef` / `jq` check in `features.json` f3; live API returned 401 for `qwen3-rerank` and 200 for `rerank` on 2026-09-26

## Test status

- Test suite: `bats tests/*.bats` (serial) -> 1655/1656 ok; the one failure is test 1257, the vendored oh-my-zsh snapshot freshness check (#1641, environmental, red on main too)
- `cd cli && go test ./internal/harness/... ./internal/doctor/... ./internal/agent/...` -> ok
- Manual smoke test: live NaN API on 2026-09-26, status codes only: `glm5.3-flash`, `qwen3.6`, `gemma4` answer 200 in 0.73-0.91s (three samples each); `rerank` 200, `qwen3-rerank` 401
- No regressions in existing test suite: yes

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Titles moved to `qwen3.6` rather than `glm5.3-flash`: they are the highest-volume calls, and `qwen3.6` has no monthly quota.
- The context-window test is the guard for the 1M/262K defect class: pi's windows are checked by the limit-drift check (HARNESS-136), so equality with pi carries that check to opencode. It also surfaced three smaller drifts, aligned here.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? no: the defect was a stale binding, not a new failure class; the guard test records it
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a routing choice inside the existing model-map contract
- [ ] New pattern candidate for `00_meta/patterns/`? no: dotfiles-only

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-044-move-off-qwen38-quota/` -> `specs/archive/AI-044-move-off-qwen38-quota/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
