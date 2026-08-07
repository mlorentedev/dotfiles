---
tags: [spec, tasks]
created: "2026-06-21"
---

# Tasks - CLI-029-dotf-tools-catalog

> Pilot the declarative package catalog with sops. PR-A = catalog + reader; PR-B
> = installer (download + checksum).

## Setup

- [x] Branch from main: `feat/dotf-tools-catalog`
- [x] `proposal.md` complete; schema decided (packages.json, per-OS asset map)

## PR-A — catalog + reader (this PR)

- [x] Write `catalog_test.go` (Load parse + error cases; AssetName per-OS incl. sops's irregular Windows name + unsupported-OS empty)
- [x] `packages.json` at repo root with the sops entry (per-OS asset map)
- [x] `cli/internal/tools/catalog.go` — `Catalog`/`Tool`/`Source`, `Load()`, `Tool.AssetName()`
- [x] `cli/internal/cmd/tools.go` — `dotf tools` + `dotf tools list` (resolves DOTFILES_DIR/packages.json; clear error if absent)
- [x] Register `newToolsCmd` in `root.go`
- [x] `go build ./...`, `go test ./internal/tools/... ./internal/cmd/...`, gofmt — all green; smoke `dotf tools list` shows sops
- [ ] PR-A opened (refs #506, does NOT close it)

## PR-B — installer (separate PR, closes #506)

- [x] Verify the exact sops release asset + checksum filenames against the live getsops/sops v3.13.1 release (assets are raw binaries; checksum manifest `sops-v{version}.checksums.txt`, GNU-coreutils format). Added `Source.Checksums` to the catalog schema.
- [x] `dotf tools install [name]`: download asset + checksums, verify sha256, place in `~/.local/bin`, chmod; install-if-missing + upgrade-if-below-pin (`decideAction`); offline/404 → command errors (setup wraps best-effort)
- [x] Testable download seam (`Fetcher`, no network in tests); table-driven tests for resolve/verify/reconcile + live end-to-end smoke
- [x] Wire `dotf tools install` into setup (best-effort, non-fatal) so sops lands on machines; deploy `packages.json` to `$DOTFILES_DIR` (both OSes)
- [ ] ~~`dotf doctor` reports catalog tools present/at-pin~~ — dropped: touches `cli/internal/doctor` (parallel doctor lane's territory); not blocking #506

## Closing

- [ ] All PR-A acceptance criteria covered (`features.json`)
- [ ] `verification.md` filled
- [ ] Both PRs merged → archive spec, close #506

## Note

This pilot does not touch `versions.conf` or the setup install loops. CLI-028 (#497) migrates the remaining tools into the catalog once the mechanism is proven here.
