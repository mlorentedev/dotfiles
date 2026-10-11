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
| Pinned single-binary CLIs (actionlint, age, bat, bats, cloudflared, delta, direnv, fd, fzf, golangci-lint, hadolint, herdr, jq, kubectx, kubens, lazygit, mkcert, shellcheck, shfmt, stern, terraform, tflint, trivy, uv, yq, zoxide) and toolchains (Go, Java, Python, Maven, Node) | mise | `versions.conf`, lines marked `# mise: cli` | `dotf tools sync`: the CLIs listed today on Linux and macOS (Windows after its session, #2013 H2), the toolchains from Wave 3 |
| The two bootstrap binaries (`dotf` and mise), plus sops | GitHub release, sha256-verified | `packages.json` | `dotf tools install` |
| Node-distributed CLIs and agents (opencode, copilot, bw, pi) | npm global | `packages.json` | `dotf tools install` |
| PyPI tools (hive, ansible, ansible-lint) | `uv tool` | `packages.json` | `dotf tools install` |
| Tools with no cross-OS channel (git, gh, eza, tmux, docker, system libraries, macOS casks) | the OS package manager: apt, winget, Homebrew | `packages.json` `source.type: system` entries (#2013 D8, P5b); Windows' winget loop in `setup-windows.ps1` until the Windows batch | `dotf tools install`, which converge's `tools` step runs; without passwordless sudo it prints the one `sudo apt-get install` command |

How a pin is read depends on the channel:

- **Catalog pins** (`packages.json`) are floors. An installed version at or above the pin is left alone, and nothing is downgraded (ADR-036 decision 1, ADR-041 decision 6).
- **mise pins** (`versions.conf` lines marked `# mise: cli`) are exact, except under `# mise: latest` (below). They are the version mise activates, and the version a committed lock will reproduce (ADR-044). An older version is only activated when the pin itself is lowered in `versions.conf`, which is a reviewed change.
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
- **Tools that track their newest release.** A `# mise: latest` line marks a tool mise keeps at its newest release instead of an exact pin; the coding agents are meant for it (ADR-044, amendment 2026-10-10). The pin is a floor: the sync renders the tool as `latest`, asks `mise outdated` and runs `mise upgrade` when the tool is behind, and fails if it ends below the floor. `dotf doctor` does not ask `mise outdated`, so a new upstream release is never a doctor fault. Only `dotf` 0.68.0 and later reads this marker, so `DOTF_VERSION` must not drop below that while a tool is marked; `tests/versions-conf.bats` enforces it.
- **Python packages.** A `# mise: python-package` line marks a library that the suite imports from the bare interpreter (PyYAML), not a CLI: `PYYAML_VERSION` is the PyPI distribution `pyyaml`. A python package needs `PYTHON_VERSION` under `# mise: cli`. The sync lists the packages in `conf.d/dotfiles-python-packages.txt`, and mise's `python.default_packages_file` setting names that file, so mise installs them into every Python it installs. A Python installed before the package was declared gets it through pip. Tools are a different case: anything you run as a command is a `uv tool` entry in `packages.json`. Only `dotf` 0.66.0 and later reads this marker, so `DOTF_VERSION` must not drop below that while a package is marked; `tests/versions-conf.bats` enforces it (#2062, lesson 373).
- **What doctor checks and fixes.** `dotf doctor` fails when a marked pin does not run at its pin through mise, or when a marked python package is missing from mise's Python. Its `[Python]` section fails when the shell's `python3` (`python` on Windows) is below 3.11 or cannot import a marked package. On a machine without mise (Windows, until its ADR-044 wave) a missing package only warns: nothing there manages it. `dotf doctor --fix` runs the sync whenever anything is pending, so it may run `mise install` for every pin, not only the one that failed. A `[Python]` failure that survives the fix names the PATH: mise has a Python that clears the floor, and the shell does not resolve it.
- **Copies left in `~/.local/bin`.** Before mise owned these CLIs, setup downloaded some of them into `~/.local/bin` (#2013 W2). An activated shell puts mise first, but any PATH that lists `~/.local/bin` without mise's shims ahead of it (a systemd unit, a launchd plist, a cron line) runs the old copy, a version no pin governs. Once every pin runs through mise, `dotf doctor` warns about each regular file in `~/.local/bin` named like an executable mise provides for a pin, companions included (`uvx`, `age-keygen`). `dotf doctor --fix` replaces each one with a link to mise's shim rather than deleting it, so a consumer that relied on `~/.local/bin` keeps finding the tool, now at its pin. It links only to a shim that exists (otherwise: `mise reshim`), in one rename, and never touches a symlink.
- **Verification.** After `mise install`, every tool must resolve through `mise which` and report a version at or above its pin, or sync fails naming it. Sync does not go through PATH, which only gains the mise shims with #2013 row W2b; until then, run a tool with `mise exec -- <tool>`.
- **The version probe.** Sync runs the binary named like the tool with `--version` and takes the first `x.y.z` it prints. Some CLIs reject that flag (kubectl, helm, argocd, kustomize, hcloud, k9s, kubeconform); their arguments live in `versionArgs` in `cli/internal/tools/mise.go`. A tool marked before the released `dotf` carries its row makes every machine's sync fail naming it, while CI, which builds `dotf` from the tree, stays green. So a new row ships first, and the pin waits for the release; `tests/versions-conf.bats` holds that order for the seven above (#2013, lesson 373).

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

`dotf` reads `source.type: "system"`: a package the OS manager owns, named once per manager. Entries ship since 0.65.0, the first release carrying the reader, became the `DOTF_VERSION` pin (#2013 P5b).

**Docker on the Mac** is Colima: the brew entries colima, docker, docker-compose and docker-buildx, then `dotf doctor --fix`, which runs `brew services start colima` once; Homebrew keeps it running across logins. The compose and buildx plugins live in `/opt/homebrew/lib/docker/cli-plugins`, outside the docker CLI's default search path; the `docker-config` deploy entry adds that directory to `~/.docker/config.json` wherever colima is installed. `dotf doctor` warns under *Docker engine* while the engine is down, and `--fix` starts it. The VM is 4 CPU and 8 GiB: *Colima VM size* warns when `~/.colima/default/colima.yaml` or the running VM differs, and `--fix` sets both keys in that file and restarts the service, which stops the running containers. On a new Mac that first `--fix` boots Colima twice, at the defaults and then at that size. **On Linux** the `docker` entry installs Ubuntu's `docker.io` and **on Windows** Docker Desktop (`Docker.DockerDesktop`, which bundles compose and buildx); a docker already on PATH from another channel, such as Docker's `docker-ce`, satisfies the entry and nothing is installed over it. After `docker.io`, join the `docker` group (`sudo usermod -aG docker $USER`, then log in again); doctor's *Docker engine* names that when the socket refuses you.

```json
{ "name": "gh", "source": { "type": "system", "apt": "gh", "brew": "gh", "winget": "GitHub.cli", "command": "gh" } }
```

- **Managers.** `apt` (linux), `brew` (darwin), `winget` (windows), and `cask` (darwin) for a Homebrew cask, a GUI app or font, in place of `brew`: brew lists and installs casks in their own namespace, so a cask named under `brew` would read as absent on every run. An entry names a formula or a cask, never both. A manager with no name skips the entry on that OS, with a `skipping` line, and `dotf tools list` shows it as not in the catalog there. `Load` rejects an entry that names no manager, an unknown key (a typo such as `pacman` would otherwise read as "no name" and skip in silence), a `version`, or a name that is a flag or has a space.
- **Commands run.** `sudo -n apt-get install -y --no-remove <pkg>` (no `sudo` when `dotf` already runs as root), `brew install <name>`, `brew install --cask <token>`, `winget install --id <id> -e --accept-source-agreements --accept-package-agreements`.
- **sudo is never prompted for.** This is the first place `dotf` itself calls `sudo`, and it uses `-n`: sudo fails instead of asking, so a run with no terminal (a scheduled converge, CI) cannot hang. When it fails because a password is needed (`sudo -n true` fails too), the tool is reported as `<name>: needs sudo; run: sudo apt-get update && sudo apt-get install -y --no-remove <pkg>`, the run does not fail, and the other tools still converge. The remedy is yours to run, once, as the setup scripts already ask: at the end of the run `dotf tools install` prints ONE `sudo apt-get update && sudo apt-get install -y --no-remove <pkg> <pkg> …` for every package that waited (the index refresh first, since dotf skipped its own when sudo refused), the converge tools step reports the same command, and `dotf doctor` (*System packages*) keeps reporting it until the packages are there, so a hint a scheduled run printed into a log is not lost (#2308). Only apt escalates: brew refuses to run as root, and winget elevates per installer through UAC. `--no-remove` holds for the whole batch, so one conflicting package (docker.io on a box running docker-ce) aborts it; apt names the package, and you install the rest without it. `dotf` does not cache credentials (`sudo -v`) and does not prompt. Any other apt failure stays an error carrying the command.
- **No pin, so presence converges.** An entry is installed when its `command` is on PATH, or else when the manager lists the package (`dpkg-query` reporting `install ok installed`, `brew list --versions` (`brew list --cask --versions` for a cask), `winget list --id <id> -e`). A package that is present is skipped and never upgraded, so a second `dotf tools install` runs no manager command. After an install the manager's record must list the package, or the run fails: a manager exiting 0 is not the package being there.
- **When to declare `command`.** Where the tool is an executable that another channel may already have provided, which then counts. Leave it out for a library or a GUI app.
- **A missing manager** (brew before the bootstrap installed it, no `apt-get`) is a skip that names it, and `--dry-run` shows `missing-manager`; the next run installs the package once the manager is there.
- **An unknown future type** is a skip with a `warning:` line, not a failed run, so a catalog written for a newer `dotf` degrades on an older one. `--dry-run` plans it as the same skip (`skip (source type "x" is not known to this dotf)`).
- `dotf tools sync` is unchanged: it renders mise and does not drive the system managers. `dotf tools install` converges them (see the spec for why).

**Coupling to the installed `dotf`.** Setup runs the `dotf` pinned in `versions.conf`, not the one in your checkout. An entry that uses a catalog feature the pinned release lacks (for example, GOOS/GOARCH keys before the release that added them) reads as "no asset" on every machine, and before that release it also failed the run. The order is: land the reader first, release it, bump `DOTF_VERSION`, and only then add the entry (#1814).

## Related

- ADR-036 (install channels, pins as floors) and its 2026-10-06 amendment; ADR-044 (mise); ADR-041 (update channels and convergence order).
- Spec `PLAT-001c-tools-via-mise`; epic #2013.
- [guide-secrets-governance.md](guide-secrets-governance.md) for the tools the secrets system needs (age, bw).
