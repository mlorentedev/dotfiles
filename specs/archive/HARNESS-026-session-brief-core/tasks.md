---
tags: [spec, tasks, templates]
created: "2026-06-17"
---

# Tasks - HARNESS-026-session-brief-core

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/session-brief-core`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (scope resolved: full agent-independent cluster)

## Implementation

> Strangler extraction. The byte-equivalence test (`session-start-config.bats`,
> 3 CWDs vs `origin/main`) is the safety net for the adapter refactor.

- [x] `tests/session-brief.bats`: `sb_*` emitters in isolation + standalone `--format` modes + unknown-format guard (16 tests)
- [x] Create `scripts/session-brief.sh`: POSIX `sh`, source-guard (`SESSION_BRIEF_LIB`), `--format` arg parse, usage→stderr + non-zero on unknown/empty format
- [x] Lift `find_vault_root` into the core; add `sb_vault_detect` (the `Obsidian vault detected: NAME (ROOT)` headline emitter)
- [x] Add `sb_vault_health` (runs `vault-health.sh`, formats the GUI-down / pass / fail / not-installed cases — verbatim text)
- [x] Add `sb_specs` (active/archived counts + `[AGENT-DRAFT]` flagging — lifted from `detect_repo_specs`)
- [x] Add `sb_vault_baseline` (skills SKILL.md + critical-files check — lifted from `check_vault_baseline`)
- [x] Implement standalone `--format=stdout` (full brief: headline → vault-health → specs → baseline) and `--format=markdown` (same, one fenced block)
- [x] Unknown `--format=xyz` exits non-zero, usage line on stderr (test)
- [x] Refactor `claude-session-start.sh`: `source` the core, replace the 4 inline blocks with `sb_*` calls at their exact legacy positions; `find_vault_root`/`detect_repo_specs`/`check_vault_baseline`/vault-health definitions removed from the hook

## Closing

- [x] Byte-equivalence test passes (3 CWDs, POST == `origin/main` PRE)
- [x] New `session-brief.bats` covers both `--format` modes + the unknown-format guard, with isolated `HOME`/vault fixtures
- [x] `shellcheck scripts/session-brief.sh` clean; `shellcheck scripts/claude-session-start.sh` clean at CI severity (SC1090 silenced via `source=` directive)
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] No unrelated changes in the diff (no scope creep — hive/path-coupled signals untouched)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See sibling `features.json`. Pass-state gating: the agent never writes `"state": "passing"` — only the harness, after running `verification` at exit 0, may.
