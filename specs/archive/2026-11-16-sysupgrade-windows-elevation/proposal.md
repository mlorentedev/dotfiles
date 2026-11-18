---
id: "2026-11-16-sysupgrade-windows-elevation"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-16"
issue: "mlorentedev/dotfiles#3553"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
wip_override: "defect: on a non-elevated Windows machine system-upgrade fails every converge and leaves 18 steps not reached (#3553) (10 active, limit 10, 2026-11-16)"
---

# 2026-11-16-sysupgrade-windows-elevation

## Why

On a Windows machine whose account cannot elevate, `dotf converge` fails at `system-upgrade` on every run: a
machine-wide winget package (the VC++ redistributable, PowerShell, Docker Desktop) cannot be upgraded from a
non-elevated process, the step joins each failed id into its error (TOOL-036 AC5), and the runner stops there. On
2026-11-16 that left 18 later steps not reached and `dotf update` recording `install-failed` (#3553). Linux and
macOS already treat a privilege the run cannot obtain as something to report, not a failure: apt without `sudo -n`
skips, a `needs_sudo` cask is left waiting. Each attempt also raises an administrator prompt on screen, once per
package and per run, that the account cannot answer.

## What

On Windows, when the process is not elevated, the step asks winget which outdated ids are installed machine-wide
(`winget list --scope machine`, read-only) and never attempts those: no installer runs, so no prompt appears. An
id it does attempt and that fails is left waiting instead of failing the step. The step exits 0, counts the ids
that did upgrade, and its detail names each waiting id with its reason (machine-wide install, or winget's exit
status) and how to run the step elevated. A plan names them too. The probe does not re-fail the step on those ids. A quiet
day's `dotf update` records `upgrade-skipped` with that detail, so `last.json` keeps showing a machine that never
upgrades them. An elevated run is unchanged: a failing id fails the step and the probe names it.

## Out of scope

- The runner's fail-fast order (a failed leaf step keeping unrelated steps from being reached). That is a
  converge-wide contract and gets its own issue and decision.
- Classifying winget exit codes. The rule is the process token, because the probe keeps no state and must reach
  the same answer as the reconcile.
- Pinning or excluding packages per machine, and any attempt to elevate from dotf.

## Risks / open questions

- A user-scope installer may still ask for elevation; winget does not say so before running it. Such an id
  prompts once per run and then waits with its status. None was observed on 2026-11-16.
- If `winget list --scope machine` fails, nothing is attempted and every outdated id waits with that reason.
- A non-elevated run also leaves waiting an id that failed for another reason (network, installer bug). Accepted:
  the id and winget's status stay in the step's detail on every run, and in `last.json` on quiet days, and an
  elevated run still fails on it.
- A full converge (a day the checkout moved) does not record `upgrade-skipped`; there the waiting ids are visible
  only in the converge report. Same as a `needs_sudo` cask on macOS today.

## Acceptance criteria

- [ ] AC1: Windows, not elevated: no `winget upgrade` runs for an id installed machine-wide, on apply or plan;
      an id that is attempted and fails does not fail the step; the other ids still upgrade and are counted.
- [ ] AC2: the step's detail names each waiting id with its reason (machine-wide install, or the exit status
      alone) under a stable marker; when the scope cannot be read, nothing is attempted.
- [ ] AC3: the probe returns nil on a non-elevated Windows run that left ids waiting.
- [ ] AC4: Windows, elevated: every outdated id is attempted, the scope is not queried, a failing id fails the
      step and the probe names it (TOOL-036 AC5 unchanged).
- [ ] AC5: a quiet day's `dotf update` reads a step that left ids waiting as a skip reason, so it records
      `upgrade-skipped`.
- [ ] AC6: the tree builds and vets for windows, linux and darwin; elevation is read from the process token on
      Windows and from the euid elsewhere.

## References

- Issue: mlorentedev/dotfiles#3553
- `specs/archive/TOOL-036-system-package-upgrade/proposal.md` (AC5, the behaviour narrowed here)
- `docs/lessons/lesson-609-a-failure-the-account-cannot-fix-is-a-report-not-an-error.md`
