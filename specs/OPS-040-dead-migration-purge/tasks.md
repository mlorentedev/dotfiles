---
tags: [spec, tasks, templates]
created: "2026-09-01"
---

# Tasks - OPS-040-dead-migration-purge

> TDD order. One task = one focused commit. Tick as you go.
>
> **Inline markers**: `[P]` safe in parallel · `[AC<n>]` satisfies that criterion.

## Setup

- [x] Branch created from main: `chore/ops-040-dead-migration-purge`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Classify before deleting

- [x] Enumerate every block by its issue-ID marker, not by the audit's line numbers — they had drifted (MEM-002 L1342→L1305, GEMINI.md L478→L500)
- [x] Classify each by **skip-cost on an unconverged machine**: dead / correcting-and-probed / correcting-and-unverifiable
- [x] Probe the correcting blocks' conditions on msi (`~/.zshrc`, `~/.bashrc`, `claude mcp get hive`, `crontab -l`, the claude-mem dirs and settings key)
- [x] Probe the *targets* of every cleanup block being deleted — this is the step that caught blocks 4 and 11

## Implementation

- [x] [AC1] Delete the `OPENROUTER_API_KEY` deploy-time export from both scripts
- [x] [AC2] Delete blocks 2–7: endpoint exports (both), pre-SDD-007 `mcp_config` orphan, legacy `GEMINI.md` (both), `init-project.sh` orphan, python symlink, Bun installer
- [x] [AC3] Delete blocks 8–9 from `setup-linux.sh`: gh-copilot alias strip, legacy hive-vault crontab removal
- [x] [AC4] Leave HIVE-118 and MEM-002 whole; leave the rc `BUN_INSTALL` exports alone
- [x] [AC7] Write `tests/guard-doctrine-target-not-deleted.bats` and prove it fail-first against `main`'s scripts **before** relying on it
- [x] [AC5] Teach `secrets-show-callsites.bats` to SKIP with its reason instead of passing on an empty sweep (C15)
- [x] Retire the seven tests that pinned deleted blocks, each replaced by a comment saying what was removed and on what evidence — never by an assertion that the deletion happened
- [x] [AC6] `bash -n`, `shellcheck`, full `bats tests/*.bats`, baselined against clean `main`
- [x] [AC8] Write lessons 256 and 257 and index them

## Spun out rather than absorbed

- [x] #1431 — MEM-002 never converged (strips a `settings.json` key Claude Code stopped writing)
- [x] HIVE-118 deferred to the batched Windows session, named in `proposal.md`
- [x] `# EXPIRES:` convention proposed to the owner in the PR body, not built

## Closing

- [x] Every acceptance criterion is covered by at least one executable check
- [x] Every criterion has a `features.json` entry with a non-vacuous verification command, each one **executed**, not asserted
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in with evidence
- [ ] PR opened referencing this spec folder
- [ ] Independent adversarial review passed (required before archive)
- [ ] Spec archived and #1333 closed in the archiving PR

## Machine-readable features

This spec emits a sibling `features.json` following [[pattern-feature-list-as-primitive]].
