---
id: "ADR-012-deploy-strategy-copy-with-drift-assertion"
type: adr
status: accepted
owner: manu
date: "2026-05-26"
tags: [architecture, decision, deploy, dotfiles, sdd-007]
created: "2026-05-26"
---

# ADR-012: Deploy strategy = copy with drift assertion

> Materialized in `<dotfiles>/specs/archive/SDD-007-iac-deploy-strategy/`. Shipped via PRs #102, #105, #108 to `mlorentedev/dotfiles`.

## Status

Accepted

## Date

2026-05-26

## Context

Until SDD-007, `setup-linux.sh` deployed managed config files (`.zshrc`, `.bashrc`, MCP configs, etc.) using `ln -sf` symlinks pointing back to the repo. The "edit in `~`, symlink keeps repo in sync" model felt elegant but had two structural failure modes:

1. **Cross-agent fragility** — BUG-100 (forum #145851, `gemini-cli` #10960): the `agy` (Antigravity) CLI v1.0.2 writes in-place to `~/.gemini/...` paths. When our deploy strategy put symlinks at those paths, `agy` either clobbered them, looped through circular link chains, or hit `EEXIST` errors. The repo's deploy was *fighting the agent's expected filesystem layout*. Adding new agents (opencode, claude-mem, future tools) would re-encounter this class of bug ad-hoc.
2. **Silent drift on direct edits** — `vim ~/.zshrc; source ~/.zshrc` was an ambiguous operation: with symlinks, edits did persist (because `~/.zshrc` IS the repo file). On migration to copies, the same reflex would silently lose the edit on the next `setup-linux.sh` run. Without an enforcement layer, the migration is net-negative.

## Decision

The `setup-{linux,windows}` scripts deploy managed config files by **atomic copy** (tempfile + `mv`, idempotent: skip if `cmp -s` matches) via a `deploy_file SRC DEST` helper in `scripts/utils.sh`. Symlinks remain ONLY for:

- Vault↔home bindings (Obsidian vault content → `~/.claude/projects/<hash>/memory/`) — these are intentional cross-system shares, not config deploy.
- Secret file deployments (`~/.ssh/id_ed25519`, `~/.config/age/key.txt`) — these need canonical absolute paths.

The two-tier deploy verification ("edit repo, run setup, then test in shell") becomes a hard contract enforced by `scripts/healthcheck.sh`'s new `check_deployed` assertion: each entry of a `DEPLOYED_FILES` registry is compared with `cmp -s` against its target in `~`. Drift = healthcheck FAIL (loud red).

## Consequences

### Positive

- Eliminates the entire BUG-100 bug class (deploy fighting agent layout). Agents that write in-place to deployed paths no longer collide with our symlinks.
- Makes direct edits in `~` impossible to lose silently: the next `setup` run + healthcheck surfaces drift loudly.
- Cross-OS uniformity: Windows junctions ≠ Linux symlinks semantically; both OS now use plain file copies, same comportment.
- Future-proof for cross-agent skill pipeline (SDD-008) — that pattern depends on copies, not symlinks.

### Negative

- First-run `setup-linux.sh` wall-time +200-400ms across 14 files (bulk `cp` vs `ln -sf`). Acceptable.
- Workflow muscle memory shift: years of "edit `~/.zshrc` directly" reflex must be retrained. Mitigation: `dot` alias opens `$DOTFILES_DIR` in `$EDITOR` (documented in `.claude/CLAUDE.md`).
- Drift detector requires `DEPLOYED_FILES` registry maintenance — every new managed file added to setup must also be added to the registry, or it escapes the drift assertion.

### Neutral

- The vault `00_meta/skills/` symlinks (3 SDD skills + claude-mem skills) remain for now; SDD-008 will reconsider them under the same lens.

## References

- Spec: `<dotfiles>/specs/archive/SDD-007-iac-deploy-strategy/proposal.md`
- Shipped in: PR [#102](https://github.com/mlorentedev/dotfiles/pull/102) (initial), [#105](https://github.com/mlorentedev/dotfiles/pull/105) (Windows StrictMode + orphan cleanup), [#108](https://github.com/mlorentedev/dotfiles/pull/108) (AGY.md content polish), [#109](https://github.com/mlorentedev/dotfiles/pull/109) (hc function fix)
- Bug closed: BUG-100 (`agy` symlink fragility) — `mlorentedev/dotfiles` issue list
- Related: ADR-006 (symlinks-vs-copies — superseded for config-deploy paths by this ADR)
- Lesson captured: `~/Projects/knowledge/10_projects/dotfiles/90-lessons.md` § "Stop fighting agent filesystem expectations" (2026-05-26)
- Follow-up: SDD-008 (cross-agent skill pipeline) extends the copy-not-symlink discipline to skills
