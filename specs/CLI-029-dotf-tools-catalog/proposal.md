---
id: "CLI-029-dotf-tools-catalog"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-21"
issue: "mlorentedev/dotfiles#506"
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-029-dotf-tools-catalog

> Pilot the declarative cross-OS package catalog (CLI-028 idea #1, from the
> WindowsDeveloperConfig analysis) on ONE tool — **sops** — to prove the
> "install list as data, dotf as consumer" mechanism before CLI-028 migrates the
> rest.

## Why

The tool/install list today is per-OS imperative code: a PowerShell array + winget loop in `setup-windows.ps1` and an apt/curl block in `setup-linux.sh`, pinned via the flat `versions.conf`. That is the cross-OS duplication ADR-021 exists to kill. **sops** is the ideal pilot: it is a documented-but-uninstalled need (`docs/lessons.md:242` — app-secrets → SOPS) and adding it the conventional way would mean editing both setups + `versions.conf`. Instead, declare it once in a catalog and let `dotf` install it.

## What

Two PRs under #506:

**PR-A (this branch, `feat/dotf-tools-catalog`) — the catalog + reader:**
- `packages.json` at repo root (sibling of `env-contract.json`): `{tools: [{name, version, profile, source}]}`.
- `cli/internal/tools` — `Load()` + `Tool.AssetName(goos, goarch)`.
- `dotf tools list` — prints each tool with the asset resolved for this OS/arch.
- sops entry only; no installer yet.

**PR-B — the installer:**
- `dotf tools install [name]` — resolve asset, download the pinned GitHub release binary, **verify against the release checksums**, place in `~/.local/bin`; install-if-missing + upgrade-if-below-pin (absorbing the `setup-windows.ps1:369-390` reconcile). Security-sensitive (download + checksum) → its own reviewable PR.
- Wire it into setup (best-effort, non-fatal) so sops actually lands on machines.

## Schema (packages.json)

```json
{ "tools": [ {
  "name": "sops", "version": "3.13.1", "profile": "full",
  "source": { "type": "github-release", "repo": "getsops/sops",
    "asset": { "linux": "sops-v{version}.linux.{goarch}",
               "darwin": "sops-v{version}.darwin.{goarch}",
               "windows": "sops-v{version}.{goarch}.exe" } } } ] }
```

**Design refinement (pre-flight materialized):** the approved single-template `asset` was changed to a **per-OS map**. Release naming is irregular — sops is `sops-v{version}.linux.{goarch}` but just `sops-v{version}.exe` on Windows (no OS/arch in the name). A single template cannot express both; the map can. Same concept, correct detail.

## Out of scope

- The installer / download / checksum (PR-B).
- Migrating any other tool off `versions.conf` + the setup loops (that is CLI-028 / #497). `versions.conf` and both setup install blocks stay untouched; the catalog is dotf-only and holds only sops.
- Adopting sops into the secrets *workflow* (age-direct vs sops+age) — a CLI-024 (#493) design decision, not this.
- Profile enforcement — no `full`/`minimal` mechanism exists yet (DX-007 R3 confirmed); `profile` is declared-but-advisory until DX-007 adds it.

## Risks / open questions

- **Exact sops asset strings must be verified against the live release in PR-B** before the downloader uses them (PR-A only renders them for display).
- Checksum source: getsops/sops ships a `sops-v{version}.checksums.txt` — PR-B verifies against it (the install-dotf / BUG-025 pattern).
- `~/.local/bin` is already on PATH (install-dotf) — reuse it.

## Acceptance criteria (PR-A)

- [ ] `packages.json` parses into `Catalog` with the sops entry; `Load()` errors clearly on missing/invalid file.
- [ ] `Tool.AssetName` resolves per-OS (`linux`/`darwin`/`windows`), handles sops's irregular Windows name, and returns "" for an unsupported OS.
- [ ] `dotf tools list` prints the catalog with the current-OS asset; clear error when `packages.json` is absent.
- [ ] `go test ./internal/tools/... ./internal/cmd/...` green; build + gofmt clean.

## References

- Issue: mlorentedev/dotfiles#506; refs #497 (CLI-028), #493 (CLI-024)
- WindowsDeveloperConfig analysis (declarative-data idea #1); `docs/lessons.md:242` (sops classification)
- ADR-020/021; the `age` install pattern (`scripts/install-dotf.*`, BUG-025)
