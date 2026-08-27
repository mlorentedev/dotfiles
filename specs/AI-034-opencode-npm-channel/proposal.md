---
id: "AI-034-opencode-npm-channel"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1294"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# AI-034-opencode-npm-channel

> **Naming**: file lives at `<repo>/specs/AI-034-opencode-npm-channel/proposal.md`. `AI-034-opencode-npm-channel` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1294: AI-034: opencode is installed through three channels and its version parse accepted 'locked.', so setup never converges the binary on PATH -->

Measured 2026-08-27 on the Windows work box: `setup-windows.ps1` printed *"OpenCode locked. below pinned minimum 1.16.2, upgrading..."* and then *"still locked. after winget install 1.16.2: another install shadows it in PATH; converge that install instead"* — a version parse that accepted a banner word as a version, and a manual instruction as the remedy. opencode reached the two machines through three channels (npm-global, winget, the curl script), seven fragile `--version` parse sites lived across both setup scripts, and no document anywhere said which channel a tool should come from. The repo already had the right primitive — `packages.json` + `dotf tools install`, with pin-as-floor and a semver-regex probe — used for `bw` and `sops` and nothing else.

## What

- **Decision recorded as ADR-036:** one install channel per tool class on every OS; npm for node-distributed agents; `packages.json` is the pin SSOT for catalog tools; version detection happens once in Go; a second copy on PATH is a doctor finding, not a thing setup converges silently.
- `packages.json` gains `opencode` (`npm`, `opencode-ai`, 1.16.2); `OPENCODE_VERSION` leaves `versions.conf`. Both setup scripts already call `dotf tools install`; their opencode install blocks and the winget loop's version-pin machinery are deleted, as is the `~/.opencode/bin` PATH line in the rc files.
- `dotf tools version <name>`: prints the first semver in `<name> --version` output, exit 1 when none. Both setup scripts use it for the hive `hive service` gate instead of `uv tool list`, which stopped seeing a healthy install once hive moved to its own installer (AI-028/#791).
- `dotf doctor`: reads opencode's pin from the catalog, and WARNs when an npm catalog tool resolves from more than one PATH directory, naming them.

## Out of scope

- Moving `pi` to the catalog: its Linux block installs with `--prefix ~/.local --ignore-scripts` (#426) and `installNpm` cannot express that yet. Follow-up on #1294.
- Pruning the leftover channel copies (`~/.opencode/bin`, winget `SST.opencode`): doctor names them; removal is the operator's, per ADR-036 §5.
- The remaining `yarn` version parse in both setups (display only, not a gate).

## Risks / open questions

- On the Linux box the curl copy in `~/.opencode/bin` stays first on PATH until removed — with the rc PATH line gone it drops off PATH at the next shell, `dotf tools install` provisions the npm copy, and doctor reports the leftover. Accepted and documented in the ADR.
- `npm install -g opencode-ai` needs node on PATH in the setup process; both setups install node before `dotf tools install` runs (as `bw` already relies on).
- `hive --version` prints `hive-vault X.Y.Z`; `ProbeVersion` takes the first semver anywhere in the output, so the prefix is fine (tested with the real banner).

## Acceptance criteria

- [ ] AC1 — `packages.json` declares `opencode` as an npm catalog tool and `versions.conf` no longer pins it; `dotf tools install` converges it on both OSes (on the work box: "opencode 1.16.2 already installed; skipping").
- [ ] AC2 — neither setup script carries an opencode install block, a winget `SST.opencode` entry, the winget version-pin machinery, or the `~/.opencode/bin` PATH export.
- [ ] AC3 — `dotf tools version <name>` prints the first semver of `<name> --version` (incl. the `OpenCode locked.` and `hive-vault 3.0.0` banners) and exits 1 when none; both setups gate `hive service` on it.
- [ ] AC4 — `dotf doctor` reads opencode's pin from `packages.json` and WARNs, naming the directories, when an npm catalog tool resolves from more than one PATH entry.
- [ ] AC5 — ADR-036 records the channel policy, with the migration consequence for the Linux box.

## References

- Bitácora: #1294 (AI-034); #145 (AI-021 umbrella), #791 (AI-028), #1265 / #1262 / #1267 (convergence + pin semantics).
- ADR-036 (this change), ADR-020, CLI-029 (`packages.json`), `pattern-setup-script-idempotence`.
