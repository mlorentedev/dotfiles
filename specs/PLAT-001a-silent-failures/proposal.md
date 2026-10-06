---
id: "PLAT-001a-silent-failures"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-10-05"
issue: "mlorentedev/dotfiles#2013"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "PLAT-001 Wave 1: silent-failure bug fixes that also break Linux arm64 (setup reports SUCCESS for non-executable binaries; no skills deploy on bash 3.2); the stalled specs are outside this session's scope (16 active, limit 10, 2026-10-05)"
---

# PLAT-001a: Stop the silent failures (Wave 1 of PLAT-001)

## Why

<!-- from issue #2013: PLAT-001: [EPIC] OS-agnostic from-zero bootstrap — macOS bring-up as the forcing function -->

We audited the bootstrap on a fresh macOS arm64 machine on 2026-10-05 (ledger in #2013). Several steps fail while reporting success, and they fail the same way on Linux arm64:

- age, eza, jq, gh and shellcheck are fetched from hard-coded `linux-amd64` URLs, so setup logs SUCCESS and leaves binaries that cannot execute ahead of working ones on PATH.
- `compile-harness.sh` needs bash 4 (`mapfile`), and its failure is downgraded to a warning, so no skills deploy.
- The catalog installs before uv exists.
- The rc files break PATH and git auth, and the first run overwrites `~/.zshrc` without a backup.

A step that can fail while reporting success needs a post-condition (`.claude/CLAUDE.md`). This track adds one wherever one is missing.

## What

One PR per row of #2013 track W. Each fix lands with the guard that would have caught it.

- **W1:** `dotf tools install` executes every tool it installs before calling the install a success.
  - For a github-release tool, the staged binary must run `--version` and report a version at or above the pin. Only then is it placed in `Dest`, so a binary that cannot execute never reaches PATH.
  - For npm and uv-tool installs, the tool must resolve and run on PATH after the package manager returns 0.
  - Any failure is an error naming the tool, not a SUCCESS line.
- **W2:** age, eza, zoxide, shellcheck and jq move into `packages.json` with per-OS/arch assets, and the hard-coded URL blocks leave `setup-linux.sh`.
- **W3:** remove bash-4-only constructs from `compile-harness.sh`, add a guard against them, and run it under bash 3.2.
- **W4:** setup fails loudly when the harness compile/deploy fails.
- **W5:** package managers are installed before `dotf tools install`.
- **W6:** `.gitconfig` resolves gh through PATH, `.bashrc` stops resetting PATH, and `.zshrc` guards oh-my-zsh.
- **W7:** back up an unmanaged rc file before the first overwrite.
- **W8:** portable `stat`, `chmod` and checksum forms.
- **W9:** the rendered `opencode.jsonc` is mode 0600.
- **W10:** a macOS CI leg.

## Out of scope

- Declaring OS as a manifest dimension, the bootstrap entrypoint and the Brewfile belong to track P (a later spec).
- Toolchain management via mise belongs to track T.
- launchd and services belong to track S.

## Risks / open questions

- **W1 runs the downloaded binary before placing it.** That is new code execution at install time, but it runs only after the sha256 gate passes, and it runs the same binary the user would run next anyway. *Resolved:* probe after the checksum, never before.
- **Some tools print no semver for `--version`.** The probe treats "no version" as a failure. Every current github-release entry (sops) prints one. W2 adds tools that must be checked one by one (`age --version` prints `v1.3.1`, which semverRE matches).
- **npm global bin may not be on PATH** (an nvm/prefix mismatch). With W1 that becomes a loud failure instead of a silent "installed". This is intended, and the message says so.

## Acceptance criteria

- [ ] AC1: a github-release install whose binary cannot execute, or reports no version, fails with an error and leaves `Dest` untouched.
- [ ] AC2: a github-release install whose binary reports a version below the pin fails with an error and leaves `Dest` untouched.
- [ ] AC3: an npm or uv-tool install that exits 0 but whose tool does not run on PATH afterwards fails with an error naming the tool.
- [ ] AC4: the existing happy paths (install, upgrade, skip, missing-manager) keep their results and messages.

## References

- Epic: #2013 (track W), ledger comment F-030, F-031, F-047, F-048
- ADR-036 (install channels; pins are floors), ADR-003 (bash 3.2 + zsh)
- `.claude/CLAUDE.md`: "A script whose failure path reports … needs a preflight"
