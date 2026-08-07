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

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (loud-vs-quiet) -> `tests/hive-upgrade-timer.bats`: `hive-upgrade.ps1 is loud and non-zero when no install is found`, `... stays silent when the install is already current`, `... reports an unreachable PyPI without failing the tick`, `... does not collapse no-install into the already-current guard`. Suite green 33/33; script ASCII-clean. B5/B6 flip once the SSOT is deployed (see the deploy caveat below).
- [ ] AC2 (bootstrap) -> commit `<hash>` + B1/B2/B3 flip on the broken box
- [ ] AC3 (bare `hive self-upgrade`, no stop/start) -> test `<name>`
- [ ] AC4 (no `uv tool list` inference on Windows) -> test `<name>` + B4 flips (task registered)
- [ ] AC5 (A1/A3 reconciliation) -> commit `<hash>`
- [ ] AC6 (Linux decision recorded) -> commit `<hash>`
- [ ] AC7 (end-to-end on the broken box) -> B1–B8 all flip; B8 is the user-visible one

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

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Split into AI-028 + AI-029 rather than one ticket** (2026-08-07). The install model and the multi-client registration SSOT are separate failures with separate blast radii; the repo's own precedent is #574 being split out of #551 for atomic-PR discipline.
- **PR1 sequenced before the model change.** The loud-vs-quiet fix does not depend on resolving the bootstrap question, and it is what makes the next recurrence visible. Shipping it first means the session produces a durable improvement even if the model migration proves deeper than expected.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? **yes** — a fault-tolerance guard and a broken-state guard that share an exit code make failure invisible; "quiet by design" and "quiet because broken" must be distinguishable at the observable surface.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no — likely no; ADR-015 in the hive repo owns the mechanism decision, this repo consumes it>
- [ ] New pattern candidate for `00_meta/patterns/`? <yes / no — candidate: "a health signal that cannot distinguish healthy-idle from broken is not a health signal". Only if it recurs outside this incident.>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-028-hive-install-model-migration/` -> `specs/archive/AI-028-hive-install-model-migration/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
