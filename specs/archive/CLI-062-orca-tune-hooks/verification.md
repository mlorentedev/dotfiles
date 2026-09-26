---
tags: [spec, verification, templates]
created: "2026-08-29"
---

# Verification - CLI-062-orca-tune-hooks

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (repair both files with backups; idempotent) -> commit `45a294f` / `TestTuneHooks_RepairsBothFilesWithBackups`, `TestTuneHooks_SecondRunChangesNothing`
- [x] AC2 (`--check`, nothing to do, generous timeout, unrecognised POST) -> `TestTuneHooks_CheckReportsAndWritesNothing`, `TestTuneHooks_NothingToDoWithoutOrca`, `TestTuneHooks_LeavesAGenerousTimeout`, `TestTuneHooks_UnrecognisedPostIsLeftAlone`
- [x] AC3 (doctor `--fix` through the package; remedy lines name the command) -> `TestCheckOrcaHook`; `checks_orca.go` imports `orca`
- [x] AC4 (setup calls the command, retires the script; four files deleted; audit amended) -> `tests/setup-windows.bats` "…through dotf orca tune-hooks, and retires the script (CLI-062)"; `git rm`; PSScriptAnalyzer 0 findings
- [x] AC5 (box) -> transcript below, Windows work box, 2026-08-29

## Test status

- Test suite: `cd cli && go test ./... -count=1` -> every package `ok`, `FAIL_COUNT=0`; `go vet` clean under `GOOS=windows` and `GOOS=linux`; `golangci-lint run` (pinned 2.12.2) `0 issues`
- `bats tests/setup-windows.bats -f CLI-062` -> 1/1; `Invoke-ScriptAnalyzer setup-windows.ps1` -> 0 findings, 0 non-ASCII
- Manual smoke test (AC5), binary built from this branch:

  ```text
  --- real files, --check ---
  ok: orca.json hook timeouts >= 30
  ok: copilot-hook.ps1 uses HttpWebRequest
  exit=0
  --- scratch copies with the DX-006 defects ---
  drift: ...\orca-scratch\orca.json has a hook timeoutSec < 30
  drift: ...\orca-scratch\copilot-hook.ps1 still uses Invoke-WebRequest
  Error: Orca's Copilot hooks need tuning — run `dotf orca tune-hooks` (DX-006)
  check exit=1
  backup     ...\orca-scratch\orca.json.bak.20260829-042250
  backup     ...\orca-scratch\copilot-hook.ps1.bak.20260829-042250
  tuned      2 fix(es) applied — restart the Copilot CLI session to pick up the new orca.json timeout
  tune exit=0
  ok: orca.json hook timeouts >= 30
  ok: copilot-hook.ps1 uses HttpWebRequest
  check-after exit=0
  timeoutSec 5 left: 0; HttpWebRequest lines: 1; backups: 2
  ```

  The real files were only read (`--check`); the repair was exercised on copies through
  `--hook-config` / `--hook-script`, the flags the script also had.
- No regressions in existing test suite: yes

## Decisions made during implementation

- **The block is assembled, not templated.** The first cut passed the `HttpWebRequest` block to `regexp.ReplaceAll` as a template; Go read PowerShell's `$req`, `$uri`, `$env:` as named groups and expanded them to nothing, so the swapped script had `Create()` where the script had `Create($uri)`. `TuneScript` now builds the block line by line with the captured indentation and the file's own line ending (the CRLF the fixture and the real file carry), through `ReplaceAllFunc`. The port's test caught it before any file did.
- **One package, three callers.** `TimeoutBelow` / `TuneTimeout` moved from doctor into `orca`, `TuneScriptFile` is the per-file entry doctor `--fix` and `TuneHooks` share, so the command and the doctor cannot drift the way the script and the doctor's regex had.
- **The script is retired, not just undeployed.** It moved from `$deployedScripts` to `$retiredScripts` so a copy left in `~\.dotfiles\scripts` by an earlier setup is swept (WIN-013's mechanism), and the bats asserts both lists.
- **`orca-tune.sh` deleted with no port needed**: `dotf orca tune` has been that function since #1274; the script had no caller.

## Retroactive review (2026-09-25, W1.4 of #1625)

Reviewed from a detached worktree at the landing commit `4718e46` (#1384) plus the f5 contract fix
already on `main` (`03ab22d`, a box-only check that now names its platform), with that commit's own
`dotf` first on `PATH`, so the reviewer's scope was `4718e46^...HEAD`. At that tree f1-f4 passed,
and f5 exits 1 off Windows, as its `behavior` declares.

**Round 1, `nan/glm5.3-flash`, FAIL.** Transcript kept outside the repo
(`~/.local/state/dotf/review-transcripts/CLI-062-r1.jsonl`). Dispositions:

| Finding | Disposition |
|---|---|
| Major: `doctor --fix`'s script half (`checkOrcaHook` through `orca.TuneScriptFile`) had no test | **Applied** in `2d5dae7`: `TestCheckOrcaHook_FixTunesTheScript` and `TestCheckOrcaHook_FixLeavesAnUnrecognisedPostAndFails`. The reviewer's mutant (a wrong path to `TuneScriptFile`) now fails. |
| Major: only one of the 14 `HttpWebRequest` block lines was pinned, although the proposal says the block is pinned by a test | **Applied** in `2d5dae7`: `TestTuneScript_WritesTheRetiredScriptsBlockByteForByte` compares the tuned script byte for byte with the retired script's block (its lines 148-161 at `4718e46^`), with the captured indentation and CRLF. The reviewer's mutant (`Timeout = 9999`) now fails. Until this commit, the proposal's claim was true for one line only. |
| Minor: apply mode printed "in sync" right after saying an unrecognised script was left unchanged | **Applied** in `2d5dae7`: no in-sync line in that case. The exit stays 0, as the retired script's did; `--check` is the gate. `TestRunOrcaTuneHooksUnrecognisedIsNotInSync` is the first cmd-level test. |
| Minor: `TestMergeAgainstTheRealDeployedSettings` reads the live `~/.claude/settings.json` and failed on the reviewer's run | **Declined**: pre-existing, in a package CLI-062 does not touch, and it passes on `main` today (25 of 25 packages ok on this branch). The `FAIL_COUNT=0` line above describes the landing run. |
| Minor THEORETICAL: `writeTuned`'s fixed `.tmp` path could race between two concurrent tuners | **Ticketed**: #1748 (CLI-086) |
| Question: the EOL is chosen per file, and `TuneTimeout` normalises the spacing of the key it replaces | **No action**: both are inherited from the retired script, and the real Orca files are uniform |

**Round 2, `nan/mimo-v2.5`, PASS WITH GAPS**, at `b1ae0c9` (the landing commit, the f5 fix, and
`2d5dae7`). `review.md` and `review-request.json` are this round's. The reviewer left the four
retired twins **staged** in the reviewed tree. They were removed before anything was copied, and
the instance is recorded on #1649. Dispositions:

| Finding | Disposition |
|---|---|
| `checkOrcaHook` CC 15, `runOrcaTuneHooks` 14, `TuneHooks` 12, against the limit of 10 | **Ticketed**: #1748 |
| `writeTuned`'s predictable `.tmp` path | **Ticketed**: #1748 (as in round 1) |
| `ScriptUsesInvokeWebRequest` compiles its regexp per call | **Ticketed**: #1748 |
| `doctor --fix`'s JSON half writes without a backup; the script half backs up | **Declined**, as the reviewer recommends: pre-existing, and `dotf orca tune-hooks`, the primary entry point, backs up both |

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? no — the `$name` template expansion is recorded in the code comment and here; it is a Go `regexp` fact, not a project class
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — ADR-020 §5 applied; audit-007 amended
- [ ] New pattern candidate for `00_meta/patterns/`? no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-062-orca-tune-hooks/` -> `specs/archive/CLI-062-orca-tune-hooks/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
