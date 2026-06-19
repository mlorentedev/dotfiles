---
id: "WIN-006-windows-dotf-install"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-18"
issue: "mlorentedev/dotfiles#451"
tags: [spec, proposal]
template_version: "1.0"
---

# WIN-006-windows-dotf-install

> The Windows half of the `dotf` bootstrap — closes the gap that forced a manual `go build` during the ADR-025 deploy.

## Why

<!-- from issue #451: setup-windows.ps1 should install dotf from the published release binary (parity with install-dotf.sh) -->

`setup-windows.ps1` does **not** install the `dotf` CLI — it only runs `dotf env generate` *if dotf is already on PATH*. On Linux/macOS, `setup-linux.sh` sources `scripts/install-dotf.sh`, which downloads the goreleaser release binary, verifies its sha256, and installs it to `~/.local/bin` — no compiler. So a Windows user was forced to build dotf from source (Go ≥1.26), which is a kludge and impossible without a working toolchain or admin rights. goreleaser **already publishes Windows binaries** (`dotf_<v>_windows_amd64.zip` / `_arm64.zip`), so nothing is compiled — the gap is purely the missing download step, which **blocks the ADR-025 cross-machine path mechanism from activating on Windows**.

## What

- **`scripts/install-dotf.ps1`** — the PowerShell twin of `install-dotf.sh`:
  - `Get-DotfArch` (PROCESSOR_ARCHITECTURE → amd64/arm64), `Get-DotfVersion` (arg → `$env:DOTF_VERSION` → versions.conf).
  - `Install-Dotf`: fetch `dotf_${version}_windows_${arch}.zip` + `checksums.txt`, verify sha256 (`Get-FileHash`), `Expand-Archive`, place `dotf.exe` in `~/.local/bin` (**user-space, no admin**). Idempotent (skip when the pinned version is on PATH; converge on drift). Never throws — returns `$true`/`$false`.
  - Standalone run-guard (`$MyInvocation.InvocationName -ne '.'`): installs when **executed** (incl. `irm … | iex`), only defines functions when **dot-sourced**.
- **`setup-windows.ps1`** dot-sources it + calls `Install-Dotf` (non-fatal, mirroring `install_dotf || log_warning`) **before** the `dotf env path` / `dotf env generate` steps, so the whole ADR-025 chain activates automatically.

## Out of scope

- **The full `curl | bash` zero-state bootstrap** (clone + setup) — that is `IDEAS-005` (#139), the umbrella; WIN-006 is a building block it consumes.
- **Bumping `DOTF_VERSION`** — release-please owns it (now `0.6.0` on main, which already carries `dotf env`).
- **file://-fixture behavioral bats** — PowerShell's `Invoke-WebRequest` has no `file://` support, so the `.sh` test's fixture path can't be mirrored; covered instead by the real-release smoke + the `.sh` mirror.

## Acceptance criteria

- [x] `install-dotf.ps1` fetches + verifies (sha256) + installs `dotf.exe` to `~/.local/bin`; idempotent; standalone **and** dot-sourceable (run-guard skips on `.`).
- [x] `setup-windows.ps1` installs dotf automatically (non-fatal) before the env steps; parse + PSScriptAnalyzer (CI settings) clean.
- [x] Real-release smoke: running the script installs `dotf 0.6.0`; `dotf version` + `dotf env path VAULT_PATH` resolve. Structural bats green.

## References

- Parity source: `scripts/install-dotf.sh` + `tests/install-dotf.bats`
- Consumers: ADR-020 (CLI bootstrap: shell fetches binary, Go owns logic) · ADR-025 (`dotf env` — the cross-machine path mechanism this unblocks on Windows)
- Work-gate: `mlorentedev/dotfiles#451` · umbrella: `IDEAS-005` (#139, curl-bash bootstrap)
