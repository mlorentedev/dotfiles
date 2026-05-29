---
id: audit-002-cross-os-duplication
type: audit
status: active
created: "2026-05-19"
---

# AUDIT-002 — Cross-OS Duplication (`.sh` ↔ `.ps1`)

> Pair-by-pair audit of every `scripts/*.sh` with a `.ps1` sibling. Baseline: [AUDIT-004](dotfiles-architecture-map.md). Generated 2026-05-19.

## TL;DR

**14 cross-OS pairs + 1 setup pair = 15 dual-maintenance surfaces. Total LOC: ~11,660** (4,807 sh + 4,898 ps1 in pairs, plus 1,955 setup). The pattern is **already partially SSOT-driven**: `doctor.{sh,ps1}` reads `env-contract.json`, `setup-{linux.sh,windows.ps1}` read `mcp-servers.json` and `versions.conf`. Where SSOT-isation has been applied, drift is invisible; where it hasn't, every cross-OS edit is two-place.

**Top extractable candidate: `claude-session-start.{sh,ps1}`** — 937 LOC combined, high change frequency (modified every time SessionStart hook gains a new injector). A `session-start-config.json` SSOT defining probes + injection order would let per-OS scripts stay thin readers, mirroring the `doctor` + `env-contract` precedent.

**One anomaly worth flagging**: `load-secrets.sh` is **1058 LOC** vs `load-secrets.ps1` at **254 LOC** (ratio 0.24). Either ps1 is missing features (cross-OS DEFECT) or sh has bloat (REFACTOR-001 will catch it). Not an SSOT-ification target, but a separate cross-OS-completeness audit candidate.

## Pair-by-pair classification

Legend:
- **Forced parallel** — different OS APIs make code structurally divergent (apt vs winget, ln -s vs Copy-Item). Logic and structure both per-OS.
- **SSOT-extractable** — structure is identical; only API calls differ. Could refactor to read a shared config + thin per-OS reader.
- **Already SSOT-driven** — the pair already reads a shared config; included for the reference pattern.

| Pair | sh LOC | ps1 LOC | Ratio | Classification | Notes |
|---|---:|---:|---:|---|---|
| `archive-spec` | 122 | 113 | 0.92 | Forced parallel | Mechanical scaffolding. Templates already in vault `00_meta/templates/`. |
| `claude-mem-heal` | 120 | 163 | 1.35 | Forced parallel | Patches specific upstream regressions (claude-mem v12/v13). Each path is OS-specific. |
| `claude-session-start` | **497** | **440** | 0.88 | **SSOT-extractable** ★ | Both inject context (vault health / `[sdd]` reminder / claude-mem context / memory archive nudge). Probe execution differs per-OS but the *injection layout* is identical. **HIGH-PRIORITY EXTRACT.** |
| `doctor` | 218 | 233 | 1.06 | **Already SSOT-driven** ✓ | Reads `env-contract.json`. **Reference pattern.** |
| `dotfiles-sync` | 155 | 175 | 1.12 | SSOT-extractable (low priority) | Sync paths could move to a `sync-manifest.json`. Low change frequency. |
| `init-project` | 460 | 576 | 1.25 | Forced parallel | New-repo bootstrap. Templates already in vault. Per-OS differences are in shell setup (`zsh` vs PowerShell profile). |
| `init-repo-agents` | 112 | 117 | 1.04 | Forced parallel | Agent file deployment. Tight pair. |
| `init-repo-github-defaults` | 97 | 93 | 0.95 | Forced parallel | `gh` CLI calls; same on both OSes. Per-OS differences minimal. |
| `init-repo-standards` | 137 | 155 | 1.13 | Forced parallel | Repo standards seeding. |
| `init-spec` | 143 | 138 | 0.96 | Forced parallel | Spec scaffolding. Templates in vault. Tight pair. |
| `knowledge-crystallize` | 289 | 264 | 0.91 | SSOT-extractable (medium priority) | Scans Claude Code project paths + computes staleness. Probe paths could move to a `crystallize-config.json`. |
| `load-secrets` | **1058** | **254** | **0.24** ⚠ | **ANOMALY** | sh is 4× ps1. Either ps1 is missing features (DEFECT) or sh has bloat (REFACTOR-001). Flagged for separate cross-OS-completeness audit. |
| `obs-cli` | 51 | 79 | 1.54 | Forced parallel | Wrapper around Obsidian CLI binary. ps1 overhead is just PowerShell verbosity. Skip. |
| `vault-maintenance-weekly` | 46 | 56 | 1.21 | Forced parallel | Cron / Task Scheduler entry. Trivial. |
| **`setup-linux.sh` / `setup-windows.ps1`** | **912** | **1043** | 1.14 | **Mostly already SSOT-driven** ✓ | Reads `versions.conf`, `env-contract.json`, `mcp-servers.json`, `sensitive/env-mapping.conf`. The remaining ~1955 LOC is genuinely OS-specific (apt vs winget, symlinks vs copies). Diminishing returns from further extraction. |

### Anomaly investigation: `load-secrets.sh` 1058 LOC

The 4× ratio is suspicious. Two hypotheses:

1. **`load-secrets.ps1` is incomplete** — the Linux version implements `secrets_show`, `secrets_refresh`, `secrets_add_file`, `secrets_rotate`, plus file-secret support (`@VAR=filename>dest`). If ps1 is missing some of these functions, the cross-OS contract is broken.
2. **`load-secrets.sh` has accumulated cruft** — REFACTOR-001 territory. The size warrants a split into smaller modules.

**Recommendation:** treat as a separate `BUG-006-load-secrets-cross-os-completeness` audit, NOT as an AUDIT-002 SSOT-ification target. Quick win: grep both files for the same public function names; any in sh but not ps1 is a defect.

## SSOT-isation priority list (by value)

Value scored as `LOC × change_frequency`. Change frequency from recent git activity + reasoning about the function's role.

| Rank | Pair | LOC saved (est.) | Change freq | Value | Action |
|:---:|---|---:|---|---|---|
| **1** | `claude-session-start.{sh,ps1}` | ~150 LOC (refactor to thin readers of `session-start-config.json`) | **HIGH** — modified every time a new injector is added (BUG-003, SDD-001, SDD-021, claude-mem heal) | HIGH × HIGH = **TOP** | **Spec it** — `SDD-XXX-session-start-config` |
| 2 | `knowledge-crystallize.{sh,ps1}` | ~50 LOC (probe paths + thresholds in `crystallize-config.json`) | MEDIUM | MEDIUM | Bundle with `dotfiles-sync` if a future spec touches them |
| 3 | `dotfiles-sync.{sh,ps1}` | ~30 LOC | LOW | LOW | Defer indefinitely |
| 4 | Spec/init-repo family (`init-spec`, `archive-spec`, `init-repo-*`) | ~80 LOC aggregate | LOW (used at repo bootstrap only) | LOW | Defer; not worth touching |

## Anti-recommendations

- **Do NOT extract** `claude-mem-heal.{sh,ps1}` — each patch is upstream-bug-specific; the duplication is necessary because the regressions differ per platform.
- **Do NOT extract** `obs-cli.{sh,ps1}` — both files just shell out to the same Obsidian binary; the duplication is API-binding plumbing, not logic.
- **Do NOT extract** more from `setup-{linux.sh,windows.ps1}` — what remains after 4 existing SSOTs is genuinely OS-specific. Diminishing returns.
- **Do NOT** target `load-secrets` via SSOT — the anomaly is a completeness / refactor issue, not a duplication issue.

## Sequenced PR list

| # | PR | Estimated diff | Risk | Notes |
|---|---|---:|---|---|
| 1 | **`SDD-XXX-session-start-config`** spec + implementation | ~300 LOC (config JSON + refactored sh + refactored ps1 + bats) | Medium — touches a hook that fires every Claude session | Spec it properly. Tests must lock pre-/post-refactor behaviour byte-equivalence. |
| 2 (deferred) | `BUG-006-load-secrets-cross-os-completeness` | ~50 LOC investigation + fix | Low | Separate audit; surface defects first, then fix. |
| 3 (deferred) | knowledge-crystallize SSOT extract | ~100 LOC | Low | Only when knowledge-crystallize gains a 3rd reason to change. |

## Observations (not action items)

- **The `doctor` + `env-contract.json` pair is the gold standard** for cross-OS factorisation. Any new feature crossing both OSes should ask "can this follow the doctor model?" first.
- **SSOT readers can be sloppy.** A common failure mode is reading the SSOT then duplicating the parsing logic per-OS. The current SSOTs (`env-contract.json`, `mcp-servers.json`, `versions.conf`) all have minimal schemas that fit in a single `jq` / `ConvertFrom-Json` call. **Keep schemas minimal** when extracting more.
- **Cross-OS test parity** (`tests/setup-windows.bats` + `tests/healthcheck.bats` etc.) is the safety net that lets these refactors land safely. AUDIT-002 candidates SHOULD ship parity asserts in the same PR.

## Closing

- [ ] PR (rank #1): `SDD-XXX-session-start-config` — open a spec and treat as a normal SDD lifecycle.
- [ ] Backlog entry: `BUG-006-load-secrets-cross-os-completeness` (cross-OS completeness investigation).
- [ ] Tick AUDIT-002 in the project task backlog with this report's path.

## References

- [AUDIT-004](dotfiles-architecture-map.md) — baseline.
- [AUDIT-001](audit-001-repo-structure.md) — repo-level structure (this audit operates *inside* `scripts/`).
- [ADR-009](adr-009-multi-agent-runtime.md) — AGENTS.md SSOT pattern (same principle, prose surface).
- Pattern precedents: `env-contract.json` + `doctor.{sh,ps1}`, `mcp-servers.json` + setup scripts, `versions.conf` + RC files.
