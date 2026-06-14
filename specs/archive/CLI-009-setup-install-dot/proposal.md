---
id: "CLI-009-setup-install-dot"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-13"
issue: "dotfiles#364"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-009-setup-install-dot

> **Naming**: file lives at `<repo>/specs/CLI-009-setup-install-dot/proposal.md`. `CLI-009-setup-install-dot` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #364: CLI-009: install dot via setup (fetch release binary) — Linux -->

The ADR-020 language boundary assigns the shell layer "thin bootstrap — detect OS/arch, **fetch binary**, PATH", but that step was never built: `setup-linux.sh` never installs `dot`, so on a fresh machine the Go CLI is unreachable and the shell twins (`init-spec`, `archive-spec`) cannot be retired (CLI-005 #339) without breaking the documented PATH fallback. With the first release (`v0.1.0`) now published by CI, setup can fetch a checksummed binary — closing the bootstrap gap and unblocking the whole go-only convergence.

## What

`setup-linux.sh` installs `dot` from its GitHub release, reproducibly and idempotently:

- `versions.conf` gains `DOT_VERSION` — the single pinned-version source, parsed the same way as `OPENCODE_VERSION` / `GO_VERSION`.
- A testable `install_dot` helper: detect host OS (`uname -s`) and arch (`uname -m` → `amd64`/`arm64`), download `dot_${DOT_VERSION}_${os}_${arch}.tar.gz` from the release, **verify its sha256 against the release `checksums.txt`**, extract `dot` into `~/.local/bin`, `chmod +x`. The release base URL is a parameter so bats can drive it against a local fixture (no network in tests).
- Idempotent + self-healing: a no-op when `dot version` already matches `DOT_VERSION`; re-installs on drift (the `opencode` convergence pattern already in this script).
- `healthcheck.sh` gains a `dot` check (presence + version match), wired into the existing tool/version/path sections.
- The `release` job in `cli.yml` gains `needs: [test, lint]` so a release is only ever published from green code — closing a latent gap now that setup *depends* on releases being trustworthy.

## Out of scope

- `setup-windows.ps1` parity (fetch the `.zip`, verify, PATH) — Windows-empirical, batched into a dedicated Windows session, not validated from Linux.
- Retiring the shell twins + their bats/Pester (`init-spec`/`archive-spec`) — **CLI-005 #339**, which this unblocks.
- Per-agent adapters calling `dot` — **CLI-006 #340**.
- Refactoring the other inline `curl` installs (gh/eza/jq) onto a shared download+verify helper — separate cleanup only if it recurs (`fix-small-debt` threshold).
- `release-please`-style auto-versioning — tag → CI-release is sufficient for a single artifact; revisit only if release cadence grows.

## Risks / open questions

- **Checksum is mandatory, not best-effort.** A download whose sha256 does not match `checksums.txt` must abort the install (leave no partial/poisoned binary on PATH), not warn-and-continue. This is the security gate; tested explicitly.
- **Arch coverage.** The release ships `amd64` + `arm64` for linux/darwin. An unmapped `uname -m` (e.g. `i686`, `armv7l`) must fail with a clear message naming the unsupported arch, not silently download a 404 HTML page and "install" it. Tested.
- **`~/.local/bin` precedence.** A `dot` already on PATH from elsewhere (e.g. a manual build) could shadow the installed one; the version check + convergence reports/handles drift rather than silently coexisting (same failure mode the `opencode` block documents).
- **Offline / release missing.** `curl` failure (no network, GitHub down, tag yanked) degrades gracefully: log a warning, leave setup non-fatal (consistent with other optional-tool installs), surface the gap in `healthcheck.sh`.
- **`sha256sum` availability.** Linux ships `sha256sum` (coreutils); the helper uses it directly. (macOS `shasum -a 256` differs — handled if/when `setup-macos` exists; out of scope here, Linux-only.)

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `DOT_VERSION` is in `versions.conf`; no hard-coded `dot` version string anywhere else.
- [ ] `install_dot` maps the host OS/arch to the correct artifact name and installs `dot` to `~/.local/bin` (the binary runs and reports `DOT_VERSION`).
- [ ] An unmapped arch → non-zero/clear failure, no install.
- [ ] A checksum mismatch → install aborts, no binary left on PATH.
- [ ] Idempotent: re-running with the pinned version present is a no-op; a drifted version converges to `DOT_VERSION`.
- [ ] `healthcheck.sh` reports `dot` and flags version drift.
- [ ] `cli.yml` `release` job is gated on `test` + `lint`.
- [ ] `shellcheck` clean on changed scripts; full bats suite green (incl. new `install_dot` tests for OS/arch mapping + checksum rejection).

## References

- GitHub issue: `dotfiles#364` (work-gate per ADR-018)
- Epic: ADR-020 (`docs/adr/adr-020-tooling-cli-go-convergence.md`) — the "fetch binary" bootstrap clause
- Release: `v0.1.0` (first CI-published `dot` release; `cli/.goreleaser.yaml`)
- Pattern precedents in-repo: the `opencode` version-pin + drift-convergence block in `setup-linux.sh`; the `gh` download-extract block
- Unblocks: CLI-005 #339 (retire shells); related CLI-006 #340 (adapters)

<!-- archived 2026-06-14 — PR: https://github.com/mlorentedev/dotfiles/pull/365 -->
