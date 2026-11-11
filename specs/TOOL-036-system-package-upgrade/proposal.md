---
id: "TOOL-036-system-package-upgrade"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-10"
issue: "mlorentedev/dotfiles#3283"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# TOOL-036-system-package-upgrade

> **Naming**: file lives at `<repo>/specs/TOOL-036-system-package-upgrade/proposal.md`. `TOOL-036-system-package-upgrade` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3283: Keep catalog system packages current: refresh, upgrade and clean up through brew, apt and winget -->

Catalog `system` entries (Homebrew formulae and casks, apt packages, winget ids) converge on presence alone, so once installed they never move. Nothing refreshes the managers either: `brew update` never runs, and on a converged Mac both the Homebrew checkout and the formula index age without limit. The owner wants every machine on current packages without running anything by hand (#3283). Pinned tools are out of scope: they keep their pins and move by PR (#3284, ADR-036).

## What

A new converge step, `system-upgrade`, runs after `tools`:

1. **Refresh** the manager's index: `brew update` (which also updates Homebrew itself), `apt-get update`, `winget source update`. A refresh is reported as info and never counted as a change, because it moves the index on every run and `changed=0` would be unreachable. A plan runs it too, since it cannot otherwise know what is outdated. It changes the manager's own metadata, never an installed package. A refresh that fails (offline, a mirror down) makes the step skip with the reason. It never fails the run, because the step runs unattended every day.
2. **Upgrade** the catalog's system entries the manager reports outdated:
   - darwin: `brew outdated --json=v2`, limited to the catalog's formulae and casks, then `brew upgrade <formulae>` and `brew upgrade --cask <casks>`.
   - linux: `apt list --upgradable`, limited to the catalog's packages, then `apt-get install --only-upgrade -y <pkgs>`.
   - windows: `winget upgrade` listing limited to the catalog's ids, then `winget upgrade --id <id> -e --silent` per id.
   Each upgraded package counts as one change.
3. **Housekeeping** after an apply that upgraded something: `brew cleanup` and `brew autoremove` on darwin, `apt-get autoremove -y` on linux. Winget has no equivalent.

The catalog's entries are the *targets*. Their dependencies follow the manager: `brew upgrade` brings a formula's dependencies along. Housekeeping is global, but it only removes what the manager already calls unneeded: superseded versions, caches, and leaf dependencies nothing requested. A package installed by hand and outside the catalog is never upgraded by name.

**Privilege.** Brew refuses root and needs none. Apt needs root. The step probes `sudo -n true` and, when that fails, skips the upgrade with "needs root: run `dotf converge --only system-upgrade` from a terminal". It never prompts, because the daily run has no terminal. A cask flagged `needs_sudo` follows the same rule. Winget runs per user; a package that needs elevation fails on its own line and does not fail the others.

**Probe** (the post-condition): after an apply, a second listing reports no outdated catalog entry, except one skipped for privilege, which is named.

`packages.json`'s `$comment` "it is never upgraded" is replaced with the rule above, and ADR-036 gets an amendment.

## Out of scope

- Pinned tools (`packages.json` versions, `versions.conf`) stay pinned. #3284 bumps them by Renovate PR.
- Upgrading packages the catalog does not name (`brew upgrade` with no arguments, `apt upgrade`).

## Risks / open questions

- A daily upgrade can pull a breaking change into a catalog package. That is accepted: these entries have no pin, so there is nothing to reproduce, and the alternative is staleness without limit. A package that needs reproducibility belongs in a pinned channel (mise, github-release).
- `brew update` takes about 5-30 s when online. That is acceptable for a daily run.
- Windows can only be checked through unit tests with the fake runner, plus `GOOS=windows go vet` and the CI `windows-latest` Go tests. Real acceptance is a `dotf vm windows` run after the release, the same posture as #3142 and #2432.
- `winget upgrade` of App Installer (winget itself) while winget runs: it is not in the catalog today, so it is not targeted. Adding it is a catalog change for later.

## Acceptance criteria

- [ ] AC1: On darwin, a plan lists the outdated catalog formulae and casks (and only those) as changes, and reports the refresh as info.
- [ ] AC2: An apply upgrades them, runs `brew cleanup` and `brew autoremove`, and a second run reports `changed=0`.
- [ ] AC3: A formula the catalog does not name is never passed to `brew upgrade`, even when outdated.
- [ ] AC4: On linux without non-interactive sudo, the step reports a skip naming the reason and the run does not fail. With sudo, `apt-get install --only-upgrade` runs on the outdated catalog packages only.
- [ ] AC5: On Windows, `winget upgrade --id <id> -e` runs once per outdated catalog id. One failing id does not stop the others; the step fails with an error that names it, and only the ids that upgraded are reported as upgraded. The id is read from the listing's Id column alone.
- [ ] AC6: A refresh that fails makes the step skip with the manager's error, and the run continues.
- [ ] AC7: `packages.json`'s `$comment` and ADR-036 state the new rule (unpinned system entries upgrade; pinned entries move by PR).
- [ ] AC8: The daily `dotf update` runs this step on a day HEAD did not move (it lands as a second PR). A fast-forward, a reinstall and every skip do not run it a second time. A failure of the step on that day exits non-zero with status `upgrade-failed`. A skip of the step exits 0 with status `upgrade-skipped` and the skip reason. Both are recorded in `last.json`, so a machine that never upgrades is visible.

## References

- Bitácora board: #3283
- Related ADR: `docs/adr/adr-036-install-channels.md` (amended here), `docs/adr/adr-044-toolchains-and-pinned-clis-via-mise.md`
- Companion: #3284 (Renovate for pinned tools)
