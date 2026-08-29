---
id: "OPS-042-npm-tools-to-catalog"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-29"
issue: "mlorentedev/dotfiles#1336"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-042-npm-tools-to-catalog

> **Naming**: file lives at `<repo>/specs/OPS-042-npm-tools-to-catalog/proposal.md`. `OPS-042-npm-tools-to-catalog` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

ADR-036 puts node-distributed tools in `packages.json`, where one installer
(`dotf tools install`, already run by both setups before these blocks) pins,
installs and upgrades them the same way on every OS, and `dotf doctor` reports
them against the pin. Two tools never moved: `obsidian-cli` and `yarn` still
carry a hand-written npm block in each setup script (four blocks), plus a
PowerShell parser that re-reads `versions.conf` for `YARN_VERSION`. The
Windows comment still names a package that 404s. `versions.conf` carries
`OBSIDIAN_VERSION=1.12.4` — the Obsidian *app* version, which nothing reads —
while the npm package is at 0.5.1: a pin that pins nothing.

## What

- Two catalog entries: `obsidian` (`obsidian-cli@0.5.1` — the binary is
  `obsidian`, which is what the installer probes with `--version`) and `yarn`
  (`yarn@1.22.22`). `dotf tools install` converges them; `dotf tools list`
  shows them.
- The four setup blocks and the PowerShell `versions.conf` parser for yarn are
  deleted. `Test-VersionAtLeast` stays: pi still uses it (#1294 follow-up).
- `YARN_VERSION` and `OBSIDIAN_VERSION` leave `versions.conf`; the doctor's
  yarn row reads the pin from `packages.json` through `catalogPin`, the way
  copilot's and opencode's rows do.
- The bats tests that asserted the npm blocks now assert the catalog entries
  and the absence of the blocks.

## Out of scope

- pi's npm block (needs `installNpm --prefix/--ignore-scripts`, #1294 follow-up).
- Changing either tool's version.
- Sync-SessionPath after install on Windows: `dotf tools install` runs early
  in setup, before anything needs `obsidian` or `yarn` in the same session,
  and the PATH refresh already happens after the winget phase.

## Risks / open questions

- The installer probes `<name> --version` and parses the first semver;
  `obsidian --version` prints `obsidian-cli v0.5.1` and `yarn --version`
  prints `1.22.22` (measured on the Windows box). RESOLVED: both parse.
- A box with `obsidian` on PATH from another channel (scoop, winget) is
  reported by `checkShadowedCatalogTools`, as for every npm catalog tool.
  RESOLVED: existing behaviour, no new case.
- `versions.conf` parsing tests use `YARN_VERSION` as a fixture key. RESOLVED:
  they test the parser, not the pin; the key stays in the fixture.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1 — `packages.json` declares `obsidian` (`obsidian-cli`, 0.5.1) and `yarn` (1.22.22) as npm tools; `dotf tools list` shows both.
- [x] AC2 — neither setup script contains an `npm install -g` for `obsidian-cli` or `yarn`, and `setup-windows.ps1` no longer parses `versions.conf` for `YARN_VERSION`.
- [x] AC3 — `versions.conf` no longer carries `OBSIDIAN_VERSION` or `YARN_VERSION`; `dotf doctor`'s yarn row matches against `packages.json`.
- [x] AC4 — on the Windows work box: `dotf tools install` reports both at pin (skip) and `dotf doctor` shows `yarn version matches packages.json (1.22.22)`.

## References

- Bitácora board: #1336. ADR-036 (node-distributed tools in packages.json); #1294 (opencode's move, the precedent); #1359 (copilot's move).
- `cli/internal/tools/install.go` (`installNpm`), `cli/internal/doctor/checks_tools.go` (yarn row), `checks_catalog.go` (`catalogPin`).
