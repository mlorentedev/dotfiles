---
tags: [spec, verification]
created: "2026-08-28"
---

# Verification - CLI-058-env-persist

## Evidence

Run on the Windows work box, 2026-08-28, worktree `dotfiles-wt-setup-cluster`,
branch `feat/env-persist`, `dotf` built from the branch.

- [x] **AC1/AC2** → `TestPersist_TouchesOnlyWhatDiffers`, `TestDrift_NamesMissingAndDifferent`,
  `TestPersist_StoreErrorsNameTheVariable`. Mutation: `Persist` treating every value as different
  → `--- FAIL: TestPersist_TouchesOnlyWhatDiffers`; restored.
- [x] **AC3** → `persist_other.go` returns `ErrUserEnvUnsupported`; `env_persist.go` prints
  `nothing to persist on <os>` and returns nil (exercised by the `GOOS=linux go vet` build and by
  the CI `test (ubuntu-latest)` job compiling both files).
- [x] **AC4** → `TestCheckPersistedEnv_ByStatus` (all persisted PASS; one missing WARN naming it;
  one different WARN naming it; store error WARN "unreadable"; nil seam → no section). Mutation:
  `Drift` never reporting → the two WARN rows fail; restored. Live: `dotf doctor` shows
  `[Persisted environment (user scope)] (1 checks, all ok)` after the persist below.
- [x] **AC5** → `tests/setup-windows.bats` "persists the contract variables at User scope after
  generating paths.ps1"; parse 0 errors; ASCII delta 0.
- [x] **AC6** → box, in order:
  1. `dotf env persist --check` → `drift: ...` × 10, exit 1 (every contract variable missing at
     User scope — the measured starting state).
  2. `dotf env persist` → `persisted ...` × 10, `user scope: 10 changed, 1 unchanged`.
  3. `dotf env persist --check` → `ok: 11 variable(s) persisted at user scope`.
  4. `dotf env persist` → `user scope: 0 changed, 11 unchanged` (idempotent).
  5. `[Environment]::GetEnvironmentVariable(n, 'User')` → DOTFILES_REPO_DIR, DOTFILES_DIR,
     VAULT_PATH, SCRIPTS_DIR, COPILOT_HOME all set to the resolved paths.
  6. `Start-Process pwsh -NoProfile -UseNewEnvironment` (a fresh environment built from the
     registry, no inheritance from the launching shell, no profile) prints the same five values —
     the profile-less consumer Copilot's tool calls are.

## Test status

```text
go build ./... && go vet ./... && GOOS=windows go vet ./... && GOOS=linux go vet ./... && go test ./internal/env/ ./internal/doctor/ ./internal/cmd/   -> ok
golangci-lint run ./...   -> 0 issues
bats tests/setup-windows.bats   -> all ok
```

- No regressions in the existing suite: yes.
- `golang.org/x/sys` becomes a direct dependency (`go mod tidy`): the registry API is the
  first production use; the earlier `bwserve_windows.go` constant stays stdlib.

## Decisions made during implementation

- **One resolver, two sinks.** `env.ResolveVars` is what `generate` renders and what `persist`
  writes, so the rc files and the registry can never disagree by construction.
- **Store as an interface, registry behind a build tag.** `UserEnvStore` keeps the logic and the
  tests OS-agnostic; `persist_windows.go` is the only file that knows about `HKCU\Environment`.
- **Broadcast, best effort.** `WM_SETTINGCHANGE` is what `setx` sends; a failure leaves the
  registry correct and only delays visibility to the next logon.
- **Doctor WARN, not FAIL.** Shells keep working without the persisted scope; the remedy is one
  idempotent command that setup now runs.
- **No-op, not error, off Windows.** Setup scripts call the same verb on both OSes without a
  per-OS branch; Linux prints one line and exits 0.

## Promotion candidates

- [ ] Lesson: no (the ticket and this spec carry the measurement).
- [ ] ADR-worthy decision: no — it implements ADR-025's cascade for a scope it did not name.

## Archive checklist

- [ ] `dotf spec review CLI-058-env-persist` PASS
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-058-env-persist/`
- [ ] Bitácora #1324 closed with the PR link
