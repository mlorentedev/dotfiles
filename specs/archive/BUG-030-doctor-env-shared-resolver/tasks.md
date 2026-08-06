---
id: "BUG-030-doctor-env-shared-resolver"
type: spec
status: implementing
created: "2026-07-10"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — BUG-030-doctor-env-shared-resolver

- [x] T1. `env.ResolveRepoFirst(name, repoDir, dotfilesDir, startDir)` +
  refactor `ResolveContractPath` to call it. Unit tests (precedence + fallbacks).
- [x] T2. `doctor.loadConfig`: resolve contract AND versions.conf via
  `ResolveRepoFirst`, repo-first; resolve `repoDir` from `DOTFILES_REPO_DIR` (real
  dir) else `.git` walk-up. Store `VersionsPath`. Remove the dead deployed-first
  helpers (`firstExisting`/`joinIf`).
- [x] T3. Provenance: `rep.Info("contract: …")` in `loadContractSection`,
  `rep.Info("versions.conf: …")` in `checkVersionMatch` (visible in non-verbose).
- [x] T4. Anti-drift guard: doctor test picks the repo copy over the deployed one
  and cross-checks `env.ResolveContractPath` agrees.
- [x] T5. Local verification: `go build`/`vet`/`test ./...`, golangci-lint clean,
  `dotf doctor` provenance observed resolving the repo copy.
- [ ] T6. CI green (`lint`, `lint-powershell`, `test`, `integration`, `spec-gate`).
