---
tags: [spec, tasks]
created: "2026-06-19"
---

# Tasks - HARNESS-113-adr025-hardening

> One PR off main. Closes #457. Surfaced during the ADR-025 / WIN-006 deploy.

## Setup
- [x] Branch `feat/adr025-path-hardening` off `origin/main` (release 0.7.0).

## Implementation
- [x] `install-dotf.ps1` + `.sh`: read `dotf version` from both streams + regex semver (StrictMode-safe).
- [x] `healthcheck.ps1` + `vault-health.sh`: resolve vault via `VAULT_PATH` + contract default.
- [x] `init-project.ps1`: contract-default fallback; template emits `$ProjectRoot`.
- [x] `profile.ps1` + `.bashrc` + `.zshrc`: auto-`dotf env generate` when `paths.*` missing.
- [x] `cli/internal/env/env.go`: `ResolveContractPath` prefers a valid `DOTFILES_REPO_DIR` checkout (+ tests).
- [x] `cli/internal/doctor/`: `System.GOOS` seam; OS-gate POSIX-only checks (+ Windows tests).
- [x] `README.md`: "Cross-machine paths" section.

## Closing
- [x] `gofmt` + `go vet` clean; `go test ./internal/env ./internal/doctor` green.
- [x] Changed `.ps1` parse clean; PSScriptAnalyzer (CI settings) = 0.
- [x] `bats tests/install-dotf-ps1.bats` green (12/12).
- [ ] PR opened referencing this spec folder; closes #457; CI green.

## Notes
- Template-drift `go test` failures are vault-present-only (CI skips them, ADR-013) — out of scope.

## Machine-readable features
`features.json` is emitted alongside; the harness sets `"state": "passing"` after a green `verification` command.
