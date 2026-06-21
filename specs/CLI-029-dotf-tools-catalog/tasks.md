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

- [ ] Verify the exact sops release asset + checksum filenames against the live getsops/sops v3.9.0 release
- [ ] `dotf tools install [name]`: download asset + checksums, verify sha256, place in `~/.local/bin`, chmod; install-if-missing + upgrade-if-below-pin; offline/404 → non-fatal
- [ ] Testable download seam (no network in tests); table-driven tests for resolve/verify/reconcile
- [ ] Wire `dotf tools install` into setup (best-effort, non-fatal) so sops lands on machines
- [ ] `dotf doctor` optionally reports catalog tools present/at-pin (nice-to-have)

## Closing

- [ ] All PR-A acceptance criteria covered (`features.json`)
- [ ] `verification.md` filled
- [ ] Both PRs merged → archive spec, close #506

## Note

This pilot does not touch `versions.conf` or the setup install loops. CLI-028 (#497) migrates the remaining tools into the catalog once the mechanism is proven here.
