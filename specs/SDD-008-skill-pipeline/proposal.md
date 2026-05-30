---
id: "SDD-008-skill-pipeline"
type: spec
status: draft
created: "2026-05-26"
tags: [spec, proposal, sdd-008, cross-os, ai-tooling, skill-distribution, pipeline, bug-100-followup]
template_version: "1.0"
---

# SDD-008: Skill pipeline

> **Naming**: file lives at `<repo>/specs/SDD-008-skill-pipeline/proposal.md`.

## Why

<!-- from 10_projects/dotfiles/11-tasks.md:193: SDD-008-skill-pipeline (Tier 1, cross-OS, ~250 LOC est.) — Compile-once-deploy-everywhere pattern for cross-agent skill distribution; eliminates symlink/junction fragility (BUG-100 precedent class). -->

Today skills are distributed via two inconsistent mechanisms: symlink/junction for vault-hosted SDD skills (`spec`, `adversarial-review`, `enrich-us`) and mechanical copy via script for opencode commands (per AI-012). BUG-100 already proved that cross-agent symlink fragility is a real bug class — agy v1.0.2 broke because our deploy strategy was "fighting the agent's expected filesystem layout" (per SDD-007 proposal). Without a unified pipeline, every new agent (Cursor, Codex, Devin, future tooling) repeats the deploy decision ad-hoc, and any improvement to a skill's content (e.g., extending `/spec` with an agent-side proactive trigger) does not propagate deterministically to the N target agents — behavior diverges silently across agents and the SSOT promise breaks without anyone noticing until the next BUG-100-class incident.

## Engine (post-ENGINE-001 reconciliation, 2026-05-29)

> This spec predates **ENGINE-001** (PR #172, merged 2026-05-29). SDD-008 is now a **consumer of that engine**, not a parallel pipeline. Wherever this document says `scripts/skills/render-all.sh`, read it as **`scripts/compile-harness.sh` extended with a new manifest target `kind: render`** — `--refresh` to render, `--check` for offline drift. Concretely:
>
> - **Skills are a new target kind.** ENGINE-001 ships `kind: native|pointer` (marker-region injection into one hand-authored file). Skills need **`kind: render`**: a whole-file/dir transform taking one vault `00_meta/skills/<name>/SKILL.md` → N agent-native whole files at agent-specific paths. Same `harness/manifest.json`, same `--refresh`/`--check` surface, same `healthcheck` drift guard. One engine, two deploy modes.
> - **Committed source-of-record resolves the CI-blindness risk.** ENGINE-001 commits its rendered output so `--check` runs fully offline; applying the same here lets dotfiles CI verify skill drift with **no vault access** — superseding this spec's original "rely on the Phase 2.6 idempotence gate" stance (see Risks, now RESOLVED).
> - **Vault SSOT + compile-commit, not symlink** — unchanged; already the engine's directionality invariant (vault upstream → committed repo artifacts → deploy as copy).

## What

After this PR, the system exhibits five observable behavior changes:

1. **Single SSOT for ALL skills**: `vault/00_meta/skills/<name>/SKILL.md` is the only source of truth for every skill (~20 total: 3 SDD already there + 17 migrated from `dotfiles/ai/skills/`). After this PR, `dotfiles/ai/skills/` either does not exist or contains only a stub `README.md` pointing to the vault. `git log --follow` preserves history because migration uses `git mv` per skill.
2. **Single render command**: `scripts/skills/render-all.sh` reads every `vault/00_meta/skills/<name>/SKILL.md` and produces N agent-native outputs at the expected paths (`~/.claude/skills/<name>/SKILL.md`, `~/.config/opencode/commands/<name>.md`, `~/.gemini/skills/<name>/<name>.md`, and a generated section in `.github/copilot-instructions.md`). Every generated file carries a header `# GENERATED — DO NOT EDIT — source: <vault-path>` plus a checksum of the source.
3. **Symlinks/junctions eliminated**: after running `setup-linux.sh` or `setup-windows.ps1`, no `~/.claude/skills/<name>` is a symlink (Linux: `find ~/.claude/skills -type l` returns empty; Windows: equivalent junction probe returns empty). All deployed skill files are regular file copies.
4. **Drift detection via setup-time render + existing idempotence gate**: `setup-{linux,windows}` invoke `scripts/skills/render-all.sh` as part of their flow, so the build IS setup. No dedicated CI workflow with vault access required (the dotfiles CI has zero visibility into the private `github.com/mlorentedev/knowledge` repo, and we deliberately don't add it). Drift is caught by the planned Phase 2.6 "run-twice-and-diff" idempotence gate (per `vault/10_projects/dotfiles/11-tasks.md`): if a developer edits a deployed skill manually OR forgets to re-render after editing vault source, the second setup run produces a diff and CI fails. One gate covers two bug classes.
5. **Agent-side proactive `/spec` trigger** (sub-task in same PR): `vault/00_meta/skills/spec/SKILL.md` declares an explicit "Agent-Side Activation Rule" section. When the agent identifies a non-trivial change in conversation, it applies the Skip-SDD heuristic *itself* and proposes `/spec init` proactively, listing the checks it ran. Observable as agent-initiated spec proposals appearing in transcripts where previously they were user-initiated.
6. **Schema-validated skill frontmatter**: `render-all.sh` runs a frontmatter validator first; any SKILL.md missing required fields (`id`, `type: skill`, `description`, `allowed-tools`) or with malformed YAML fails the render with a clear file:line error. Prevents silent deploy of malformed skills that break agent discovery. Schema lives at `vault/00_meta/templates/skill-frontmatter.schema.json`.
7. **Per-skill agent opt-out via manifest**: each skill MAY declare `targets: [claude, opencode, agy, copilot]` in its frontmatter (default = all four). Skills that only make sense for one agent (e.g., `claude-mem-*` family is Claude-only) ship only to declared targets. The render dispatcher honors the manifest; outputs for non-targeted agents are never written.
8. **Atomic migration with rollback**: the 17-skill repatriation from `dotfiles/ai/skills/` → `vault/00_meta/skills/` runs as a single script (`scripts/skills/migrate-to-vault.sh`) that is transactional — if any of the 17 moves fails (permission, conflict, vault unwritable), the script restores all prior moves from a pre-migration snapshot and exits non-zero. No half-migrated state possible.
9. **Post-render smoke test**: `tests/skills-pipeline.bats` includes a smoke test that invokes a known skill (`/spec`) from the deployed location and asserts non-error exit; equivalent test for opencode (`opencode run -m <model> /spec --help` or similar). Validates the full round-trip vault → render → deploy → agent invocation, not just the file generation step.

## Out of scope

Per user directive (Q3 answer): the PR MUST repatriate and migrate ALL skills to the single SSOT — partial migration is explicitly rejected. The items below are what falls outside that complete-migration scope.

- **Adapters for agents beyond Claude / OpenCode / agy / Copilot** — adding Cursor, Codex, Devin, or other future agents is one new `render-<agent>.sh` script each; deferred to per-agent PRs as those agents enter the workflow.
- **Empirical validation of the agent-side proactive trigger behavior in production sessions** — the rule ships as content in `vault/00_meta/skills/spec/SKILL.md`, but observing whether agents actually apply Skip-SDD heuristic correctly across N sessions is observational work tracked separately.
- **Migration of vault commit-and-sync mechanics** — this PR assumes the existing `obsidian-git` auto-commit on the vault is sufficient for SSOT durability. Tightening vault commit cadence, adding pre-commit hooks on the vault, or building a vault-side CI is out of scope.
- **Cross-OS CI matrix** for skills pipeline — this PR exercises Linux runner only. Windows-runner validation is tracked separately under Phase 2.6 (idempotence GHA, planned). Adding a Windows matrix purely for skills here would duplicate effort.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- **🟢 RESOLVED 2026-05-26 — Bootstrap dependency, option (b) chosen**: `setup-{linux,windows}` performs preflight check for BOTH prerequisites before invoking `render-all.sh`:
  1. **Vault clone present**: `~/Projects/knowledge` (Linux/macOS) or `%USERPROFILE%\Projects\knowledge` (Windows) exists with `00_meta/skills/` populated (test: `[ -d ~/Projects/knowledge/00_meta/skills ]` / `Test-Path "$env:USERPROFILE\Projects\knowledge\00_meta\skills"`).
  2. **Obsidian binary installed**: detect via `command -v obsidian` (Linux/macOS) / `Get-Command obsidian -ErrorAction SilentlyContinue` OR Application bundle probe (Windows). Reason: vault editing/sync requires Obsidian GUI; setup already checks for it in `healthcheck` section 7/7 — keep that contract.

  If either missing, setup aborts with **exit code 2** and prints an actionable error block:
  ```
  [FATAL] Skill pipeline requires Obsidian + vault clone.
    → Install Obsidian: https://obsidian.md
    → Clone vault:      git clone <vault-repo> ~/Projects/knowledge
    → Re-run:           scripts/setup-{linux,windows}.{sh,ps1}
  ```

  Rationale: vault is **private repo** (cannot vendor into public dotfiles → option c rejected). Option (a) `install.sh` is not yet shipped (IDEAS-005 still backlog → coupling deferred). New-machine bootstrap is rare; explicit 2-step is acceptable cost. Tracked: `vault/10_projects/dotfiles/11-tasks.md § Cross-Provider Harness Wave 2`.
- **🟢 RESOLVED 2026-05-29 — Phase 2.6 coupling no longer needed**: superseded by ENGINE-001's committed source-of-record. Drift is detected by `compile-harness.sh --check` (offline, no vault) wired into `scripts/healthcheck.sh` exactly as the no-attribution blocks are (ADR-013). The dotfiles CI does not need vault access and does not need the run-twice idempotence GHA for *this* gate — `--check` compares committed rendered outputs against the committed source-of-record. AC3 below is the drift gate.
- **🟡 Migration history preservation across repos**: 17 cross-repo moves from `dotfiles/ai/skills/<name>/` to `vault/00_meta/skills/<name>/`. `git mv` does not preserve history across separate repos. Options: (a) accept history loss + add reference table in each skill README pointing to pre-migration dotfiles commit hash, (b) `git filter-repo` extraction + injection (technically complex; can corrupt repos if mishandled), (c) migration script that copies content + commits each with a `Co-authored-by`-style reference to the original dotfiles hash. Choose at implementation time. Non-blocker for spec authoring.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1 — Zero symlinks/junctions in deployed skill paths**: after `setup-linux.sh`, `find ~/.claude/skills ~/.gemini/skills ~/.config/opencode/commands -type l` returns empty. After `setup-windows.ps1`, equivalent PowerShell junction probe returns empty. **Verification**: `tests/skills-pipeline.bats` + `tests/skills-pipeline.Tests.ps1`.
- [ ] **AC2 — Single SSOT post-migration**: `vault/00_meta/skills/` contains exactly the migrated set (3 original SDD + 17 repatriated = 20 skill directories, each with `SKILL.md`); `dotfiles/ai/skills/` either does not exist or contains only `README.md` pointing to vault. **Verification**: structural test counting directories + grep for SKILL.md in vault, asserting absence/stub in repo.
- [ ] **AC3 — Drift detection via `--check` mode**: running `scripts/skills/render-all.sh --check` against a clean working tree post-setup exits 0; the same command after editing any `vault/00_meta/skills/<name>/SKILL.md` without re-rendering exits non-zero with a diff summary pointing to the stale outputs. **Verification**: bats fixture test mutating a sample SKILL.md and asserting exit code transition.
- [ ] **AC4 — Generated-file provenance header**: every output file produced by `render-all.sh` contains a header `# GENERATED — DO NOT EDIT — source: <vault-relative-path> — sha256: <16-char-prefix>` as the first non-shebang line. Editing the source updates the sha; editing the output without re-rendering creates a mismatch detectable by `--check`. **Verification**: grep + sha256sum compare across all outputs in a bats test.
- [ ] **AC5 — Frontmatter schema validation**: corrupting any SKILL.md frontmatter (delete `id`, malformed YAML, unknown `type` value) causes `render-all.sh` to fail with file:line error pointing to the offending field. Valid frontmatter renders silently. **Verification**: bats fixture mutates a skill copy, asserts non-zero exit + error message format. Schema file `vault/00_meta/templates/skill-frontmatter.schema.json` exists and is referenced from `pattern-cross-agent-skill-pipeline.md`.
- [ ] **AC6 — Per-skill target manifest honored**: a skill with `targets: [claude]` in frontmatter renders ONLY to `~/.claude/skills/<name>/SKILL.md`; no output appears at opencode/agy/copilot paths. A skill without `targets:` defaults to all four. **Verification**: fixture skill with explicit `targets:`, post-render `ls` assertion on all four paths.
- [ ] **AC7 — Pattern documentation promoted to vault**: `vault/00_meta/patterns/pattern-cross-agent-skill-pipeline.md` exists, links bidirectionally with `pattern-spec-driven-development.md` (this pipeline serves SDD discipline), documents the render dispatch + manifest + schema + drift detection design. **Verification**: file exists check + grep for cross-reference link.
- [ ] **AC8 — End-to-end smoke test post-render**: after `setup-linux.sh` completes, invoking `/spec init` in a Claude session (or `claude --help` listing skills) shows `spec` available; equivalent assertion for OpenCode (`opencode --help` lists `spec` command); agy / copilot smoke depends on their CLI surfaces. Validates that the full vault→render→deploy→discovery chain works, not just file presence. **Verification**: bats test wrapping CLI invocations; failures point to which agent's discovery broke.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (backlog entry, line 193)
- Related ADR: `10_projects/dotfiles/30-architecture/adr-009-multi-agent-runtime.md`, `adr-010-agent-harness-parity.md`
- Related patterns: `00_meta/patterns/pattern-spec-driven-development.md` (this spec extends it), to-be-created `00_meta/patterns/pattern-cross-agent-skill-pipeline.md`
- Precedent: SDD-007 / BUG-100 (symlink fragility with agy), AI-012 (skills→opencode mechanical port — partial precedent for this pattern)
