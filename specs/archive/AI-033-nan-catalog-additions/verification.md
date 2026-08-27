---
tags: [spec, verification, templates]
created: "2026-08-27"
---

# Verification - AI-033-nan-catalog-additions

## Evidence

- [x] AC1 (models resolve) -> commit `1188c9f` (`ai/pi/models.json`, `ai/pi/settings.json`)
      + `tests/pi-config.bats` "nan/* models all resolve to an id in models.json"
- [x] AC2 (README matches) -> commit `1188c9f` (`ai/pi/README.md`)
      + `tests/pi-config.bats` "README.md model list matches settings.json enabledModels"
- [x] AC3 (opencode + interleaving) -> commit `1188c9f` (`ai/opencode/opencode.jsonc`,
      `tests/opencode.bats`) + `tests/opencode.bats` "exposes 6 chat NaN models" and
      "maps NaN reasoning_content via interleaved on all 6 chat models"
- [x] AC4 (not wired as default) -> manual grep, `ai/pi/settings.json` `defaultModel` and
      `ai/opencode/opencode.jsonc` `model`/`small_model` unchanged from `qwen3.6`
- [x] AC5 (live-verified functional) -> manual: two `scripts/nan-debug.sh` runs per model
      (smoke prompt + this repo's planted `((count++))`/`set -e` bug), both models
      responded and both diagnosed the bug correctly; recorded on issue #1244

## Test status

- `~/.local/bin/bats tests/pi-config.bats tests/opencode.bats tests/pr-agent-config.bats` -> 88/88 ok
- `~/.local/bin/bats tests/*.bats` (full suite, isolated worktree, post-merge of #1246/#1248/#1252) -> 1519/1519 ok, exit 0
- `cd cli && go build ./... && go vet ./...` -> clean (unaffected by this diff; re-run for regression confidence)
- Manual smoke test: `scripts/nan-debug.sh -m qwen3.8-flash` and `-m glm5.3-flash`, both
  the trivial "responde solo: pong" prompt and the planted `((count++))` bug — both models
  responded correctly and both emitted `reasoning_content`
- No regressions in existing test suite: yes

## Decisions made during implementation

- Diff crossed the spec-gate's 50-LOC threshold (140 LOC) after the config work was
  already done and verified; this spec folder was created retroactively against a
  same-day issue (#1254) rather than reworking the change to duck under the threshold.
  `tasks.md` records the actual implementation order.
- Committed in an isolated worktree (`dotfiles-wt-nan-catalog`, branched fresh off
  `origin/main`) rather than the shared worktree the config edits were originally made
  in, because another live session was concurrently committing unrelated work
  (`fix/pi-subagent-shadowing`, merged as #1248) in that shared worktree. A patch of only
  the 5 touched files was applied there instead, to guarantee zero overlap.
- Deliberately left `harness/model-map.json`, `.pr_agent.toml` and
  `harness/reviewer-pool.json` untouched (see proposal's "Out of scope") — quality is not
  the blocker per the functional verification above, but NaN's allocation for both models
  is promotional and time-boxed; #1244 tracks the revisit.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? No — this is routine catalog maintenance
      following the existing `mimo-v2.5` precedent; nothing new was learned about the
      mechanism itself.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No — no architectural
      decision, catalog-only per existing conventions.
- [ ] New pattern candidate for `00_meta/patterns/`? No — single-repo, single-occurrence.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-033-nan-catalog-additions/` -> `specs/archive/AI-033-nan-catalog-additions/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
