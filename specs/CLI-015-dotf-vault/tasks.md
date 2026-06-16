---
tags: [spec, tasks, templates]
created: "2026-06-16"
---

# Tasks - CLI-015-dotf-vault

> TDD order. One task = one focused commit. Tick as you go. Two-PR sequence (see `proposal.md` "What"): **PR1** ships `dotf vault work` (closes #388); **PR2** extracts `dotf vault project` + rewires `dotf init` (#395).

## Setup

- [x] Branch created from main: `feat/dotf-vault`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## PR1 — `dotf vault work` (closes #388)

> New `cli/internal/vault` package + `cli/internal/cmd/vault.go`. Reuses the embed+drift pattern of `cli/internal/spec` / `cli/internal/initrepo`.

### Templates (SSOT first)

- [x] Author the 3 work-SDK templates from the removed `init-project.sh --work-sdk` heredocs (`git show e861b41~1:scripts/init-project.sh`): `work-sdk-context.md`, `work-sdk-memory.md`, `work-sdk-family.md`, with `{{component}}` / `{{family}}` / `{{date}}` tokens.
- [x] Vendor them byte-identical into `cli/internal/vault/templates/` (embedded) AND `~/Projects/knowledge/00_meta/templates/` (vault SSOT — committed to vault `master` `63004bbe`, `--ff-only`, isolated-worktree discipline).

### Renderer (TDD)

- [x] Test: `WriteWorkEntry` creates `50_work/45-development/<family>/<component>/{00-context.md,memory/MEMORY.md}` under a temp vault, with tokens substituted.
- [x] Implement `vault.WriteWorkEntry` + `renderTemplate` core (render embedded template → dest, token-replace) + `//go:embed templates`.
- [x] Test: parent `<family>/00-context.md` created only when absent (pre-existing family file untouched, even under `--force`).
- [x] Implement family-context handling.
- [x] Test: skip-if-present leaves an existing `00-context.md` untouched; `--force` regenerates it.
- [x] Implement skip/force.
- [x] Test: empty/unresolvable vault → error naming `$VAULT_PATH` (NOT a silent skip — `dotf vault` is vault-only).
- [x] Implement `ResolveVaultStrict` (vault package's own resolver; `initrepo.ResolveVault` folds in at PR2).
- [x] Bonus: `TestWriteWorkEntryRejectsPathTraversal` — `validateSlug` guards `family`/`component` (untrusted path segments).

### Drift guard

- [x] Add `cli/internal/vault/drift_test.go`: embedded templates `bytes.Equal` the vault SSOT; `t.Skip` when the vault is absent (ADR-013), mirroring `spec`/`initrepo`. Runs real (PASS) on this machine.

### Command wiring

- [x] `newVaultCmd()` — parent runnable via `RunE: cmd.Help` (first-class "Available Command"); `vault work <family> <component>` subcommand with `--force`; cobra `Args: ExactArgs(2)`.
- [x] `root.AddCommand(newVaultCmd())` in `cli/internal/cmd/root.go`.
- [x] Command test (`cmd/vault_test.go`): arg-count validation, `--force` plumbed, vault-absent error surfaced.

### Closing (PR1)

- [x] `go test ./...` green; `gofmt -l` + `go vet` clean.
- [x] Smoke: built binary scaffolds a throwaway `<family>/<component>` under a temp `$VAULT_PATH`; re-run skips; `--force` regenerates.
- [x] `verification.md` filled (evidence: test names, smoke, drift-real proof).
- [ ] PR1 opened referencing this spec folder; closes #388.

## PR2 — `dotf vault project` + rewire `dotf init` (#395) — DEFERRED

> Separate PR (atomic-PR). Behavior-preserving refactor. Not started until PR1 merges.

- [ ] Move `initrepo.WriteVaultEntry` (+ `linkMemory`, `vaultEntryFiles`, the 3 `vault-*` templates) into `cli/internal/vault` as `WriteProjectEntry` / `vault project <repo>`.
- [ ] Rewire `dotf init` orchestrator to call `vault.WriteProjectEntry`; delete the `initrepo` copy (strangler).
- [ ] Parity: `dotf init` produces a byte-identical `10_projects/<repo>/` before/after (golden compare; existing orchestrator/vault tests green).
- [ ] PR2 opened referencing this spec folder; closes #395. Archive CLI-015 after PR2 merges.

## Machine-readable features

`features.json` is emitted alongside this file per [[pattern-feature-list-as-primitive]]; the harness (not the agent) sets `"state": "passing"` after capturing a green `verification` command.
