# Run ops-053-20261017T0807Z

**Verdict:** ended `queue-empty` at about 15:05Z. Started 08:07Z; expiry 20:07Z. 5 PRs merged, 1 item parked, 1 not taken, 0 unmerged.

Waiting on a human: the OPS-053 independent review and archive, then one release and one role declaration on the Mac. Nothing irreversible is pending. Everything landed is code behind an opt-in role (ADR-019) or a converge step that writes only the global git config.

## 1. Waiting on you, irreversible first

1. **Already landed, under the merge grant** (each PR has a `## Review triage` comment, CI green on the merged head):
   - #2682 `bc0185d0`: OPS-053 PR 1, the `schedule` package plus the launchd, systemd --user and schtasks backends.
   - #2693 `ba750094`: OPS-053 PR 2, the `self-deploy` converge step and the `self-deploy` machine role. The Linux timer block and the Windows twin were deleted from `setup-linux.sh` and `setup-windows.ps1`.
   - #2695 `adb56ad0`: OPS-053 PR 3. `dotf update` records its last run to `$XDG_STATE_HOME/dotfiles/update/last.json`, and doctor gets a *Self-deploy* section (trigger state plus last run).
   - #2694 `57a2dcb7`: PLAT-001c, the 16 casks from the macOS runbook.
   - #2697 `099961b5`: PLAT-001c / #2201, the `git-lfs` converge step and a doctor *Git LFS* section. Closes #2201. Its review fix (`2001e284`) was **not re-reviewed** by pr-agent (below its 3-commit threshold), which the triage discloses.
2. **OPS-053 archive gate:** run `dotf spec review OPS-053-self-deploy-trigger` (the reviewer must not be me), then `dotf spec archive` on PASS. That frees the `--over-wip-limit` override. #2502 stays open until then.
3. **The Mac's self-deploy, after the next release is installed:** `dotf env role add self-deploy`, then `dotf converge --only self-deploy`. **Not before the release:** an unreleased role in the real `machine.json` breaks every installed `dotf converge` on the machine. That happened during this run and was rolled back within the hour (lesson 468).
4. **The Mac's lfs filter:** `dotf converge --only git-lfs --plan` on the real HOME reports `1 to change`. The first converge with a release that carries the step writes it. Nothing to do by hand.
5. **Live `changed=0` on Linux and Windows** for self-deploy and git-lfs needs msi and the work-lab box. Here they are covered by unit tests plus `GOOS=windows go vet ./...` only, as each PR says.

## 2. Parked, with the question restated

- **#2399** (CLI-108, the Windows half of the Claude MCP registration twin). Board Status is Blocked (verified by GraphQL). Question: (A) take it in #2013's Windows batch, after hive's Windows catalog entry, at the work-lab box; or (B) narrow it to a guard against `claude mcp add` growing back in setup-linux.sh, and file the Windows deletion under #2013. The cheapest unblock is the owner choosing A or B on the issue.

## 3. Not taken

- **#1331** (CLI-060, verifying scheduled jobs). It needs its own spec, and the WIP limit is full (15 active against a limit of 10; OPS-053 used the one override). Its scope was narrowed on the issue: self-deploy is covered by #2695, so vault-maintenance and hive-upgrade remain, reusing `schedule.Backend.Inspect` and `System.Scheduler`.

## 4. Filed while working (debt recorded, not work invented)

- #2685: env-persist reads any `launchctl print` failure as an unloaded agent.
- #2696: the Windows self-deploy task drops `Job.Env`, so a session-only XDG home never reaches the 03:00 run. Options on the issue; A is recommended.

## 5. Done but unmerged

None.

## 6. Evidence index

| Item | Verification | Result |
|---|---|---|
| #2682, #2693, #2695 | `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...`, pinned golangci-lint 2.12.2; bats for every file that references a changed script (41 files for #2693) | exit 0; 0 issues; bats exit 0. Detail in `specs/OPS-053-self-deploy-trigger/verification.md` |
| OPS-053, launchd half live | plan, apply, `changed=0`, `launchctl print gui/$(id -u)/<label>` on the macmini (dev build), then rolled back | recorded in the OPS-053 verification.md and lesson 468 |
| #2694 casks | from-zero (macos-latest) log: 16 casks installed via brew-cask | CI green |
| #2697 git-lfs | Go gates as above; 2 mutations killed; live in a scratch HOME and scratch XDG_CONFIG_HOME: plan, applied, `0 changed`; differing global key installs nothing; differing system key applies, then `0 changed` | exit 0 each; `specs/PLAT-001c-tools-via-mise/verification.md` |
| CI on each merged head | `gh api …/check-runs` | all success or skipped. #2694's first test-windows run hit a winget source outage (0x8a15000f); it passed on re-run |

## Host

At disarm a `caffeinate -i -t 32000` wake-lock was active on the host (pid 36157, about 8.9h). This report does not verify that it was this run's or that it covered the window up to 20:07Z. GitHub GraphQL was avoided: reads went through `gh api` REST, per the arming brief.
