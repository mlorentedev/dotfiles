---
tags: [spec, verification]
created: "2026-06-21"
---

# Verification - CLI-029-dotf-tools-catalog

## Evidence (PR-A)

- [x] **Catalog parses** → `packages.json` + `tools.Load`; `TestLoad` asserts the sops entry, `TestLoad_Errors` covers missing + invalid JSON.
- [x] **Per-OS asset resolution** → `Tool.AssetName`; `TestAssetName` covers linux/darwin (amd64+arm64), the irregular Windows `sops-v3.13.1.amd64.exe` (arch before `.exe`, no OS token), and an unsupported OS → "".
- [x] **`dotf tools list` works** → smoke output on this Windows box:
  ```
  NAME  VERSION  PROFILE  ASSET (windows/amd64)
  sops  3.13.1   full     sops-v3.13.1.amd64.exe
  ```
- [x] **Absent-catalog error** → `dotf tools list` errors clearly when `DOTFILES_DIR/packages.json` is missing.

## Test status

- `go -C cli build ./...` → ok. `go -C cli test ./internal/tools/... ./internal/cmd/...` → **ok**. gofmt clean for the 3 new files.
- Smoke: `DOTFILES_DIR=$(pwd) dotf tools list` → sops row with the correct Windows asset.
- (Repo-wide `go test ./...` still shows the pre-existing #461 `TestEmbeddedTemplatesMatchVault` drift in `internal/spec`/`internal/vault` — unrelated; SKIPs on CI.)

## Decisions made during implementation

- **Per-OS `asset` map, not a single template.** The approved single-template schema could not express sops's irregular Windows asset (`sops-v{version}.exe`, no OS/arch) alongside `sops-v{version}.linux.{goarch}`. The map is the minimal correct shape — a pre-flight failure mode that materialized on first contact.
- **`dotf tools` is a new noun, not `dotf setup`.** Setup is CLI-028 (last); the pilot needs a small standalone consumer.
- **Installer deferred to PR-B.** Download + checksum verification is security-sensitive and deserves its own reviewable PR + fresh focus.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **maybe at PR-B** — "declarative catalog: per-OS asset maps beat single templates because release naming is irregular." Hold until the installer lands.
- [ ] ADR-worthy? Borderline — if the catalog becomes the canonical install mechanism (CLI-028), an ADR may record it then; not now.
- [ ] New cross-project pattern? no.

## Archive checklist (after PR-B merges)

- [ ] `proposal.md` `status: archived`
- [ ] Folder → `specs/archive/CLI-029-dotf-tools-catalog/`
- [ ] Close #506 (PR-B link)
