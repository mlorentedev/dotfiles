---
tags: [spec, verification]
created: "2026-06-19"
---

# Verification - HARNESS-027-adr025-hardening

## Evidence

- [x] **install-dotf** → `scripts/install-dotf.ps1` + `.sh` capture `dotf version 2>&1` and
  regex-extract the semver. Root cause confirmed empirically on this machine:
  `dotf version` → STDOUT empty, STDERR `dotf version 0.6.0`. `bats tests/install-dotf-ps1.bats` → **12/12**.
- [x] **vault consumers** → `healthcheck.ps1` (L381) + `vault-health.sh` (L24) + `init-project.ps1`
  (L121) resolve `VAULT_PATH` with the contract default as fallback; no hardcoded machine layout.
- [x] **zero-touch profile** → `profile.ps1` / `.bashrc` / `.zshrc` gate `dotf env generate` on
  `paths.*` missing + dotf on PATH. Verified: a simulated fresh login resolves
  `VAULT_PATH`/`DOTFILES_REPO_DIR`/`HIVE_VAULT_PATH` to the Workspace values from `paths.ps1`
  with NO User-scope env (stopgap retired).
- [x] **resolver prefer-repo** → `env.go ResolveContractPath`. Tests `TestResolveContractPathPrefersRepoCheckout`
  + `TestResolveContractPathRepoMissingFallsThrough`. `go test ./internal/env` green.
- [x] **doctor Windows parity** → `System.GOOS` seam (`system.go`); `checkCoreTools`,
  `checkVersionMatch`, `checkToolHomeEnvVars`, `checkSymlinks`, `checkTmux` SKIP on Windows.
  `TestOSGatingSkipsLinuxOnlyChecksOnWindows` covers 4 functions; existing Linux tests unchanged.
  `go test ./internal/doctor` green.
- [x] **README** → "Cross-machine paths (ADR-025)" section added.

## Test status

- `go build ./...` ok; `gofmt -l` clean; `go vet ./...` exit 0.
- `go test ./internal/env ./internal/doctor` → ok.
- Changed `.ps1` parse OK; PSSA (CI settings, Error+Warning) = 0 on all four.
- `bats tests/install-dotf-ps1.bats` → 12 ok / 0 failed.

## Decisions made during implementation

- **`runtime.GOOS` → `System.GOOS` seam.** First cut used `runtime.GOOS` directly; it broke
  existing doctor tests when run on Windows (they assume Linux behavior). Fix: inject the OS via
  `System.GOOS` (zero value `""` = POSIX), so CI-Linux tests are untouched and Windows-path tests
  set `GOOS="windows"` explicitly.
- **install-dotf reads both streams.** Robust to `dotf version` going to stderr (0.6.0) OR stdout
  (0.7.0 source); the regex extract is StrictMode-safe regardless.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **yes** — "OS-conditional code must inject the OS (mockable
  seam), or it breaks the test suite the moment it runs on the other OS."
- [ ] Lesson — "a release binary may print `version` to stderr; installers that grep stdout
  silently never detect 'already installed' and (under StrictMode) crash."

## Archive checklist

- [ ] PR merged, closes #457; CI green.
- [ ] `proposal.md` `status: archived`; folder → `specs/archive/HARNESS-027-adr025-hardening/`.
- [ ] Bitácora #457 → Done.
