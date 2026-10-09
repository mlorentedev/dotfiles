---
id: "dotfiles-runbook-tool-installation"
type: runbook
status: active
tags: [runbook, dotfiles, tools, installation, catalog, mise]
created: "2026-02-22"
updated: "2026-10-06"
owner: manu
---

# Tool Installation

How the tools this repository depends on reach a machine, and how to add one. None of the steps here is manual. Every tool has exactly one channel, declared in the repository and converged by `dotf` (ADR-036, ADR-044). If you find yourself typing `apt install`, `brew install` or `curl | sh` for a tool listed below, its declaration is missing: add it instead.

## Channels

| Tool class | Channel | Declared in | Converged by |
|---|---|---|---|
| Pinned single-binary CLIs (age, bats, direnv, fzf, golangci-lint, herdr, jq, lazygit, shellcheck, uv, zoxide) and toolchains (Go, Java, Python, Maven, Node) | mise | `versions.conf`, lines marked `# mise: cli` | `dotf tools sync`: the CLIs listed today on Linux and macOS (Windows after its session, #2013 H2), the toolchains from Wave 3 |
| The two bootstrap binaries (`dotf` and mise), plus sops | GitHub release, sha256-verified | `packages.json` | `dotf tools install` |
| Node-distributed CLIs and agents (opencode, copilot, bw, pi) | npm global | `packages.json` | `dotf tools install` |
| PyPI tools (hive) | `uv tool` | `packages.json` | `dotf tools install` |
| Tools with no cross-OS channel (git, gh, uv, system libraries, macOS casks) | the OS package manager: apt, winget, Homebrew | the setup script for each OS today; `packages.json` `source.type: system` entries once a release that reads the type is the pin (#2013 D8) | setup today; then `dotf tools install` |

How a pin is read depends on the channel:

- **Catalog pins** (`packages.json`) are floors. An installed version at or above the pin is left alone, and nothing is downgraded (ADR-036 decision 1, ADR-041 decision 6).
- **mise pins** (`versions.conf` lines marked `# mise: cli`) are exact. They are the version mise activates, and the version a committed lock will reproduce (ADR-044). An older version is only activated when the pin itself is lowered in `versions.conf`, which is a reviewed change.
- **A hand-written `~/.config/mise/config.toml`** pinning the same tool takes precedence over the synced file. Measured with mise 2026.10.3: a newer version pinned there is kept, and an older one makes `dotf tools sync` fail, naming the tool.

`setup-linux.sh` still installs some CLIs from fixed URLs. Those blocks are #2013 row W2, removed once a `dotf` release carrying `tools sync` is the `DOTF_VERSION` pin.

## Day to day

```bash
dotf tools list                 # the catalog, with the asset resolved for this OS/arch
dotf tools install --dry-run    # what install would do, tool by tool
dotf tools install              # converge every catalog tool
dotf tools install sops         # one tool
dotf tools version sops         # the version the tool on PATH reports
dotf tools sync --dry-run       # the mise config and the CLIs sync would install
dotf tools sync                 # install the pinned CLIs through mise
dotf doctor                     # pins, drift and leftovers, per tool
```

`install` checks each release asset against the release's checksum manifest. It then runs the staged binary before placing it: a binary that does not execute here, or that reports a version below the pin, is refused and nothing is placed (lesson 337). A tool with no asset for this OS/arch is skipped, with a message saying so, and does not fail the run.

## Adding or bumping a CLI that mise installs

Put the pin in `versions.conf` with the marker on the line before it:

```sh
# mise: cli
LAZYGIT_VERSION=0.66.0
```

- **Name.** `NAME_VERSION` becomes mise's `name`, with underscores turned into hyphens (`GOLANGCI_LINT_VERSION` becomes `golangci-lint`). The short name must resolve to an aqua backend; check it with `mise registry | grep '^name '`.
- **Marker placement.** The marker goes on its own line, never after the value. Every reader of this file (the shell, doctor, `setup-windows.ps1`) skips comment lines, but a trailing comment would be read as part of the version. A marker that is not followed by a `NAME_VERSION=<pin>` line makes `sync` fail.
- **What sync writes.** `dotf tools sync` renders the marked pins into `~/.config/mise/conf.d/dotfiles.toml`; it honours `MISE_CONFIG_DIR` and `XDG_CONFIG_HOME`. That file is generated, so do not edit it. It also sets `disable_update_warning`: mise is pinned in `packages.json`, and its own notice says to run `mise self-update`, which would move it past that pin. Bump `mise` in `packages.json` instead. Sync never touches a hand-written `~/.config/mise/config.toml`; one that pins the same tools is a leftover for the doctor check of #2013 row T3.
- **Copies left in `~/.local/bin`.** Before mise owned these CLIs, setup downloaded some of them into `~/.local/bin` (#2013 W2). An activated shell puts mise first, but a GUI app, launchd or cron finds the old copy and runs a version no pin governs. Once every pin runs through mise, `dotf doctor` warns about each regular file in `~/.local/bin` that has the name of an executable mise provides for a pin, companions included (`uvx`, `age-keygen`). `dotf doctor --fix` removes them. A symlink is left alone, because someone made it on purpose.
- **Verification.** After `mise install`, every tool must resolve through `mise which` and report a version at or above its pin, or sync fails naming it. Sync does not go through PATH, which only gains the mise shims with #2013 row W2b; until then, run a tool with `mise exec -- <tool>`.

## Adding a GitHub-release tool to the catalog

This channel is only for `dotf`, mise, and tools mise cannot install. Everything else goes through mise (ADR-044).

```json
{
  "name": "mise",
  "version": "2026.10.3",
  "profile": "full",
  "source": {
    "type": "github-release",
    "repo": "jdx/mise",
    "asset": {
      "linux/amd64":   "mise-v{version}-linux-x64",
      "linux/arm64":   "mise-v{version}-linux-arm64",
      "darwin/amd64":  "mise-v{version}-macos-x64",
      "darwin/arm64":  "mise-v{version}-macos-arm64",
      "windows/amd64": "mise-v{version}-windows-x64.exe",
      "windows/arm64": "mise-v{version}-windows-arm64.exe"
    },
    "checksums": "SHASUMS256.txt"
  }
}
```

- **`asset`** maps a platform to the release's raw-binary filename.
  - A key is either a GOOS (`linux`, `darwin`, `windows`) or a GOOS/GOARCH pair, where GOARCH is `amd64` or `arm64`. The pair wins over the GOOS key.
  - Templates expand `{version}` and `{goarch}`, using Go's arch names.
  - Use pairs when the release spells arches or OSes its own way (`x64`, `macos`).
  - A platform with no key is not supported. `dotf` rejects a key that names no known platform, so a typo fails loudly instead of being skipped everywhere.
- **`checksums`** is the release's sha256 manifest. Its lines may list the asset as `name`, `./name` or `*name`. A tool whose release publishes no manifest cannot use this channel.
- **`platforms`** (optional) limits a tool to some OSes. It is meant for npm and uv-tool sources; a release tool expresses the same thing through its `asset` keys.
- Before relying on the entry, check it with `dotf tools list` and `dotf tools install --dry-run` on each OS you declared. The dry run runs the same entry checks as install: a release with no `checksums`, or an npm or uv-tool source with no `package`, shows as `refused (<reason>)`, and the dry run exits non-zero. An `unsupported` row names its cause: no asset for this OS/arch, or an OS the entry's `platforms` leaves out.

## Adding a system package to the catalog

`dotf` reads `source.type: "system"`: a package the OS manager owns, named once per manager. **No entry ships yet.** The installed `dotf` fails on a type it does not know, so entries wait for the release that carries this reader to be the `DOTF_VERSION` pin (#2013 P5b).

```json
{ "name": "gh", "source": { "type": "system", "apt": "gh", "brew": "gh", "winget": "GitHub.cli", "command": "gh" } }
```

- **Managers.** `apt` (linux), `brew` (darwin), `winget` (windows). A manager with no name skips the entry on that OS, with a `skipping` line, and `dotf tools list` shows it as not in the catalog there. `Load` rejects an entry that names no manager, an unknown key (a typo such as `pacman` would otherwise read as "no name" and skip in silence), a `version`, or a name that is a flag or has a space.
- **Commands run.** `sudo -n apt-get install -y <pkg>` (no `sudo` when `dotf` already runs as root), `brew install <name>`, `winget install --id <id> -e --accept-source-agreements --accept-package-agreements`.
- **sudo is never prompted for.** This is the first place `dotf` itself calls `sudo`, and it uses `-n`: sudo fails instead of asking, so a run with no terminal (a scheduled converge, CI) cannot hang. When it fails because a password is needed (`sudo -n true` fails too), the tool is reported as `<name>: needs sudo; run: sudo apt-get install -y <pkg>`, the run does not fail, and the other tools still converge. The remedy is yours to run, once, as the setup scripts already ask. `dotf` does not cache credentials (`sudo -v`) and does not prompt. Any other apt failure stays an error carrying the command.
- **No pin, so presence converges.** An entry is installed when its `command` is on PATH, or else when the manager lists the package (`dpkg-query` reporting `install ok installed`, `brew list --versions`, `winget list --id <id> -e`). A package that is present is skipped and never upgraded, so a second `dotf tools install` runs no manager command. After an install the manager's record must list the package, or the run fails: a manager exiting 0 is not the package being there.
- **When to declare `command`.** Where the tool is an executable that another channel may already have provided, which then counts. Leave it out for a library or a GUI app.
- **A missing manager** (brew before the bootstrap installed it, no `apt-get`) is a skip that names it, and `--dry-run` shows `missing-manager`; the next run installs the package once the manager is there.
- **An unknown future type** is a skip with a `warning:` line, not a failed run, so a catalog written for a newer `dotf` degrades on an older one. `--dry-run` plans it as the same skip (`skip (source type "x" is not known to this dotf)`).
- `dotf tools sync` is unchanged: it renders mise and does not drive the system managers. `dotf tools install` converges them (see the spec for why).

**Coupling to the installed `dotf`.** Setup runs the `dotf` pinned in `versions.conf`, not the one in your checkout. An entry that uses a catalog feature the pinned release lacks (for example, GOOS/GOARCH keys before the release that added them) reads as "no asset" on every machine, and before that release it also failed the run. The order is: land the reader first, release it, bump `DOTF_VERSION`, and only then add the entry (#1814).

## Related

- ADR-036 (install channels, pins as floors) and its 2026-10-06 amendment; ADR-044 (mise); ADR-041 (update channels and convergence order).
- Spec `PLAT-001c-tools-via-mise`; epic #2013.
- [guide-secrets-governance.md](guide-secrets-governance.md) for the tools the secrets system needs (age, bw).
