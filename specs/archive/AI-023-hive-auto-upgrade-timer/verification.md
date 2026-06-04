---
tags: [spec, verification]
created: "2026-06-03"
---

# Verification - AI-023-hive-auto-upgrade-timer

## Evidence

- [x] Timer cadence + catch-up -> `grep -qF 'OnCalendar=*:0/15' systemd/hive-upgrade.timer && grep -qF 'Persistent=true' systemd/hive-upgrade.timer`
- [x] Oneshot runs the upgrade via absolute uv -> `grep -qF 'Type=oneshot' systemd/hive-upgrade.service && grep -qF 'ExecStart=%h/.local/bin/uv tool upgrade hive-vault' systemd/hive-upgrade.service`
- [x] Linux deploys units + enables timer inside the version gate -> `enable --now hive-upgrade.timer` line number > `sort -V` line number
- [x] Linux strips the legacy cron (single owner) -> `grep -qF "grep -v 'uv tool upgrade hive-vault' | crontab -" setup-linux.sh`
- [x] Windows registers the daily DotfilesHiveUpgrade task, gated -> `Register-ScheduledTask -TaskName $hiveUpgradeTask` line number > `[version]'1.32.0'` line number
- [x] `setup-linux.sh` syntax + lint -> `bash -n setup-linux.sh` clean; `shellcheck` adds no findings in the added block (842-895)
- [x] Windows block ASCII-only -> bats test 16 (PSScriptAnalyzer non-ASCII guard)
- [ ] (post-merge, maintainer) Real fleet rollout -> on each machine: `systemctl --user list-timers hive-upgrade.timer` active; `uv tool list` tracks latest; the legacy 9am cron is gone

## Test status

- `bash -n setup-linux.sh` -> OK
- `shellcheck -f gcc setup-linux.sh` -> no findings in the added block (lines 842-895); pre-existing findings elsewhere untouched
- `bats tests/hive-upgrade-timer.bats` -> 18/18 passing
- `bats tests/setup-linux.bats` -> 82/82 passing (no regression)
- `bats tests/setup-windows.bats` -> passing (PSScriptAnalyzer/parse skipped: pwsh unavailable on this host; covered by lint-powershell CI + the ASCII bats guard)
- pwsh parse of `setup-windows.ps1` -> not run here (pwsh unavailable); the block mirrors the existing `Register-ScheduledTask` self-healing pattern already in the file (weekly vault maintenance task) and stays ASCII-only.
- Manual smoke (Linux, this machine): deferred to the maintainer's rollout (running the setup mutates `~/.config/systemd/user/` + the crontab).

## Decisions made during implementation

- **Wall-clock + catch-up, not interval-since-last-run.** `OnCalendar=*:0/15` +
  `Persistent=true` (maintainer's call) means a suspended laptop converges to the
  latest hive-vault on resume instead of drifting. Matches the "always latest" goal.
- **Static unit files in `systemd/`, not an inline heredoc.** SSOT + greppable by
  bats + reviewable in the PR, consistent with `mcp-servers.json` / `env-contract.json`.
  `ExecStart` uses the absolute `%h/.local/bin/uv` because a `--user` oneshot gets a
  minimal PATH that may omit `~/.local/bin`.
- **Co-located in the existing `hive >= 1.32.0` gate.** The upgrade timer is the
  policy that feeds the daemon's restart-on-upgrade; installing them together (or
  not at all) keeps "daemon + feeder" a single unit and honours "never route an
  auto-upgrade onto an old hive".
- **Cron strip only after a successful enable.** An old-hive machine that skips the
  gate keeps its manual cron, so it is never left with zero upgrade owners.
- **Non-fatal everywhere.** No systemd `--user` / Task Scheduler downgrades to a
  warning; the setup `prerequisite_command` still upgrades hive on every run, so the
  machine never goes stale silently.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? maybe -- "co-locate the policy timer with the
  mechanism it feeds, under one version gate" is a small reusable shape; capture only
  if it recurs.
- [ ] ADR-worthy? no -- ADR-011 in the hive repo owns the daemon-model decision; this
  is its fleet-side policy arm.
- [ ] New pattern? no -- single-project rollout mechanics.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/AI-023-hive-auto-upgrade-timer/`
- [ ] hive #176 ticked / closed with the PR link after multi-machine rollout
