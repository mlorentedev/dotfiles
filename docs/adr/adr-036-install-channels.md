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
   | node-distributed agents and CLIs (`pi`, `opencode`, `bw`, `copilot`) | npm global | `npm` |
   | static single-binary releases (`sops`, `age`, `eza`, `zoxide`, `dotf`) | GitHub release, sha256-verified | `github-release` |
   | tools with no cross-OS channel (`git`, `gh`, `jq`, `uv`) | the OS package manager (`winget` / apt / official installer) | setup script |

   A tool in the first two classes is declared in `packages.json` and converged by `dotf tools install` on both OSes; the setup scripts stop carrying per-OS install blocks for it.

2. **`packages.json` is the pin SSOT for every catalog tool.** A catalog tool's version appears nowhere else; `versions.conf` keeps only the pins the shell layer still consumes directly. `dotf doctor` reads the catalog pin for those tools.

3. **Version detection happens once, in Go.** `dotf tools version <name>` returns the first semver in `<name> --version`'s output; setup scripts call it instead of parsing. `dotf tools install` already probes the same way.

4. **A second copy on PATH is a finding, not a state to converge silently.** `dotf doctor` WARNs when an npm catalog tool resolves from more than one PATH directory and names them; the catalog converges the copy it owns and the operator removes the other channel's. Setup never prints a manual instruction for it.

5. **Migration is by the same mechanism.** Retiring a channel (winget `SST.opencode`, the opencode curl script, the `~/.opencode/bin` PATH line in the rc files) leaves the old copy where it is; the next `dotf tools install` converges the declared channel and doctor reports the leftover.

## Amendment 2026-08-28 (AI-038, #1321)

`copilot` was listed as "no cross-OS channel" on the day of this ADR; measured on
the Windows work box, GitHub publishes the CLI as `@github/copilot` on npm
(1.0.81), the binary self-updates behind winget's back (registry 1.0.78 vs
binary 1.0.80), and Linux boxes installed nothing at all. It moves to the npm
class: one `packages.json` entry, `dotf tools install` on every OS, and the
shadowed-install WARN for a leftover winget copy. Turning the CLI's own
`autoUpdate` off belongs to its user settings file (`settings.json`, which the
repo does not deploy yet — `config.json` is rewritten by the CLI, measured on
the same box) and is tracked under #1322.

## Amendment 2026-10-03 (#1993, AI-028)

A fourth class: **PyPI-distributed tools**, installed with `uv tool install
<package>==<pin>` (source type `uv-tool`). The first one is `hive`
(`hive-vault`), whose only installer until now was a side effect of Claude's MCP
registration (`prerequisite_command` in `mcp-servers.json`).

The class has one twist that the first two do not: its channel is not the same
on every OS. On Windows hive owns a versioned install layout, because Windows
cannot replace an executable that is in use (hive ADR-019). So the source
declares `platforms: [linux, darwin]`. On a platform it does not list, `dotf
tools install` reports the tool as unsupported and skips it; a skip is not a
failure. `Source.Platforms` is available to every source type. A
`github-release` tool already gets the same effect from its `asset` map.

The pin is a floor, as for every catalog tool. That matches hive ADR-020, under
which hive notifies and the user upgrades: setup no longer upgrades hive on
every run. The twins keep running `prerequisite_command` until the cutover
deletes their MCP loops.

## Consequences

- `setup-linux.sh` and `setup-windows.ps1` lose their opencode install blocks and the winget loop's version-pin machinery (no winget tool carries a pin any more). The hive `hive service` gate probes `hive --version` through `dotf tools version` instead of `uv tool list`, which stopped seeing a healthy install when hive moved to its own installer (AI-028, #791).
- `pi` is not moved in the same change: its Linux block installs with `--prefix ~/.local --ignore-scripts` for the Orca per-node-version PATH trap (#426), a behaviour `installNpm` does not yet carry. It follows once `packages.json` can express those flags (tracked on #1294's follow-up).
- The Linux box will report the orphaned `~/.opencode/bin` copy the first time doctor runs after this lands; removing it is the migration.

## References

- AI-034 (#1294), CLI-029 (`packages.json`, `dotf tools`), AI-028 (#791), #1265 / #1262 / #1267 (convergence and pin semantics), `pattern-setup-script-idempotence`.

## Amendment 2026-09-30 (ADR-041, #1843)

`dotf` itself gains a second channel for its own binary: `stable`, the pinned release, and `edge`, the newest tested build of `main` (ADR-041). The pin is a floor on both: convergence never downgrades an installed build, and only an explicit `dotf self-update --version` goes down.

## Amendment 2026-10-06 (ADR-044, #2013)

Toolchains (Java, Go, Python, Maven, Node) and the pinned single-binary CLIs (age, zoxide, shellcheck, jq, bats, golangci-lint, sops, direnv, fzf, herdr, lazygit) move to **mise**, rendered from `versions.conf` by `dotf tools sync` and installed with `mise install --locked` against a committed `mise.lock` (ADR-044). The `github-release` source type keeps the two bootstrap binaries, mise and `dotf`. The npm, uv-tool and class-3 rows are unchanged; mise replaces Homebrew as a toolchain channel on macOS, and what Homebrew may still install there is #2013 D4, still proposed. A pin that moves to mise lives only in `versions.conf`, so decision 2 still holds with that file as the single place.

## Amendment 2026-10-21 (AI-028 PR6, #791)

The PyPI class gets its Windows half. The 2026-10-03 amendment declared hive `platforms: [linux, darwin]`, so the catalog installed nothing on Windows. The setup script then filled that gap with `uv tool install --upgrade hive-vault`, which is the copy hive ADR-019 rules out, and on Windows Arm64 the install failed in silence (AI-028 Baseline 2). This amendment supersedes that `platforms` line.

A `uv-tool` source may now declare **`self_install`**: the OSes on which the package installs itself through its own installer instead of `uv tool`, and that installer's arguments.

```json
"source": {"type": "uv-tool", "package": "hive-vault",
           "self_install": {"platforms": ["windows"], "args": ["self-upgrade", "{version}"]}}
```

On a listed OS, `dotf tools install` runs `uv tool run --refresh-package <package> --from <package>==<pin> [--python <series>] <name> <args>` (`uvx` is that command), with `{version}` expanded to the pin. Everywhere else it runs `uv tool install` as before. One entry and one class still cover hive on every OS. This is a field of the class and not a new source type: the class already says its channel differs by OS, and a type named after one tool would be the only one of its kind.

Four rules are part of the contract:

1. **The pin is a floor, and the probe enforces it.** hive's `self-upgrade <v>` repoints `current` at `<v>` even when a newer version is current. The catalog never runs it when `<name> --version` already reports the pin or higher. That probe is what keeps decision 2's pin-as-floor; the installer alone would downgrade.
2. **The arguments must carry `{version}`.** A self-installer without a version resolves the newest release itself, which is the unattended upgrade hive ADR-020 forbids. `Load` rejects `args` without it. Installing a version the owner pinned and committed is an install, not an upgrade on the user's behalf.
3. **The tool is probed where the installer puts it, before the install and after.** A self-installer adds its launcher to the per-user persistent PATH (`HKCU\Environment`), which a process started earlier does not see. The probe resolves `<name>` on the process PATH, then on the registered PATH a new process gets (`env.RegisteredPath`, which converge's tools step already reads). Before the install this is what rule 1 stands on: an installed hive that reads as absent would be `self-upgrade`d down to the pin. After it, a fresh install is not reported as failed, and a launcher that never landed is not reported as installed. A registered PATH that cannot be read is a refusal, never an absent tool.
4. **uv is still the manager.** `uvx` comes with uv, so an entry without uv on PATH waits for it, as every `uv-tool` entry does (`PlanMissingManager`).
5. **`no_build` names what uv must not build from source in that run.** uv resolves the package's dependencies before the tool starts, so no guard inside the tool reaches them. Each name becomes `--no-build-package <name>`, and uv takes the newest release with a wheel. hive declares `cryptography`, which has no `win_arm64` wheel at 50.0.2 and whose build needs `link.exe` (AI-028 V6, hive#523). On x64 the flag changes nothing: 50.0.2 has a wheel there.

`platforms` keeps its meaning: an entry with `self_install` and no `platforms` installs on every OS. hive's Windows install stops coming from `mcp-servers.json`'s `prerequisite_command`, and with hive's line removed no entry uses that field. The field's reader in the Windows setup loop is dead code from here, and goes with that loop under CLI-096 (#2000).
