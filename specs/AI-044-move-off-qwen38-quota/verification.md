---
tags: [spec, verification, templates]
created: "2026-09-26"
---

# Verification - AI-044-move-off-qwen38-quota

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 -> commit `0b18c9a` / test `opencode.jsonc default model is nan/glm5.3-flash, and titles use unmetered qwen3.6 (AI-044)`
- [x] AC2 -> commit `0b18c9a` / test `pi-nan-package: opencode.jsonc and the package snapshot declare the same context window for every NaN model both carry` in `tests/pi-nan-package.bats`. It failed on main for four models. Since AI-046 (#1764), pi's side is the pinned `@gtrabanco/pi-nan-provider` in `ai/pi/packages.json`, not `ai/pi/models.json`. The test needs a real pi, so it runs in the `pi-nan-package` CI job; locally, run it with `PI_BIN` set.
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

## Review round 1 (FAIL, 2026-09-30) — dispositions

`review-round-1.md` (`nan/deepseek-v4-flash`) failed on two Majors:

- **Major F1, f2's command selected nothing: applied.** The test had moved to `tests/pi-nan-package.bats`, and `bats -f 'context window' tests/opencode.bats` matched no test and exited 0 (lesson 309). f2 now runs the real test with `PI_NAN_PACKAGE_REQUIRED=1`, so it can fail:
  - exit 0 with a real pi at `0a584d4`;
  - exit 1 when the reviewer's mutant sets the three opencode windows to `999999`;
  - exit 1 with no pi, instead of a silent skip.
- **Major F2, no offline guard: deferred to #1866.** The reviewer's own next steps call it "a follow-up ticket, not a blocker". The property is checked in CI but not enforced. The `pi-nan-package` job runs on every PR that touches `ai/opencode/opencode.jsonc` (its path filter) and on every push to main, but it is not a required check. #1866 records the root cause and the snapshot design for a local guard, and now also whether the job becomes required.
- **Minor, the evidence named `ai/pi/models.json`: applied.** The AC2 line above now names the test file and the pinned package.
- **Minor SPECULATIVE, the low chain falls to `claude:haiku` with no alarm: out of scope.** It is declared out of scope in `proposal.md` and tracked in AI-047 (#1766).
- **Question, the launcher's base range: answered.** The attribution to `75eb855` is right. The inferred base is tracked in #1551 and #1645.

## Review round 2 (PASS-WITH-GAPS, 2026-09-30) — dispositions

`review.md` (`nan/glm5.3-flash`) confirmed that f2 now fails on the mutant, and re-graded F2 from Major to Minor, with the reasoning stated. Its four Minors:

- **`pi-nan-package` is not a required check (REAL): deferred to #1866.** Whether to make it required, or put it behind an aggregate, is recorded there. The path filter and supervised merges make a drift PR visibly red today.
- **"enforced in CI" overstated it (REAL): applied.** The round-1 disposition above now says "checked in CI but not enforced".
- **No offline guard (REAL, carried from round 1): deferred to #1866**, as in round 1.
- **The low chain degrades with no quota alarm (SPECULATIVE): out of scope, tracked in AI-047 (#1766).**

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
