---
tags: [spec, verification, templates]
created: "2026-06-25"
---

# Verification - CLI-025-secrets-render

Delivered across 2 PRs (see `tasks.md`). PR-A adds `dotf secrets render` + wires
the setups; PR-B deletes the twins + `env-mapping.conf` and closes #587.

## Evidence

- [x] **AC1** (substitute mapped, leave unmapped intact) -> `secrets/render_test.go`
  `TestRender_SubstitutesMapped_LeavesUnmappedIntact`, `TestRender_NoPlaceholders_NoOp`,
  `TestRender_FileSecretVar_LeftIntactNotMaterialized`; cmd `TestSecretsRender_SubstitutesInPlace`;
  real smoke: rendered a fixture against the live registry+age store -> `{env:NAN_API_KEY}`
  substituted (0 remaining), `{env:HOME}` + an unmapped var left intact (1 each).
- [x] **AC2** (atomic write, 0600, no trailing-newline drift) -> `secrets/render_test.go`
  `TestRender_Mode0600_NoTrailingNewlineDrift`; smoke confirmed last byte is `}` (no newline added).
- [x] **AC3** (setups materialize via `dotf secrets render`; twins gone) -> PR-A wired
  both setups to `dotf secrets render`; PR-B deleted `substitute_env_placeholders`
  (`utils.sh`) + `Substitute-EnvPlaceholders` (`utils.ps1`) + `tests/sdd-009-*` and
  simplified the setup fallback to render-or-literal. *Verify:* grep-clean of the twins
  (only descriptive comments remain); `bash -n` + PSScriptAnalyzer parse pass.
- [x] **AC4** (`env-mapping.conf` deleted; drift-guard test removed) -> `git rm
  sensitive/env-mapping.conf` + `ParseMapping` + `registry_seed_test.go` + `TestParseMapping`.
  Its 3 other live consumers were migrated to the registry: the doctor
  (`checks_pat`/`checks_deploy`) and `github-secrets-manager.sh`, via the new
  `dotf secrets ls --pairs` (`TestSecretsLs_Pairs_EnvOnly`).
- [x] **AC5** (cross-OS green; #587 closes) -> `go test ./...` 11 ok; `Closes #587` on PR-B.

Parity bonus added beyond the twins: a var exposed by two distinct secrets is a
non-deterministic registry and is rejected fail-fast (`TestRender_DuplicateVar_FailsFast`);
a mapped-but-undecryptable secret is left intact (not fatal), matching twin behaviour
(`TestRender_UnresolvedDecryptError_LeftIntact`).

## Test status

- `go test ./internal/secrets/ ./internal/cmd/` -> ok (render unit + cmd-wiring tests green).
- Pre-existing, unrelated: `TestEmbeddedTemplatesMatchVault` fails in `internal/{spec,vault,initrepo}`
  on this machine (local vault ahead of vendored templates) -> confirmed identical on untouched
  `main`, i.e. not introduced here; passes in CI.
- `go vet ./...` clean; `gofmt -l` clean on changed Go files; `bash -n setup-linux.sh` OK.
- Manual smoke: `dotf secrets render <fixture>` end-to-end over the real age store (above).

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Reuse `EnvFor` per-var, not in batch.** `Render` decrypts by calling the existing
  `Loader.EnvFor` (one decrypt+strip path shared with run/show) one entry at a time, so a
  decrypt failure degrades to "unresolved, leave placeholder intact" instead of `EnvFor`'s
  fail-fast (correct for `run`, wrong for setup which must complete). It also filters to env
  entries so file secrets are never materialized to disk as a render side effect.
- **Two-PR split kept.** PR-A leaves the twins in place as a bootstrap fallback so the new
  path can be validated by a real setup/integration run before PR-B removes the safety net.
- **PR-B scope grew on a grep sweep.** `env-mapping.conf` was assumed to have only the render
  twins as consumers; a pre-deletion grep found 4 more (doctor `checks_pat`/`checks_deploy`,
  `github-secrets-manager.sh`, `dotfiles-sync.{sh,ps1}`, a setup-windows deploy step). Rather
  than ship a broken `dotf doctor` / secrets-uploader, all were migrated to the registry SSOT
  in the same PR, with `dotf secrets ls --pairs` added as the machine-readable enabler. The
  full runbook/troubleshooting rewrite was the one piece deferred (→ #600) to keep this bounded.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-025-secrets-render/` -> `specs/archive/CLI-025-secrets-render/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
