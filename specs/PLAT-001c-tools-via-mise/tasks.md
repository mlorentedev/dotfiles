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
- [ ] [AC4] Windows: **after the first release carrying the Windows bats/direnv reader is the `DOTF_VERSION` pin**, drop `platforms` only after the full catalog and all marked pins pass a fresh `dotf converge` tools step and a second run has `changed=0`; retire the duplicated winget installs and the uv installer in that change. The entrypoint `install.ps1` already delegates to `dotf converge`, whose tools step owns catalog → mise sync → catalog; do not add another sync to `setup-windows.ps1`. The 0.67.0 release still renders the aqua bats channel on Windows, so changing the catalog platform gate in this reader change would break from-zero installs.
- [x] T5 Windows Java spike: mise 2026.9.13 installs `java@temurin-21.0.4+7.0.LTS` in an isolated data directory; `java.exe -version` reports 21.0.4 and a second install changes nothing.
- [x] Windows CLI parity in the new reader: `aqua:bats-core/bats-core@1.13.0` rejects Windows; render `npm:bats`, require its `.cmd` launcher, install pinned Node before bats, and scope `MISE_NPM_PACKAGE_MANAGER=npm` to the bats install child so other npm tools and hand-written mise config keep their installer. Keep Windows excluded until the reader ships.
- [x] Diagnose the isolated Windows direnv failure: the aqua recipe installed a PE binary without `.exe`; both mise 2026.9.13 and 2026.10.8 could not resolve it. With the pinned mise, `github:direnv/direnv` and `bin = "direnv.exe"` select the Windows release, run 2.37.1 through `mise which direnv`, and a second install adds 0 tools. The Windows config now declares that channel, without a post-install rename.
- [x] In a short, isolated Windows fixture with the catalog-pinned mise 2026.9.13, the current renderer installed and probed all 27 marked CLIs plus PyYAML; a second `dotf tools sync` reported `nothing to do`, its dry-run said `unchanged`/`nothing to install`, and `mise bin-paths --bin-names` found `bats.cmd`, `direnv.exe`, `herdr.exe`, and `uv.exe`.
- [x] Node 24.11.1 pinned via the existing `# mise: cli` channel on every OS; Windows sync refuses bats without a marked Node pin. In a fresh Windows PATH without npm, `dotf tools sync` installs all 28 pins and PyYAML, then reports `nothing to do` on a second run; the rendered mise config retains `npm.package_manager = auto`.
- [x] A short Windows catalog fixture containing only the real checksum-verified mise entry and four pins (`bats`, `direnv`, `node`, `uv`) passes `dotf converge --only tools` from no mise or npm on PATH: first pass installs mise and the four pins; second pass reports `0 changed, 1 ok`. This proves first-run ordering and shim reachability without installing the full production catalog into a live machine. The full 28-pin sync and the smaller from-zero catalog check are separate isolated tests, not yet a full production catalog rollout.

### W2 (first half) — uv through mise; setup runs the sync (stacked on T1b)

- [x] `UV_VERSION=0.12.18` under `# mise: cli` (aqua:astral-sh/uv, the short name resolves there; so does herdr's, which answers H1's open question). It also provides `uvx` for the uvx MCP servers
- [x] `setup-linux.sh`: `dotf tools install` (mise and release binaries), mise's shims onto PATH ahead of `~/.local/bin`, `dotf tools sync`, `dotf tools install` again (the uv-tool entries). Order asserted in `tests/setup-linux.bats`. The unpinned `curl … astral.sh/uv/install.sh | sh` block and the poetry block are deleted (31 lines)
- [x] poetry 2.5.1 is a `packages.json` uv-tool entry on every OS; `setup-windows.ps1` loses its poetry block (26 lines). Windows keeps its uv installer until the Windows session measures `mise install uv` there
- [x] `systemd/hive-upgrade.service` hard-coded `%h/.local/bin/uv`, where uv's own installer put it and mise does not. It runs `/usr/bin/env mise exec -- uv`, with a PATH that finds mise and no shims dir (a moved mise data dir would leave a hard-coded one stale)
- [x] From zero on the Mac (empty HOME, isolated mise/XDG/uv dirs): `tools install mise` → shims on PATH → `tools sync` installed uv 0.12.18 → `tools install pre-commit` installed and ran 4.6.2 through that uv
- [x] Second half: `setup-linux.sh` has no installer of its own for a marked pin. Deleted: age and jq (linux-amd64 curl, which ran BEFORE `dotf tools sync` and so put a second, unpinned copy in `~/.local/bin` on every fresh box), and zoxide, direnv, shellcheck and bats (after the sync, so dead). eza and gh stay behind the host gate: neither is in mise. On a non-amd64 host the leftover cleanup still removes an unrunnable age, jq or shellcheck a past run placed
- [x] Guard: `tests/setup-linux.bats` fails when setup logs "Installing <name>" for any name `versions.conf` marks for mise (mutation-checked with a reintroduced direnv line). It replaces the per-tool "installs X if missing" assertions, which asserted the opposite
- [x] The integration suite's PATH carries mise's shims, as a shell with `mise activate` does: jq now exists only there, and two `verify-setup.bats` tests resolve it from PATH or `~/.local/bin`

### W2 (last part) — eza and gh leave setup; X2's guard

- [x] eza and gh are `system` entries since P5b, and converge's `tools` step installs the catalog on every OS before `legacy-setup` runs (P10), so `setup-linux.sh`'s linux-amd64 downloads of both were a second, unpinned channel (`releases/latest`) that won on every box with no sudo. Deleted, with the host gate, `host_is_linux_amd64` and `remove_unrunnable_tool` (`scripts/utils.sh`). The Mac runs no setup (ADR-045), so the leftover cleanup only reached a non-amd64 Linux host; doctor reporting a stale `~/.local/bin` copy of a `system` entry is #2320
- [x] Guard (#2013 X2): `tests/setup-no-arch-downloads.bats` fails when a code line of an installer (both setup scripts, both install scripts, `scripts/*.sh`) names a release asset by OS and arch together, either order and any separator (`linux-amd64`, `linux_x86_64`, `x86_64-unknown-linux`, `darwin-arm64`, `aarch64-apple-darwin`), and when any named installer is missing; a second test proves it sees each spelling on a code line and skips a comment and install.sh's arch detection (PR-Agent on #2323). Mutation-checked: a reintroduced eza URL in `scripts/utils.sh` turns it red. CI's workflows and `tests/Dockerfile.integration` are outside it: they run on amd64 runners by construction
- [x] The tests that asserted the downloads (`setup-linux.bats` "installs eza/gh if missing", "skips tools already installed"; `setup-windows.bats` eza/gh parity) now read the catalog's apt row; the integration container has no sudo, so the apt rows skip there with the sudo command named, as for tmux
- [x] The documented `~/.local/bin/shellcheck` and `~/.local/bin/bats` paths are gone from CLAUDE.md, `specs/CI-002` verification commands and `guard-no-gui.bats`, whose shellcheck test skipped on every machine set up through mise; ADR-004 carries an update note. CI installs its own bats and is unchanged
- [x] Review triage: the guard catches an installer by any of three shapes: its log line, the `~/.local/bin/<name>` path it writes, or a `/<name>/releases/download/` asset. Mutation-checked with one installer of each shape. The mark format needs no second check: `tools.ParseMiseTools` refuses a mark that is not directly followed by a pin, so the guard and dotf read the same set
- [x] Review triage: the integration PATH also carries `~/.local/bin`, as a shell after setup does. Without it, the three dotf tests in `verify-setup.bats` (#696, #915) skipped on every run, `main` included, while the image builds dotf from the checkout into `~/.local/bin`. A missing dotf now fails them instead of skipping
- [x] doctor reports a leftover `~/.local/bin` copy of a marked pin (placed by the deleted blocks on existing machines; shadowed by mise in a shell, but what a GUI-launched or cron process finds first). Go, needs a release. Shipped in #2196 (`90549a9e`, released in 0.66.0, the current pin): a WARN naming each copy, and `dotf doctor --fix` links it to mise's shim
- [x] `disable_update_warning` in the rendered mise config: a catalog-pinned mise prints "run mise self-update" on every call, and self-update steps outside the pin. Shipped in #2198 (`44db9fc5`, released in 0.66.0): `cli/internal/tools/mise.go` writes it under `[settings]`, pinned by `mise_test.go`

### H1 — herdr through mise (#2013 track H)

- [x] `HERDR_VERSION=0.9.3` under `# mise: cli` (`aqua:herdrdev/herdr`). That entry checks the release digest only; upstream's GitHub attestations for the linux and macOS assets are left to H3 (`gh release verify-asset`), and ADR-044 now says aqua verifies what its registry entry declares
- [x] Guard: `tests/verify-setup.bats` asserts that every marked pin is installed through mise at its version (`mise where name@pin`). Doctor's equivalent check gates only the Windows leg, so Linux had none. On the Mac the same loop reported uv and herdr missing before `dotf tools sync`, and nothing after
- [x] On the Mac: `dotf tools sync` installed herdr and uv, `herdr --version` is 0.9.3, a second run reported nothing to do
- [ ] Windows: H2 (does `mise install herdr` keep `conpty/` next to `herdr.exe`), at the Windows box

### H3 — herdr's attestations and pin drift (#2013 H3)

- [x] Failing test first (`TestRenderMiseConfig_RoutesHerdrToTheBackendThatVerifiesAttestations`), then `attestedBackend`: the rendered `conf.d/dotfiles.toml` carries `[tool_alias] herdr = "github:herdrdev/herdr"` on linux and darwin. mise's github backend verifies the release digest and the GitHub artifact attestations; aqua's registry entry verified the digest only. An alias keeps the short name, so `mise which`, `where`, `bin-paths`, doctor and `verify-setup.bats` are unchanged
- [x] Not on Windows: the herdr zip there carries no attestation, and mise refuses an asset without one (as it refused `python@3.12.6`); Windows herdr is H2's
- [x] Pin drift needs no new check: herdr is a `# mise: cli` pin, so `checkMiseTools` already FAILs when it does not run at its pin, on every OS mise manages; `update.version_check = false` landed with H4 (#2417)
- [ ] An existing aqua install of herdr 0.9.3 is kept: mise treats the version as installed. The next pin bump installs through the github backend, attestations included

### H6 — herdr's agent integrations (#2013 H6)

- [x] Failing tests first (`cli/internal/herdr`): `ParseStatus` reads `herdr integration status`, `Current` is read on the positive (an outdated or re-worded state reinstalls), `Apply` installs only for installed harnesses whose integration is not current, a plan installs nothing, a harness herdr has no integration for is reported, one failed install is named and the rest proceed. Mutation-checked (five mutants killed)
- [x] A third half of `agent-extensions`: the harnesses are the ones `harness/model-map.json` routes to (`harness.Harnesses`), agy is herdr's `antigravity-cli`, herdr absent is a named skip. A harness that is not installed is never touched: herdr would create its config directory (claude and copilot "ensure settings"), the #1312 class. codex is not in the routing map, so it gets none
- [x] Coexistence: `harness bind` keeps herdr's SessionStart group, verbatim from 0.9.3, and a second bind changes nothing (`TestMergeHooksKeepsHerdrsSessionStartGroup`). herdr writes `tui.jsonc` beside the deployed `tui.json` and leaves it byte-identical; opencode merges both (`config/tui.ts`). Measured on the Mac: herdr's own merge added one SessionStart group to `settings.json` and its uninstall removed only that group
- [x] Cost when herdr is not the parent: the claude hook exits after `mktemp` and `cat` (7 ms), and 20 ms with a dead socket
- [x] On the Mac, with this branch's build (2026-10-14): `dotf converge --only agent-extensions` reported `herdr: 5 integration(s) installed`, and a second run `0 installed, 5 current`. `herdr integration status` shows all five current. `records-bind` stays in sync, and the only change to `settings.json` is herdr's one SessionStart group

### H4a — `dotf deploy` merges TOML (#2013 H4, owner decision 2026-10-08)

herdr writes its own `config.toml` (onboarding writes `onboarding = false`; the Settings screen saves into it), so a `replace` entry would overwrite the owner's in-app choices on every deploy and show drift until then: the `.gitconfig` class (lesson 366). The owner chose a TOML merge over `replace` and seed-if-missing.

- [x] Failing tests first (`merge_toml_test.go`): a merge keeps the keys the tool wrote and writes the managed ones; a reformatted destination (comments, key order, inline tables) is in sync and is not rewritten; an absent destination is created and a second run is in sync; an unreadable destination fails naming the config and is untouched; `paths` on a TOML entry is refused at parse time
- [x] `mergeFormat`: the destination's extension picks the codec (`.toml` is TOML, anything else JSON as before), and `deepMerge` is shared. Dependency `github.com/pelletier/go-toml/v2` v2.2.4. Mutation-checked: with `.toml` read as JSON, all five tests fail
- [x] `ai/deploy.json` documents the TOML merge. No manifest version bump: a `dotf` that predates this refuses a TOML merge entry loudly ("source is not a JSON object") rather than replacing the file
- [x] Review triage: a merge source that names no key is refused for every format (`source manages no key`). An empty or comment-only TOML file parsed as an empty table, so the deploy wrote nothing and reported success, or failed on a confusing `stat` when the destination was absent; JSON `{}` had the same hole (`TestDeploy_MergeRefusesASourceThatManagesNoKey`)
- [x] Review triage: a merge entry whose source and destination differ in format is refused when the manifest is parsed. The merge reads the source in the destination's format, so the slip failed on every machine at deploy time with a misleading `source is not a TOML table`; `replace` never parses and may still cross formats (`TestParseManifest_RefusesAMergeWhoseSourceAndDestinationFormatsDiffer`)
- [x] Review triage: the `paths` refusal reads the source's extension, not the destination's. `paths` rewrites the source as JSON before it is installed, so `replace` from `a.toml` to `b.json` passed the parse and failed at deploy on every machine, while `a.json` to `b.toml` was refused although it works (`TestParseManifest_RefusesPathsOnATOMLSource`, mutation-checked)
- [x] Review triage: "manages no key" looks below the top level. A source of empty tables (`[keys]` with no assignment, `{"a": {}}`) named no value and passed; it is refused now, and an empty list still counts as a value
- [x] H4 (the `herdr` entry) lands after the release carrying this is the `DOTF_VERSION` pin (#1814 class): 0.68.0, which carries the TOML merge, is the pin. A machine still on an older dotf (the Mac runs 0.67.0) fails that one entry loudly until it updates; it never replaces the file
- [x] On the Mac, once it runs dotf >= 0.68.0 (the owner installs 0.69.0 next): `dotf deploy herdr`, then `herdr config check` on the deployed file, a second deploy in sync
  - Done on the Mac on 2026-10-13, with dotf 0.69.0 and herdr 0.9.3. `dotf deploy herdr --dry-run` and `dotf deploy herdr` both exit 0 with `in sync  herdr  ~/.config/herdr/config.toml`. `herdr config check` on the deployed file exits 0 with `config: ok`
  - `ai/herdr/config.toml`: prefix `ctrl+s` and worktrees in `~/Projects/.worktrees` (owner, 2026-10-13), Orca's tab keys as `ctrl+alt+[`/`]` beside herdr's prefix ones, `update.version_check = false`, `experimental.pane_history = false`. `herdr config check` (0.9.3) accepts it; `tests/herdr-config.bats` runs it and proves it rejects an unknown key
  - The dst is `{HERDR_CONFIG_DIR}/config.toml`, not the planned `{HERDR_CONFIG_PATH}`: a merge's format comes from the dst's extension when the manifest is parsed, so an extensionless token made every shipped dotf refuse the whole manifest (lesson 411). `HERDR_CONFIG_DIR` in `env-contract.json` holds herdr's default directory per OS; herdr's own `HERDR_CONFIG_PATH` stays unset
  - `TestDeploy_TheShippedHerdrEntryKeepsHerdrsOwnKeys`: the shipped entry and file, deployed over a config herdr wrote, keep `onboarding`, a Settings key and a non-managed binding, and a second plan is in sync
  - `TestManifest_EveryRequiredCommandIsInstalledWhereTheEntryDeploys` counts a `# mise:` pin as installed wherever the catalog installs mise, so herdr needs no exemption on linux and darwin; `claude`'s exemption narrows to Windows, where `setup-windows.ps1` runs the official installer
  - Windows: H2 and mise in the Windows catalog (`herdr/windows` exemption)

### H9 — lazygit in a herdr popup (#2013 H9)

lazygit replaces Orca's diff viewer. Its install was done with the mise pins above (`LAZYGIT_VERSION`, tasks 104-105); this is the binding.

- [x] Failing test first: `TestDeploy_TheShippedHerdrEntryKeepsHerdrsOwnKeys` wants a lazygit popup in the shipped `keys.command`, and was red before the stanza
- [x] `[[keys.command]]` in `ai/herdr/config.toml`: `prefix+d` (d for diff; herdr 0.9.3 leaves it unbound, `prefix+g` is goto, `prefix+shift+g` is new_worktree), `type = "popup"`, `command = "lazygit"`. No alt chord: macOS terminals send Option as Alt only when configured to
- [x] No path argument: herdr 0.9.3's `spawn_popup_command` (`src/app/popup.rs`) falls back to the focused pane's cwd, so the popup shows that pane's worktree, and the bare command also runs under Windows' `cmd.exe /d /c`
- [x] An array of tables through the TOML merge: `TestDeploy_TOMLMergeUnionsAnArrayOfTables` keeps a binding the user added beside the repo's, the merged file still holds tables, and a second merge is in sync
- [x] On the Mac, into a scratch `HERDR_CONFIG_DIR` holding a user binding: `dotf deploy herdr` wrote both, a second deploy said `in sync`, and `herdr config check` on the merged file said `config: ok`
- [x] `herdr config check` does not report a binding that collides with a default (a `prefix+g` command passed it), so the key was chosen against 0.9.3's default keymap (`config-reference.json` at the `v0.9.3` tag). A later herdr that binds `prefix+d` itself would show up as a changed keymap in that file's diff, which a pin bump reads anyway
- [x] The popup runs the bare `lazygit` with herdr's own environment, so it finds lazygit where herdr was started from a shell: W2b puts mise's shims on PATH for every shell we configure. A herdr started with no shell at all is W2b's open PR 2 (`dotf env persist`)
- [x] The test pins the key itself, not only a lazygit popup: `prefix+x` in the stanza fails it
- [x] The merge keeps a stale binding when one is edited, because lists union by equality: #2455 (CLI-113) phase 1. The reader learns manifest version 5 and `listKeys` (path -> identity field): a managed table replaces the destination's table with the same identity, collapses the duplicates the union left, and a managed element without the field is an error. `TestDeploy_MergeReplacesAListElementByIdentity` and its siblings; eleven mutations killed (verification.md lists them)
- [x] #2455 phase 2, after a release carries the version 5 reader (#1814): `ai/deploy.json` to version 5, `"listKeys": {"keys.command": "key"}` on the herdr entry, and the "Until #2455" comment in `ai/herdr/config.toml` removed. Landed as version 6, together with CLI-118 PR 7b (owner, 2026-10-16: one fleet update); removal of a retired binding is #2570.
- [x] #2455's removal half, BUG-149 (#2570): a merge entry records the source it last merged and retires what it no longer declares (ADR-051).
  - [x] Record at `<StateDir>/deploy/merged/<name>.json`: `{version, dst, source}`, the source text after `paths`, decoded by the codec that merged it.
  - [x] Retire table keys, union-list elements by value and `listKeys` elements by identity, only while the destination still holds the recorded value; an emptied table goes too.
  - [x] No record, another `dst` or an unknown version removes nothing and writes a baseline; a record that does not parse is an error naming the file and the recovery.
  - [x] Written after a landed or in-sync merge, never on a dry run; atomic, and an unchanged record is not rewritten.
  - [x] `dotf deploy`, converge's `configs` and doctor's deploy check read the same records (`RunOptions.Records`, `PlanConfigWith`); a state directory that cannot be placed fails only a run that merges (`MergeRecords`).
  - [x] `env.StateDirFor`, so doctor and converge place state from their own seams and no test writes into the machine's state directory.
  - [x] Mutations, thirteen, all killed (verification.md).
- [ ] In daily use on the Mac, recorded at gate V

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
- [x] Docker on the Mac is Colima (owner, #2013 P5b): brew-only `system` entries for colima, docker, docker-compose and docker-buildx. On the Mac the dry-run plans install for all four
- [x] Docker on Linux and Windows (owner, 2026-10-09): the `docker` entry also names apt `docker.io` and winget `Docker.DockerDesktop`, which bundles compose and buildx, so those two entries have no winget name and skip on Windows. Its `command: docker` means a box whose docker came from another channel (Docker's `docker-ce`) is satisfied and nothing is installed over it. Core tools' FAIL for docker names `dotf tools install` on every OS
- [ ] apt `docker-compose-v2` and `docker-buildx` on the compose and buildx entries wait for two things. First, a release carrying `--no-remove`: both packages depend on `docker.io`, which displaces `docker-ce` and `containerd.io`, and `apt-get install -y` removes conflicting packages without asking. Second, a look at msi, whose compose plugin was v2.39.1 on 2026-09-02, which may well be Docker's `docker-compose-plugin`: on a `docker-ce` box those rows fail on every run, because a CLI plugin is not a `command` on PATH and `dpkg-query` does not find Ubuntu's package name. They land only where the box runs `docker.io`
- [x] `dotf tools install` runs `apt-get install -y --no-remove`, so an apt entry can never remove a package to satisfy its dependencies; it fails with apt's message instead
- [x] Doctor's *Docker engine* on Linux tells a socket that refuses this user (`permission denied`, the state apt's `docker.io` leaves until the user joins the `docker` group) from a stopped daemon, and names `sudo usermod -aG docker $USER` instead of `systemctl start` (pr-agent on #2237)
- [x] Colima runs as Homebrew's launchd service: `brew services start colima`, once, restarts it at login. That is the class-3 manager's own mechanism, so no plist is written here; a converge reconciler can take it over later. Doctor's new *Docker engine* section warns while the engine is down, and `dotf doctor --fix` runs that command itself, so no manual step is left, and its Core tools FAIL for docker on the Mac names `dotf tools install`
- [x] Colima's VM size is 4 CPU and 8 GiB (owner, 2026-10-09). It is not a deploy entry: colima treats a `colima.yaml` without `runtime` as empty, so a merged partial file is ignored on first start and then overwritten with defaults, and a `.yaml` merge would fail the installed 0.66.0's `dotf deploy` (it reads every non-TOML merge as JSON). Doctor's *Colima VM size* compares the declared size, a constant beside the check, with the instance's `colima.yaml` and with `colima list --json`. `--fix` rewrites the two top-level lines in colima's own file, which keeps every comment and blank line, and reads the result back before an atomic rename. It then restarts the Homebrew service and reports that the containers stopped; a VM that runs outside the service is left alone, with the command to run. On a new Mac one `dotf doctor --fix` boots Colima twice, first at the defaults and then at the declared size, because colima writes the file only on its first start
- [x] `docker-config` in `ai/deploy.json` merges `cliPluginsExtraDirs: /opt/homebrew/lib/docker/cli-plugins` into `~/.docker/config.json`, because Homebrew's compose and buildx plugins sit outside the docker CLI's default search path. It `requires: colima`, which only the Mac's brew entry provisions, so it needs no `platforms` and the manifest stays at version 3; the dry-run on the Mac skips it until colima is installed
- [x] kubectl 1.35.8 and helm 3.21.4 as `# mise: cli` pins (Linux and macOS; Windows waits for Windows mise, #2013), with doctor's kubectl remedy `dotf tools sync` there (`TestCheckCoreTools_HintsTheProvisioningCommand`, red before the remedy). Was: kubectl 1.35.8 and helm 3.21.4 as `# mise: cli` pins on every OS (kubectl within one minor of kubelab's k3s v1.34.4, helm on kubelab's 3.x line) wait on a release whose version probe reads them: `kubectl --version` and `helm --version` are unknown flags, so `dotf tools sync` 0.66.0 reports both "not running at their pin" after installing them. The per-tool probe table lands with the toolset change (`feat/toolset-infra`). The pins (helm 3.21.4 and kubectl 1.35.8, owner 2026-10-09) and doctor's kubectl remedy (`dotf tools sync`) follow the release that carries it. The #2190 guard covers every marked pin unchanged, and mise.lock stays with Wave 3
- [x] Cask reader (#2013 P5b): a `cask` key, darwin only, never beside `brew`. Measured on the Mac (Homebrew 7.0.9): `brew list --versions appcleaner` exits 1 for an installed cask and `brew list --cask --versions appcleaner` prints `appcleaner 3.7`, so a cask under `brew` would plan an install on every run and fail the post-condition. Tests `TestInstallSystem_CaskIsQueriedAndInstalledAsACask`, `TestLoad_AcceptsACaskEntry` and two `TestLoad_RejectsMalformedSystemEntries` rows; mutations (formula query for a cask, install without `--cask`, the both-keys check off) each turn a test red
- [x] A cask whose app was installed outside Homebrew counts as present (pr-agent on #2234): `brew list --cask` does not record a drag-installed app and `brew install --cask` refuses to overwrite its bundle, so the entry failed on every run. Presence falls back to the cask's `app` artifact from `brew info --cask --json=v2` (shape measured on the Mac: `{"app": ["Obsidian.app"], "target": ...}`), looked up in `/Applications` and `~/Applications`. Test `TestInstallSystem_CaskAppInstalledOutsideBrewIsPresent`; dropping the fallback turns 3 cases red
- [x] Cask entries: the 16 casks of the owner's list in the vault runbook `runbook-macos-workstation-setup` §5.2 (pin 0.73.0 carries the reader). `brew list --cask` on the Mac (2026-10-17) holds all 16 plus `veracrypt`, already declared, and `macfuse`, its dependency, which is left to brew. `tailscale-app` and `google-drive` install a `pkg`, so they carry `needs_sudo` like veracrypt (#2669). A `--plan --only tools` from the branch plans no catalog install with both the dev build and the installed 0.73.0, which loads the catalog; renaming the `vlc` entry's cask to an unknown token makes the same plan report `catalog to install: vlc`, so the entries are read, not skipped

### W2b — shims on PATH (after #2013 P6/P7)

- [x] `mise activate` in `.zshrc`, `.bashrc` and the PowerShell profile, guarded on mise, before direnv and zoxide (#2013 P6, PR #2043)
- [x] The shims directory for non-interactive callers that read a startup file: `.zsh/path.sh` puts mise's shims, then `~/.local/bin`, at the head of PATH, sourced from `.zshenv` (`zsh -c`), the end of `.profile` (`bash -lc`) and the top of `.bashrc`, before its interactive check (bash run by sshd for a command); `zshenv` and `shell-path` are deploy entries on linux and darwin (`tests/shell-path-head.bats`)
  - `bash -c` and `sh -c` read no file and inherit their parent's PATH: covered when the parent is one of the shells above, by construction never by a file of ours (lesson 410)
  - cron is S1's: its jobs run `/bin/sh -c` with cron's own PATH, so the PATH belongs in the job definition S1 renders, not in a startup file
- [ ] PR 2: the PATH a process gets without any shell, persisted by `dotf env persist` with a managed record of the entries it adds: the user environment (HKCU) on Windows, `launchctl` for GUI apps on the Mac
- [x] The doctor's pin half: the pinned CLIs resolve through `mise which`, T3 below. That answers from the mise config whatever the caller's PATH, so it does NOT show the shims directory reaches PATH
- [x] The doctor probe that the shims directory is on PATH for a non-interactive caller: `zsh -c` and `bash -lc`, each started from HOME and a bare system PATH, must list the shims ahead of `~/.local/bin`, else a WARN naming `dotf deploy` (`checkNonInteractivePath`, linux and darwin; Windows waits for PR 2)

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
- [x] PR2: see the next section

### Python — PR2, the pins (after v0.66.0 carried the parser)

- [x] Mark `PYTHON_VERSION=3.13.16` (`# mise: cli`) and `PYYAML_VERSION=6.0.3` (`# mise: python-package`)
- [x] Guard: `tests/versions-conf.bats` now requires `DOTF_VERSION >= 0.66.0` while python is marked, and that both lines stay marked; six mutations (0.65.0, 0.100.0, 1.0.0, package marker removed but python still marked under 0.65.0, package marker removed, python unmarked) give the expected verdict
- [x] CI's macOS job reads `PYYAML_VERSION` and installs `pyyaml==` that pin; `actions/setup-python`'s manifest carries 3.13.16 (checked 2026-10-09), so the job keeps the exact pin rather than a floor
- [x] Windows (owner decision 2026-10-09): `test-windows` failed `[Python]` with `python cannot import at their pin: pyyaml (install mise, ...)`, a remedy Windows cannot take with mise disabled. Without mise a missing package now warns, as `Pinned CLIs (mise)` does for the CLIs; the 3.11 floor still fails everywhere (`TestCheckPython_APackageMissingWithoutMiseWarns`, red before the change)
- [x] On the Mac, after merge: `dotf tools sync`, `pip3 uninstall --user pyyaml` from the system 3.9, then `dotf doctor` passes `[Python]` and `[Pinned CLIs (mise)]`

### Toolset A — kubelab essentials (owner decision on #2013, 2026-10-09)

- [x] `versionArgs` in `cli/internal/tools/mise.go`: the sync probes argocd, helm, hcloud, k9s, kubeconform, kubectl and kustomize with the arguments they take; all seven reject `--version` as an unknown flag (measured on darwin-arm64 in an isolated mise). `TestProbeVersionArgs` (real banners), `TestSync_ProbesAToolWithoutVersionFlagThroughItsArgs` (fails with the kubectl row removed)
- [x] Guard: `tests/versions-conf.bats` refuses those seven under `# mise: cli` while `DOTF_VERSION` <= 0.66.0. Five mutations give the expected verdict: argocd marked under 0.66.0 fails, an indented kubectl marker under 0.66.0 fails, argocd marked under 0.67.0 and 0.100.0 passes, argocd unmarked passes
- [x] Marked for mise, in their own delimited group: terraform 1.16.4 (sha256), mkcert 1.4.4 (release digest only), cloudflared 2026.9.3 (release digest only), trivy 0.74.0 (checksums + cosign)
- [x] Catalog: `ansible` (uv-tool `ansible-core` 2.21.4) and `ansible-lint` 26.9.0, linux and darwin only (no native Windows control node, kubelab lesson-296); `aws` as `system` (brew `awscli`, winget `Amazon.AWSCLI`), no apt entry, because apt carries CLI v1 where it carries it at all and v2 ships only through AWS's installer, so Linux takes that installer
- [x] From the worktree on the Mac: `dotf tools sync --dry-run` lists `to install: cloudflared, mkcert, terraform, trivy`; `dotf tools install --dry-run` plans `install` for ansible 2.21.4, ansible-lint 26.9.0 and aws
- [x] After the release carrying `versionArgs` is the `DOTF_VERSION` pin: argocd 3.5.3, kustomize 5.8.1, hcloud 1.69.0 (Toolset A), k9s 0.51.0 and kubeconform 0.8.0 (Toolset B), and kubectl 1.35.8 and helm 3.21.4 (P5b) are marked. versionArgs (#2238, `f6557b46`) is in v0.67.0 and v0.68.0 (`git merge-base --is-ancestor`), and 0.68.0 is the pin, so the guard's `last` stays 0.66.0 and still holds the floor: with `DOTF_VERSION=0.66.0` it fails naming all seven. Measured in an isolated mise on darwin-arm64: all seven install through aqua and report their pin through their versionArgs; kubectl and helm verify the sha256 file upstream publishes, the other five the release digest only (aqua names no signer for argocd's SLSA provenance)
- Windows: every mise pin waits for Windows mise (#2013); the catalog entries above install there today only where they name a winget package

### Toolset B — k8s daily and lint (stacked on Toolset A)

- [x] Marked for mise in the toolset group: kubectx and kubens 0.11.0, stern 1.34.0 and shfmt 3.14.1 (sha256); yq 4.53.6 (sha512 + cosign); tflint 0.64.0, hadolint 2.15.1 and actionlint 1.7.12 (checksums + GitHub artifact attestations). Each answers `--version` with its version (measured on darwin-arm64 in an isolated mise), so the released dotf counts them
- [x] k9s 0.51.0 and kubeconform 0.8.0 reject `--version` and wait with the rest of the Toolset A deferral
- [x] `gopls` as a `system` entry, brew only. No mise registry entry, Go is not under mise, and apt's gopls lags the pinned Go by several releases, so Linux and Windows have no declared channel for it until Go moves to mise (Wave 3)
- [x] No setup script installs any of these, so each has one owner (`tests/packages-json.bats`, #2195)
- [x] From the worktree on the Mac: `dotf tools sync --dry-run` adds actionlint, hadolint, kubectx, kubens, shfmt, stern, tflint and yq to the install list; `dotf tools install --dry-run` plans `install` for gopls
- Windows: every mise pin waits for Windows mise (#2013)

### Toolset C — ergonomics (stacked on Toolset B)

- [x] Marked for mise in the toolset group: delta 0.19.2, fd 10.5.0 and bat 0.26.1. Their aqua entries declare no checksum, so mise checks the release digest only (the herdr precedent). Each answers `--version` with its version, measured on darwin-arm64 in an isolated mise
- [x] Debian's packages rename two of them (`fdfind`, `batcat`); through mise they keep their own names on every OS, and no rc file aliases either
- [x] `btop` as a `system` entry (apt, brew): aqua's btop carries a Linux asset only, and upstream ships no Windows build (btop4win is a separate project), so Windows has no entry
- Not in scope: making delta git's pager. Declaring the tool does not change `core.pager`; that is a config decision for the owner
- [x] From the worktree on the Mac: `dotf tools sync --dry-run` adds bat, delta and fd to the install list; `dotf tools install --dry-run` plans `install` for btop
- Windows: every mise pin waits for Windows mise (#2013)

### Agents at latest (#2013 D9, row P5c)

- [x] PR1: `# mise: latest` marker; render `latest`; `mise outdated --json` then `mise upgrade` in the sync and converge; doctor plans without asking; `tests/versions-conf.bats` holds the marker behind DOTF_VERSION 0.68.0
- [x] PR1: dry-run on the Mac with a scratch versions.conf marking claude and agy: `claude latest (at least 2.1.296)`, `agy latest (at least 1.3.2)`, `to install: agy, claude`
- [x] PR2 (after the release): mark CLAUDE, PI and AGY under `# mise: latest`; Claude's auto-updater off; delete the agent blocks in `setup-linux.sh` (the pi package reconcile stays); re-check whether aqua's agy has caught up with upstream (1.3.2 vs 1.3.3 on 2026-10-10)
  - [x] Measured on the macmini 2026-10-13: installed claude 2.1.296, agy 1.3.4, pi 1.2.0 (all in `~/.local/bin`); `mise latest` serves claude 2.1.296, agy 1.3.3, pi 1.1.0. aqua still lags upstream agy by one patch. `mise activate` puts its install dirs and shims ahead of `~/.local/bin`, so on an existing machine the mise copy wins and the hand-installed one is shadowed, never run
  - [x] `CLAUDE_VERSION=2.1.296` and `AGY_VERSION=1.3.3` under `# mise: latest`. Each floor is what aqua serves, never upstream's: a floor above aqua fails doctor on every machine
  - [x] pi deferred: aqua serves 1.1.0 while the npm package the machines run is 1.2.0, and `PI_VERSION` has four consumers (setup's `version_gte`, `setup-windows.ps1`, doctor's `matchPinFloor`, CI's exact pi reconcile, BUG-104). It moves when aqua catches up, in its own change
  - [x] Claude's updater off through the mise config, not `ai/claude/settings.json`: that file deploys on Windows too, where `Install-AgentBinary` installs once and the vendor updater is the only one. `RenderMiseConfig` writes `[env] DISABLE_AUTOUPDATER = "1"` when it manages claude (`TestRenderMiseConfig_TurnsOffTheOwnUpdaterOfAToolMiseManages`). Measured with an isolated `XDG_CONFIG_HOME`: `mise env` exports it and `mise exec` (the shims' path) sees it
  - [x] `setup-linux.sh` loses the agy and Claude vendor-installer blocks; the agy config block and the pi package reconcile stay. Guard: `tests/setup-windows.bats` fails when `setup-linux.sh` runs `claude.ai/install.sh`, `antigravity.google/cli/install.sh` or `@anthropic-ai/claude-code` outside a comment (mutation-checked); the old parity assertion now reads the markers
  - [x] X1 (`from-zero.yml`): `The agents resolve on PATH` runs `command -v` and `--version` for claude and agy in the interactive shell, between the first converge and doctor (`tests/from-zero-doctor-gate.bats`)
  - [x] Lands after the macmini runs 0.68.0: 0.67.0 rejects `# mise: latest` as a near-miss, so the tools step and doctor would fail there, and converge never upgrades dotf itself (#2381, CLI-104). The macmini runs 0.70.0 (`dotf version`, 2026-10-15) and `dotf converge --plan` reads `tools 38 pinned CLI(s) at their pin` there. Nothing is left in `setup-linux.sh` to delete: its vendor installers went in #2392 and its pi install arm in #2452; Windows keeps `Install-AgentBinary` (the row below)
  - [x] Evidence: from-zero run 38376819599 on the branch (merged with main after BUG-126) is green on both runners. `The agents resolve on PATH` prints claude and agy from `~/.local/share/mise/installs/<tool>/latest/` on each; the gates read `linux: 8 known, 0 new, 0 stale` and `darwin: 3 known, 0 new, 0 stale`; the second converge is `changed: 0` on both
  - [x] The macOS plugin step this row expected does not exist: converge runs the Claude MCP and plugin steps only through the bare `dotf deploy` that `setup-linux.sh` calls, so macOS never runs them. Filed as BUG-127 (#2387), out of this PR
  - [x] agy on macOS un-skips doctor's Antigravity section, whose master `mcp_config.json` only the setup twins write: filed as CLI-105 (#2384) and listed in `from-zero-known-failures.txt` with that owner
- [x] pi under `# mise: latest`, the change deferred above (#2278, branch `feat/pi-under-mise`). Re-measured 2026-10-14: `mise ls-remote aqua:earendil-works/pi` ends at 1.2.0, the version npm ships and the Mac runs, so aqua has caught up. #2278 planned an npm catalog entry with a new `ignore_scripts` option; the aqua artifact is a native binary, so no npm script runs and the option is not needed. The forcing bug is #2449's red from-zero: on Linux `setup-linux.sh` installed pi in `legacy-setup`, after `agent-extensions`, so herdr's pi integration landed on the second converge Landed in #2452 (`0d7677bf`, released in 0.70.0)
  - [x] `PI_VERSION=1.2.0` under `# mise: latest`; `setup-linux.sh` loses its pi install arm and `$PI_BIN` (the config block and the package reconcile stay); `setup-windows.ps1` keeps its npm install at the floor, because Windows has no mise yet (#2013)
  - [x] Backend: aqua, no `attestedBackend` entry. `mise install -v github:earendil-works/pi@1.2.0` -> `No GitHub artifact attestations found`, so the github backend would verify nothing more than aqua's digest check
  - [x] pi's update notice off under mise: `PI_SKIP_VERSION_CHECK = "1"` in the rendered `[env]` (pi has no updater, but the notice says `pi update`, a second channel outside mise)
  - [x] `dotf pi packages apply` resolves pi on PATH before `~/.local/bin/pi`, so a pre-mise npm copy left there is a fallback, never the one reconciled; doctor's pi hints name `dotf converge --only tools` (Windows: `setup-windows.ps1`) instead of an npm command
  - [x] Guards: the `setup-linux.sh` installer guard in `tests/setup-windows.bats` now also rejects `@earendil-works/pi-coding-agent`; the parity assertion reads `PI_VERSION` under the marker; X1 asserts pi on PATH (`tests/from-zero-doctor-gate.bats`); the integration test asks mise for pi instead of `~/.local/bin/pi`
  - [x] Evidence: from-zero run 38532622634 on #2449, rebased after #2452 merged (`adca557c`), is green on both runners. `The agents resolve on PATH` passes for claude, agy and pi after the first converge; the second converge is `changed: 0` on both; `agent-extensions` reads `herdr: 5 integration(s) installed, 0 current` on the first pass and `herdr: 0 integration(s) installed, 5 current` on the second, on macos-latest and ubuntu-latest alike
- [x] A `latest` tool upstream has not released is behind, not failed (BUG-147, #2526). mise skips a release younger than `minimum_release_age` (24h by default), and near that cutoff `outdated` and `upgrade` disagree: a fresh macOS runner failed `from-zero` on a claude release about a day old. Measured on the Mac on 2026-10-16: `mise upgrade --dry-run claude` warned that 2.1.302 was ignored by `minimum_release_age (24h)` and would install 2.1.301.
  - [x] Apply's post-condition for a `latest` tool is "the upgrade ran and nothing is older than before". A tool still behind is `SyncPlan.Behind`, reported as `still behind their newest release, retried next run`. A regression is still an error (`TestSync_FailsWhenTheUpgradeRegressesALatestTool`).
  - [x] The converge probe re-plans with `CheckUpgrades` off, as doctor does, so it never asks upstream (`TestToolsSync_ALatestToolUpstreamHasNotReleasedIsBehindNotFailed`). Pinned tools keep the strict check. Lesson 461.
  - [x] The 24h delay stays. It is mise's own supply-chain guard, so `latest` means the newest release mise considers safe.
- Windows: the agents stay on `Install-AgentBinary` until Windows mise is measured (#2013)

### apt index refresh (#2013 X1 finding, #2332)

The first X1 run on ubuntu-latest failed its first converge: the runner's apt
lists named a libgit2 build the mirror had dropped, so `apt-get install` 404ed.
A fix to the AC7 path, so it lives here and not in a spec of its own.

- [x] `refreshAptIndex`: one `apt-get update` per Installer, before the first apt install; none on a run with nothing to install
- [x] When sudo wants a password the refresh is not attempted, like the install it precedes
- [x] A refresh that fails for another reason warns and the install tries the index as it is
- [x] The command handed to the person refreshes first: `SudoInstallCommand` prefixes `sudo apt-get update &&`, and the per-package line and the plan note reuse it (pr-agent finding on #2332: an install alone meets the same stale lists)
- [x] X1 on ubuntu-latest passes its first converge once #2332 is on main: from-zero push run 38671157122 on main (`6d92b561`), ubuntu-latest's `The first converge succeeded` step passed

### mise installs no pin asks for (#2375 half b, owner decision 2026-10-14)

A tool installed through mise by hand and since abandoned leaves a shim that
cannot run (`No version is set for shim: pwsh` on the Mac). The owner placed
the fix in the converge, visible in its plan, rather than in a hand cleanup.

- [x] `MiseSync.Prune`: `Plan` lists each install `mise ls --prunable --json` reports whose tool is no pin, as `tool@version`; `Apply` runs `mise prune --yes --tools <names>` after the install and the upgrade, and fails if one is still there (`TestSync_PrunesAnInstallNoPinAsksForAndIsIdempotent`)
- [x] Only the converge sets it: `dotf tools sync` and doctor never ask mise what to delete (`TestSync_DoesNotPruneUnlessAsked`)
- [x] The `tools` step names each orphan in `--plan` (`to prune, asked for by no pin: …`), counts it as a change, and its probe reports one left behind (`TestToolsSync_PlansReportsAndPrunesAnOrphanedInstall`)
- [x] Doctor WARNs on an orphan and names `dotf converge --only tools`; `--fix` removes nothing (`TestCheckMiseTools_AnInstallNoPinAsksForWarnsNamingTheConverge`)
- [x] Non-goals, by construction: a tool another project's tracked `mise.toml` requires is not in mise's prunable list; a superseded version of a pinned tool is left to the removal `mise upgrade` already scheduled
- [x] Evidence on the Mac: plan named `powershell@7.5.11`, the apply removed it, the next plan has nothing to prune and `pwsh` no longer resolves (verification.md)

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
