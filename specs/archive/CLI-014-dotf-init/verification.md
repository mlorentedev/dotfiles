---
tags: [spec, verification, templates]
created: "2026-06-14"
---

# Verification - CLI-014-dotf-init

## Evidence

- [x] **AC1** scaffold with no vault / no `~/.claude`, zero `$VAULT_PATH` leak (closes #248) -> Steps 1-4; render-time `AgentsSnippet` guard fails closed on a surviving `$VAULT_PATH`; e2e smoke confirmed zero literals.
- [x] **AC2** `dotf init agents` / `github` independently re-runnable + idempotent -> `agents_test.go`, `github_test.go` (Steps 2-3).
- [x] **AC3** AGENTS.md is the SSOT, CLAUDE.md a thin pointer (closes #299) -> Step 4 (`vault.go` / scaffold writes the pointer).
- [x] **AC4** embedded templates drift-tested against the vault SSOT -> `drift_test.go` (`bytes.Equal`), Step 1.
- [x] **AC5** vault entry + GitHub defaults auto-skip with a `[skip]`/`[warn]`, exit 0 -> Step 4 orchestrator step statuses.
- [x] **AC6** `scripts/init-{project,repo-agents,repo-standards,repo-github-defaults}.sh` + their bats removed; guard-grep `init-(project|repo)` returns only provenance -> **this session**: `git rm` of 4 `.sh` (+ `init-repo-standards.ps1`, dropped) + `tests/init-project.bats`; guard-grep leaves only CHANGELOG/ADR/specs/lessons provenance, the 3 kept Windows `.ps1` (#380), and Go port-provenance comments.
- [x] **AC7** `setup-linux.sh` + docs repoint to `dotf init` -> **this session**: removed the `init-project.sh` -> `~/.claude` deploy; dropped the dead `project-init` alias (`.bashrc`/`.zshrc`); repointed README, `ai-tools-setup.md`, `ai-tools.md`, `guide-bitacora-setup.md`, `bitacora-rollout.sh`. (AGENTS.md carried no `init-project` reference to repoint.)
- [x] **AC8** `go test ./...` covers scaffold + drift; `dotf init` smoke-tested e2e -> Steps 1-4 + prior-session e2e on a throwaway no-vault checkout.

## Test status

- Affected unit suite (this session): `bats tests/init-project-ps1.bats tests/init-repo-github-defaults.bats` -> **34/34 ok** (pwsh-gated PSScriptAnalyzer/syntax tests `skip` locally; CI runs them). Includes regression guards that `init-project.ps1` no longer wires `init-repo-standards` nor carries `-WorkSdk`.
- Static: `bash -n` + `zsh -n` clean on every changed `.sh`/rc; `shellcheck setup-linux.sh scripts/bitacora-rollout.sh` adds zero new findings (only pre-existing SC1091/SC2015/SC2016 infos, none on changed lines).
- Full suite to run on commit/CI: `bats tests/*.bats`, `go test ./cli/...`, PSScriptAnalyzer (`init-project.ps1` lint), Pester.
- Manual smoke test: `dotf init` e2e exercised in the prior session (full structure, AGENTS.md zero `$VAULT_PATH`, go.mod/Makefile/CI/pre-commit; vault + github auto-skip).
- No regressions: guard-grep is the completeness oracle — no live reference to a deleted `.sh` survives outside provenance + the Windows `.ps1` retained under #380.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Step 1 (package + embed + drift guard).** Templates are vendored **whole-file, byte-identical** to the vault SSOT (`embed` + `drift_test.go` does pure `bytes.Equal`), exactly like `cli/internal/spec`. Per-output transforms (snippet extraction, `$VAULT_PATH` stripping for #248) happen at *render* time in later steps, not in the embedded bytes — keeping the drift guard a clean equality check.
- **`init` parent is runnable via `RunE: cmd.Help`** (the same idiom `root.go` uses) so it is listed under "Available Commands" rather than demoted to cobra's "Additional help topics" — a non-runnable, childless command is treated as a help topic. Step 4 swaps this `RunE` for the orchestrator.
- **Vault-content drift flagged for Step 2 (not silently dropped):** the embedded `agents-spec-section.md` snippet (a) carries `$VAULT_PATH` literals — the #248 leak, stripped at render time per AC #1 — and (b) still references the deprecated `init-spec`/`archive-spec` shell scripts instead of `dotf spec`. (b) is stale *vault* content; updating the vault SSOT (then re-vendoring) is Step 2 work done via the isolated-worktree vault-commit discipline.
- **Step 2 (`dotf init agents`).** Per the approved design (vault was clean), the **vault SSOT `agents-spec-section.md` was rewritten self-contained** (points at `dotf spec`, zero `$VAULT_PATH`, drops the stale `init-spec` refs) and re-vendored — fixing #248 at the source so every consumer (the `.sh`/`.ps1` twins too) benefits, and keeping the drift-test a pure `bytes.Equal`. A **render-time guard** (`AgentsSnippet` fails closed if `$VAULT_PATH` survives) backs it (incident→guard). `BootstrapAgents` ports the script's awk section-surgery (create/append/no-op/`--force` replace-in-place) as an explicit, table-tested state machine.
- **Strangler deletion sequencing — `init-repo-agents.sh` deletion DEFERRED to Step 4 (decided).** The `.sh` is still called by `init-project.sh` (the orchestrator), which Step 4 ports and deletes wholesale alongside all init twins + their bats. Deleting the helper earlier would either regress `init-project.sh` on `main` or churn Step-4-doomed assertions; deferring keeps every intermediate commit regression-free. The branch accumulates Steps 1–4 and merges together, so the transient `.sh`+`.ps1`+`dotf` coexistence never reaches `main`. Honors the strangler *intent* (deletion tracked, lands at the caller's port), not silent debt.
- **Step 4 (orchestrator), decisions a–e.** (a) **CI scaffold = a stack-appropriate stub** (go: build/vet/test; python: ruff/pytest; node: npm ci/test) + the SDD convention in the generated AGENTS.md — *not* an enforced spec-gate, because dotfiles' `check-spec-gate.sh` is non-portable; a portable `dotf spec gate` enforcer is deferred follow-up (user-approved). (b) **vault entry drops `11-tasks.md`** — task state lives in the bitácora (ADR-018 supersedes ADR-022's literal `{…,11-tasks,…}` list) (confirmed). (c) **stack init is lightweight + offline** — `go mod init`+Makefile, `uv`/`poetry init`, `npm init -y`, but NOT init-project.sh's opinionated network dep installs (typer/rich/pytest, typescript/vitest): a scaffold must be fast/offline-safe and the project declares its own deps. (d) **`init` parent becomes a real orchestrator** (`RunE` runs the scaffold); host-coupled steps degrade to `[skipped]/[warn]` so it always leaves a usable repo (C7). (e) Verified e2e on a throwaway repo: full structure, AGENTS.md with zero `$VAULT_PATH`, go.mod/Makefile/CI/pre-commit; vault + github auto-skip.
- **`--work-sdk` — DECIDED for Step 4f: removed (course-corrected from "extract").** init-project.sh's vault-only mode (`--work-sdk <family> <component>` → a `50_work/45-development/…` vault entry, no repo) is *not* repo-scaffolding — it is vault work, so it has no place in the `dotf init` orchestrator. The prior session's plan was to **extract** it to a standalone transitional `init-work-sdk.sh`; mid-implementation that was re-weighed against the ADR-021 north star (shrink the per-OS script surface) and **changed to outright removal** — minting a new `.sh` to delete four others is a poor trade for an infrequently-used capability (onboarding a new work-SDK component). Resolution: the mode is **removed on both OSes** (the Linux `.sh` dies with `init-project.sh`; the Windows `init-project.ps1 -WorkSdk` block + its `-Family`/`-Component` params are stripped), and **#388 tracks restoring it in `dotf vault`** — its proper home per ADR-021 step 3, cross-platform with no `.ps1` twin. Rejected: "extract" (a transitional artifact fighting the north star); "port into CLI-014" (scope-bleeds the init flagship into the vault flagship). Note: `claude-session-start.{sh,ps1}`'s `find_work_sdk_project` only *discovers* existing entries; it never invoked the script, so no session-start reference needed repointing (correcting an earlier handoff assumption).

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? yes - (1) gitignored //go:embed template builds locally but is absent in a fresh CI checkout; (2) extract-to-a-new-script fights a reduce-the-surface north star, prefer remove + ticket-restore.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no - ADR-022 already records the `dotf init` flagship; no further architectural decision surfaced.
- [x] New pattern candidate for `00_meta/patterns/`? no - both lessons are repo-local (embed/gitignore mechanics, strangler surface trade); neither recurs cross-project yet.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-014-dotf-init/` -> `specs/archive/CLI-014-dotf-init/`
- [x] Backlog entry in vault `11-tasks.md` ticked with PR link
- [x] Promotions above executed (if any)
