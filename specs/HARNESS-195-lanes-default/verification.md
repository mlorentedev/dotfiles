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
- [ ] AC9 -> sessions met, Windows activation open. Under the owner's per-agent reading (2026-11-15), a real session of each agent with a carrier (claude, pi, opencode, agy) started in a lane and quoted AC3's line, and claude also did so on Linux; copilot and codex have no carrier (see AC3). Activation through `dotf converge` alone is shown on macOS and Linux (ace2). On the Windows guest it is not: converge skipped `lanes` because the guest gets an archive with no `.git`, and the activation below needed a hand-grafted clone. AC9 stays open until #3545 lets `dotf vm windows` (or the lab box) activate lanes through converge alone. Merged worktrees were swept on the Mac on 2026-11-15 (`dotf worktree sweep` reaped 3; the 29 left hold unmerged work):
  - macOS (macmini), 2026-11-14: `dotf update` installed 0.81.0 and converged. The run reported `[CHANGE] lanes 2.9s applied: lanes created and activated (4 change(s))` and `4 changed, 17 ok, 0 opaque, 4 skipped, 0 failed`. `.git/dotf-lanes-active` holds the committed marker (#3514), and a re-plan reports `0 to change, 20 ok`.
  - Real sessions, macOS, 2026-11-14, see the table below. In a lane (`dotf lane switch chore/3432-ac9-evidence` -> `feat-lane-secondary`), claude, pi and opencode each quoted `` lane `feat-lane-secondary` ACTIVE on `chore/3432-ac9-evidence` ``. In the main checkout each quoted `this repository runs lanes: start with `dotf lane switch <branch>``. That second line cannot be guessed from the prompt, so the model read it from the carrier rather than making it up. One earlier pi run answered inconsistently (it quoted the line, then retracted it in the same reply); the rerun and the control are both clean.
  - Linux (ace2, Ubuntu, dotf 0.81.0), 2026-11-15. ace2's converge used to stop at `tools`, because kubelab's `dev_node` role pinned node and python in `~/.config/mise/config.toml`, which outranks dotfiles' `conf.d`. kubelab#3564 fixed that, and the role was applied with `make provision NODE=ace2 ENV=staging`. A full `dotf converge` then reported `5 changed, 13 ok, 1 opaque, 6 skipped, 0 failed`, and wrote `.git/dotf-lanes-active` with the committed marker. A re-plan reports `[ OK ] tools 45 pinned CLI(s) at their pin`, `[ OK ] lanes lanes active (.dotf/lanes.json on origin/HEAD)`, and `0 to change, 17 ok`. Note that `dotf update` alone reported `already current (origin/main) — no deploy needed` and never reached `lanes`.
  - Linux carrier and guard, ace2, 2026-11-15. The deployed Claude SessionStart hook (`dotf mem session-start`), fed the hook's own JSON from the main checkout, emits `this repository runs lanes: start with `dotf lane switch <branch>``. `git commit --allow-empty` on main is refused with `ERROR: direct commits on 'main' at the main checkout are blocked in lane-managed repository` and a `FIX:` line naming `dotf lane switch`. At first no real session could run: `claude -p` answered `Not logged in`, and the secrets store is locked on that box (no age key or Bitwarden), so pi and opencode cannot get a model key either.
  - Real session, Linux (ace2), 2026-11-15, after the owner logged claude in. In the main checkout, `claude -p` quoted `this repository runs lanes: start with `dotf lane switch <branch>``. After `dotf lane switch chore/3432-ac9-ace2-probe` (routed to `fast-track-lane`), `claude -p` in the lane quoted `` lane `fast-track-lane` ACTIVE on `chore/3432-ac9-ace2-probe` ``. The probe lane was then parked (`dotf lane park fast-track-lane --force`; the branch had no commits).
  - agy, macOS, 2026-11-15. The first runs answered NONE. The cause was not the carrier: the main checkout was at 7e4bafa7, before #3497, so `records-bind` had never written the `session-line` `PreInvocation` entry into `~/.gemini/config/hooks.json`. `dotf update` reported `[CHANGE] records-bind ... written: .gemini/config/hooks.json`. Afterwards `agy -p` (agy 1.5.9) in `fast-track-lane` quoted `` lane `fast-track-lane` ACTIVE on `chore/3432-ac9-evidence` ``, and in the main checkout it quoted `this repository runs lanes: start with `dotf lane switch <branch>``. So agy's `PreInvocation` output does reach the model, which settles the open question in proposal.md's agy row.
  - Windows guest (Windows 11 Pro 10.0.26300 Arm64, the `dotf vm windows` clean-base overlay, session windows-1794776896, dotf 0.81.0), 2026-11-15:
    - **Setup:** `dotf vm windows --ref origin/main --keep` exited 0 with a second converge at 0 changed, but reported `[SKIP] lanes ... is not a git checkout yet`, because the guest receives the tree as an archive with no `.git` (#3545). A clone's `.git` was grafted in by hand (`git clone --no-checkout`, then `git reset --hard origin/main` at 900029db). That step is test setup, not a `dotf` step.
    - **Activation:** a full `dotf converge` then reported `[CHANGE] lanes 11.0s applied: lanes created and activated (4 change(s))` and `4 changed, 14 ok, 1 opaque, 5 skipped, 0 failed`. `.git\dotf-lanes-active` holds the committed marker. `dotf converge --plan` reports `[ OK ] lanes lanes active (.dotf/lanes.json on origin/HEAD)` and `0 to change, 18 ok`.
    - **Carrier:** the deployed SessionStart hook (`"C:\Users\dotf\.local\bin\dotf.exe" mem session-start`), fed the hook's JSON from the main checkout, emits `this repository runs lanes: start with `dotf lane switch <branch>``.
    - **Guard:** `git commit --allow-empty` on main exits 1 with `ERROR: direct commits on 'main' at the main checkout are blocked in lane-managed repository` and the `FIX:` line. No real session ran in the guest, which has no agent login.

## Observed launches (CLI-119 rule)

One row per agent, per machine: the date, the argv, and the `LANE-READY` answer, or "not installed".

| Agent | Machine | Date | Argv | Result |
|---|---|---|---|---|
| copilot 1.0.81 | macOS, herdr 0.9.3 | 2026-11-14 | `copilot -i "Reply with exactly the word LANE-READY and do nothing else."` via `dotf lane exec --agent copilot --prompt-file p.md` typed into a fresh split pane by `herdr pane run` | started interactive, answered `LANE-READY`, pane `agent: copilot` (the agent is the foreground process, no `HERDR_AGENT` needed) |
| opencode 1.16.2 | macOS, herdr 0.9.3 | 2026-11-14 | `dotf secrets run --only NAN_API_KEY -- opencode --prompt "<same prompt>"` via `dotf lane exec --agent opencode ...`, same way | `--prompt` submits the prompt: answered `LANE-READY` (Build, GLM 5.3 Flash). Pane `agent: null` before the #3430 fix, `agent: opencode` after it |
| pi | macOS, herdr 0.9.3 | 2026-11-14 | `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi "<same prompt>"` via `dotf lane exec --agent pi ...` (entry from CLI-119) | started; pane `agent: pi` with the #3430 fix |
| claude 2.1.325 | macOS (macmini), dotf 0.81.0 | 2026-11-14 | `claude -p "<quote the line starting with lane>"` in lane `feat-lane-secondary`; control from the main checkout | quoted the ACTIVE line in the lane, and the "this repository runs lanes" line in the main checkout (SessionStart hook) |
| pi 1.5.2 | macOS (macmini), dotf 0.81.0 | 2026-11-14 | `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi -p --no-session "<same>"`, same two places | same two lines (deployed `~/.pi/agent/extensions/dotfiles-gate.ts`) |
| opencode 1.16.2 | macOS (macmini), dotf 0.81.0 | 2026-11-14 | `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY,OPENAI_API_KEY -- opencode run "<same>"`, same two places | same two lines (deployed plugin, build agent on glm5.3-flash) |
| agy 1.5.9 | macOS (macmini), dotf 0.81.0 | 2026-11-15 | `agy -p "<quote the line starting with lane>"` in `fast-track-lane`; control from the main checkout | quoted the ACTIVE line in the lane and the "this repository runs lanes" line in the main checkout (`PreInvocation` hook from #3497, once the binding ran) |
| claude 2.1.325 | Linux (ace2), dotf 0.81.0 | 2026-11-15 | `claude -p "<same>"` in `fast-track-lane` and the main checkout | same two lines (SessionStart hook) |
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
