---
id: "PLAT-001c-tools-via-mise"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-06"
issue: "mlorentedev/dotfiles#2013"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "PLAT-001 track T: the macOS machine has no pinned CLIs on PATH until mise installs them through dotf (ADR-044); the other stalled specs are outside this session's scope (17 active, limit 10, 2026-10-06)"
---

# PLAT-001c: Pinned CLIs through mise, orchestrated by `dotf` (track T, Wave 1)

## Why

<!-- from issue #2013: PLAT-001: [EPIC] OS-agnostic from-zero bootstrap — macOS bring-up as the forcing function -->

ADR-044 (accepted, #2017) moves toolchains and pinned single-binary CLIs to mise, with `dotf` as the orchestrator and verifier. Nothing implements it yet, and the gap shows on the Mac:

- **The CLIs are missing.** age, jq, direnv, zoxide and the other pinned CLIs are absent from the macOS machine.
- **Hand-installed tools are not on PATH.** go, bats, shellcheck and golangci-lint were installed through mise by hand on 2026-10-05, and they do not resolve on PATH.
- **Linux still hard-codes amd64.** `setup-linux.sh` fetches five CLIs from hard-coded `linux-amd64` URLs (F-030).

On a fresh machine the owner's rule applies: nothing by hand, one engine for every OS.

## What

One PR per row of #2013 track T (Wave 1, CLIs only):

- **T1a — installer support** (this spec's first PR):
  - A `github-release` asset map may key an entry by `goos/goarch`, which wins over `goos`. mise names its arches `x64`/`arm64` and its OS `macos`, which `{goarch}` cannot express.
  - The checksum manifest accepts the `sha256sum` forms `./name` and `*name`; mise's `SHASUMS256.txt` uses `./`.
  - A `github-release` tool with no asset for this OS/arch is **skipped** by `Install`, as `Plan` already reports it (`unsupported`). Today `Install` errors, so the plan and the apply disagree.
- **T1b — mise as a catalog tool:**
  - A checksummed, exec-probed `github-release` entry keyed by `goos/goarch`.
  - It lands only after a `dotf` release that carries T1a is the pin (`DOTF_VERSION`). An older binary would read the entry as "no asset" and, before T1a, fail `dotf tools install`.
- **T2 — `dotf tools sync`:**
  - Renders the pinned CLIs from `versions.conf` into `~/.config/mise/conf.d/dotfiles.toml`, a file `dotf` owns end to end. The hand-made `~/.config/mise/config.toml` is never touched; the toolchains move in Wave 3.
  - Runs `mise install`, then probes every tool with `mise which <tool>` plus `--version` at or above the pin.
  - A `tools` reconciler in `dotf converge` runs it after the records step.
- **P5a — `source.type: system`, reader only** (#2013 D8):
  - A catalog entry names its package per OS manager (`apt`, `brew`, `winget`); `dotf tools install` and `--dry-run` converge it through that manager. A manager with no name skips the entry on that OS, through the same skip as a `platforms` miss.
  - Presence, not a pin, is the convergence rule, so a second run changes nothing and a system package is never upgraded.
  - An unknown source type is skipped with a warning instead of failing the run, so a catalog written for a newer `dotf` degrades on an older one.
  - No `system` entry ships in `packages.json` in this PR: the installed `dotf` errors on an unknown type, so entries wait for the release carrying this reader (P5b).
- **W2b — the shims on PATH:** `mise activate` in the rc files and the shims directory for non-interactive callers, with a doctor check. This row waits on the rc files moving to deploy entries (#2013 P6, P7).

## Out of scope

- **Toolchains through mise** (Java, Go, Maven, Node): Wave 3, Mac first (ADR-044 decision 6). Python is the exception; see the amendment below.
- **Deleting the setup scripts' `linux-amd64` blocks** (#2013 W2): after T2 lands on every OS.
- **Archive extraction in the installer** (#649): superseded by ADR-044, see the comment on #649.

## Risks / open questions

- **Binary/record coupling (#1814 class).** A catalog entry the installed `dotf` cannot read correctly breaks `dotf tools install` on every machine until the pin moves. *Resolved:* T1a ships the reader first; T1b waits for the release.
- **`mise.lock`.** ADR-044 promises a committed lock that CI regenerates for linux-x64, macos-arm64 and windows-x64. `mise lock --platform` produces a multi-platform lock from one host (checked on 2026-10-06, mise 2026.10.3). The T2 PR verifies it end to end before it commits a lock.
- **eza.** It has no aqua entry and no macOS asset (F-056). T2 tries `vfox:jdx/vfox-eza` on each OS; if that fails, eza is class 3 on darwin.
- **Date versions.** mise's own version (`2026.10.3`) matches the installer's semver pattern and compares numerically, so no `min_version` field is needed.

## Acceptance criteria

- [ ] AC1: a `github-release` asset keyed `goos/goarch` is chosen over a `goos` key, and a `goos` key still resolves when no specific key exists.
- [ ] AC2: a checksum manifest line in the `./name` or `*name` form verifies the asset `name`.
- [ ] AC3: `dotf tools install` skips a `github-release` tool with no asset for this OS/arch, says so, and exits 0, matching `--dry-run`'s `unsupported`.
- [ ] AC4: `dotf tools install mise` installs the pinned mise on linux, darwin and windows, checksum-verified and exec-probed.
- [ ] AC5: `dotf tools sync` renders `conf.d/dotfiles.toml` from `versions.conf`, installs, and fails naming any tool that does not run at or above its pin; a second run changes nothing.
- [ ] AC6: `dotf converge` runs the tools step after the records step, and `--plan` reports the tools it would install.
- [ ] AC7: a `system` entry is installed through the OS manager (`sudo -n apt-get install -y --no-remove`, `brew install`, `winget install --id <id> -e` with both agreement flags), skipped with a message on an OS whose manager it does not name, a sudo password is reported as "needs sudo" with the command to run and does not fail the run, and a second run runs no manager command.
- [ ] AC8: `Load` rejects a `system` entry that names no manager, names an unknown key, or declares a `version`, naming the entry; `Install` skips a source type it does not know instead of failing.

- [ ] AC9: `dotf tools sync` installs mise's Python with the packages `versions.conf` marks `# mise: python-package`, through mise's `python.default_packages_file` on a fresh install and through pip into a Python already at its pin; `dotf doctor` fails when the shell's Python is below 3.11 or cannot import one of them at its pin, naming the remedy that clears it.

## Amendment 2026-10-09: Python, the Wave 3 canary (#2062)

The owner made Python >= 3.11 a hard dependency of `dotf` (#2062): the Mac's `python3` was the system 3.9.6, and the suite's TOML and YAML readers need 3.11 and PyYAML. Python moves to mise ahead of the other toolchains, in two PRs:

- **PR1 (this one):** the reader and the check. `ParseMisePins` reads a second marker, `# mise: python-package`. `dotf tools sync` writes those packages to a file named by mise's `python.default_packages_file` and pip-installs any still missing. `dotf doctor` gains a Python section, `--fix` on the mise section runs the sync, and the dead `PYTHON_HOME` layout leaves doctor and the rc files. `versions.conf` is unchanged.
- **PR2, after the release carrying PR1 is the `DOTF_VERSION` pin:** mark `PYTHON_VERSION` (moved to 3.13.16, an attested build) and `PYYAML_VERSION`. The released parser rejects the new marker, and a released sync would install a Python without its packages, so neither line can land earlier. `tests/versions-conf.bats` enforces the order.

## References

- ADR-044 (decisions 1–6), ADR-036 (and its 2026-10-06 amendment), ADR-045 decision 4; lesson 337.
- Epic #2013 (rows T1, T2, W2, W2b; ledger F-030, F-056); #649; #1814.
