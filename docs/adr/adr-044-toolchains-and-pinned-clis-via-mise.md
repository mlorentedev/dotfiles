---
id: "ADR-044-toolchains-and-pinned-clis-via-mise"
type: adr
status: accepted
owner: manu
date: "2026-10-06"
supersedes: []
extends: [adr-036-install-channels, adr-020-tooling-cli-go-convergence, adr-041-update-channels-and-convergence-order]
issue: mlorentedev/dotfiles#2013
tags: [architecture, decision, tooling, install, toolchains, mise, macos, cross-os]
created: "2026-10-06"
---

# ADR-044: Toolchains and pinned CLIs install through mise; `dotf` orchestrates and verifies

## Context

The first `dotf doctor` run on a factory-fresh macOS arm64 machine (2026-10-05, ledger on #2013) failed on the toolchain model itself, not on a missing package:

- **F-003.** Java, Maven, Python, Go and Minikube are expected as tarballs unpacked into `~/Applications/<tool>-<ver>`, with a `*_HOME` variable per tool. That layout is a Linux convention that `setup-linux.sh` hand-rolls. Nothing installs it on macOS, and `setup-windows.ps1` carries a second, different implementation.
- **F-041.** On macOS `~/Applications` is a real system folder, and a `JAVA_HOME` that points at a directory that does not exist breaks `/usr/bin/java`, which is a stub that reads `JAVA_HOME`.
- **F-030.** Five CLIs (age, eza, jq, gh, shellcheck) are downloaded from hard-coded `linux-amd64` URLs. On any other OS or architecture, setup reports SUCCESS and leaves binaries that cannot execute. W1 (#2014) made `dotf tools install` refuse such a binary, but the shell blocks are outside the catalog.

ADR-036 gave each tool class one install channel, and the `github-release` source type in `packages.json` covers static binaries. It has three gaps for this job:

- It has no notion of a toolchain: a JDK or a Python is a tree, not one binary.
- Every new tool needs a hand-written per-OS/arch asset map.
- It has no lock file, so two machines with the same pin can still install different bytes when an upstream re-publishes an asset.

Research for #2013 (epic comment *Toolchain research*) compared five managers against three hard requirements: native Windows, pinning with a lock, and verified provenance.

| Manager | Native Windows | Lock with per-platform checksums | Verified provenance | Result |
|---|---|---|---|---|
| mise | yes | `mise.lock` (URL + checksum per platform), `--locked` | aqua backend: cosign / SLSA / GitHub attestations | **chosen** |
| Homebrew | no | no (the Brewfile lock was removed by design) | bottles only | rejected as a toolchain channel (it cannot pin) |
| asdf | no | no | no | rejected |
| SDKMAN | no | no | no | rejected |
| Nix | no | yes | yes | rejected (no native Windows) |

## Decision

1. **mise installs toolchains and pinned CLIs on every OS.**
   - Toolchains: Java (`temurin-21`; the vendor is always named), Go, Python (mise's python-build-standalone), Maven (aqua), Node.
   - Pinned CLIs through the aqua backend, which verifies what each tool's aqua registry entry declares (a checksum or release digest always; an attestation only where the entry declares one, whatever upstream publishes): age, zoxide, shellcheck, jq, bats, golangci-lint, sops, direnv, fzf, herdr, lazygit. herdr publishes GitHub attestations for its linux and macOS assets, but `aqua:herdrdev/herdr` checks the release digest only (measured on 0.9.3, mise 2026.9.13); verifying those attestations is #2013 H3's job (`gh release verify-asset`).
   - eza has no aqua entry and no macOS release asset. It goes through `vfox:jdx/vfox-eza` if that works on all three OSes, otherwise it is a class-3 tool on darwin (ADR-036 decision 1).
2. **`dotf` orchestrates and verifies; mise does not decide.**
   - `versions.conf` stays the source of truth for pins. `dotf tools sync` renders the global mise config from it and runs `mise install --locked`.
   - `mise.lock` is committed. CI regenerates it for linux-x64, macos-arm64 and windows-x64, so the lock never depends on the machine that last ran mise.
   - mise's exit 0 is not trusted. Every tool is exec-probed after install (`mise which <tool>` plus `--version` at or above the pin), the post-condition W1 introduced for the catalog (lesson 337).
3. **mise itself is a catalog tool.** It is a checksummed `github-release` entry in `packages.json`, not a brew formula and not mise's own curl installer. mise's documentation advises against package-manager installs that lag its release cadence. mise is date-versioned, so its entry carries a `min_version` floor.
4. **ADR-036's classes after this ADR:**

   | Tool class | Channel |
   |---|---|
   | toolchains and pinned single-binary CLIs | mise, rendered by `dotf tools sync` |
   | node-distributed agents and CLIs | npm global, `packages.json` (unchanged) |
   | PyPI-distributed tools | `uv tool`, `packages.json` (unchanged) |
   | mise and `dotf` themselves | `github-release`, `packages.json` / the installers (ADR-041 decision 7) |
   | tools with no cross-OS channel | OS package manager, class 3 (unchanged; what Homebrew may install on macOS beyond this ADR is #2013 D4, still proposed) |

   uv keeps virtual environments and `uv tool`; mise does not manage Python packages.
5. **Environment comes from mise, not from hand-written `*_HOME` blocks.** `mise activate` (bash, zsh, pwsh) sets `JAVA_HOME` and PATH in shells. `dotf` writes a static env file for GUI apps and IDEs, which do not run a shell rc. The shims directory is on PATH for non-interactive callers; without it, `dotf tools sync` succeeds and nothing is reachable, which is a new false success (W2b).
6. **Rollout is split by risk.**
   - Wave 1, every OS: the CLIs only. They land in a directory nothing else owns, so there is no legacy layout to collide with.
   - Toolchains go to the Mac first, as the canary. It has no `~/Applications/<tool>-<ver>` installs, so nothing can break. Linux and Windows move only after the Mac has run `dotf doctor` clean.
   - Windows Java via mise is undocumented upstream. A spike (T5) gates the Windows half; if it fails, the JDK stays a winget class-3 install on Windows and mise handles everything else.
   - Legacy `~/Applications/<tool>-<ver>` installs are reported by doctor as leftovers, and removed only after the mise version passes its probe.

## Consequences

- The five hard-coded `linux-amd64` URL blocks and the tarball/`*_HOME` blocks leave both setup scripts (W2, T4). That is code removed from the twins, not ported into them (ADR-020 §5).
- Doctor stops checking `~/Applications/<tool>-<ver>` and `*_HOME`. It compares `mise ls --json` with the pins and probes each tool (T3).
- `packages.json` shrinks to npm, uv-tool and the two bootstrap binaries. A pin that moves to mise appears only in `versions.conf`, so ADR-036 decision 2 ("a catalog tool's version appears nowhere else") still holds, with `versions.conf` as the one place.
- mise is a new runtime dependency on every machine. It is a single static binary with no daemon, installed and verified like any other catalog tool, and removable with its data directory.
- `DX-007-orca-cli-bootstrap` is abandoned in the same change: Orca is retired in favour of herdr on every OS (#2013 D6), and herdr installs through mise under this ADR.

## Alternatives rejected

- **Extend `github-release` with toolchain support.** It reimplements a version manager inside `dotf` (archive layouts, per-vendor JDK naming, a lock format) for less coverage than mise has today.
- **Homebrew for toolchains on macOS.** It cannot pin a version, and a macOS-only channel is the per-OS branching this epic removes.
- **mise's own installer (`curl https://mise.run | sh`).** It is a second unverified shell pipe on a fresh machine; the catalog entry is checksummed and exec-probed by the same code as every other tool.

## References

- Epic #2013 (decisions D3, D7; rows T0 to T6, W2, W2b), ledger F-003, F-030, F-041, F-056.
- ADR-020, ADR-036, ADR-041; lesson 337 (a checksum proves the bytes, not that they run here).
- mise documentation: `mise.jdx.dev/dev-tools/mise-lock.html`, `/installing-mise.html`, `/dev-tools/backends/aqua.html`, `/lang/java.html`.

## Clarification 2026-10-06 (PLAT-001c T2)

A mise pin is an exact version, not a floor. `dotf tools sync` renders it into `~/.config/mise/conf.d/dotfiles.toml`, and mise activates exactly that version, which is also what the committed lock reproduces. This differs from ADR-036 decision 1, where catalog pins are floors that are never downgraded. An older version reaches a machine only when the pin is lowered in `versions.conf`, which is a reviewed change. mise reads a hand-written `~/.config/mise/config.toml` with precedence over `conf.d` (measured, mise 2026.10.3): a newer version pinned there is kept, and an older one makes the sync's post-condition fail, naming the tool.

## Clarification 2026-10-09 (#2062, Python as a hard dependency)

The owner made Python >= 3.11 a hard dependency of `dotf` (#2062), and Python is the Wave 3 canary. Decision 4's "mise does not manage Python packages" keeps its meaning for environments: uv still owns virtual environments and `uv tool`. The exception is narrow. The libraries the suite itself imports from the bare interpreter (PyYAML today) are marked `# mise: python-package` in `versions.conf`, and `dotf tools sync` lists them in a file that mise's `python.default_packages_file` setting names. mise installs them into every Python it installs, and the sync pip-installs them into a pinned Python installed before the package was declared. Both paths are measured on the Mac (mise 2026.10.3): mise reads the setting from `conf.d`, and a `mise install python@3.13.16` with no `dotf` involved leaves `import yaml` at 6.0.3.

`dotf doctor` fails when the Python a shell resolves is below 3.11, or cannot import a declared package at its pin. When mise already has a Python that clears the floor, the failure names the PATH, because a sync would change nothing.

Two constraints decide the version and the order:

- **The pin must be an attested build.** mise verifies GitHub artifact attestations for python-build-standalone by default, and `python@3.12.6` predates them (`No GitHub artifact attestations found`). The pin moves to 3.13.16. Turning the verification off is rejected.
- **The markers wait for the release.** The released `dotf` up to v0.65.0 rejects every `# mise:` comment except `# mise: cli`. So the pins are marked only once `DOTF_VERSION` carries the parser; `tests/versions-conf.bats` enforces it.

## Amendment 2026-10-10 (#2013 D9, the coding agents track latest)

The owner decided that the coding agents (Claude Code, pi and agy) install through mise like every pinned CLI, but stay at their newest release instead of an exact pin. Their upstreams ship almost daily, and the owner wants the newest one on every machine without a bump PR per release.

- **A third marker, `# mise: latest`.** The pin under it is a floor, so the other readers of `versions.conf` (the setup scripts, doctor's floor match, CI's pi reconcile) keep comparing a version. `dotf tools sync` renders the tool as `latest`, and requires it to run at or above the floor after the sync, as it does for exact pins.
- **The sync upgrades.** `mise install` keeps an installed version of a tool requested as `latest`, so the sync also asks `mise outdated --json <tools>` and runs `mise upgrade <tools>` for the ones it names. `mise upgrade` schedules the version it replaced for removal rather than deleting it at once: mise 2026.10.3 lists it as `pruned in 1d`, so an agent does not pile up a copy per release. A run that turns an exact pin into `latest` upgrades in the same run.
- **Doctor does not ask.** `mise outdated` reaches the network, and a release upstream is not a fault on the machine, so doctor plans without it. `dotf converge` and `dotf tools sync` ask.
- **Accepted costs.** Two machines can run different versions until each converges, and a broken upstream release arrives untested. The alternative the owner kept in reserve is an exact pin with a bot that opens the bump PR, which the from-zero CI job (#2013 X1) would test.
- **Release order.** Every released `dotf` up to v0.67.0 rejects the marker as a near miss, which fails the whole tools step. `versions.conf` uses it only once `DOTF_VERSION` carries the parser; `tests/versions-conf.bats` enforces it.
