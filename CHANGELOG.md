# Changelog

## [0.17.1](https://github.com/mlorentedev/dotfiles/compare/v0.17.0...v0.17.1) (2026-06-24)


### Bug Fixes

* **vault:** scaffold number-free context/roadmap filenames (KPM-P) ([#572](https://github.com/mlorentedev/dotfiles/issues/572)) ([4c08b72](https://github.com/mlorentedev/dotfiles/commit/4c08b7270a0a83a74ed6d4261fee46f2f7058fdf))

## [0.17.0](https://github.com/mlorentedev/dotfiles/compare/v0.16.0...v0.17.0) (2026-06-24)


### Features

* **mem:** assemble the Claude session-start adapter + golden gate (CLI-025) ([#569](https://github.com/mlorentedev/dotfiles/issues/569)) ([dd95039](https://github.com/mlorentedev/dotfiles/commit/dd95039855db89a31709520e2be562f979c31cd5))
* **mem:** Claude session-start injectors (CLI-025) ([#566](https://github.com/mlorentedev/dotfiles/issues/566)) ([ecfff9d](https://github.com/mlorentedev/dotfiles/commit/ecfff9d256ac22cd1437cfde179d224f1be693da))
* **mem:** cut over the SessionStart hook to dotf mem session-start, delete the shell cluster (CLI-025) ([#570](https://github.com/mlorentedev/dotfiles/issues/570)) ([0a18373](https://github.com/mlorentedev/dotfiles/commit/0a18373a37682c58f64fac9d3b555cc1dd430903))
* **memlink:** OS-agnostic vault-&gt;memory link primitive (CLI-025) ([#557](https://github.com/mlorentedev/dotfiles/issues/557)) ([edec57c](https://github.com/mlorentedev/dotfiles/commit/edec57c5607deb2e49c3edc1db3545e813b6712c))

## [0.16.0](https://github.com/mlorentedev/dotfiles/compare/v0.15.0...v0.16.0) (2026-06-24)


### Features

* **doctor:** provision knowledge-vault git hooks (OPS-016) ([#553](https://github.com/mlorentedev/dotfiles/issues/553)) ([ca02475](https://github.com/mlorentedev/dotfiles/commit/ca0247542ff484a4b4ab0f5afd69de0771e0521d))
* **mem:** port session-brief agnostic core to dotf mem session-start (CLI-025) ([#554](https://github.com/mlorentedev/dotfiles/issues/554)) ([0cadeac](https://github.com/mlorentedev/dotfiles/commit/0cadeac3c91943535e9d9172b1bf5fe7708ebdbf))

## [0.15.0](https://github.com/mlorentedev/dotfiles/compare/v0.14.1...v0.15.0) (2026-06-24)


### Features

* **mem:** port session-handoff to dotf mem session-end, delete shell twins (CLI-025) ([#546](https://github.com/mlorentedev/dotfiles/issues/546)) ([75c40ea](https://github.com/mlorentedev/dotfiles/commit/75c40eae947666270f97f8cef17ccb94d57ef41d))

## [0.14.1](https://github.com/mlorentedev/dotfiles/compare/v0.14.0...v0.14.1) (2026-06-23)


### Bug Fixes

* **session-handoff:** write records to the project folder, not 00_meta/sessions ([#542](https://github.com/mlorentedev/dotfiles/issues/542)) ([1a185b1](https://github.com/mlorentedev/dotfiles/commit/1a185b1e8a46cbeff39a027b28f920d0efa22227))

## [0.14.0](https://github.com/mlorentedev/dotfiles/compare/v0.13.0...v0.14.0) (2026-06-22)


### Features

* **tools:** dotf tools install — download + checksum-verify catalog tools (CLI-029) ([#526](https://github.com/mlorentedev/dotfiles/issues/526)) ([9d6f2ed](https://github.com/mlorentedev/dotfiles/commit/9d6f2ed28b7c7e55779c6b5450120e938d4b6a08))

## [0.13.0](https://github.com/mlorentedev/dotfiles/compare/v0.12.0...v0.13.0) (2026-06-21)


### Features

* **doctor:** port healthcheck section 4 deployed-config checks ([#522](https://github.com/mlorentedev/dotfiles/issues/522)) ([7f9f3b6](https://github.com/mlorentedev/dotfiles/commit/7f9f3b66ad8caf47283e36c025811d5293afc484)), closes [#509](https://github.com/mlorentedev/dotfiles/issues/509)

## [0.12.0](https://github.com/mlorentedev/dotfiles/compare/v0.11.0...v0.12.0) (2026-06-21)


### Features

* **doctor:** port repo↔deploy-dir drift check (CLI-019 PR-A) ([#513](https://github.com/mlorentedev/dotfiles/issues/513)) ([699e34c](https://github.com/mlorentedev/dotfiles/commit/699e34c17e1694c0ccbc02ab9a998aa142014cac))

## [0.11.0](https://github.com/mlorentedev/dotfiles/compare/v0.10.0...v0.11.0) (2026-06-21)


### Features

* **bash:** opt-in userspace ssh-agent autoload ([#507](https://github.com/mlorentedev/dotfiles/issues/507)) ([cbe1c78](https://github.com/mlorentedev/dotfiles/commit/cbe1c787d9cb735e8deba2a0c24d0485f53ac72d))
* **tools:** declarative package catalog + dotf tools list (CLI-029 pilot) ([#508](https://github.com/mlorentedev/dotfiles/issues/508)) ([8332630](https://github.com/mlorentedev/dotfiles/commit/8332630260d9c4c8cb671f77bc09a33b985c4163))


### Bug Fixes

* **harness:** CRLF-robust refresh + reconcile skill records with vault SSOT ([#511](https://github.com/mlorentedev/dotfiles/issues/511)) ([7328965](https://github.com/mlorentedev/dotfiles/commit/7328965e6981a5a1db8c6f4b063c620367ceed30))

## [0.10.0](https://github.com/mlorentedev/dotfiles/compare/v0.9.4...v0.10.0) (2026-06-21)


### Features

* **doctor:** port the Orca Copilot hook (DX-006) check into dotf doctor ([#505](https://github.com/mlorentedev/dotfiles/issues/505)) ([3701936](https://github.com/mlorentedev/dotfiles/commit/3701936bedf68df6d3ddf4c86119d3999620ed30))
* **handoff:** cache-stable block placement + agnostic lessons-staleness signal ([#502](https://github.com/mlorentedev/dotfiles/issues/502)) ([9de802b](https://github.com/mlorentedev/dotfiles/commit/9de802b6a760019bccf90bd51ec28d13adf748c5))
* **ssh:** add *-ext bastion aliases for off-LAN fleet access ([#503](https://github.com/mlorentedev/dotfiles/issues/503)) ([260008f](https://github.com/mlorentedev/dotfiles/commit/260008f1fe4de6003e625056ac2c4cd3b3f6d4c4))

## [0.9.4](https://github.com/mlorentedev/dotfiles/compare/v0.9.3...v0.9.4) (2026-06-20)


### Bug Fixes

* **profile:** resolve nan-debug.sh via DOTFILES_REPO_DIR, not a hardcoded literal ([#482](https://github.com/mlorentedev/dotfiles/issues/482)) ([2a23355](https://github.com/mlorentedev/dotfiles/commit/2a2335562cf5a292f59a4794b911c4c596a056fc))
* **shell:** rename gp-&gt;gpr (collision) and source utils.sh declaratively ([#484](https://github.com/mlorentedev/dotfiles/issues/484)) ([e1c0090](https://github.com/mlorentedev/dotfiles/commit/e1c00900a1dd93988020962042f51f09e98bdc35))

## [0.9.3](https://github.com/mlorentedev/dotfiles/compare/v0.9.2...v0.9.3) (2026-06-20)


### Bug Fixes

* **setup:** enforce versions.conf pins as minimums, not presence or exact match ([#480](https://github.com/mlorentedev/dotfiles/issues/480)) ([a9fdd60](https://github.com/mlorentedev/dotfiles/commit/a9fdd60f8872e6e60642806ff23577d083536913))

## [0.9.2](https://github.com/mlorentedev/dotfiles/compare/v0.9.1...v0.9.2) (2026-06-20)


### Bug Fixes

* **setup:** align Linux deploy strategy with Windows always-overwrite ([#476](https://github.com/mlorentedev/dotfiles/issues/476)) ([d653db3](https://github.com/mlorentedev/dotfiles/commit/d653db30608695ada867e36803022d57aafa919b))
* **setup:** correct Compare-Object -SyncId errors and add pi version drift check ([#474](https://github.com/mlorentedev/dotfiles/issues/474)) ([58cd9e3](https://github.com/mlorentedev/dotfiles/commit/58cd9e3389b3c735754c8dd571ee0c5a479ff3db))

## [0.9.1](https://github.com/mlorentedev/dotfiles/compare/v0.9.0...v0.9.1) (2026-06-20)


### Bug Fixes

* **ci:** add PRs to bitácora board via gh CLI ([#470](https://github.com/mlorentedev/dotfiles/issues/470)) ([01d7bb5](https://github.com/mlorentedev/dotfiles/commit/01d7bb5e9c7c17724a3aec33f0aebb5b483262f4))
* **ci:** rewrite add-to-project PR step with GraphQL API ([#473](https://github.com/mlorentedev/dotfiles/issues/473)) ([f2312a5](https://github.com/mlorentedev/dotfiles/commit/f2312a525a7055f7a5dcc90a6d3cf5cf6068387f))

## [0.9.0](https://github.com/mlorentedev/dotfiles/compare/v0.8.1...v0.9.0) (2026-06-20)


### Features

* **setup:** autostart the hive daemon via Startup folder when Task Scheduler is blocked ([#467](https://github.com/mlorentedev/dotfiles/issues/467)) ([12c8d50](https://github.com/mlorentedev/dotfiles/commit/12c8d502350d15d6262d11c5ca47f5034b7469fc))

## [0.8.1](https://github.com/mlorentedev/dotfiles/compare/v0.8.0...v0.8.1) (2026-06-20)


### Bug Fixes

* **session-start:** match Claude Code's path encoding for the memory junction ([#466](https://github.com/mlorentedev/dotfiles/issues/466)) ([298fb60](https://github.com/mlorentedev/dotfiles/commit/298fb602fe90fd1647b7f573efb88923973a59eb))
* **setup:** install Bun on Windows so the claude-mem worker can start ([#464](https://github.com/mlorentedev/dotfiles/issues/464)) ([2785235](https://github.com/mlorentedev/dotfiles/commit/27852355ca610ff400aefb904faec5adcd82b1e2))

## [0.8.0](https://github.com/mlorentedev/dotfiles/compare/v0.7.0...v0.8.0) (2026-06-19)


### Features

* **harness:** harden ADR-025 cross-machine path resolution end-to-end (HARNESS-027, [#457](https://github.com/mlorentedev/dotfiles/issues/457)) ([#458](https://github.com/mlorentedev/dotfiles/issues/458)) ([6594288](https://github.com/mlorentedev/dotfiles/commit/6594288720d4724ec33ff79c3d7ca831f31554d4))


### Bug Fixes

* **windows:** re-apply Orca Copilot hook fix idempotently (DX-006) ([#456](https://github.com/mlorentedev/dotfiles/issues/456)) ([6f045a4](https://github.com/mlorentedev/dotfiles/commit/6f045a43da42bd90d95263ec5ddd4b574bc3f6d1))

## [0.7.0](https://github.com/mlorentedev/dotfiles/compare/v0.6.0...v0.7.0) (2026-06-19)


### Features

* **setup:** install dotf from the published release binary on Windows (WIN-006, [#451](https://github.com/mlorentedev/dotfiles/issues/451)) ([#453](https://github.com/mlorentedev/dotfiles/issues/453)) ([1f22769](https://github.com/mlorentedev/dotfiles/commit/1f227693e67501a5b0a6f6ae8f5a6873a6aa943a))

## [0.6.0](https://github.com/mlorentedev/dotfiles/compare/v0.5.1...v0.6.0) (2026-06-19)


### Features

* **cli:** cross-machine path resolution via dotf env generate (CLI-016, [#445](https://github.com/mlorentedev/dotfiles/issues/445)) ([#447](https://github.com/mlorentedev/dotfiles/issues/447)) ([60d120b](https://github.com/mlorentedev/dotfiles/commit/60d120bdd4d10eb75368b4fb6abcf560df57af48))
* **setup:** wire resolved vault path into setup + hive daemon (HARNESS-024, [#446](https://github.com/mlorentedev/dotfiles/issues/446)) ([#448](https://github.com/mlorentedev/dotfiles/issues/448)) ([4d8ce18](https://github.com/mlorentedev/dotfiles/commit/4d8ce184af2fe7bf1d4f3307e32b649be7ef0119))

## [0.5.1](https://github.com/mlorentedev/dotfiles/compare/v0.5.0...v0.5.1) (2026-06-18)


### Bug Fixes

* **setup:** install pi into ~/.local so GUI/ADE launchers resolve it ([#440](https://github.com/mlorentedev/dotfiles/issues/440)) ([f22e425](https://github.com/mlorentedev/dotfiles/commit/f22e425e47e6da45dcbfda9da3b4434ab3f693aa)), closes [#426](https://github.com/mlorentedev/dotfiles/issues/426)

## [0.5.0](https://github.com/mlorentedev/dotfiles/compare/v0.4.0...v0.5.0) (2026-06-18)


### Features

* **doctor:** detect expiring or invalid GitHub PATs before they break CI ([#427](https://github.com/mlorentedev/dotfiles/issues/427)) ([52695f3](https://github.com/mlorentedev/dotfiles/commit/52695f32e1de33b46f1e24a3f90322fb6b95d7db)), closes [#422](https://github.com/mlorentedev/dotfiles/issues/422)


### Bug Fixes

* **ci:** deterministic Windows tool install — age, eza, zoxide (BUG-025/024) ([#425](https://github.com/mlorentedev/dotfiles/issues/425)) ([fdb27f8](https://github.com/mlorentedev/dotfiles/commit/fdb27f854ed2dfb16d937975c5ff21d0f978d0aa))
* **doctor:** resolve PAT from any mapped env alias, not just the first ([#429](https://github.com/mlorentedev/dotfiles/issues/429)) ([add7d1d](https://github.com/mlorentedev/dotfiles/commit/add7d1de3104fd61bafd15fb523fc6d586ae5b63))

## [0.4.0](https://github.com/mlorentedev/dotfiles/compare/v0.3.0...v0.4.0) (2026-06-17)


### Features

* **ci:** adopt release-please (version + changelog + tag automation) ([#416](https://github.com/mlorentedev/dotfiles/issues/416)) ([a17c917](https://github.com/mlorentedev/dotfiles/commit/a17c917ba66d08b9d75932c1ff7291b963430590)), closes [#369](https://github.com/mlorentedev/dotfiles/issues/369)
* **guard:** complete GUARD-001 single-sink (gitignore, global install, AGENTS.md) ([#415](https://github.com/mlorentedev/dotfiles/issues/415)) ([a4cd005](https://github.com/mlorentedev/dotfiles/commit/a4cd005964480c87de36f0f12c2dfd76f6399068))
* **session-start:** extract agent-agnostic session-brief core (ADR-023) ([#413](https://github.com/mlorentedev/dotfiles/issues/413)) ([5f34eee](https://github.com/mlorentedev/dotfiles/commit/5f34eeeef18e28d81f3cd15baa82a1d5f1c6221c))


### Bug Fixes

* **ci:** point release-please at the existing RELEASE_TOKEN secret ([#419](https://github.com/mlorentedev/dotfiles/issues/419)) ([242f6d1](https://github.com/mlorentedev/dotfiles/commit/242f6d1ff1a61ca65ad72041f4b6ad634e8fdd2c)), closes [#369](https://github.com/mlorentedev/dotfiles/issues/369)
* **guard:** deploy the memory-sink dispatcher + wire core.hooksPath ([#418](https://github.com/mlorentedev/dotfiles/issues/418)) ([#420](https://github.com/mlorentedev/dotfiles/issues/420)) ([3d551a6](https://github.com/mlorentedev/dotfiles/commit/3d551a690b5fc0a941953276c6246ebbce9ce493))

## Changelog

Maintained by [release-please](https://github.com/googleapis/release-please) from Conventional Commits. Do not edit by hand.

## Features

- 2026-05-17: feat(AI-012): port Claude skills to OpenCode commands (d326954)
- 2026-05-17: feat(aliases): add oclog for live opencode log tailing (91ebdf7)
- 2026-05-16: feat(agents-md): salvage MCP rules from stale refactor branches (c6e049b)
- 2026-05-16: feat(AI-011): bootstrap opencode + canonical AGENTS.md migration (0d7fed8)
- 2026-05-15: feat(doctor): SessionStart silent doctor + binary version pinning (4e9798a)
- 2026-05-15: feat(doctor): declarative env contract + doctor.sh/ps1 with --check/--fix (d6ced62)
- 2026-05-14: feat(scripts): add init-repo-standards generator (SDD-010) (adfd638)
- 2026-05-14: feat(scripts): SessionStart hook surfaces repo specs/ state (SDD-016) (eb32d75)
- 2026-05-14: feat(scripts): vault working-tree integrity check at session start (SDD-017) (c39e6f6)
- 2026-05-14: feat(scripts): add init-repo-agents bootstrap for AGENTS.md (SDD-013) (a7f9b9a)
- 2026-05-13: feat(claude-md): add claude-mem MCP rules + dual-memory protocol pointer (c8a93f6)
- 2026-05-13: feat(setup): auto-link vault-hosted skills into ~/.claude/skills/ (2137ffa)
- 2026-05-13: feat(scripts): add init-spec + archive-spec for SDD per-feature workflow (d77021b)
- 2026-05-12: feat(scripts): opt-in shell startup profiling (8a83e1c)
- 2026-05-12: feat(scripts): add changelog-gen.sh + initial CHANGELOG.md (63d4a36)
- 2026-05-12: feat(scripts): add diff-check.sh to detect repo ↔ deploy-dir drift (3f7af6a)
- 2026-05-12: feat(tmux): add focus-events, vi visual-mode bindings, slower status refresh (c23ec99)
- 2026-05-11: feat(tmux): copy selection to system clipboard via xclip (fd361f6)
- 2026-05-11: feat(tmux): integrate tmux with versioned config and Linux install (239e715)
- 2026-05-08: feat(scripts): add claude-mem-heal for upstream v12/v13 packaging bugs (053bad8)
- 2026-03-29: feat: skills ecosystem overhaul — 23 to 17 skills, CSO audit, Standing Orders (61c4b38)
- 2026-03-27: feat: add obs-cli wrapper for Obsidian CLI (Linux + Windows) (91ba19d)
- 2026-03-26: feat: unified workflow protocol — area-agnostic CLAUDE.md, full vault entry in init-project, work SDK detection in session hooks (3884d4f)
- 2026-03-26: feat(hooks): auto-create memory junction/symlink on session start (999a478)
- 2026-03-26: feat(setup): bidirectional memory sync on Windows via junctions (ef53bd4)
- 2026-03-25: feat(ai,secrets): add engineering discipline rules, secrets reconciliation, cleanup (a9fb76a)
- 2026-03-24: feat(claude): add self-maintaining memory system (0204133)
- 2026-03-16: feat(setup): auto-install 10 developer tools on Linux and 7 on Windows (e1e4746)
- 2026-03-10: feat(ai): add aider integration with 3-tier OpenRouter model config (c515c69)
- 2026-03-07: feat(setup): add hive MCP server with auto-upgrade to both Linux and Windows (3ddc041)
- 2026-02-28: feat(ai): add kc / kca shortcuts for quick access as aliases (e153f7e)
- 2026-02-28: feat(ai): knowledge crystallization system — bash + PowerShell + auto-discovery (067ee84)
- 2026-02-27: feat(setup): auto-register Claude Code SessionStart hook (03cb64e)
- 2026-02-27: feat: add Claude Code SessionStart hook for vault health context (917c03b)
- 2026-02-27: feat: add vault-health.sh and integrate Obsidian CLI checks (fc85d1d)
- 2026-02-27: feat(shell): add obsidian alias with --no-sandbox for Linux AppImage (dfe4c37)
- 2026-02-26: feat(ci): add container-based integration test for setup-linux.sh (465fd2a)
- 2026-02-26: feat: add versions.conf and healthcheck.sh (P1 backlog) (72fdb29)
- 2026-02-26: feat(shell): standardize set -euo pipefail across standalone scripts (fd5ef7e)
- 2026-02-26: feat(ai): add auto-memory to Neural Hive context sync phase (b01af3e)
- 2026-02-26: feat: persist MCP servers globally and auto-memory via vault (46abe35)
- 2026-02-26: feat: persist MCP servers and auto-memory across machines (ffd56fa)
- 2026-02-23: feat(claude): add no Co-Authored-By policy to global CLAUDE.md (4befe55)
- 2026-02-23: feat: set nano as default UNIX editor (3fa1bea)
- 2026-02-22: feat(ai): implement neural hive protocol and standardize vault (6c32d26)
- 2026-02-22: feat: add PSScriptAnalyzer linting to CI for PowerShell scripts (34b94d5)
- 2026-02-22: feat: add PSScriptAnalyzer linting to CI for PowerShell scripts (4197b4d)
- 2026-02-22: feat: add secrets_show command and SSH config deployment (9712196)
- 2026-02-21: feat: add file-based secrets support for kubeconfig and multiline files (9da08d8)
- 2026-02-21: feat: add prd, qa-plan, prd-to-issues skills and automate plugin installation (e1a9ad9)
- 2026-02-21: feat: add prd skill for interactive requirements gathering (ce7263e)
- 2026-02-18: feat: auto-install claude-mem plugin in setup scripts (8af28a7)
- 2026-02-16: feat: apply Anthropic Claude Code best practices across project and global config (5bb8aae)
- 2026-02-16: feat: apply Anthropic Claude Code best practices across project and global config (0fd3557)
- 2026-02-13: feat: add POLLEX_API_KEY (4e2bafe)
- 2026-02-11: feat: add excalidraw MCP server registration to setup scripts (2148176)
- 2026-02-10: feat: auto-create python symlink in setup for version-agnostic command (37950fe)
- 2026-02-10: feat: upgrade Go to 1.26.0 and prepend tool paths for system override (83e938a)
- 2026-02-08: feat: add bun PATH to .bashrc and .zshrc for persistent installation (c7d7590)
- 2026-02-04: feat: add USB backup of secrets with VeraCrypt support (032965b)
- 2026-02-02: feat: add Windows PowerShell support and rename setup scripts   - Add setup-windows.ps1, powershell/profile.ps1, scripts/init-project.ps1   - Rename install.sh → setup-linux.sh, change claude-init → project-init   - Delete obsolete .bat files   - Update all documentation (5a568af)
- 2026-02-01: feat: consolidate AI configuration and implement Claude Code skills   - Refactor CLAUDE.md and GEMINI.md   - Restructure skills to official format (SKILL.md with YAML frontmatter)   - Add skills: audit, refactor, test, doc, docker   - Update install.sh to copy skill directories and extract Gemini prompts   - Update init-project.sh for new skill structure   - Add docs/AI.md with complete setup and workflow guide   - Clean up deprecated versioned files (6037ada)
- 2026-01-14: feat: implement dotfiles sync and enhance secrets management 	- Add `dotfiles-sync` workflow for local vs repo synchronization 	- Update `github-secrets-manager` to support uploading from `env-mapping.conf` 	- Add auto-sync capabilities to `secrets_add` and `secrets_rotate` 	- Register `PYPI_TOKEN` in secrets mapping 	- Update documentation and test suite (87eab20)
- 2026-01-14: feat: implement dotfiles sync and enhance secrets management 	- Add `dotfiles-sync` workflow for local vs repo synchronization 	- Update `github-secrets-manager` to support uploading from `env-mapping.conf` 	- Add auto-sync capabilities to `secrets_add` and `secrets_rotate` 	- Register `PYPI_TOKEN` in secrets mapping 	- Update documentation and test suite (290c362)
- 2026-01-11: feat: add secrets availability in bash based on encrypted files with a config file mapping, add test suite as part of precommit hook. (09235cb)
- 2025-11-19: feat: introduce Claude AI aliases and prompt function,  and restructure documentation. (6280f4c)
- 2025-11-19: feat: initial commit with Claude optimization (25e7324)
- 2025-11-18: feat: support gemini-cli with custom GEMINI.md and prompts for easy use common to all projects (5018898)
- 2025-03-23: feat: add env file path as input parameter to setup-gh-secrets (5a3f6c6)

## Bug Fixes

- 2026-05-17: fix(BUG-001): update integration test for detect-and-act Copilot logic (5dfed05)
- 2026-05-17: fix(BUG-001): correct Copilot verification + gate config on extension presence (22fe726)
- 2026-05-17: fix(tmux): pass truecolor through to ghostty (TERM=xterm-ghostty) (fa36796)
- 2026-05-16: fix(lint): replace em dash with ASCII in profile.ps1 OpenCode comment (9d284b9)
- 2026-05-16: fix(AI-011): update CLAUDE/GEMINI deployment marker after AGENTS.md migration (a875103)
- 2026-05-15: fix(setup-windows): replace em dash with ASCII to satisfy PSScriptAnalyzer (464eecf)
- 2026-05-15: fix(setup): idempotent claude plugin install to stop .claude.json truncation (0d805f9)
- 2026-05-15: fix(setup): stop deleting vault content via symlink follow in skill sync (022d535)
- 2026-05-15: fix(setup-linux): close mcp-servers.json + doctor.sh parity gaps from Windows-side work (2775aa8)
- 2026-05-15: fix(doctor): persist structural env vars in profiles + section summaries (3176804)
- 2026-05-15: fix(scripts): claude-mem-heal Windows parity (zod/v3 plugin bug) (1b4288b)
- 2026-05-15: fix(setup): idempotent MCP registration + self-healing scheduled task (4f8a2c6)
- 2026-05-15: fix(setup): self-healing SessionStart hook + LF gitattributes (closes #20) (b5a3f6c)
- 2026-05-13: fix(setup): use ASCII hyphen in Write-Warn to satisfy PSScriptAnalyzer (001efa3)
- 2026-05-12: fix(secrets): keep env, deployed, and repo in sync after every mutation (295b6f3)
- 2026-05-08: fix(scripts): remove unused cwd_slug local in claude-session-start (f030959)
- 2026-03-29: fix(ci): update skill count threshold from 18 to 15 after ecosystem overhaul (103ec41)
- 2026-03-29: fix(ci): guard crontab call for environments without cron (8262808)
- 2026-03-27: fix(ci): replace remaining non-ASCII chars in init-project.ps1 (27dc6af)
- 2026-03-26: fix(ssh): aws1 uses MagicDNS instead of hardcoded Tailscale IP (c6ab454)
- 2026-03-26: fix(ci): replace non-ASCII chars in init-project.ps1 to pass PSScriptAnalyzer (60417ce)
- 2026-03-25: fix(setup): always deploy AI config regardless of CLI presence (c3be688)
- 2026-03-25: fix(tests): skip Gemini integration tests when CLI not installed (13c07a1)
- 2026-03-25: fix(tests): update healthcheck section count from 7 to 8 (f2c1c13)
- 2026-03-22: fix(ssh): sync config with live host inventory (14edd90)
- 2026-03-18: fix(setup): symlink .gitconfig instead of copying (6c60d20)
- 2026-03-17: fix(sync): replace git pull with rsync for local installation (73cdc2e)
- 2026-03-16: fix(ci): resolve 3 CI failures from developer tools addition (971e01e)
- 2026-03-12: fix: resolve 26 bugs and close Windows/Linux parity gaps (82481ef)
- 2026-02-28: fix(ci): remove non-ascii characters to resolve PSScriptAnalyzer BOM error (b36fac7)
- 2026-02-26: fix(ci): remove false CLAUDE.md assertion from integration tests (e2cbd36)
- 2026-02-23: fix: close setup parity gaps between Linux and Windows (866314e)
- 2026-02-22: fix: harden AI rules deployment and fix SSH directory copy (0736c74)
- 2026-02-18: fix: replace non-interactive claude plugin install with manual instruction (add9285)
- 2026-02-16: fix: handle zsh nomatch error in secrets_clean glob patterns (791c0e6)
- 2026-02-16: fix: install zsh in CI for zsh compatibility tests (409c347)
- 2026-02-07: fix: harden all shell scripts for POSIX/zsh compatibility and add 95 bats-core tests (196a4e5)
- 2026-01-15: fix: add shellcheck disable for zsh-specific syntax (544a8fe)
- 2026-01-14: fix: remove local keyword, skip GITHUB_ secrets, add CI workflow (6886c3b)
- 2025-03-23: fix: issue in .bashrc (4496a06)

## Refactoring

- 2026-05-15: refactor(claude-md): compact MCP Server Usage Rules to bullets + links (f96afd8)
- 2026-05-15: refactor(claude-md): trim Neural Hive Loop phases + Vault Structure (e6b4b8e)
- 2026-05-15: refactor(claude-md): replace duplicated standards with vault pointers (b7422b0)
- 2026-02-28: refactor(ai): align project init and agent prompts with Neural Hive protocol (2e1eda8)
- 2025-11-28: refactor: standardize shell configs and fix APPS_HOME path (ed02dc3)

## Documentation

- 2026-03-12: docs(readme): update test count, add aider aliases and new scripts (6e687b1)
- 2026-01-12: docs: add SECRETS.md and reorganize documentation structure (c84292e)

## Tests

- 2026-02-28: test(ci): pass PSScriptAnalyzer settings to bats test (b4b178c)

## Chores

- 2026-05-17: chore(TERM-001): scaffold + filled proposal for Ghostty Linux bootstrap (11270f3)
- 2026-05-17: chore(AI-011): archive spec + correct opencode.jsonc Layer 1 comment (f9d2497)
- 2026-05-16: chore(AI-011): full aider sunset — Linux, Windows, README, env-contract (1a6a15f)
- 2025-12-05: chore: add sops age key file path as env variable (f4a58a7)
- 2025-11-19: chore: Remove claude boost.sh script and its installation command from install.sh. (7f96a05)
- 2025-11-14: chore: prioritize ~/go/bin in $PATH to use correct go-task v3 from v2 (ad8a9a8)
- 2025-08-29: chore: add zoho codes (bdc088a)
- 2025-08-29: chore: add cloudflare token (9ac5a47)
- 2025-08-26: chore: add pre-commit hooks and validation script (117dbc9)
- 2025-08-26: chore: add sensitive scripts and refactor documentation (141b1f3)
- 2025-06-20: chore: add age encryption for secrets (3f7ba2d)
- 2025-03-23: chore: add gh secrets configuration from env file (12391d1)
- 2025-03-23: chore: add node dependencies checkout (e91d2f2)
- 2025-03-23: chore: add dependencies checkout for aliases (79dacc2)
- 2025-03-22: chore: add custom functions to zsh terminal (32de27b)

## Other

- 2026-03-01: secrets: add openrouter api key (0df59eb)
- 2026-02-22: doc: migrate docs/ to private vault and extract ADRs (5a045cb)
- 2026-02-21: doc: minor details (01abc2b)
- 2026-02-18: doc: remove excalidraw MCP server from all configs and docs (babdcb1)
- 2026-02-10: git commit -m "feat: add drawio MCP server registration to setup scripts" (7340e37)
- 2026-02-08: doc: update LLM files with obsidian vault info (d002719)
- 2026-02-07: doc: add backlog file (4e88cd0)
- 2026-02-04: doc: add claude code plugins installation (81e88dd)
- 2025-03-24: bug: remove --icon flag from eza aliases (250bd8c)
- 2025-03-23: bug: fix issue in nvm alias inizialiation (fc73945)
- 2024-11-01: First commit (c40614f)
- 2024-11-01: Initial commit (3cb97d3)
