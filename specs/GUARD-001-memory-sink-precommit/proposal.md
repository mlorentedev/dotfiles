---
id: "GUARD-001-memory-sink-precommit"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-16"
issue: "dotfiles#398"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, memory, single-sink, git-hooks, cross-agent]
template_version: "1.0"
---

# GUARD-001-memory-sink-precommit

> **Naming**: file lives at `<repo>/specs/GUARD-001-memory-sink-precommit/proposal.md`. Keystone of the cross-agent single-sink memory convention (decided 2026-06-16, D1–D4).

## Why

<!-- from issue #398: GUARD: global pre-commit blocking memory artifacts in non-vault repos (cross-agent single-sink) -->

The single-sink convention says agent memory lives **only** in the vault — never in a code repo or machine-local dir. But nothing enforces it. Every guard today is either vault-scoped (gitleaks on the vault) or per-repo opt-in: `dotf init` scaffolds a `.pre-commit-config.yaml`, so the guard exists only in repos that went through the scaffolder. The leak that motivated this (a non-Claude agent committed `MEMORY.md` into the ts-bridge repo root, recovered via ts-bridge#147) happened precisely in a repo that **never ran `dotf init`** — so no local guard could have caught it. Until a machine-wide, agent-agnostic guard exists, the convention is a documented wish, not an invariant, and memory will keep leaking out of the sink.

## What

A machine-wide git guard that makes it impossible to commit memory artifacts to any repository that is not the vault.

- **Global `pre-commit` dispatcher** installed via `core.hooksPath` (machine-wide `git config --global`). On every commit in every repo: detect whether the repo is the vault; if not, reject the commit when staged paths include memory artifacts (`MEMORY.md`, a `memory/` directory, or `00_meta/sessions/` records), with a message pointing the author to the vault as the only sink. Then **chain** to the repo-local hook so existing per-repo guards (gitleaks) still run.
- **`dotf init` gitignore block**: every newly-scaffolded repo is born convention-correct with `MEMORY.md` and `memory/` ignored (defence in depth — stops accidental `git add`).
- **Idempotent install**: setup / `dotf doctor` wires and repairs the global `core.hooksPath` without clobbering an existing one.
- **AGENTS.md** states the single-sink rule once (Hive is the memory *API* over the same sink — no duplicated wording).

After this PR: `git commit` staging `MEMORY.md` in any non-vault repo is rejected; the same commit in the vault succeeds; a repo with a pre-existing gitleaks hook runs **both** the GUARD and gitleaks; new `dotf init` repos already ignore the artifacts.

## Out of scope

- **Per-agent session-end hooks** (`MEMORY-001-mirror`) — those *feed* the sink; this *guards* it. Sibling task, separate spec.
- **The obsidian-git-bypasses-gitleaks vault issue** (knowledge#114) — that is vault-internal (the vault is the *allowed* repo); this guard is about *non-vault* repos. Different threat surface.
- **Server-side / CI enforcement** — the true backstop for `--no-verify` and libgit2/isomorphic-git bypass. Noted as a follow-up; local hooks are the first line, not the only line.
- **Recovering already-leaked memory** — ts-bridge / iris / nan recoveries are done (34/34 symlinked).

## Risks / open questions

- **R1 — `core.hooksPath` vs. the per-repo `pre-commit` framework (load-bearing). ✅ DECIDED (2026-06-16): Option A — multi-hook chaining dispatcher.** A global `core.hooksPath` makes *every* repo resolve **all** hook types to the global dir, disabling per-repo `.git/hooks/` machine-wide — and `pre-commit install` respects `core.hooksPath`, so it would write into the global dir and clobber the per-repo gitleaks guard. **Resolution:** declare the global `core.hooksPath` in dotfiles `.gitconfig` (IaC) pointing at a tracked `git-hooks/` dir holding **one dispatcher per hook type** the ecosystem uses (`pre-commit`, `commit-msg`, `prepare-commit-msg`, `pre-push`). Each dispatcher runs its global concern (only `pre-commit` carries the memory guard) and then execs the **literal** `<toplevel>/.git/hooks/<type>` if present. Mechanical note for implementation: because the framework's `pre-commit install` would target the global dir, the per-repo layer is installed as a stable `.git/hooks/<type>` that calls `pre-commit run --hook-stage <stage>` — the dispatcher chains to that literal path, not to the `core.hooksPath`-resolved one. So `install-precommit.sh` / `dotf init` write the local hook to `.git/hooks/` directly rather than relying on `pre-commit install`'s location.
- **R2 — vault detection.** How the hook knows "this is the vault" (to allow): (a) repo root == `$VAULT_PATH` (`$HOME/Projects/knowledge`) — simple but path-coupled; (b) a sentinel at repo root (`.obsidian/` dir or a `.vault-root` marker) — cross-machine robust; (c) remote-URL match. **Recommendation:** sentinel (`.obsidian/` present at repo root), with `$VAULT_PATH` as fallback — robust across machines, no hardcoded path in the hot path.
- **R3 — bypass paths.** `--no-verify`, libgit2/isomorphic-git tools (obsidian-git), and fresh clones before the hook is wired all skip a local hook. **Mitigation:** gitignore as defence in depth + a documented CI backstop (follow-up); a deliberate `--no-verify` is outside the threat model (the convention is for honest agents, not adversaries).
- **R4 — false positives.** A repo that legitimately needs a file named `MEMORY.md`. **Mitigation:** the match is narrow (root-level `MEMORY.md`, `memory/` dirs, session-record shape) and `--no-verify` is the documented escape hatch with user approval.
- **R5 — cross-OS.** `core.hooksPath` + a POSIX hook script work on both Linux and Git-for-Windows (git invokes the hook via its bundled `sh`). The gitignore + `dotf init` parts are already cross-OS Go. No `.ps1` mirror needed for the hook itself; assert parity in tests.

## Acceptance criteria

- [ ] **AC1** — Committing `MEMORY.md` (or a `memory/` path) to a non-vault repo is rejected (exit ≠ 0) with a message naming the vault. *Verify:* bats fixture repo.
- [ ] **AC2** — Committing the same memory artifact inside the vault is allowed (exit 0). *Verify:* bats fixture with the vault sentinel.
- [ ] **AC3** — In a repo with a pre-existing local pre-commit hook (e.g. gitleaks), both the GUARD and the local hook run; neither is clobbered. *Verify:* bats chaining fixture.
- [ ] **AC4** — `dotf init` writes the `MEMORY.md` / `memory/` gitignore block. *Verify:* Go test on scaffold output.
- [ ] **AC5** — The global hook installs idempotently (setup / `dotf doctor`); re-running is a no-op and never clobbers an unrelated existing `core.hooksPath`. *Verify:* bats idempotence.
- [ ] **AC6** — AGENTS.md states the single-sink rule once (no Hive-note duplication). *Verify:* grep assertion.

## References

- Issue: dotfiles#398 (work-gate)
- Sibling specs: `MEMORY-001-cross-agent-session-bridge` (feeds the sink, Claude core shipped), `MEMORY-001-mirror` (per-agent hooks, draft)
- Vault decisions: single-sink D1–D4 (handoff 2026-06-16); ts-bridge recovery (ts-bridge#147)
- Related: knowledge#114 (obsidian-git bypasses gitleaks — vault-internal, distinct surface)
- Existing infra (build on, do not duplicate): `scripts/install-precommit.sh`, `cli/internal/initrepo/` (gitleaks `.pre-commit-config.yaml` scaffold), `scripts/claude-session-start.sh` (memory symlink — to be extracted, sibling task)
