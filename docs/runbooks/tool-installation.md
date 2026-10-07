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
| Pinned single-binary CLIs (age, bats, direnv, fzf, golangci-lint, jq, lazygit, shellcheck, zoxide) and toolchains (Go, Java, Python, Maven, Node) | mise | `versions.conf`, lines marked `# mise: cli` | `dotf tools sync`: the CLIs listed today, herdr with track H of #2013, the toolchains from Wave 3 |
| The two bootstrap binaries (`dotf` and mise), plus sops | GitHub release, sha256-verified | `packages.json` | `dotf tools install` |
| Node-distributed CLIs and agents (opencode, copilot, bw, pi) | npm global | `packages.json` | `dotf tools install` |
| PyPI tools (hive) | `uv tool` | `packages.json` | `dotf tools install` |
| Tools with no cross-OS channel (git, gh, uv, system libraries, macOS casks) | the OS package manager: apt, winget, Homebrew | the setup script for each OS | setup |

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
- **What sync writes.** `dotf tools sync` renders the marked pins into `~/.config/mise/conf.d/dotfiles.toml`; it honours `MISE_CONFIG_DIR` and `XDG_CONFIG_HOME`. That file is generated, so do not edit it. Sync never touches a hand-written `~/.config/mise/config.toml`; one that pins the same tools is a leftover for the doctor check of #2013 row T3.
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
- Before relying on the entry, check it with `dotf tools list` and `dotf tools install --dry-run` on each OS you declared.

**Coupling to the installed `dotf`.** Setup runs the `dotf` pinned in `versions.conf`, not the one in your checkout. An entry that uses a catalog feature the pinned release lacks (for example, GOOS/GOARCH keys before the release that added them) reads as "no asset" on every machine, and before that release it also failed the run. The order is: land the reader first, release it, bump `DOTF_VERSION`, and only then add the entry (#1814).

## Related

- ADR-036 (install channels, pins as floors) and its 2026-10-06 amendment; ADR-044 (mise); ADR-041 (update channels and convergence order).
- Spec `PLAT-001c-tools-via-mise`; epic #2013.
- [guide-secrets-governance.md](guide-secrets-governance.md) for the tools the secrets system needs (age, bw).
