---
id: audit-007-cli-convergence-state
type: audit
status: dated-snapshot
date: "2026-06-20"
parent: adr-021-cli-orchestration-roadmap
related: [adr-020-tooling-cli-go-convergence, adr-021-cli-orchestration-roadmap, audit-002-cross-os-duplication, audit-005-scripts-classification, audit-bug-006-load-secrets-cross-os]
tags: [cli, dotf, convergence, strangler-fig, cross-os, execution-plan]
---

# AUDIT-007 — CLI convergence state + execution roadmap

> **Status note:** `status: dated-snapshot` — the plan below is half-executed as of
> this writing (`secrets`, `mem`, and `update`/`sync` have since shipped Go nouns
> this snapshot marked as gaps). Treat the plan as historical direction, not a
> current TODO list; `dotf --help` and `cli/README.md` are the current source of
> truth for what's shipped.
>
> Dated current-state snapshot of the ADR-020/021 strangler-fig migration, with an
> ordered, PR-by-PR plan to drive the `.sh`/`.ps1` surface down to the ADR-021 end
> state. Produced 2026-06-20 by a multi-agent audit (8 per-noun maps + synthesis +
> a completeness critic), then human-verified against the live tree. ADR-021 is the
> strategy; this is the execution map for finishing it.

## TL;DR

The migration is **half-done and asymmetric**. For the two nouns already built in
Go (`doctor`, `init`), **Linux repointed its callers and deleted the `.sh` twins,
but Windows still deploys + invokes the `.ps1` twins** — even though the Windows
`dotf` binary is now installed (`install-dotf.ps1`, WIN-006). Five domains
(vault-knowledge, secrets, mem, spec-gates, sync, harness) have **no Go
implementation at all** — they are build-first, not finish-the-strangler. Net:
**20 `.ps1` remain** (ADR-021 end state target: ~3).

The `.ps1` count never dropped because **no Windows caller was ever cut over** to
`dotf`, and several shell clusters (the `vault.sh` dispatcher → vault-health →
check-backlog-*/crystallize; claude-session-start → claude-mem-heal) are
interdependent graphs that must port as units.

## Corrected facts (verified against the live tree)

The audit surfaced three errors in prior premises; corrected here:

1. **Windows HAS `dotf` installed.** One per-noun agent claimed "no Windows `dotf`
   install path" — false. `setup-windows.ps1:521-534` runs `install-dotf.ps1`
   (WIN-006 / #451) into `~/.local/bin`, and `dotf env generate` already runs at
   `:1556-1557`. So `doctor`/`init` are **unblocked** on Windows — they only need
   their invocations repointed, not a new install path.
2. **True script counts: 34 `.sh` + 20 `.ps1` = 54** (not the "51 / ~21" the
   synthesis estimated).
3. **`init-repo-standards.{sh,ps1}` never existed** — only `init-repo-agents` and
   `init-repo-github-defaults` twins were ever built (ADR-021's table lists
   `standards` as a *target*, never implemented).

## Current state by noun

| Noun | Go status | `.sh` left | `.ps1` left | Windows wired to `dotf`? |
|---|---|---|---|---|
| `doctor` | exists-partial (Windows prod wiring deferred `fs.go:23-34`; no drift; no Win-only checks) | diff-check, vault-health | healthcheck, doctor, diff-check | **no** (still runs `.ps1`) |
| `init` | **exists-at-parity** (ADR-022/CLI-014; 10 impl files, tested) | — | init-project, init-repo-agents, init-repo-github-defaults | **no** (profile.ps1:113 + setup-windows.ps1:1436 still run `.ps1`) |
| `vault` | **not-built for this domain** — `dotf vault` is an entry-scaffolder (project/work), DISJOINT from crystallize/health/maintain | knowledge-crystallize, vault-maintenance-weekly, vault-health, obs-cli, check-md-escapes | knowledge-crystallize, vault-maintenance-weekly, obs-cli | n/a |
| `secrets` | not-built | load-secrets, github-secrets-manager, age-* | load-secrets | n/a |
| `mem` | not-built | claude-mem-heal, claude-session-start | claude-mem-heal, claude-session-start | n/a |
| `spec-gates` | not-built (POSIX-only; no `.ps1`, no Go) | check-spec-gate, check-backlog-integrity, check-backlog-merged, check-md-escapes | — | n/a |
| `harness` | not-built | compile-harness | (twin lives inside setup-windows.ps1 `Deploy-SkillRecord`) | n/a |
| `sync/update` | not-built | dotfiles-sync, dotfiles-selfupdate | dotfiles-sync, dotfiles-selfupdate | n/a |

## Execution plan (12 PRs)

Two phases: **A — finish the strangler for already-built nouns (drops `.ps1`
immediately)**; **B — build the missing Go nouns, then delete twins on contact.**
Every deletion is gated behind a parity proof + a guard-grep completeness oracle;
no Windows-load-bearing script is removed while still referenced or not-at-parity.

### Phase A — finish the strangler (built nouns)

| # | id | Scope | Deletes | Risk |
|---|---|---|---|---|
| 1 | HARNESS-030-changelog-dangling-cleanup | Remove dangling refs to already-deleted changelog-gen / skills-to-opencode; honest ADR-021 table + architecture.md:40 | 0 (refs only) | nil |
| 2 | CLI-018-dotf-doctor-windows-repoint | Repoint setup-windows.ps1 (1942-1974 invocations, 1488-1514 deploys) + ci.yml:220-233 + profile `hc` to `dotf doctor`; delete doctor/healthcheck `.ps1` | healthcheck.ps1, doctor.ps1 (+ tests) | medium |
| 3 | CLI-019-doctor-drift-absorb-diffcheck | Implement repo-deploy drift in `dotf doctor` (`checks_deploy.go:350` deferred); delete diff-check | diff-check.{sh,ps1} (+ tests) | medium |
| 4 | CLI-020-dotf-init-windows-repoint | Repoint profile.ps1:113 `project-init` + setup-windows.ps1:1436-1443 to `dotf init`; delete the 3 init `.ps1` (one runtime cluster) | init-project.ps1, init-repo-agents.ps1, init-repo-github-defaults.ps1 (+ tests) | medium |

### Phase B — build-first nouns

| # | id | Scope | Risk |
|---|---|---|---|
| 5 | CLI-021-dotf-vault-build-knowledge | Resolve the `dotf vault` noun collision (scaffolder vs knowledge); build `vault crystallize/maintain/health`. Build+test only, no deletes | medium |
| 6 | CLI-022-dotf-spec-gate-build | Build `dotf spec gate` + backlog/md-escape subcommands beside the shell; flip nothing (critical merge-path gate) | high |
| 7 | CLI-023-vault-specgate-cutover-delete | Repoint CI/pre-commit/vault-dispatcher/cron/aliases; delete the 7-script vault+spec-gate cluster + 2 `.ps1` | high |
| 8 | CLI-024-dotf-secrets-build-and-shim | Build `dotf secrets` (reconstruct the Linux SUPERSET, not the `.ps1` 5-fn subset); `eval "$(dotf secrets env)"` shims; delete load-secrets twins | high |
| 9 | CLI-025-dotf-mem-heal-and-session-start | Build `dotf mem heal/session-start`; thin SessionStart hook to a shim; delete the 4 scripts. Porting also closes a live Windows gap (the `.ps1` lacks the BUG-019 context-hook neuter) | medium |
| 10 | CLI-026-dotf-harness-engine | Port compile-harness to `dotf harness {refresh,deploy,check}`; replace the `Deploy-SkillRecord` block inside setup-windows.ps1 | medium |
| 11 | CLI-027-dotf-sync-update | Build `dotf sync`/`update` (reconcile rsync-model Linux vs git-pull-model Windows); keep scheduler shims; delete the 4 sync/selfupdate scripts | medium |
| 12 | CLI-028-dotf-setup-last | Move the setup orchestrator body to `dotf setup`; leave only curl bootstrap + install-dotf + the floor | high |

**Projected reduction:** `.ps1` 20 → ~5 after PR 11 (→ ~3 after PR 12); cross-OS
pairs ~18 → ~2 — matching the ADR-021 end state.

## Completeness gap — 12 scripts unowned (MUST resolve before declaring done)

The completeness critic found `allScriptsClassified = FALSE`: 12 scripts fall
outside every per-noun map and PR. **Two are live twin pairs that would recreate
the exact orphan bug this audit exists to fix** if executed as-is:

| Script(s) | Why it's a risk | Disposition |
|---|---|---|
| `session-handoff.{sh,ps1}` | LIVE twin pair — SessionEnd hook (setup-linux.sh:1244, setup-windows.ps1:1461-1467/1667). Structurally identical to session-start, which `mem` migrates, but **no noun owns it** | Add to **PR 9 (`mem`)** as a `dotf mem session-end` sibling shim |
| `orca-tune.sh` / `orca-hook-tune.ps1` | ~~LIVE twin pair. The plan named the `.ps1` as "floor" but **omitted the `.sh`** — the mirror image of the orphan bug~~ **Resolved as a pair, both ported (CLI-062, #1338, 2026-08-29):** `orca-tune.sh` was already `dotf orca tune` (#1274) with no caller left, deleted; `orca-hook-tune.ps1` — a different function, the DX-006 hook repair — is `dotf orca tune-hooks`, which setup and `doctor --fix` both call; script, Pester and bats deleted | Ported |
| `session-brief.sh` | Sourced by claude-session-start.sh ("session-brief core"); would orphan when session-start is deleted | Fold into **PR 9 (`mem session-start`)** explicitly |
| `ensure-memory-symlink.sh` | Auto-memory link logic; overlaps `mem session-start` | Fold into **PR 9** |
| `install-git-hooks.sh`, `install-precommit.sh` | Hook installers; sourced live (setup-linux.sh:272-277) but never named as floor | Mark **floor** explicitly |
| `nan-bench.sh`, `nan-debug.sh`, `nan-quality-bench.sh`, `bitacora-rollout.sh` | `ai/nan` project tooling / archived-spec one-shot | Mark **out-of-scope** (project-local) explicitly |

`scripts/test.sh` (the bats harness) retires implicitly as its consumers leave
(ADR-021 consequences) — borderline floor-vs-unaccounted; name it explicitly.

## Irreducible shell floor (correctly stays shell — ADR-021 boundary)

`install-dotf.{sh,ps1}`, `shell-profile.sh`, `windows-defaults.ps1`,
`profile-heal.ps1`, `utils.{sh,ps1}`, the curl bootstrap (IDEAS-005), the
secrets env-export shim and OS-scheduler unit files. (`orca-tune.sh` /
`orca-hook-tune.ps1` left the floor with CLI-062: both ported, see the table.)

## Immediate next move

Per ADR-021 the order is `doctor` first, but **on risk, `init` (PR 4) is the
cleanest first real deletion**: it is `exists-at-parity` (Go fully built), so it
drops **3 `.ps1`** by repointing just two Windows callers (`profile.ps1:113`,
`setup-windows.ps1:1436-1443`). `doctor` (PR 2) is `exists-partial` — its Windows
production wiring is deferred (`fs.go:23-34`) and it lacks drift + Windows-only
checks, so it needs a parity verification before any deletion. Recommended start:
**PR 1 (zero-risk dangling cleanup) → PR 4 (`init`) → PR 2-3 (`doctor`).**

## References

- [ADR-020](adr-020-tooling-cli-go-convergence.md) — the convergence decision.
- [ADR-021](adr-021-cli-orchestration-roadmap.md) — the roadmap this executes.
- [AUDIT-005](audit-005-scripts-classification.md) — the `scripts/` classification (§5 superseded by ADR-021).
- [audit-bug-006](audit-bug-006-load-secrets-cross-os.md) — the load-secrets gap (BUG-008/009/010 ps1-parity superseded by ADR-021: `dotf secrets` deletes the `.ps1`).
