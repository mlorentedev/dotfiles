---
id: audit-005-scripts-classification
type: audit
status: complete
date: 2026-05-21
parent: REFACTOR-001
related: [audit-001-repo-structure, audit-002-cross-os-duplication, audit-bug-006-load-secrets-cross-os, dotfiles-architecture-map]
tags: [scripts, refactor, classification, dotfiles, vault-deliverable]
---

# AUDIT-005 — scripts/ classification & refactor roadmap

> **Scope:** read-only classification + duplication + dead-code audit of `/home/manu/Projects/dotfiles/scripts/` (45 files: 24 `.sh` Linux + 21 `.ps1` Windows). Output: this report + downstream multi-PR roadmap. Source ticket: **REFACTOR-001** (parent, this audit closes it).
> **Method:** Explore-agent dispatch + manual ID reconciliation against existing backlog.
> **Companion to:** AUDIT-002 (cross-OS duplication, surfaced SDD-004), BUG-006 (load-secrets ratio anomaly, surfaced BUG-008/009/010).

## 1. Inventory Table

| Script | LOC | Purpose | Pair? | PS1 LOC | Category |
|--------|-----|---------|-------|---------|----------|
| utils.sh | 726 | Foundation library (logging, paths, command validation, secrets helpers) | No | — | Foundation |
| load-secrets.sh | 1,058 | Decrypt age-encrypted secrets, export env vars (20 functions) | Yes | 254 | Secrets |
| test.sh | 567 | Comprehensive dotfiles test suite (109 tests) | No | — | Dev tooling |
| claude-session-start.sh | 497 | Claude Code SessionStart hook (context, SDD reminder, vault health) | Yes | 440 | Claude integration |
| init-project.sh | 460 | Initialize new Claude Code projects (dual AI config) | Yes | 576 | Repo-init |
| healthcheck.sh | 448 | Post-setup tool/version verification (12 sections) | Yes | 520 | Diagnostics |
| knowledge-crystallize.sh | 289 | AI knowledge maintenance (MEMORY.md dates, health, checklist) | Yes | 264 | Vault |
| vault-health.sh | 247 | Knowledge vault health check via Obsidian CLI | No | — | Vault |
| doctor.sh | 218 | Verify env vars / PATH / binaries against env-contract.json | Yes | 233 | Diagnostics |
| claude-mem-heal.sh | 213 | Idempotently repair claude-mem plugin (BUG-014/015/016/017) | Yes | 269 | Claude integration |
| check-spec-gate.sh | 198 | SDD Tier 4 PR-size gate (LOC threshold + spec-folder check) | No | — | CI/Gate |
| github-secrets-manager.sh | 194 | Upload secrets to GitHub repo secrets via `gh` CLI | No | — | Secrets |
| skills-to-opencode.sh | 167 | Convert Claude Code skills → OpenCode commands | No | — | AI tooling |
| dotfiles-sync.sh | 155 | Bidirectional sync: repo ↔ `~/.dotfiles` (deploy-dir) | Yes | 175 | Lifecycle |
| init-spec.sh | 143 | Scaffold `specs/<id>/` folder + proposal/tasks/verification | Yes | 138 | SDD tooling |
| init-repo-standards.sh | 137 | Bootstrap repo standards & templates | Yes | 155 | Repo-init |
| changelog-gen.sh | 136 | Regenerate CHANGELOG.md from conventional commits | No | — | Repo tooling |
| archive-spec.sh | 122 | Archive SDD spec post-merge + selective vault promotion | Yes | 113 | SDD tooling |
| diff-check.sh | 116 | Detect drift: repo vs `~/.dotfiles` | Yes | 157 | Diagnostics |
| init-repo-agents.sh | 112 | Bootstrap AGENTS.md with SDD snippet from vault | Yes | 117 | Repo-init |
| age-standalone.sh | 108 | Standalone age encrypt/decrypt (USB recovery, zero deps) | No | — | Secrets |
| init-repo-github-defaults.sh | 97 | Bootstrap GitHub repo config defaults | Yes | 93 | Repo-init |
| age-encrypt-decrypt.sh | 94 | Encrypt/decrypt `*.secret` files using age | No | — | Secrets |
| shell-profile.sh | 92 | Measure shell startup time, per-function profiling | No | — | Dev tooling |
| check-md-escapes.sh | 79 | Detect Hive vault_patch corruption in markdown | No | — | Vault |
| backup-secrets-to-usb.sh | 69 | Backup secrets + standalone age script to USB | No | — | Secrets |
| install-precommit.sh | 61 | Install pre-commit hooks for current git repo | No | — | Repo tooling |
| obs-cli.sh | 51 | Thin Obsidian CLI wrapper (--no-sandbox, default vault, GUI check) | Yes | 79 | Vault |
| vault-maintenance-weekly.sh | 46 | Weekly cron: crystallize + vault health | Yes | 56 | Vault |

**Totals:** 29 `.sh` files counted (`init-spec.sh` was missing from agent's first-pass; reconciled here). Linux LOC: ~6,898. Windows LOC across paired `.ps1`: ~4,030. Combined: ~10,928. Pairs: 18 cross-OS, 11 Linux-only.

## 2. Role Classification (9 categories)

- **Foundation library** (1): `utils.sh` — sourced by 11+ scripts; defines `log_*`, `pass`/`fail`/`skip`, path helpers, command checks, secrets primitives.
- **Secrets** (5): `load-secrets`, `age-encrypt-decrypt`, `age-standalone`, `github-secrets-manager`, `backup-secrets-to-usb`. Only `load-secrets` is paired (with 4.2× ratio defect, see §3).
- **Diagnostics & health** (4): `doctor`, `healthcheck`, `diff-check`, `vault-health`. First three are paired; `vault-health` is Linux-only.
- **SDD & CI** (3): `check-spec-gate` (Linux-only), `init-spec`, `archive-spec`.
- **Claude integration** (2): `claude-session-start`, `claude-mem-heal`. Both paired.
- **Knowledge/vault** (5): `vault-maintenance-weekly`, `knowledge-crystallize`, `obs-cli`, `vault-health`, `check-md-escapes`. 3 paired, 2 Linux-only.
- **Repo initialization** (5): `init-project`, `init-repo-agents`, `init-repo-github-defaults`, `init-repo-standards`, `init-spec`. All paired.
- **Lifecycle/sync** (1): `dotfiles-sync` (bidirectional sync). Paired.
- **Dev tooling** (5): `test`, `shell-profile`, `changelog-gen`, `skills-to-opencode`, `install-precommit`. All Linux-only.

## 3. Cross-OS pair ratio table

Healthy range: `[0.7, 1.3]`. **14/18 in range; 4 flagged.**

| Pair | SH | PS1 | Ratio | Status |
|------|----|----|------|--------|
| **load-secrets** | 1,058 | 254 | **0.24** | **CRITICAL** (BUG-006; BUG-008/009/010 queued) |
| obs-cli | 51 | 79 | 1.54 | DRIFT (verbose Windows help/check) |
| diff-check | 116 | 157 | 1.35 | DRIFT (extra Windows file output) |
| init-project | 460 | 576 | 1.25 | OK (edge of band) |
| claude-mem-heal | 213 | 269 | 1.26 | OK (edge of band; heal complexity) |
| vault-maintenance-weekly | 46 | 56 | 1.22 | OK |
| healthcheck | 448 | 520 | 1.16 | OK |
| dotfiles-sync | 155 | 175 | 1.13 | OK |
| init-repo-standards | 137 | 155 | 1.13 | OK |
| doctor | 218 | 233 | 1.07 | OK (reference pair) |
| init-repo-agents | 112 | 117 | 1.04 | OK |
| init-spec | 143 | 138 | 0.96 | OK |
| init-repo-github-defaults | 97 | 93 | 0.96 | OK |
| archive-spec | 122 | 113 | 0.93 | OK |
| knowledge-crystallize | 289 | 264 | 0.91 | OK |
| claude-session-start | 497 | 440 | 0.89 | OK |

Plus 2 not in agent's table: `init-spec`, `archive-spec` (now included above).

**Key findings:**
1. `load-secrets`: confirmed CRITICAL — already vault-tracked as BUG-006, downstream BUG-008/009/010 queued (Windows-empirical).
2. `obs-cli` and `diff-check` drift is mild (~10-30 LOC of legitimate Windows-specific divergence); reasonable to leave as-is unless consolidated in a polish PR.

## 4. Dead-code / orphan candidates

| Script | External refs | Verdict |
|--------|--------------|---------|
| `shell-profile.sh` | 1 (chmod +x in setup-linux.sh only) | **Orphan** — never invoked. Candidate for deletion or revival via alias. |
| `backup-secrets-to-usb.sh` | 2 (chmod + README mention) | Minimal — manual invocation only. Keep (recovery tooling). |
| `init-repo-{agents,standards,github-defaults}.sh` | Each called only by `init-project.sh` | Private helpers — consolidation candidate. |

**Strong recommendation:** delete or alias-document `shell-profile.sh`. It exists in the deploy chain but no entry point calls it. Either:
- Add an alias `profile-shell` to `.zsh/aliases.zsh` + documentation in CLAUDE.md, OR
- Delete (~92 LOC) and reclaim the LOC budget.

## 5. Multi-PR refactor roadmap

> **Reviewer notes on IDs:** the Explore-agent's initial draft used `SDD-004/005/006/...` for new specs, which **collides with existing backlog allocations** (SDD-004 = session-start-config queued; SDD-005 = github-copilot-instructions-sync merged PR #62; SDD-006 = vault-integrity-check merged PR #63). All downstream items below use **fresh free IDs** in the appropriate prefix class (REFACTOR for code reshape, FEAT for new capabilities, CHORE for deletes, BUG for cross-OS parity gaps).

| ID | Scope | P | LOC | Risk | Notes |
|----|-------|---|-----|------|-------|
| **BUG-008/009/010** | load-secrets.ps1 parity (3 ports) | P1 | ~380 | High | **Already queued** — Windows-empirical, 10 missing functions. This audit confirms BUG-006 scope. No renaming needed. |
| ~~**REFACTOR-004**~~ | ~~Wire 3 init-repo-* helpers into init-project default flow~~ | P2 | +95 | Low | ✅ **Closed 2026-05-21 (PR [#91](https://github.com/mlorentedev/dotfiles/pull/91))** — implementation pivoted from "consolidate as subcommands" to "wire into default flow with opt-out flags" after verify-before-act discovered the audit's premise was wrong (init-repo-* are standalone CLIs, not private helpers). The 3 scripts remain bit-identical; init-project.{sh,ps1} gains the wiring + --skip-* flags. |
| **REFACTOR-005** | Vault tooling unification: merge `vault-health` + `vault-maintenance-weekly` + `check-md-escapes` behind single CLI | P2 | ~150-200 | Med | `vault-cli {health,maintain,check-md}`. Risk: medium because vault-maintenance-weekly is a cron entry — must preserve invocation contract. |
| **REFACTOR-006** | `obs-cli.ps1` + `diff-check.ps1` parity polish (drop verbose to match `.sh`) | P3 | ~40-60 | Low | Drift is mild; defer until adjacent work. |
| ~~**CHORE-001**~~ | ~~Delete or alias-document `shell-profile.sh`~~ | P3 | +10 | Low | ✅ **Closed 2026-05-21 (PR [#92](https://github.com/mlorentedev/dotfiles/pull/92))** — user chose alias + README docs; `shell-profile.sh` kept as dormant-but-load-bearing diagnostic. |
| **FEAT-001** | Port `check-spec-gate.sh` → `.ps1` (Tier 4 Windows parity) | P3 | ~200-300 | Med | SDD enforcement currently Linux-only. Tier 4 CI gate already enforces in GitHub Actions regardless; this is only for local pre-push parity. Low value unless Windows authors hit it. |
| **FEAT-002** | Port `shell-profile.sh` → `.ps1` (Windows shell startup profiling) | P3 | ~80-120 | Low | Speculative value — depends on whether user profiles PowerShell startup. Probably skip. |
| **POLISH-001** | Extract repeated `get_script_dir()` + file-sourcing fallback into `utils.sh` | P3 | -50 to -70 | Low | 11 scripts repeat the BASH_SOURCE pattern. Move into utils.sh to reduce surface. |

**Recommended execution order (highest leverage first):**
1. **REFACTOR-004** (init-repo consolidation) — quick win, low risk, reduces script count visibly. ~1-2 hour PR.
2. **REFACTOR-005** (vault tooling unification) — medium effort, clear value, prepares for further vault work.
3. **CHORE-001** (shell-profile decision) — user input gate.
4. **POLISH-001** (utils.sh extraction) — bundled with any of the above as opportunistic refactor.
5. **REFACTOR-006**, **FEAT-001/002** — defer unless adjacent work touches the files.

## 6. Shared-boilerplate-extraction notes

Three patterns repeated across 5+ scripts; candidates for utils.sh extraction:

1. **`get_script_dir()`** — 11 scripts repeat `SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"`. Move to utils.sh as `get_script_dir()`. Net: -15 LOC.
2. **utils.sh source-fallback** — 8 scripts try repo-local then `~/.dotfiles/` fallback for utils.sh source. Worth making utils.sh self-locating. Net: -50 to -70 LOC + improved resilience.
3. **Obsidian GUI check** — 3 vault scripts repeat the GUI-process check. Move to `utils.sh::check_obsidian_gui()`. Net: -10 LOC.

Bundle into **POLISH-001** above.

### POLISH-001 follow-up (2026-05-21 evening) — DEFERRED / WONTFIX

Investigation during AUDIT-005 closeout revealed the audit overestimated the net value. Most of the "repeated boilerplate" is **bootstrap code** — it runs BEFORE `utils.sh` is sourced, so a helper inside `utils.sh` cannot replace it (chicken-and-egg). The only path to extract it would be a new `scripts/_bootstrap.sh` file that each script sources via a 1-liner, but this:

- introduces a new file in `scripts/` (cognitive overhead);
- couples 10 scripts to its continued existence (breakage if renamed);
- nets only ~-10 LOC overall after adding the bootstrap file's content;
- adds non-trivial risk of subtle script-loading regressions (the bootstrap path differs slightly under bash vs zsh, with vs without `BASH_SOURCE`, etc.).

**Decision:** WONTFIX. The current `SCRIPT_DIR` + `if [ -f ... utils.sh ]` boilerplate (5 lines per script × 10 scripts = ~45 LOC) stays. Trade-off rejected: the file-add cost + breakage risk exceeds the LOC saving.

**Reusable lesson:** when an audit proposes "extract boilerplate", classify the boilerplate first — **bootstrap** (chicken-and-egg, runs before the helper would exist) is inextricable; **logic** (runs after the helper is loaded) is extractable. The audit-005 agent did not make this distinction, leading to a misleadingly high LOC-saving estimate.

The `check_obsidian_gui()` extraction (item 3) IS extractable (3 vault scripts already source utils.sh before invoking the GUI check, so the function can live in utils.sh). Splitting it out as a sibling micro-PR is possible if motivated, but the upside is ~-10 LOC across 3 files — likely below the effort threshold.

## Conclusion

Scripts collection is **well-structured** (9 clear categories, strong cross-OS parity in 14/18 pairs). Top actionable findings:

1. **Confirmed CRITICAL:** `load-secrets` cross-OS gap (already vault-tracked, downstream BUG-008/009/010 queued — no new work surfaced).
2. **NEW REFACTOR-004:** 3 redundant `init-repo-*` scripts → consolidate into `init-project` subcommands. **Top quick-win.**
3. **NEW REFACTOR-005:** Vault tooling fragmentation (3 overlapping scripts) → unify behind `vault-cli`.
4. **NEW CHORE-001:** `shell-profile.sh` is orphan (deploy-only, never invoked) → user decision needed.
5. **NEW POLISH-001:** ~75-85 LOC of boilerplate extractable into utils.sh.

**Sprint slot:** add REFACTOR-004 + REFACTOR-005 + CHORE-001 to the active sprint backlog (`11-tasks.md`). Total estimated ~250-350 LOC of net change across 3 atomic PRs.
