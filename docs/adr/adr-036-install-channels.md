---
id: "ADR-036-install-channels"
type: adr
status: accepted
owner: manu
date: "2026-08-27"
supersedes: []
extends: [adr-020-tooling-cli-go-convergence, adr-025-cross-machine-paths]
issue: mlorentedev/dotfiles#1294
tags: [architecture, decision, tooling, install, packages, windows, linux]
created: "2026-08-27"
---

# ADR-036: Install channels — one channel per tool class, on every OS, pinned in `packages.json`

## Context

Measured 2026-08-27 on the Windows work box: `opencode` was installed three ways across the two machines — npm-global `opencode-ai` (scoop's node `bin`, first on PATH), winget `SST.opencode` (what `setup-windows.ps1` pinned), and the curl install script into `~/.opencode/bin` on Linux. Setup parsed the version as "last token of the first output line", accepted `locked.` as a version, "upgraded" the winget copy, and printed *"another install shadows it in PATH; converge that install instead"* — a manual instruction — on every run. Seven such parse sites existed across both setup scripts (`opencode`, `pi`, `hive`, `yarn`), and no document anywhere recorded which channel a tool was supposed to come from. `specs/archive/AI-014-opencode-windows-bootstrap` had considered npm as a fallback and never decided.

The repository already carried the right primitive: `packages.json` + `dotf tools install` (CLI-029) implements pin-as-floor reconciliation with a semver-regex probe for two source types, `github-release` (sha256-verified into `~/.local/bin`) and `npm` (`npm install -g <pkg>@<version>`), used by `sops` and `bw`. Nothing said when a tool belongs there.

## Decision

1. **Every tool this repository provisions has exactly one install channel**, chosen by the tool's own distribution shape. For the first two classes below that channel is the same on every OS and lives in `packages.json`; the third class has no cross-OS channel by definition and takes the OS package manager or the official installer, one per OS, in the setup script:

   | Tool class | Channel | Source type |
   |---|---|---|
   | node-distributed agents and CLIs (`pi`, `opencode`, `bw`) | npm global | `npm` |
   | static single-binary releases (`sops`, `age`, `eza`, `zoxide`, `dotf`) | GitHub release, sha256-verified | `github-release` |
   | tools with no cross-OS channel (`git`, `gh`, `jq`, `copilot`, `uv`) | the OS package manager (`winget` / apt / official installer) | setup script |

   A tool in the first two classes is declared in `packages.json` and converged by `dotf tools install` on both OSes; the setup scripts stop carrying per-OS install blocks for it.

2. **`packages.json` is the pin SSOT for every catalog tool.** A catalog tool's version appears nowhere else; `versions.conf` keeps only the pins the shell layer still consumes directly. `dotf doctor` reads the catalog pin for those tools.

3. **Version detection happens once, in Go.** `dotf tools version <name>` returns the first semver in `<name> --version`'s output; setup scripts call it instead of parsing. `dotf tools install` already probes the same way.

4. **A second copy on PATH is a finding, not a state to converge silently.** `dotf doctor` WARNs when an npm catalog tool resolves from more than one PATH directory and names them; the catalog converges the copy it owns and the operator removes the other channel's. Setup never prints a manual instruction for it.

5. **Migration is by the same mechanism.** Retiring a channel (winget `SST.opencode`, the opencode curl script, the `~/.opencode/bin` PATH line in the rc files) leaves the old copy where it is; the next `dotf tools install` converges the declared channel and doctor reports the leftover.

## Consequences

- `setup-linux.sh` and `setup-windows.ps1` lose their opencode install blocks and the winget loop's version-pin machinery (no winget tool carries a pin any more). The hive `hive service` gate probes `hive --version` through `dotf tools version` instead of `uv tool list`, which stopped seeing a healthy install when hive moved to its own installer (AI-028, #791).
- `pi` is not moved in the same change: its Linux block installs with `--prefix ~/.local --ignore-scripts` for the Orca per-node-version PATH trap (#426), a behaviour `installNpm` does not yet carry. It follows once `packages.json` can express those flags (tracked on #1294's follow-up).
- The Linux box will report the orphaned `~/.opencode/bin` copy the first time doctor runs after this lands; removing it is the migration.

## References

- AI-034 (#1294), CLI-029 (`packages.json`, `dotf tools`), AI-028 (#791), #1265 / #1262 / #1267 (convergence and pin semantics), `pattern-setup-script-idempotence`.
