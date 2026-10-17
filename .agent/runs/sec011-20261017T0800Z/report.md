# Run sec011-20261017T0800Z: report

**Verdict:** the run ended `queue-empty` at 10:10Z, after 6 queue items in about 2h10m.

- 2 PRs landed: #2675 and #2692. #2374 closed.
- 3 PRs held, by the owner's own rules: #2672, #2683 and #2681.
- 3 items parked with a question: #2388, #2464 and row 215.
- Nothing was applied to any host. Every apply in this run is still the owner's, at a sudo prompt or on the lab box.

## Waiting on you (what landed first)

**Landed on main under the standing grant.** Both are code only; no apply has run.

- #2675, `8f395195` (squash of `ae5833c`): the macOS AC power profile through `pmset` (row 210, power sub-row).
  - Reviewed twice by pr-agent. The triage is on the PR.
  - The row stays `[ ]` until you run the attended apply on the macmini. Then run it a second time and check it reports `changed=0`.
  - It reaches the installed `dotf` only once the release that includes it is cut (release-please PR #2680 is yours).
- #2692, `31df521`: the "mesh does not come back after an unattended reboot" note, in the plan and as a WARN in doctor. This merge closed #2374.
  - #2374 was closed by a read-only measurement on msi: Ubuntu's override sets `sleep-inactive-ac-timeout = 0`, the type is `'nothing'`, and there have been 0 suspends on AC since 2026-10-13.
  - The runbook names the step change that a GNOME without that override would need.

**Held PRs.** None of these is merged by an agent.

- #2672: the Windows half (sshd, keys, firewall). Rebased on 0.73.0 and triaged. Since then:
  - `e1e11251` applies one finding: a missing sshd is now read from sc.exe's exit code, not from its output.
  - One finding was declined with evidence: `net.ParseCIDR` rejects dotted masks.
  - It closes only with acceptance on the work-lab box. The open conditions are in `verification.md`, row 214.
  - It touches `cli/internal/converge/records.go`, as #2682 (self-deploy) does. Whoever lands second rebases.
- #2683: the Windows power policy through `powercfg` (standby, hibernate, lid action; AC only). Stacked on #2672 and triaged.
  - `500dc172` tightens the index parser.
  - The lidless-box question is left for the lab: skipping a missing setting needs powercfg's localized answer, measured first.
- #2681 (draft): the pf anchor with its LaunchDaemon and commit-confirm timer. Retargeted to main and brought up to date with `9674941b`.
  - Nothing loads pf until your three root measurements are recorded on #2356.

**The attended run that closes the pmset row.** Once `dotf` carries #2675:

1. `dotf converge --only workstation-target` on the macmini, at the sudo prompt.
2. The same command again, which must report `changed=0`.

The rollback line is in the runbook: `pmset -c disksleep 10 autorestart 0`, the macmini's measured state.

## Parked, with the question

- **#2388, SEC-014: retire the target when the role is removed.** Board: Blocked. Park comment 6231614858.
  - (A) A PLAT-001b row: retire only what a witness file says the step wrote; record the listener state before the role; refuse to stop the listener `SSH_CONNECTION` arrived on. Two PRs, no new spec.
  - (B) Its own spec once WIP frees a slot.
  - Recommended: (A).
  - Cheapest unblock: say "A".
- **#2464, SEC-015: the dead `*-lan` jump aliases.** Board: Blocked. Comments 6231643209 and 6231646050.
  - (A) rpi4 becomes the declared LAN jump host: kubelab gets a `Match User manu` block with `AllowTcpForwarding yes` and a `PermitOpen` for the four nodes.
  - (B) Drop the aliases. The onward hop then needs a key on rpi4, because the hardening also sets `AllowAgentForwarding no`.
  - Re-measured tonight: still `administratively prohibited`.
  - Cheapest unblock: choose; (A) is a kubelab change.
- **Row 215, host keys pinned.** Not started: checking the fingerprints out of band is your act, not an agent's.

## Done but unmerged

#2672, #2683 and #2681 above. All three are held by rule, none for lack of work.

## Evidence index

| Item | Command | Exit |
|---|---|---|
| powercfg (#2683) | `go build ./... && go vet ./... && GOOS=windows go vet ./...`; `go test ./...`; `golangci-lint run` and `GOOS=windows golangci-lint run` on sshtarget, converge, doctor | 0 / 0 / 0 issues |
| sc.exe fix (#2672) | `go test ./...`; `golangci-lint run` (both GOOS) on sshtarget | 0 / 0 issues |
| pmset review fixes (#2675) | `go test ./internal/sshtarget/... ./internal/converge/... ./internal/doctor/...`; lint | 0 / 0 issues |
| #2675 land | `dotf pr triage-queue --pr 2675` then `dotf pr land 2675` | 0, `[MERGED] #2675 at ae5833c` |
| mesh-service on main (#2692) | build, vet (both GOOS), `go test ./...`, lint | 0 / 0 issues |
| #2692 land | `dotf pr triage-queue --pr 2692` then `dotf pr land 2692` | 0, `[MERGED] #2692 at 31df521` |
| pf merged with main (#2681) | build, vet (both GOOS), `go test ./...`, lint | 0 / 0 issues |
| #2374 | `gh api repos/.../issues/2374` | `closed / completed` |

Review triage comments:
- #2672: 6232223332
- #2675: 6232429525
- #2692: 6232581225
- #2683: 6232613188

## Observations, not acted on

- **pr-agent's round 2 on #2675 (`ae5833c4`) re-reported two round-1 findings that the reviewed commit had already fixed.** At `ae5833c4` the row reads `[ ]` and the test calls `Apply`. Both were declined as stale, with the file content quoted.
  - I did not file a ticket. I could not tell whether the cause is the review-state carry-forward or the model, and it was one case. If it recurs, it belongs in #1923.
- **The push gate skips a single post-triage commit.** The threshold is 3. I asked for a `/review` on #2675 so the fixes did not land unreviewed. This matches #1756's measurement.
- **From 08:19Z to about 08:45Z, the installed `dotf` 0.73 refused every converge.** A peer session's unreleased build had written the role `self-deploy` into `machine.json`. That peer removed it and owns the lesson. In the meantime this run used a scratch `XDG_CONFIG_HOME` for its read-only plans.

## Budget and host

- Items: 6 of `max_items` 7.
- Consecutive failures: 0.
- The envelope was never widened.
- No frozen path was touched. Every changed path is under `cli/internal/sshtarget/**`, `converge/workstation*`, `doctor/checks_workstation*`, the PLAT-001b spec, the runbook, and this report.
- Host: the macmini was on AC power with `caffeinate -i -t 41400` held for the whole window.

## Knowledge

- Lesson: none. Tonight's failure classes are already written up: lesson 225 (stacked PRs re-conflict after their base squash-merges) and lesson 349 (an exit error decides, not its text).
- ADR: none. Everything implements owner decisions already recorded on #2356.
- Runbook: `docs/runbooks/guide-workstation-ssh-target.md`, changed by #2675 and #2692 (on main), and by #2683 (held).

## Next action

Cut the release that carries #2675. Then run the attended `dotf converge --only workstation-target` twice on the macmini, and tick the pmset row with the output.
