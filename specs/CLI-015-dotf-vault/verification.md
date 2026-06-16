---
tags: [spec, verification, templates]
created: "2026-06-16"
---

# Verification - CLI-015-dotf-vault

> This spec ships as a 2-PR sequence. **PR1** (this PR, #388) delivers `dotf vault work`; AC2/AC6 below belong to **PR2** (#395, deferred) and are marked accordingly.

## Evidence

- [x] **AC1** `dotf vault work <family> <component>` scaffolds `50_work/45-development/<family>/<component>/{00-context.md,memory/MEMORY.md}` + parent `<family>/00-context.md` create-if-absent -> `vault.WriteWorkEntry`; tests `vault/work_test.go:TestWriteWorkEntryCreatesFilesWithTokensSubstituted` + `TestWriteWorkEntryFamilyContextNotClobberedBySecondComponent`; smoke below. Ported byte-faithfully (modulo `{{date}}`) from `init-project.sh --work-sdk` (`git show e861b41~1:scripts/init-project.sh`).
- [ ] **AC2** `dotf vault project <repo>` parity with `dotf init` — **PR2 / #395 (deferred)**.
- [x] **AC3** skip-if-present + `--force`; vault-absent = error naming `$VAULT_PATH` -> `TestWriteWorkEntrySkipsExistingThenForceRegenerates`, `TestResolveVaultStrictErrorsWhenAbsent`, `cmd/vault_test.go:TestVaultWorkErrorsWhenVaultAbsent` + `TestVaultWorkForceFlagPlumbed`.
- [x] **AC4** `00_meta/templates/work-sdk-*.md` embedded + drift-tested -> `vault/drift_test.go:TestEmbeddedTemplatesMatchVault` (runs real, PASS, on this machine; skips where vault absent — ADR-013). Vault SSOT committed: `mlorentedev/knowledge@63004bbe`.
- [x] **AC5** `dotf vault` parent is a first-class "Available Command" (runnable via `cmd.Help`) -> `cmd/vault_test.go:TestVaultListedInRoot` + `TestVaultParentPrintsHelp`; smoke: `dotf --help` lists `vault`.
- [ ] **AC6** `initrepo.WriteVaultEntry` deleted (guard-grep clean) — **PR2 / #395 (deferred)**.

## Test status

- `go test ./...` (from `cli/`): all `ok` — `internal/vault` (new), `internal/cmd`, `internal/spec`, `internal/initrepo`, `internal/doctor`, `cmd/dotf`. No regressions.
- `gofmt -l internal/vault internal/cmd`: empty (clean). `go vet ./...`: clean.
- **Drift guard runs real here** (vault present): `go test ./internal/vault -run TestEmbeddedTemplatesMatchVault -v` -> `PASS` (not skip), proving the embedded templates are byte-identical to the vault SSOT.
- **Path-traversal guard**: `TestWriteWorkEntryRejectsPathTraversal` — `../../etc`, `..`, `a/b`, `.hidden`, empty all rejected; nothing written under `45-development/`.
- **Manual smoke** (built binary, throwaway `$VAULT_PATH`): `dotf vault work acme-sensors edge-fw` -> creates the 3 files; tokens substituted (`acme-sensors-edge-fw`, `2026-06-16`), literal `${PROJECTS_PATH}`/`${ONEDRIVE_PATH}` preserved, zero unrendered `{{...}}`; re-run skips both component files + family `[exists]`; `--force` regenerates.

## Decisions made during implementation

- **`dotf vault` is the unified vault-entry noun, entry-type as the dimension** (broadened from #388's narrow "restore work-SDK" scope, by design decision this session): `vault work` now, `vault project` extracted from `dotf init` in PR2 (#395). Recorded in `proposal.md`.
- **2-PR decomposition** to honor the atomic-PR cap: PR1 = `vault work` (this); PR2 = `vault project` + `dotf init` rewire + delete the `initrepo` copy. PR1 ships a self-contained, shippable capability and closes #388.
- **`vault.ResolveVault` is the new canonical resolver** (with `ResolveVaultStrict` for the vault-only error). `initrepo.ResolveVault` folds in during PR2 — a brief, ticketed coexistence (one pure 15-line func), not silent duplication.
- **Family context is create-if-absent, never overwritten — even under `--force`** — because it accumulates the family's hand-maintained repo table across components (faithful to the old script; guarded by a test).
- **Path-traversal validation on `family`/`component`** (untrusted args become directory names): start-alphanumeric + no `..`, rejecting `../../etc` and friends (security HALT, AGENTS.md).

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? maybe at archive — "broaden a port's scope when it exposes a latent duplication (init's inlined vault entry) rather than restoring the narrow capability in isolation". Decide when PR2 lands.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — ADR-021 step 3 already names `dotf vault`; this executes it.
- [ ] New pattern candidate for `00_meta/patterns/`? no — repo-local (embed/drift + cobra noun-verb, already established).

## Archive checklist

> Archive only after **both** PR1 (#388) and PR2 (#395) merge — the spec spans the sequence.

- [ ] `proposal.md` frontmatter set to `status: archived` (after PR2)
- [ ] Folder moved: `specs/CLI-015-dotf-vault/` -> `specs/archive/CLI-015-dotf-vault/` (after PR2)
- [ ] Bitácora #388 + #395 closed -> `Done` (task state lives in the bitácora, ADR-018)
- [ ] Promotions above executed (if any)
