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
- [x] AC2 (bootstrap, rewritten 2026-10-21) -> guest run 2026-10-22 on #2922's `68ed76db` (Arm64, hive 4.4.2, `--dotf` a source build of that commit): V2, V4 and V5 flip, and the second converge changes nothing. See "Guest run on Arm64" below
- AC3 -> *withdrawn 2026-10-21* (hive ADR-020); AC9 replaces it
- [x] AC4 (no uv install or `uv tool list` inference on Windows; no silent skip, rewritten 2026-10-21) -> the setup gate half is `016bf1a8` (#1311); the catalog half is #2901 (`a4e46b1e`): Windows installs hive through `hive self-upgrade`, never `uv tool install`, and an install that places nothing fails. V3 flipped on the x64 `test-windows` runner in the run that installed hive (run 39721972641)
- AC7 -> *withdrawn 2026-10-21*: Baseline 2 replaces the broken-box table
- [ ] AC9 (no unattended upgrade, added 2026-10-21) -> PR5 (#2876, `ec5170ff`): `tests/hive-daemon-gate.bats` `no setup script installs an unattended hive upgrade`, plus the retire step's changed=0 rerun. That rerun needs a machine that still holds the old trigger: a clean CI runner or guest never had one, so its changed=0 proves nothing. The guest cannot be seeded: the old setup installed `DotfilesHiveUpgrade` only when `Get-Command hive` succeeded, and on Arm64 it never got hive installed (V3, V6). Each `dotf vm windows` run also starts from a fresh clean-base. So the rerun needs msi (powered off as of 2026-10-22) or the work-lab box, in the batched Windows session. On the guest the step reports `no unattended hive upgrade is installed` on both runs, which is the clean-machine half. The removal itself is unit-proven against a fake `schedule.Backend` (`cli/internal/converge/hiveupgrade_test.go`: `TestHiveUpgradeRetire_RemovesTheTimerThenIsConverged`, `TestHiveUpgradeRetire_WindowsRemovesTheTaskAndTheScriptCopy`), so the live changed=0 is AC9's last open step
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

### PR5: the timer deletion (#2857), 2026-10-21

```
$ bats tests/hive-daemon-gate.bats        # red: the guard copied onto d233110d^, before the deletion
not ok 2 no setup script installs an unattended hive upgrade (#2857, hive ADR-020)
#   `[ "$status" -eq 1 ]' failed
$ bats tests/hive-daemon-gate.bats        # the same guard with setup-linux.sh moved away: grep exits 2
not ok 1 no setup script installs an unattended hive upgrade (#2857, hive ADR-020)

$ bats tests/hive-daemon-gate.bats        # green, head
ok 1 parity: both setups probe hive's version through dotf tools version, not uv tool list (AI-034, #791)
ok 2 no setup script installs an unattended hive upgrade (#2857, hive ADR-020)

$ dotf lint powershell          -> exit 0
$ bash -n setup-linux.sh; echo "bash -n exit=$?"; zsh -n setup-linux.sh; echo "zsh -n exit=$?"
bash -n exit=0
zsh -n exit=0

# every bats file that names a changed path, run under bash so the list splits
$ bash -c 'files=$(for p in $(git diff --name-only origin/main...HEAD); do git grep -l "$(basename $p)" -- "tests/*.bats"; done | sort -u); bats $files > out.txt 2>&1; echo "exit=$?"; head -1 out.txt; grep -c "^not ok" out.txt'
exit=0
1..840
0
```

Those 840 tests in 50 files pass. That is the repo half of AC9: nothing installs a timer any more. The machine half is the `hive-upgrade-retire` converge step:

```
$ go test ./internal/converge/ -run HiveUpgradeRetire   -> ok (7 tests)
$ go test ./... && go vet ./... && GOOS=windows go vet ./...   -> ok
$ golangci-lint run ./internal/converge/...   -> 0 issues
```

The tests run against a fake scheduler. The live proof is two runs on msi (Linux) and on the work-lab box (Windows, elevated), after a release carrying the step; both are open tasks in PR5.

### PR6: hive self-installs on Windows (2026-10-21)

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

On the x64 `test-windows` runner (#2901, run 39721972641, job 119637916180), the setup run's own log:

```
hive self-upgrade: switched none -> 4.4.1.
hive 4.4.1 installed via hive self-upgrade 4.4.1 (hive-vault)
[SUCCESS] Installed hive daemon service (Scheduled Task, v4.4.1)
```

So on x64, V2 and V3 flip in one setup run. That job was red for another reason: the spec audit flagged GUARD-003 as a zombie, because #2898 closed #1052 after this branch last merged main. AC2's guest run is below.

## Guest run on Arm64 (2026-10-22)

`dotf vm windows --ref 68ed76db --dotf <windows/arm64 build of 68ed76db>`. Clean Windows 11 Arm64 guest, standard user, hive pinned at 4.4.2 with `self_install.no_build: ["cryptography"]`. The quoted lines, with their log line numbers, are in `.agent/runs/ai028-20261022t0735z/report.md` on `chore/autopilot-ai028-20261022t0735z`. The full 1909-line log is not committed.

| # | Baseline 2 (main, 2026-10-21) | This run |
|---|---|---|
| V2 | `hive is not installed on windows by this catalog; skipping` | `hive self-upgrade: switched none -> 4.4.2.` / `hive 4.4.2 installed via hive self-upgrade 4.4.2 (hive-vault)` (log:725-726); second run `hive 4.4.2 already installed; skipping` (log:1484) |
| V4 | `Registered MCP hive` with no hive on PATH | `[SUCCESS] Registered MCP hive` (log:943), and hive is on PATH |
| V5 | `[FAIL] mcp-servers.json registers hive as hive client, but hive is not on PATH` | no such row; `hive daemon` section `(1 checks, all ok)`, which is `hive service status` reporting healthy (log:1271, 1754) |
| V6 | `Failed to build cryptography==50.0.2` … `link.exe not found` | no source build: `Downloading cryptography (2.8MiB)` is a wheel |
| verdict | converged (hive silently absent) | `==> converged, and the second run changed nothing` (log:1909) |

The first attempt at 07:10Z failed for an unrelated reason. The guest clock runs 2 hours behind, and registry.npmjs.org's certificate was then under 2 hours old, so all four npm installs failed with `certificate is not yet valid`. The hive rows were the same in that attempt. Filed as #2932; the rerun after the certificate passed 2 hours of age is the run above.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Split into AI-028 + AI-029 rather than one ticket** (2026-08-07). The install model and the multi-client registration SSOT are separate failures with separate blast radii; the repo's own precedent is #574 being split out of #551 for atomic-PR discipline.
- **Resumed and re-scoped, not abandoned** (owner, 2026-10-21). It was triaged as an abandon candidate to free a WIP slot (12 active against ADR-043's 10, with PLAT-005 waiting). Baseline 2 then showed hive is never installed on the Arm64 guest while converge stays green, and that the guest removes the hardware gate that had stalled it. PLAT-005 waits a cycle. *Narrowed the same day:* the diagnostic run (V6) traced the guest's failure to a missing `win_arm64` wheel, so the "every fresh Windows machine" reading that informed the decision holds for Arm64 only. The x64 shape, a uv copy that ADR-019 rules out, is inferred, not measured. RESUME still rests on AC4's uv-on-Windows, AC9 and #1993. The owner re-confirmed it the same day, and decided the three open questions: PR6 is a `hive-self` catalog source, Arm64 is fixed in hive (hive#515), and the retire step is a converge step.
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
