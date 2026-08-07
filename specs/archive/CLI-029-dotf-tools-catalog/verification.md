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

## Evidence (PR-B — installer)

- [x] **Live release verified** → `gh release view v3.13.1 --repo getsops/sops` confirms the PR-A asset strings (`sops-v3.13.1.linux.{arch}`, `.darwin.{arch}`, `.{arch}.exe`) and the checksum manifest `sops-v3.13.1.checksums.txt` (GNU-coreutils `<sha256>  <file>` format). Assets are **raw binaries**, not archives → no extraction step (unlike install-dotf's tar/zip).
- [x] **Schema extended** → `Source.Checksums` template (single file, all OSes) added to `packages.json` + `catalog.go`; `Tool.ChecksumsName` resolves it. `TestChecksumsName` covers expansion + the no-template empty case.
- [x] **Download seam** → `tools.Fetcher func(url, dest string) error`; default `HTTPFetch` (real GET, non-200 → error). All installer tests inject a fixture-serving fetcher — **no network**.
- [x] **Reconcile policy** → `decideAction` (pin = minimum): absent→install, below→upgrade, at/above→skip (never downgrade — REFACTOR-011/013). `TestDecideAction` table incl. exact-match, newer, unparseable.
- [x] **Verify gate** → `verifyChecksum` matches the asset's sha256 against its manifest entry; `TestInstall_ChecksumMismatch` (wrong hash → error, nothing placed) and `TestInstall_MissingChecksumEntry` (asset absent from manifest → error).
- [x] **Place** → raw binary copied to `~/.local/bin`, renamed to the command name (`sops`/`sops.exe`), `0755`. `TestInstall_Fresh` (content + exec bit on POSIX hosts), `TestInstall_WindowsBinaryName` (`.exe`).
- [x] **Failure paths** → `TestInstall_DownloadFailure` (offline/404 → error), `TestInstall_UnsupportedOS` (no asset for OS → error).
- [x] **Command** → `dotf tools install [name]`; `selectTools` (all vs named, unknown → error), `installAll` (best-effort + aggregate). `TestRunToolsInstall_{All,UnknownTool,AggregatesFailure}` + `TestToolsInstall_MissingCatalog`.
- [x] **Live end-to-end smoke** (this Windows box, `DOTFILES_DIR=$(repo)`):
  ```
  $ dotf tools list      → sops  3.13.1  full  sops-v3.13.1.amd64.exe
  $ dotf tools install   → sops 3.13.1 installed to C:\Users\mlorente\.local\bin\sops.exe
  $ sops --version       → sops 3.13.1 (latest)
  $ dotf tools install   → sops 3.13.1 already installed; skipping   (reconcile/skip path)
  ```
- [x] **Setup wiring (best-effort, non-fatal)** → `setup-linux.sh` (`dotf tools install || log_warning`, guarded on dotf on PATH) + `setup-windows.ps1` (`Get-Command dotf` guard + `$LASTEXITCODE` warn), each preceded by deploying `packages.json` to `$DOTFILES_DIR` so the catalog resolves on a deployed machine. shellcheck clean; PowerShell parses.

## Decisions / debt (PR-B)

- **Pin = minimum, not exact.** Reproduces REFACTOR-011/013 (`setup-windows.ps1:369`): an exact-match reconcile would downgrade a newer install.
- **`packages.json` deployed to `$DOTFILES_DIR`.** PR-A's `tools` resolver reads only `$DOTFILES_DIR/packages.json`, unlike `env-contract.json`'s richer repo-first cascade (`DOTFILES_REPO_DIR` → deploy → walk-up). Deploying the file (beside `versions.conf`) is the minimal fix; unifying `tools` onto the env cascade (`env.ResolveRepoFile`) is a follow-up, kept out of PR-B to stay atomic.
- **Version-compare duplicated.** `compareVersions`/`atLeast`/`component` mirror `doctor`'s unexported copies; consolidating into a shared package is deferred to avoid touching `cli/internal/doctor` (owned by a parallel session this cycle).
- **`dotf doctor` catalog reporting dropped** (was a PR-B "nice-to-have"): it would touch `cli/internal/doctor`; left to the doctor lane.

## Test status

- `go -C cli build ./...` → ok. `go -C cli test ./internal/tools/... ./internal/cmd/...` → **ok** (PR-A + PR-B: parser, asset/checksum resolution, decideAction, fetch/verify/place, command select + best-effort loop). `go vet` clean; gofmt clean for all new/edited files.
- Live smoke (above) exercises the real download → sha256 verify → place → reconcile/skip chain against the published getsops/sops v3.13.1 release.
- (Repo-wide `go test ./...` still shows the pre-existing #461 `TestEmbeddedTemplatesMatchVault` drift in `internal/{initrepo,spec,vault}` — unrelated; SKIPs on CI.)

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
