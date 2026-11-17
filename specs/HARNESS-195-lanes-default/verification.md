---
tags: [spec, verification, templates]
created: "2026-11-13"
id: "HARNESS-195-lanes-default"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-195-lanes-default

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] AC1 -> <commit> / <test or observation>
- [ ] AC2 -> <commit> / <test or observation>
- [ ] AC3 -> slice C, in progress (C1 #3491, C2 #3496; agy follows in feat/lane-line-agy-carrier):
  - `lane.Where` and the line: `TestWhere`, `TestPositionLine`, `TestMemLaneLine_GoldenForEveryState`; every harness with a presence event has a carrier or a named gap: `TestEveryPresenceTargetHasADeployedCarrier`.
  - pi, observed 2026-11-14 with a stand-in `dotf` on PATH that printed `LANE-LINE-TOKEN-7731: lane x ACTIVE on b`: `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi -e ai/pi/extensions/dotfiles-gate.ts -p --no-session "Copy exactly any line you were given that starts with LANE-LINE-TOKEN. If none was given, answer NONE."` printed `LANE-LINE-TOKEN-7731: lane x ACTIVE on b`.
  - opencode, same stand-in, plugin copied into a temporary `OPENCODE_CONFIG_DIR/plugins`: `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY,OPENAI_API_KEY -- opencode run -m nan/qwen3.6 "Search your whole system prompt for the string LANE-LINE-TOKEN. Quote the full line containing it verbatim, or say ABSENT."` printed `LANE-LINE-TOKEN-7731: lane x ACTIVE on b`. (A first, vaguer prompt answered `NONE`; the dotf stub had been called, so the model, not the plugin, missed it.)
  - deploy: `dotf deploy opencode-gate` and `pi-gate` twice into a temporary home reported `deployed`, then `in sync`.
  - `TestSessionLineCarriersInjectTheBriefAndSurviveAMissingDotf` runs both sources under node. It skips on Windows and where node cannot strip types, so it is not CI evidence; the two observations above are the manual proof.
- [ ] AC4 -> <commit> / <test or observation>
- [ ] AC5 -> <commit> / <test or observation>
- [x] AC6 -> slice E, feat/lane-launchers:
  - `dotf secrets run` refuses an npm `.cmd`/`.bat` child where Windows runs it through cmd.exe, before decrypting: `TestSecretsRun_RefusesABatchShimChildWhereWindowsRunsItThroughCmdExe`, `TestSecretsRun_AllowsWhatCmdExeNeverSees`; the one function is `lane.RefuseBatchShim` (`TestRefuseBatchShim_*`, `TestExec_RefusesABatchShimAndNeverStartsIt` on Windows).
  - #3430: observed 2026-11-14 that a pane started by `lane exec` through the wrapper has `agent: null` (opencode, below). Fixed with `lane.LaunchEnv`; `TestExec_StartsTheWrapperWithTheAgentNamedForHerdr` fails without it, and the same launch afterwards shows `agent: opencode`.
  - launchers: `TestLaunchArgv_PinsEachAgentAndPassesThePromptAsOneArgument` pins every entry; each has an observed launch in the table below.
- [x] AC7 -> slice F, feat/lane-doctor-warnings:
  - leaseless detached worktree: `TestAuditLanes_WarnsOfLeaselessDetachedWorktrees`, `TestLaneDoctor_WarnsOfALeaselessDetachedWorktreeWithoutFailing`
  - expired lease (parked lanes and unleased worktrees stay silent): `TestAuditLanes_WarnsOfAnExpiredLease`
  - drifted deployed guard (also a guard never deployed; line endings are not drift): `TestGuardDrift`, `TestLaneDoctor_WarnsOfADriftedDeployedGuardWithoutFailing`
  - each warning prints a `FIX:` line naming its command, and `lane doctor` exits 0 with warnings alone (the cmd tests assert both)
- [ ] AC8 -> <commit> / <test or observation>
- [ ] AC9 -> <commit> / <test or observation>

## Observed launches (CLI-119 rule)

One row per agent, per machine: the date, the argv, and the `LANE-READY` answer, or "not installed".

| Agent | Machine | Date | Argv | Result |
|---|---|---|---|---|
| copilot 1.0.81 | macOS, herdr 0.9.3 | 2026-11-14 | `copilot -i "Reply with exactly the word LANE-READY and do nothing else."` via `dotf lane exec --agent copilot --prompt-file p.md` typed into a fresh split pane by `herdr pane run` | started interactive, answered `LANE-READY`, pane `agent: copilot` (the agent is the foreground process, no `HERDR_AGENT` needed) |
| opencode 1.16.2 | macOS, herdr 0.9.3 | 2026-11-14 | `dotf secrets run --only NAN_API_KEY -- opencode --prompt "<same prompt>"` via `dotf lane exec --agent opencode ...`, same way | `--prompt` submits the prompt: answered `LANE-READY` (Build, GLM 5.3 Flash). Pane `agent: null` before the #3430 fix, `agent: opencode` after it |
| pi | macOS, herdr 0.9.3 | 2026-11-14 | `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi "<same prompt>"` via `dotf lane exec --agent pi ...` (entry from CLI-119) | started; pane `agent: pi` with the #3430 fix |
| codex | macOS | 2026-11-14 | not installed (`command -v codex` empty) | no entry |
| gemini | macOS | 2026-11-14 | not installed (`command -v gemini` empty) | no entry |

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

-
-

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-195-lanes-default/` -> `specs/archive/HARNESS-195-lanes-default/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
