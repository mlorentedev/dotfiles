---
tags: [spec, verification, templates]
created: "2026-08-07"
---

# Verification - AI-028-hive-install-model-migration

## Baseline: the broken state, captured before any change (2026-08-07)

This spec is unusual in having a machine sitting in exactly the failure state it targets. The before-state is recorded here verbatim so the after-state is a comparison against evidence, not against memory. Host: maintainer's Windows 11 box, `hive-vault` 1.43.0 latest on PyPI.

| # | Probe | Before (broken) |
|---|---|---|
| B1 | `hive --version` | `error: uv trampoline failed to canonicalize script path` |
| B2 | `uv tool list` | `aider-chat v0.86.2`, `pre-commit v4.6.0` — **no hive-vault** |
| B3 | `Test-Path $env:LOCALAPPDATA\hive` | `False` — A3 layout never created |
| B4 | `Get-ScheduledTask` matching hive | only `DotfilesHiveUpgrade`; **no `HiveVaultDaemon`** |
| B5 | `powershell -File ~/.claude/scripts/hive-upgrade.ps1` | *no output*, `exit 0` |
| B6 | `Get-ScheduledTaskInfo DotfilesHiveUpgrade` | `LastTaskResult: 0`, `LastRunTime: 8/7/2026 11:34:34 AM`, `NextRunTime: 11:49:49 AM` — green while doing nothing |
| B7 | `~/.claude.json` → `mcpServers.hive` | `C:\Users\mlorente\.local\bin\hive.exe` (the dead trampoline) |
| B8 | `ToolSearch` for `vault_query` in a Claude session | no match — **zero hive MCP tools registered** |
| B9 | `python -c "shutil.which('hive')"` | `C:\Users\mlorente\.local\bin\hive.exe` — what `_resolve_exec()` would register (hive#328) |

Supporting facts captured at the same time:

- The 3 live `hive-vault.exe` processes resolve to `...\uv\cache\archive-v0\BWm4N5bVOd2KpnaOdNMHk\Scripts\hive-vault.exe` — ephemeral `uvx` environments belonging to *other* clients, not the daemon and not the uv tool. See AI-029.
- `uvx --from hive-vault hive --version` → `hive-vault 1.43.0`, rc=0. The bootstrap path works today.
- `~/.local/share/hive` holds 295 unrotated `hive-<pid>.log` files dating to 2026-05-19.

## Baseline 2: a clean Windows 11 guest, measured 2026-10-21

The box above no longer holds its before-state, so the re-scope (proposal, 2026-10-21) takes a reproducible one: `dotf vm windows --ref 856f49fc` (main), a throwaway copy of the PLAT-003 clean-base, **Windows 11 Arm64**, with no hive and nothing on PATH but what converge puts there. V1–V5 line numbers are from that run's 1878-line log. V6 is from a second run on a local, never-pushed branch that only printed what main discards (the `prerequisite_command` output, `$env:Path`, and whether `~\.local\bin\hive.exe` and `%LOCALAPPDATA%\hive` exist).

**This is an Arm64 baseline.** x64 is unmeasured. There `cryptography` has a wheel, so the same `uv tool install` most likely succeeds and leaves the uv copy that ADR-019 rules out: a different defect, still AC4's.

| # | Observation | On main |
|---|---|---|
| V1 | Run verdict | `==> converged, and the second run changed nothing`. First run: `9 changed, 1 ok, 1 opaque, 7 skipped, 0 failed` (log:1392) |
| V2 | The tools step on hive | `hive is not installed on windows by this catalog; skipping` (log:948, both runs) |
| V3 | `hive service install` and the `DotfilesHiveUpgrade` task | never run: no "hive service" or "hive-upgrade task" line in the whole log. The block in `setup-windows.ps1` is gated on `Get-Command hive`, and its `elseif` is gated on it too, so a missing hive skips both in silence |
| V4 | MCP registration | `[SUCCESS] Registered MCP hive` (log:994): agents launch `hive client` against a binary that is not there, the same shape as B7 |
| V5 | `dotf doctor`, both runs | `[FAIL] mcp-servers.json registers hive as hive client, but hive is not on PATH` (log:1248, 1724), with a fix that names `dotf tools install hive`, which V2 shows skips Windows (#2856) |
| V6 | Why hive is missing (diagnostic run) | `uv tool install --upgrade hive-vault` exits **1** on both runs: `Failed to build cryptography==50.0.2` … `error: linker link.exe not found` (Python reports `cp313-win_arm64`). `setup-windows.ps1` pipes this to `Out-Null`. Afterwards `~\.local\bin\hive.exe` and `%LOCALAPPDATA%\hive` do not exist, and `uv tool list` shows only `poetry v2.5.1` |

The run is green while the doctor fails: the converge has no step that owns hive on Windows, so nothing can report it missing. That is lesson-159's class ("quiet when idle and quiet when broken") on the converge path.

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (loud-vs-quiet) -> `tests/hive-upgrade-timer.bats`: `hive-upgrade.ps1 is loud and non-zero when no install is found`, `... stays silent when the install is already current`, `... reports an unreachable PyPI without failing the tick`, `... does not collapse no-install into the already-current guard`. Suite green 33/33; script ASCII-clean. B5/B6 flip once the SSOT is deployed (see the deploy caveat below). *Superseded 2026-10-21:* PR5 (#2857) deletes the script and this suite, so this line is history: it proves #796 at the time, and nothing replaces it, because AC9 removes what AC1 tested.
- [ ] AC2 (bootstrap, rewritten 2026-10-21) -> a guest run on PR6's ref where V2, V4 and V5 flip, and the second converge changes nothing
- AC3 -> *withdrawn 2026-10-21* (hive ADR-020); AC9 replaces it
- [ ] AC4 (no uv install or `uv tool list` inference on Windows; no silent skip, rewritten 2026-10-21) -> the setup gate half is `016bf1a8` (#1311); the rest flips V3 on PR6's ref
- AC7 -> *withdrawn 2026-10-21*: Baseline 2 replaces the broken-box table
- [ ] AC9 (no unattended upgrade, added 2026-10-21) -> PR5 (#2857): `tests/hive-daemon-gate.bats` `no setup script installs an unattended hive upgrade`, plus the retire step's changed=0 rerun
- [x] AC5 (A1/A3 reconciliation) -> `AI-022/tasks.md` supersession block + `AI-023` forward pointer
- [x] AC6 (Linux decision recorded) -> Linux keeps `uv tool`; recorded in both `proposal.md` and `AI-022/tasks.md`

## Test status

```
$ bats tests/hive-upgrade-timer.bats
33 tests, 0 failures          # 4 new (cases 26-29), 29 pre-existing unchanged

$ LC_ALL=C grep -nP '[^\x00-\x7F]' windows/hive-upgrade.ps1
(no output)                   # ASCII clean for PSScriptAnalyzer CI
```

- Manual smoke test: pending for PR2/PR3. PR1 is a text-contract change to a PowerShell script; bats asserts the contract, and the live flip of B5/B6 needs the SSOT deployed to `~/.claude/scripts/` first.
- No regressions: the 29 pre-existing cases in `hive-upgrade-timer.bats` are untouched and green, including `only acts when a newer version is published`, which pins the fast-path contract this change had to preserve.

> **Do not verify against the deployed copy.** `~/.claude/scripts/hive-upgrade.ps1` is deployed from `windows/hive-upgrade.ps1` by setup. Re-deploy (or invoke the SSOT directly) before treating B5 as flipped, or a stale copy will read as a pass.

## PR6 evidence (2026-10-21)

```
$ go test ./internal/tools/ -run 'SelfInstall|TheRepoCatalogDeclaresHive' -v | grep -E '^--- '
--- PASS: TestTheRepoCatalogDeclaresHiveSelfInstalledOnWindows
--- PASS: TestLoad_SelfInstall
--- PASS: TestInstallUvTool_SelfInstall
--- PASS: TestInstallUvTool_SelfInstallRunsOnThePinnedPython
--- PASS: TestInstallUvTool_SelfInstallProbesThePersistentPath
--- PASS: TestInstallUvTool_SelfInstallThatPlacesNothingFails
$ go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...   -> ok
$ golangci-lint run   -> 0 issues

# the argv against a real uv (Mac, ephemeral: nothing is installed)
$ uv --version
uv 0.12.18 (01cb90c1a 2026-09-22 aarch64-apple-darwin)
$ uv tool run --refresh-package hive-vault --from hive-vault==4.4.1 --python 3.13 hive --version
hive-vault 4.4.1        # exit 0

# mutation: current() without the registered-PATH fallback
--- FAIL: TestInstallUvTool_SelfInstallProbesThePersistentPath

# mutation: packages.json self_install args "latest" instead of "{version}"
$ jq -e '<the tests/packages-json.bats self_install filter>' mutated.json
false        # exit 1
```

The tests run against a fake uv and a fake probe. Nothing here has run `hive self-upgrade` on Windows: that is AC2's guest run, gated on hive#515.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Split into AI-028 + AI-029 rather than one ticket** (2026-08-07). The install model and the multi-client registration SSOT are separate failures with separate blast radii; the repo's own precedent is #574 being split out of #551 for atomic-PR discipline.
- **Resumed and re-scoped, not abandoned** (owner, 2026-10-21). It was triaged as an abandon candidate to free a WIP slot (12 active against ADR-043's 10, with PLAT-005 waiting). Baseline 2 then showed hive is never installed on the Arm64 guest while converge stays green, and that the guest removes the hardware gate that had stalled it. PLAT-005 waits a cycle. *Narrowed the same day:* the diagnostic run (V6) traced the guest's failure to a missing `win_arm64` wheel, so the "every fresh Windows machine" reading that informed the decision holds for Arm64 only. The x64 shape, a uv copy that ADR-019 rules out, is inferred, not measured. RESUME still rests on AC4's uv-on-Windows, AC9 and #1993, and the owner re-confirms it on return.
- **PR1 sequenced before the model change.** The loud-vs-quiet fix does not depend on resolving the bootstrap question, and it is what makes the next recurrence visible. Shipping it first means the session produces a durable improvement even if the model migration proves deeper than expected.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? yes: docs/lessons/lesson-159-a-guard-that-is-quiet-when-idle-and-quiet-when-bro.md (captured 2026-08-07, *"A guard that is quiet when idle and quiet when broken is not a guard"*; Baseline 2 shows the class again on the converge path)
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes, for PR6: the catalog's `self_install` field is this repo's own contract, recorded as `docs/adr/adr-036-install-channels.md`'s 2026-10-21 amendment. The mechanism and the upgrade policy stay hive's (hive ADR-015, ADR-019, ADR-020)
- [x] New pattern candidate for `00_meta/patterns/`? no: lesson-159 holds the insight, and a second instance inside the same spec is not a recurrence outside the incident

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-028-hive-install-model-migration/` -> `specs/archive/AI-028-hive-install-model-migration/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
