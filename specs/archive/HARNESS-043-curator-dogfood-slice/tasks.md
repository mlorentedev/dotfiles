---
tags: [spec, tasks]
created: "2026-06-25"
---

# Tasks - HARNESS-043-curator-dogfood-slice

> TDD order. Decision resolved: **presence = uniform marked-region injection** across claude/opencode/pi/copilot (agy out). The SessionStart-hook build was replaced. One task ≈ one focused commit.

## Setup

- [x] Branch from main: `feat/HARNESS-043-curator-dogfood-slice`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Decision resolved: injection over hook; 4 harnesses; presence-only (Action/dispatch → H-045)

## Implementation

### AC1 — neutral curator definition (vault)
- [x] `00_meta/agents/definitions/curator/AGENT.md` — neutral frontmatter, `targets: [claude, opencode, pi, copilot]`, system-prompt body; record copy under `harness/agents/curator/`

### AC2 — manifest + schema (dotfiles)
- [x] `harness/agent-frontmatter.schema.json` (required: name, description, kind)
- [x] `harness/manifest.json` `agents` block: `deploy[]` (claude→agent-md) + `presence[]` (claude/opencode/pi/copilot → instructions file)

### AC3 — agent-md render engine (dotfiles, TDD)
- [x] bats: `--refresh` writes a verbatim AGENT.md record (no provenance, no $HOME)
- [x] bats: `--deploy` renders `agent-md` to `~/.claude/agents/<name>.md` — keeps name/description, drops neutral-only keys, adds `generated_*`
- [x] bats: `--check` validates the record renders offline; fails on missing required key

### AC4 — uniform presence injection (dotfiles, TDD)
- [x] engine: `AGENT_BEGIN_PREFIX`/`AGENT_END_MARKER` (distinct namespace), `build_agent_presence`, `inject_agent_presence` (replace-in-place or append; skip absent file), `deploy_agent_presence` (iterate `presence[]`)
- [x] removed `deploy_agent_hooks` + `merge_session_start_hook` (SessionStart)
- [x] bats: `--deploy` injects a presence region (forced skills) into every harness instructions file
- [x] bats: injection is idempotent (one region across re-deploys) and leaves the patterns region + user content intact
- [x] bats: append-when-absent; per-harness `targets[]` respected (persona only appears for harnesses it targets)

### AC5 — proof
- [x] `bats tests/compile-harness.bats` green (agent subset 9/9; full suite 36/36 executed, see verification.md)
- [x] sandbox-`$HOME` deploy proof: real script injects into seeded claude/opencode/pi/copilot files, all `presence=1 generated=1 skills=yes`, patterns+user intact, idempotent on re-deploy
- [ ] PR referencing this spec (on user's word)
- deferred (not this slice): live-session capture; injection is text-only + cross-OS, so no OS-axis command form to verify

## Closing

- [ ] Every acceptance criterion covered by ≥1 bats test or a sandbox artifact
- [ ] `bats tests/compile-harness.bats` green; pre-existing tests unaffected
- [ ] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] Pattern doc updated (presence=injection; plugins=Action level)
- [ ] PR opened referencing this spec folder
