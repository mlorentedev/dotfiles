---
tags: [spec, verification, templates]
created: "2026-06-24"
---

# Verification - OPS-016-vault-hooks-doctor-check

## Evidence

All criteria proven by table tests in `cli/internal/doctor/checks_vault_hooks_test.go` (commit `1ac5820`), plus a real end-to-end lifecycle run.

- [x] **AC1** (installed → PASS) → `TestCheckVaultHooks_BothInstalledPasses`
- [x] **AC2** (missing, no --fix → FAIL, no install) → `TestCheckVaultHooks_MissingFailsWithoutFix`
- [x] **AC3** (--fix installs, idempotent) → `TestCheckVaultHooks_InstalledOnFix` + `TestCheckVaultHooks_BothInstalledPasses` (no reinstall) + `TestCheckVaultHooks_InstallFailureIsReported`
- [x] **AC4** (--fix, no pre-commit → loud FAIL) → `TestCheckVaultHooks_MissingToolFailsLoudOnFix`
- [x] **AC5** (no vault → SKIP) → `TestCheckVaultHooks_NoVaultSkips`
- [x] **AC6** (VAULT_PATH seam, no hardcode) → every test injects `VAULT_PATH` at a temp dir and the check resolves it (shared idiom with `checkVault`)

## Test status

- Test suite: `go test ./internal/doctor/` → `ok` (6/6 new tests pass; full package green, no regressions)
- Static: `go vet ./internal/doctor/` clean; `go build ./...` exit 0; CI `lint` + `test (ubuntu/windows)` green
- Manual smoke (real binary, real `pre-commit` 4.6.0):
  - Fresh temp vault, no hooks → `dotf doctor` reports `[FAIL] vault secret gate INACTIVE … run dotf doctor --fix`
  - `dotf doctor --fix` → `[FIX ] installed vault pre-commit + pre-push hooks`; both hook files written under `.git/hooks/`
  - `dotf doctor` again → `(1 checks, all ok)` (idempotent PASS)
  - Provisioned machine's real vault (`~/Projects/Workspace/knowledge`) → section reports `all ok`
- No regressions in existing suite: yes

## Decisions made during implementation

- **Spec authored after implementation.** The change was initially (wrongly) judged below the SDD bar and built directly; the `spec-gate` CI correctly flagged the 297-LOC diff with no `specs/` folder. This spec documents the completed work — the gate did its job.
- **CLI check, not a bootstrap script.** First pass added shell scripts (`install-vault-hooks.{sh,ps1}`) + setup wiring; reverted in favor of a `dotf doctor` check — consistent with the CLI-consolidation direction (ADR-021/022). Provisioning is pure behavior, so it does not hit the ADR-020 C7 "bootstrap-deploys-assets" constraint.
- **Report-only bootstrap (decision A).** Bootstrap stays `dotf doctor` (not `--fix`); the gate is provisioned by `--fix`, same as GUARD. The bootstrap-auto-fix question rides with HARNESS-040 (#551).
- **`CommandOutputDir` seam.** `pre-commit install` resolves the git repo from cwd, which the existing `CommandOutput` seam can't set — added a minimal dir-aware sibling, faithful to the existing seam.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — "vault-hook provisioning is pure behavior → a `dotf doctor` check, not a bootstrap script; and a ~120-LOC change is over the SDD bar regardless of how 'obvious' it looks — write the spec first."
- [ ] ADR-worthy decision? **no** — applies existing decisions (checkGuardHooks model, CLI consolidation ADR-021).
- [ ] New pattern for `00_meta/patterns/`? **no** — vault-specific, does not recur across >1 project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/OPS-016-vault-hooks-doctor-check/` -> `specs/archive/OPS-016-vault-hooks-doctor-check/`
- [ ] Backlog entry ticked with PR link (#553)
- [ ] Promotion (lesson above) executed into `docs/lessons.md`
