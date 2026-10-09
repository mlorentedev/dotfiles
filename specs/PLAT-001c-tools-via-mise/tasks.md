---
tags: [spec, tasks, templates]
created: "2026-10-06"
---

# Tasks - PLAT-001c-tools-via-mise

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/plat-001c-installer-goarch` (T1a); each later PR gets its own branch from `main`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> One PR per row. Go: `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` and the pinned `golangci-lint`.

### T1a — installer support (this PR)

- [x] [AC1] Failing test, then `AssetName`: a `goos/goarch` key wins over `goos`; `goos` still resolves alone
- [x] [AC2] Failing test, then `expectedChecksum`: `./name` and `*name` manifest lines verify `name`
- [x] [AC3] Failing test, then `Install`: a `github-release` tool with no asset for this OS/arch is skipped with a message and no error, as `Plan` reports it; `Load` rejects an asset key that names no known platform, so a misspelt key cannot turn into a silent skip everywhere

### T1b — mise in the catalog (after the release carrying T1a is the `DOTF_VERSION` pin)

- [x] [AC4] `packages.json` entry `mise` 2026.9.13, keyed `goos/goarch` (linux, darwin and windows on amd64 and arm64; the raw binaries, so no archive extraction and no dependency on #649), `SHASUMS256.txt`. With an empty HOME on the Mac it downloaded, verified, probed `2026.9.13 macos-arm64`, and a second run skipped; on the real HOME the hand-installed 2026.10.3 is above the floor and is skipped
- Deviation: no `MISE_VERSION` in `versions.conf`. The catalog entry carries the pin, as every `packages.json` tool does, and a second copy would have no reader
- Linux and darwin only for now (`platforms`). On the Windows runner the catalog installed mise, `setup-windows.ps1` runs no `dotf tools sync`, and doctor turned its "mise not on PATH" WARN into a FAIL for the nine pins that winget provides there. The six asset names are verified against the release's `SHASUMS256.txt`, so the Windows keys stay
- [ ] [AC4] Windows, at the Windows box (batched with the other Windows-empirical work): drop `platforms`, add `dotf tools sync` to `setup-windows.ps1` next to `dotf tools install`, and retire the winget installs mise then owns. Measure first that `mise install` works there for bats, shellcheck and direnv

### W2 (first half) — uv through mise; setup runs the sync (stacked on T1b)

- [x] `UV_VERSION=0.12.18` under `# mise: cli` (aqua:astral-sh/uv, the short name resolves there; so does herdr's, which answers H1's open question). It also provides `uvx` for the uvx MCP servers
- [x] `setup-linux.sh`: `dotf tools install` (mise and release binaries), mise's shims onto PATH ahead of `~/.local/bin`, `dotf tools sync`, `dotf tools install` again (the uv-tool entries). Order asserted in `tests/setup-linux.bats`. The unpinned `curl … astral.sh/uv/install.sh | sh` block and the poetry block are deleted (31 lines)
- [x] poetry 2.5.1 is a `packages.json` uv-tool entry on every OS; `setup-windows.ps1` loses its poetry block (26 lines). Windows keeps its uv installer until the Windows session measures `mise install uv` there
- [x] `systemd/hive-upgrade.service` hard-coded `%h/.local/bin/uv`, where uv's own installer put it and mise does not. It runs `/usr/bin/env mise exec -- uv`, with a PATH that finds mise and no shims dir (a moved mise data dir would leave a hard-coded one stale)
- [x] From zero on the Mac (empty HOME, isolated mise/XDG/uv dirs): `tools install mise` → shims on PATH → `tools sync` installed uv 0.12.18 → `tools install pre-commit` installed and ran 4.6.2 through that uv
- [x] Second half: `setup-linux.sh` has no installer of its own for a marked pin. Deleted: age and jq (linux-amd64 curl, which ran BEFORE `dotf tools sync` and so put a second, unpinned copy in `~/.local/bin` on every fresh box), and zoxide, direnv, shellcheck and bats (after the sync, so dead). eza and gh stay behind the host gate: neither is in mise. On a non-amd64 host the leftover cleanup still removes an unrunnable age, jq or shellcheck a past run placed
- [x] Guard: `tests/setup-linux.bats` fails when setup logs "Installing <name>" for any name `versions.conf` marks for mise (mutation-checked with a reintroduced direnv line). It replaces the per-tool "installs X if missing" assertions, which asserted the opposite
- [x] The integration suite's PATH carries mise's shims, as a shell with `mise activate` does: jq now exists only there, and two `verify-setup.bats` tests resolve it from PATH or `~/.local/bin`
- [x] The documented `~/.local/bin/shellcheck` and `~/.local/bin/bats` paths are gone from CLAUDE.md, `specs/CI-002` verification commands and `guard-no-gui.bats`, whose shellcheck test skipped on every machine set up through mise; ADR-004 carries an update note. CI installs its own bats and is unchanged
- [x] Review triage: the guard catches an installer by any of three shapes: its log line, the `~/.local/bin/<name>` path it writes, or a `/<name>/releases/download/` asset. Mutation-checked with one installer of each shape. The mark format needs no second check: `tools.ParseMiseTools` refuses a mark that is not directly followed by a pin, so the guard and dotf read the same set
- [x] Review triage: the integration PATH also carries `~/.local/bin`, as a shell after setup does. Without it, the three dotf tests in `verify-setup.bats` (#696, #915) skipped on every run, `main` included, while the image builds dotf from the checkout into `~/.local/bin`. A missing dotf now fails them instead of skipping
- [ ] doctor reports a leftover `~/.local/bin` copy of a marked pin (placed by the deleted blocks on existing machines; shadowed by mise in a shell, but what a GUI-launched or cron process finds first). Go, needs a release
- [ ] `disable_update_warning` in the rendered mise config: a catalog-pinned mise prints "run mise self-update" on every call, and self-update steps outside the pin

### H1 — herdr through mise (#2013 track H)

- [x] `HERDR_VERSION=0.9.3` under `# mise: cli` (`aqua:herdrdev/herdr`). That entry checks the release digest only; upstream's GitHub attestations for the linux and macOS assets are left to H3 (`gh release verify-asset`), and ADR-044 now says aqua verifies what its registry entry declares
- [x] Guard: `tests/verify-setup.bats` asserts that every marked pin is installed through mise at its version (`mise where name@pin`). Doctor's equivalent check gates only the Windows leg, so Linux had none. On the Mac the same loop reported uv and herdr missing before `dotf tools sync`, and nothing after
- [x] On the Mac: `dotf tools sync` installed herdr and uv, `herdr --version` is 0.9.3, a second run reported nothing to do
- [ ] Windows: H2 (does `mise install herdr` keep `conpty/` next to `herdr.exe`), at the Windows box

### H4a — `dotf deploy` merges TOML (#2013 H4, owner decision 2026-10-08)

herdr writes its own `config.toml` (onboarding writes `onboarding = false`; the Settings screen saves into it), so a `replace` entry would overwrite the owner's in-app choices on every deploy and show drift until then: the `.gitconfig` class (lesson 366). The owner chose a TOML merge over `replace` and seed-if-missing.

- [x] Failing tests first (`merge_toml_test.go`): a merge keeps the keys the tool wrote and writes the managed ones; a reformatted destination (comments, key order, inline tables) is in sync and is not rewritten; an absent destination is created and a second run is in sync; an unreadable destination fails naming the config and is untouched; `paths` on a TOML entry is refused at parse time
- [x] `mergeFormat`: the destination's extension picks the codec (`.toml` is TOML, anything else JSON as before), and `deepMerge` is shared. Dependency `github.com/pelletier/go-toml/v2` v2.2.4. Mutation-checked: with `.toml` read as JSON, all five tests fail
- [x] `ai/deploy.json` documents the TOML merge. No manifest version bump: a `dotf` that predates this refuses a TOML merge entry loudly ("source is not a JSON object") rather than replacing the file
- [x] Review triage: a merge source that names no key is refused for every format (`source manages no key`). An empty or comment-only TOML file parsed as an empty table, so the deploy wrote nothing and reported success, or failed on a confusing `stat` when the destination was absent; JSON `{}` had the same hole (`TestDeploy_MergeRefusesASourceThatManagesNoKey`)
- [x] Review triage: a merge entry whose source and destination differ in format is refused when the manifest is parsed. The merge reads the source in the destination's format, so the slip failed on every machine at deploy time with a misleading `source is not a TOML table`; `replace` never parses and may still cross formats (`TestParseManifest_RefusesAMergeWhoseSourceAndDestinationFormatsDiffer`)
- [x] Review triage: the `paths` refusal reads the source's extension, not the destination's. `paths` rewrites the source as JSON before it is installed, so `replace` from `a.toml` to `b.json` passed the parse and failed at deploy on every machine, while `a.json` to `b.toml` was refused although it works (`TestParseManifest_RefusesPathsOnATOMLSource`, mutation-checked)
- [x] Review triage: "manages no key" looks below the top level. A source of empty tables (`[keys]` with no assignment, `{"a": {}}`) named no value and passed; it is refused now, and an empty list still counts as a value
- [ ] H4 (the `herdr` entry) lands after the release carrying this is the `DOTF_VERSION` pin, so no machine runs a `dotf` that refuses it (#1814 class)

### T2 — `dotf tools sync` (this PR, stacked on T1a)

- [x] [AC5] Failing tests, then `tools.ParseMiseTools`: pins marked `# mise: cli` on the line before them, name from `NAME_VERSION`; a marker that marks no pin is an error
- [x] [AC5] `tools.MiseSync`: renders `<mise config dir>/conf.d/dotfiles.toml` (owned end to end, written atomically), runs `mise install` through an injected runner (lesson 335), probes every tool through `mise which` and `--version` at or above the pin; a second run writes and installs nothing; the hand-written `config.toml` is never touched
- [x] [AC5] `dotf tools sync [--dry-run] [--versions]`; `env.ResolveVersionsPath`, with the checkout-first rule extracted to one helper (`resolveInCheckout`) shared with the catalog and the secret store
- [x] `versions.conf` marks age, bats, shellcheck, golangci-lint, zoxide and adds pins for jq, direnv, fzf and lazygit; eza stays unmarked (F-056)
- [x] On the Mac: `dotf tools sync` installed age, direnv, fzf, jq, lazygit and zoxide, each runs at its pin, and a second run reported nothing to do

### T2b — the converge tools step (this PR)

- [x] [AC6] The `tools` reconciler in `dotf converge`, after the records step. Its plan is `MiseSync.Plan`, its apply `MiseSync.Apply`, and its probe re-plans and requires nothing pending. Without mise on PATH it is skipped, naming the remedy, not failed; `Result.Skip` lets any reconciler say so. The mise runners move to `tools.HomeRunners`, shared with `tools sync`
- [x] On the Mac: `dotf converge` -> records-mirror, records-harness, tools all `[ OK ]`, `9 pinned CLI(s) at their pin`

### Wave 3 — committed `mise.lock`

- [ ] Commit `mise.lock` and install with `--locked`. Measured 2026-10-06: mise keeps one lockfile for the whole global config, and `--locked` fails when any configured tool (the Mac's hand-written `config.toml`) is missing from it, so the lock lands when `conf.d/dotfiles.toml` is the only tools config (comment on #2013). The aqua backend already verifies each asset's checksum and attestation

### P5a — `source.type: system`, reader only (this PR)

> Tests first, each failing before its implementation. The release constraint: no `system` entry ships in `packages.json` here (#2013 P5b follows the release that carries this reader).

- [x] [AC8] Failing tests, then `Load` validation: a `system` entry with no `apt`/`brew`/`winget`, with an unknown key, or with a `version` is an error naming the entry
- [x] [AC7] Failing test, then `SupportsOS`: a `system` entry with no name for this OS is skipped by `Install` with a message and reported `unsupported` by `Plan`
- [x] [AC7] Failing tests, then `Install`: the exact argv per manager (apt as root and through sudo, brew, winget), run through the injected runner; no manager on PATH is a skip naming it
- [x] [AC7] Failing test, then the second run: a package the manager (or the declared command) reports is skipped and no manager command runs; a manager that exits 0 without installing the package is an error
- [x] [AC7] Failing test, then `Plan`: install, skip and missing-manager rows, never upgrade; `tools list` shows `apt:`/`brew:`/`winget:` names
- [x] [AC8] Failing test, then `Install` of an unknown source type: a skip with a warning, exit 0; `Plan` still says `unsupported`
- [x] Runbook `tool-installation.md` names the type; `packages.json` documents it in a `$comment`, with no entry added
- [x] Mutation check: invert the presence skip and confirm the second-run test goes red
- [x] [AC7] Failing tests, then `sudo -n`: `-n` is in the apt argv, and a refusal because a password is needed is a named `needs sudo; run: <exact command>` skip that does not fail the run (`installAll` continues). No `sudo -v` caching, no prompt
- [x] [AC8] Failing test, then `Plan` of an unknown source type: a skip carrying the same "not known to this dotf" words as `Install`, not `unsupported`

> **Decision (P5a): which command converges system entries.** `dotf tools install` does, and `dotf tools sync` stays mise-only. #2013 D8 says "`dotf tools sync` drives the managers"; P5b uses `dotf tools install`, because `install` is the catalog's converge command (`packages.json` is what `install` reads, and `sync` reads `versions.conf` and renders mise), setup already calls it, and a system package is installed by a manager that may need privilege and may be absent, which `sync`'s all-or-nothing `mise install` and its converge step (a reconciler with a plan and a probe) have no place for yet. Widening `sync` is a separate change once the converge step can plan a sudo-gated install; D8's wording is read as "the tools command", not that verb.

### P5b (first half) — class-3 CLIs as `system` entries (0.65.0, carrying P5a, is the pin)

- [x] `packages.json` declares gh, git-lfs (#2201), tmux, xclip (apt only), parallel, wget, eza and zoxide, each with the manager names it has and the command it puts on PATH, so a copy from another channel (setup's linux-amd64 download of gh and eza, Git for Windows' git-lfs) satisfies the entry
- [x] `setup-linux.sh` loses its tmux and xclip presence checks and hand-written `sudo apt install` hints; `tests/setup-linux.bats` asserts the catalog carries both and setup no longer prints the hint
- [x] On the Mac, the dry-run plans install for tmux, wget and eza and skip for the rest, xclip `unsupported`. Each entry was then installed through brew, and a second plan reports skip for all three
- [ ] The winget rows that move here (gh, eza) leave `setup-windows.ps1`'s loop in the Windows-empirical batch: `dotf tools install` there is not followed by a PATH refresh, which only a run on the Windows box can prove harmless. The catalog declares the same ids, so both channels agree until then
- [x] Review triage on #2195: zoxide left `packages.json`. It is a mise-marked CLI in `versions.conf`, so setup's first `dotf tools install`, which runs before `dotf tools sync`, installed an unpinned OS copy on a fresh box before mise placed the pinned one. One owner per tool is now a test (`tests/packages-json.bats`). Its Windows winget row moves to mise with the Windows batch
- [x] `bash`, brew only and with no `command` (/bin/bash 3.2 would satisfy one), for `env bash` on the Mac (#2202). The PATH half was already true: `.zshrc` evals `brew shellenv`, which puts `/opt/homebrew/bin` (or `/usr/local/bin` on Intel) ahead of `/bin`, and `dotf env` declares path variables, not PATH. The 3.2 guard is unchanged: CI's macOS leg runs bats with `/bin` first and `--expect-bash 3`; the guard's comment now says so
- [ ] Second half: `git lfs install` as a converge reconciler beside `git-config` once #2208 lands (#2201); the darwin casks from the owner's list in the vault runbook `runbook-macos-workstation-setup`
- [ ] Out of P5b: docker and kubectl wait on the owner's runtime choice (Docker Desktop, OrbStack or Colima), and doctor keeps reporting them

### W2b — shims on PATH (after #2013 P6/P7)

- [x] `mise activate` in `.zshrc`, `.bashrc` and the PowerShell profile, guarded on mise, before direnv and zoxide (#2013 P6, PR #2043)
- [ ] The shims directory for non-interactive callers (setup, cron, an agent's shell), which read no rc file
- [x] The doctor's pin half: the pinned CLIs resolve through `mise which`, T3 below. That answers from the mise config whatever the caller's PATH, so it does NOT show the shims directory reaches PATH
- [ ] The doctor probe that the shims directory is on PATH for a non-interactive caller; lands with the shims directory above, and until it does a green "at their pin through mise" says nothing about cron or an agent's shell

### T3 — doctor reads a darwin machine as darwin, and checks the mise CLIs (stacked on #2043)

- [x] Failing tests, then one gate for the `~/Applications/<tool>-<version>` toolchain layout: it applies on linux only (`platform.Supports`), and each other OS names how it gets its toolchains instead: winget on windows, mise on darwin (ADR-044 Wave 3)
  - `checkVersionMatch` (the versioned directories) and `checkToolHomeEnvVars` (the `*_HOME` variables must be set) consult it; on the Mac they reported 11 FAILs for a layout darwin does not use
  - `checkVersionedPaths` stays on every OS: a `*_HOME` that is set must point at a real toolchain, which is F-041 (a `JAVA_HOME` pointing at nothing breaks macOS's `/usr/bin/java`)
- [x] tmux is class 3 (ADR-044): the not-installed remedy names the OS's package manager, not `apt` on every POSIX host. `checkTmux` keeps the installed and version half only; `~/.tmux.conf` is the `tmux` deploy entry's, checked by `checkDeployManifest` since #2043
- [x] W2b doctor check: the CLIs marked `# mise: cli` in `versions.conf` run at their pin through `mise which`, the same plan `dotf tools sync --dry-run` prints, on every OS; a changed rendered config is a WARN naming the sync
- [x] On the Mac: the doctor run before and after, the FAIL count and what remains (recorded in `verification.md`)

### mise's own update notice

- [x] Failing test first: the rendered `conf.d/dotfiles.toml` sets `disable_update_warning = true` under `[settings]`, after the pins (`TestRenderMiseConfig_TurnsOffMiseOwnUpdateNotice`)
- [x] Why: mise is pinned in `packages.json` and installed by `dotf tools install`. Its notice, `mise version 2026.10.4 available / To update, run mise self-update`, printed on stderr in a `mise bin-paths` call on the Mac (2026-10-08), points the user past that pin
- [x] Measured: mise 2026.10.3 reads the setting from a `conf.d` file (`MISE_CONFIG_DIR=<tmp> mise settings get disable_update_warning` gives `true`; the live config gives `false`)

### W2c — doctor reports the copies in `~/.local/bin` that shadow mise's

The installers W2 deletes left copies on every machine they ran on. Measured on the Mac (2026-10-08): `bats`, `direnv`, `zoxide`, `uv` and `uvx` sit in `~/.local/bin` beside mise's pinned ones.

- [x] Failing tests first: a regular file named like an executable mise provides warns and names it (a companion such as `age-keygen` counts); `--fix` replaces it with a link to mise's shim, keeps a symlink and a file mise does not provide, keeps the copy when no shim exists, and a second run is clean; nothing is touched while a pin does not run through mise; a failing `mise bin-paths` warns naming the command
- [x] `checkMiseTools` asks mise what each pin provides (`mise bin-paths --bin-names name@version`), not the pin's name, because uv brings uvx and age brings age-keygen
- [x] The check runs only after every pin is proven at its pin through mise: that is what makes replacing a copy safe. It is a WARN, never a FAIL. Owner decision: `dotf doctor --fix` removes the copies
- [x] Review triage: `--fix` replaces each copy with a link to mise's shim instead of deleting it (`MiseShimsDir`, one rename, only to a shim that exists). A deleted copy would leave a consumer whose PATH lists `~/.local/bin` explicitly with nothing, which is worse than the old version; the link keeps it resolvable, now at the pin. The copy's content is still gone, as the owner decided
- [x] Mutation-checked: dropping the symlink exemption, running the check before the pins are verified, or linking without checking the shim exists turns tests red
- [x] Runbook: `docs/runbooks/tool-installation.md` says where the copies come from and how `--fix` treats them
- [x] Review triage: an entry from `mise bin-paths --bin-names` that is a path rather than a file name (a separator, `.` or `..`) is reported and left alone, so `--fix` can never rename over a file outside `~/.local/bin` (`TestCheckMiseTools_AnEntryThatIsAPathIsNeverTouched`, mutation-checked). The doc comment on `checkMiseTools` now says the fix links, not removes
- [x] Review triage: the pin-guard test was vacuous. Its fixture had no shim, so with the guard removed `--fix` still left the copy, for a different reason. The fixture now writes the shim and asserts the copy is still a regular file; deleting the guard's `return` turns it red (measured 2026-10-09)

### Python, the Wave 3 canary (#2062) — PR1, the reader and the check

> The release constraint: `versions.conf` is unchanged here. The released `dotf` (v0.65.0) rejects any `# mise:` comment but `# mise: cli`, so the pins are marked in PR2, once `DOTF_VERSION` carries this parser.

- [x] [AC9] Failing tests, then `ParseMisePins`: both markers, sorted; a package with no `PYTHON_VERSION` under `# mise: cli` is an error; a near-miss names both markers
- [x] [AC9] Failing tests, then the sync: the packages file is written before `mise install` and named by `python.default_packages_file` under `[settings]`; a Python already at its pin gets the packages through pip, once; a failed pip and a pip that installs nothing are errors naming the package; a package below its pin is missing
- [x] [AC9] Failing tests, then doctor's Python section: the shell's `python3` (`python` on Windows) below 3.11, or absent, fails; a package missing or below its pin fails; the remedy is the sync, the PATH when mise already has a Python that clears the floor, or "install one" when nothing pins it
- [x] `checkMiseTools` reports packages missing from mise's Python, and `--fix` runs the sync: one owner for mise's config and installs
- [x] The dead `~/Applications/python-<ver>` layout leaves doctor (`versionedHomes`, `versionMatches`, `toolHomeVars`) and the rc files: nothing installs it
- [x] Guard: `tests/versions-conf.bats` refuses the python-package marker and a marked `PYTHON_VERSION` until PR2 relaxes it; both mutations fail it
- [x] On the Mac, through a scratch `versions.conf` (`--versions`): `python@3.12.6` is refused for lacking GitHub attestations, `3.13.16` installs with PyYAML 6.0.3, a second run does nothing, and `mise install python@3.13.16` alone also leaves `import yaml` working
- [ ] PR2, after the release: mark `PYTHON_VERSION=3.13.16` and `PYYAML_VERSION=6.0.3`, relax the guard to the `DOTF_VERSION` condition, read the pins in CI's macOS job (check first that `actions/setup-python`'s manifest has 3.13.16: it lags upstream, so the job may need a floor instead of the exact pin), and remove the user-site PyYAML the Mac's system 3.9 carries

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/PLAT-001c-tools-via-mise/features.json`):

```json
[
  {
    "id": "PLAT-001c-tools-via-mise-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
