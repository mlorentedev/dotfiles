---
id: "CLI-091b-machine-convergence"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-29"
issue: "mlorentedev/dotfiles#1843"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-091b: Machine convergence (track B of #1843)

## Why

<!-- from issue #1843: CLI-091: [EPIC] Fixes reach machines and repos by themselves, and dotf becomes usable outside this repo -->

A machine converges today by running a 3,920-line pair of setup scripts. They decide what to install by presence, not by version (#1265). They copy files unconditionally, so a re-run cannot report `changed=0` (#1266). Doctor and `dotf tools` answer "what is in the catalog" from different files (#1381). Nobody can ask a machine what a convergence run *would* change without running it. Track B moves that decision into `dotf`: a plan that can be read before it is applied, from one source of truth, so that ADR-041's convergence run has something safe to call.

## What

Track B lands row by row (#1843). Each row adds its acceptance criteria here before its PR starts, and ticks them in `tasks.md` when it merges.

Rows B4 and CLI-067 (#1381), in this first PR:

- `dotf tools install --dry-run [name]` prints what `install` would do for each selected tool (the installed version, the pin and the action) and changes nothing.
- `dotf tools list` and `dotf tools install` read `packages.json` from the checkout first and fall back to the deploy mirror, as doctor does. The checkout itself is found cwd first (`env.RepoDir`), and doctor still finds it `DOTFILES_REPO_DIR` first, so one shared resolver waits for #1418.

Later rows, whose criteria are added when they start: B5 (pins as floors in doctor and setup, #1262 and #1265), then B1-B3 and B6-B10.

## Out of scope

- The order doctor uses to resolve the checkout itself (`DOTFILES_REPO_DIR` before the cwd, the reverse of `env.RepoDir`). That is #1418, and changing it touches every doctor check.
- Any change to the setup scripts. They keep calling `dotf tools install` as they do today.
- A machine-readable (`--json`) plan. It belongs with `dotf converge --plan` (B6).

## Risks / open questions

- A plan is only honest if it runs the same decision code as the apply. The dry run must call the same `decideAction` and version probe as `Install`, not a copy of them.
- Resolving the catalog checkout-first changes which file `dotf tools` reads when it runs from a checkout whose mirror is stale. That is the intended fix (#1381), and setup, the production caller, copies the catalog into the mirror before calling `dotf tools install`, so its behaviour does not change.

## Acceptance criteria

- [ ] **AC1:** `dotf tools install --dry-run` prints one row per selected tool with its installed version (or `absent`), its pin and the action `install` would take (`install`, `upgrade`, `skip`, or `unsupported` when the tool has no build for this platform). The action comes from the same `decideAction` as the apply path. It fetches nothing, runs no package manager, writes nothing and exits 0.
- [ ] **AC2:** `dotf tools list` and `dotf tools install` read `packages.json` from the checkout that contains the working directory when it has one, and from the deploy mirror otherwise. A test with different catalogs in the checkout and the mirror shows the checkout wins, and a test with no checkout shows the mirror is used.

## References

- EPIC #1843, track B; ADR-041 (convergence order); ADR-036 (pins are floors); ADR-030 (checkout-first resolution)
- #1381 (CLI-067), #1262, #1265, #1418
