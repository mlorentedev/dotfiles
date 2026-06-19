---
tags: [spec, verification]
created: "2026-06-18"
---

# Verification - CLI-016-dotf-env

## Evidence

- [x] **AC1 — cascade** (`env → machine.json → contract[GOOS]`, override>default, skip no-default, darwin fallback, home expand) → `cli/internal/env/env_test.go`: `TestResolveOverrideBeatsDefaultAndSkipsNoDefault`, `TestResolveDarwinFallsBackToLinuxDefault`, `TestResolveWindowsExpandsUserprofile`.
- [x] **AC2 — generate write/idempotent/check** → `TestGenerateWriteIdempotentAndCheck` (write, no-rewrite on identical content, drift after manual edit). CLI smoke: `dotf env generate` → `wrote …\paths.ps1`; `dotf env generate --check` → `ok: … up to date`.
- [x] **AC3 — already-set value preserved** → `TestRenderShKeepsAlreadySetValue`, `TestRenderPs1OnlyAssignsWhenUnset`. End-to-end: `dotf doctor --verbose` shows `[OK] VAULT_PATH=…\Workspace\knowledge` etc. when the seam is set.
- [x] **AC4 — doctor drift check** → `checkPathFiles` (doctor.go `Run` wiring). End-to-end: `[OK] paths.ps1 up to date (…)`; with a stale file the section FAILs.
- [x] **AC5 — vault.ResolveVault via cascade, no hardcoded fallback** → `cli/internal/vault/vault.go` uses `env.ResolvePath("VAULT_PATH")`; `go test ./internal/vault/...` green (no regressions); `path/filepath` import dropped.
- [x] **AC6 — consumers wired** → diff: `.bashrc`/`.zshrc`/`profile.ps1` source the generated file with bootstrap fallback; `claude-session-start.{sh,ps1}` read `$VAULT_PATH`; `setup-{linux,windows}` run `dotf env generate`.
- [x] **AC7 — toolchain green** → below.

## Test status

- `go build ./...` → exit 0. `go vet ./...` → exit 0.
- `go test ./...` (from `cli/`) → all `ok`: `cmd/dotf`, `internal/cmd`, `internal/doctor`, `internal/env` (new), `internal/initrepo`, `internal/spec`, `internal/vault`. No regressions.
- `gofmt -l ./internal/...` → empty (clean, after `gofmt -w` on the new + edited files).

## Manual smoke (this machine, Windows)

- `DOTFILES_DIR=<repo> dotf env generate --stdout` → 9 guarded `if (-not $env:X)` lines; invariant paths (`~/.X`) from the contract, `DOTFILES_REPO_DIR`/`VAULT_PATH`/`HIVE_VAULT_PATH` from `~/.config/dotfiles/machine.json` (the `…\Workspace\…` overrides).
- `dotf doctor --verbose` (seams set, as a sourced shell would have): `[OK] DOTFILES_REPO_DIR=…\Workspace\dotfiles (path exists)`, `[OK] VAULT_PATH=…\Workspace\knowledge`, `[OK] HIVE_VAULT_PATH=…\Workspace\knowledge`, `[OK] paths.ps1 up to date`.
- Removing `path_exists` from the two vault vars confirmed: an unset optional vault no longer FAILs the generic contract sweep (the dedicated vault section still reports presence).

## Decisions made during implementation

- **`env` does not import `doctor`** — the focused contract view (name + default) lives in `env`; doctor imports env for the drift check (one direction, no cycle).
- **Resolve to absolute paths at generate time** (expand `$HOME` / `$env:USERPROFILE`) so the path file has no cross-shell expansion ambiguity; the file is per-machine (deployed to `~/.dotfiles`, gitignored).
- **`VAULT_DIR` retired** — doctor's third name for the vault collapsed into the canonical `VAULT_PATH`.
- **No `path_exists` on the vault vars** — optional; existence is the dedicated `checkVault`'s job, avoiding false FAILs on vault-less machines.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **yes (at archive)** — "an env-var seam is inert until something actually sets it; a hardcoded fallback that happens to match reality masks the broken seam until the assumption shifts."
- [ ] ADR-worthy? **already** — ADR-025 is the decision; this spec implements it.
- [ ] New pattern for the vault `00_meta/patterns/`? **maybe** — "contract-defaults + per-machine-override + render-at-setup" as a reusable cross-machine config pattern.

## Archive checklist

- [ ] PR merged, closes #445.
- [ ] Deploy verified on ≥1 machine (setup run → `dotf doctor` path-file section PASS); stopgap User env vars removed.
- [ ] `proposal.md` `status: archived`; folder → `specs/archive/CLI-016-dotf-env/`.
- [ ] Bitácora #445 → Done.
- [ ] Promotions above executed (if any).
