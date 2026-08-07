---
id: "AI-022-pi-harness-slot"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-10"
issue: "dotfiles#161"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# AI-022: Pi harness slot

> **Naming**: file lives at `<repo>/specs/AI-022-pi-harness-slot/proposal.md`. `AI-022-pi-harness-slot` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #161: AI-022: Install Pi coding agent (native AGENTS.md reader; registry slot) -->

AI-025 made pi a managed agent (pinned npm install, `~/.pi/agent/` config deploy, healthcheck, NaN key substitution), but the **registry slot** half of #161 is still open: pi is not in `harness/manifest.json`, so the unified engine deploys vault skills to claude, opencode, and agy — and pi gets none. The skills pi has today are ad-hoc symlinks created by its own installer, invisible to the harness lifecycle (refresh/deploy/check/prune).

## What

- `harness/manifest.json` gains a `skills.deploy[]` entry: `{agent: "pi", render: "skill", dir: ".pi/agent/skills"}`. Both engines (`compile-harness.sh --deploy` on Linux, the `Deploy-SkillRecord` port in `setup-windows.ps1`) are manifest-driven, so vault skills land in `~/.pi/agent/skills/<name>/SKILL.md` as regular copies on both OSes, honoring per-skill `targets:` frontmatter and pruning stale outputs.
- `tests/skills-pipeline.bats` covers pi end-to-end (deploy renders /spec for pi; copies, not symlinks; `targets:`-restricted skills excluded).
- `scripts/healthcheck.sh` documents why `~/.pi/agent/skills` is deliberately NOT in the strict symlink sweep: pi's own installer manages sibling skills as symlinks into `~/.agents/skills` — flagging those would fight the agent's filesystem expectations (docs/lessons.md lesson 13; ADR-012 rationale).

This completes the residual scope of #161 (install + AGENTS.md deploy + healthcheck shipped in AI-025).

## Out of scope

- Managing or versioning pi-installed skills (`~/.pi/agent/skills/*` symlinks into `~/.agents/skills`) — user-installed runtime state.
- A `command`/`prompt` render variant for pi — pi consumes SKILL.md dirs natively; `skill` render is correct.
- Windows empirical validation — stays on #297 (batch-Windows session).

## Risks / open questions

- Name collision between a vault skill and a pi-installed symlink of the same name: the engine de-symlinks the destination before copying (existing BUG-100 behavior), so ours wins deterministically. Current names don't collide.
- pi skill discovery path: confirmed empirically on this box (`~/.pi/agent/skills/` holding SKILL.md dirs/symlinks).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `harness/manifest.json` lists pi under `skills.deploy[]` with `render: skill`, `dir: .pi/agent/skills`.
- [ ] `compile-harness.sh --deploy` with a fake `$HOME` renders /spec to `.pi/agent/skills/spec/SKILL.md` as a regular file (not a symlink).
- [ ] A `targets:`-restricted skill that excludes pi is NOT deployed to pi.
- [ ] Full bats suite green; shellcheck clean on changed scripts.

## References

- GitHub issue: `dotfiles#161` (work-gate)
- Related ADR: `docs/adr/adr-012-deploy-strategy-copy-with-drift-assertion.md`
- Related spec: `specs/archive/AI-025-pi-coding-agent/` (install + config deploy)
