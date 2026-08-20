---
id: "REFACTOR-006-setup-phase-decompose"
type: spec
status: abandoned # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
review: waived
review_waived_reason: "Abandoned in owner triage 2026-08-20. Never started: setup-linux.sh has zero phase functions and the tasks.md still holds the untouched template placeholders. Cold since 2026-06-10."
---

# REFACTOR-006-setup-phase-decompose

> **Naming**: file lives at `<repo>/specs/REFACTOR-006-setup-phase-decompose/proposal.md`. `REFACTOR-006-setup-phase-decompose` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: Split `setup-{linux.sh,windows.ps1}` (~2.6k LOC combined) into numbered phase modules (`scripts/setup/01-prereqs.sh`, `02-dirs.sh`, ..., `07-post.sh`) fmontes-style. Pure structural; behavior byte-identical. Effort: L. Anti-scope: no behavior change, no new flags, public entrypoint unchanged. *Note: prior research/dotfiles-survey rejected holman-topical-discovery for setup but did NOT evaluate fmontes numbered-phase decomposition.* -->

`setup-linux.sh` is 1175 LOC and `setup-windows.ps1` is 1478 LOC. Both blow past the AGENTS.md class-length threshold (<250 lines) by ~5x at file-as-module level. Every BUG-XXX in the setup scripts (BUG-005, BUG-020, BUG-021, BUG-022, ...) pays a tax: locating the right line, understanding cross-phase ordering, modifying without breaking earlier phases. Prior `research/dotfiles-survey.md` considered holman-style topical discovery and rejected it (correct call — find-glob is fragile cross-OS). But fmontes-style **numbered phase modules** (`01-prereqs`, `02-dirs`, ..., `07-post-install`) are explicit, ordered, no glob — and were never evaluated.

## What

Refactor both setup scripts so they become thin entrypoints (~50 LOC) that call numbered phase modules in order. Proposed phase boundaries:

- `01-prereqs.{sh,ps1}` — tool presence checks (git, age, pwsh, gh).
- `02-dirs.{sh,ps1}` — create `~/.dotfiles`, `~/.zsh`, `~/.claude`, `~/.gemini`, etc.
- `03-deploy-shellrc.{sh,ps1}` — `.zshrc`, `.bashrc`, `profile.ps1`, ssh config.
- `04-ai-configs.{sh,ps1}` — Claude settings merge, opencode.jsonc, agy, gemini.
- `05-secrets.{sh,ps1}` — age install hint, secrets directory, `load-secrets` wiring.
- `06-tools.{sh,ps1}` — opportunistic installs (Ollama, Antigravity, mise).
- `07-post-install.{sh,ps1}` — healthcheck, drift baseline, log summary.

Each phase is a separate file under `scripts/setup/`. The entrypoints `setup-linux.sh` and `setup-windows.ps1` keep their names (public contract) and become orchestrators (~50 LOC each). Behavior must be byte-identical to current (verified by healthcheck + drift baseline against `main`).

## Out of scope

- **Any new behavior, flag, or default** — pure structural movement.
- **Changing the public entrypoint names** — `setup-linux.sh` and `setup-windows.ps1` stay.
- **Re-architecting `utils.{sh,ps1}` sourcing** — still dot-sourced from each phase.
- **Topical/discovery glob** — explicitly rejected (see research/dotfiles-survey.md).
- **Splitting `load-secrets.sh`** (1058 LOC) — a separate REFACTOR ticket.
- **Profile system (DX-001)** — orthogonal; that adds `-Profile minimal|standard|full`, this just decomposes the file.

## Risks / open questions

- **R1**: Phase boundaries are subjective. Two reviewers may disagree on whether opencode init goes in `04-ai-configs` or `06-tools`. Mitigation: phases NAMED for what they DO, not for files they touch. Document the boundary rationale in `scripts/setup/README.md`.
- **R2**: Cross-phase env vars (e.g., `DOTFILES_DIR`, `ScriptsDir`) must be exported by `01-prereqs` to be visible downstream. Each phase becomes a sourced function scope, NOT a subshell. PowerShell side: dot-source `.` not `&` invocation.
- **R3**: Regression risk is real. Validation strategy: run healthcheck on a clean VM before AND after; byte-compare deployed `.zshrc`/`.bashrc`/`.gitconfig`/Claude settings. Use existing drift detector. Block merge on any diff.
- **R4**: Bats tests reference setup-linux.sh / setup-windows.ps1 by path. Some tests may need updates if they `grep` for inline content that moved into a phase module.
- **R5**: Sequenced PR vs. single PR. L scope (~2.6k LOC of movement). Decompose as Linux first (1175 LOC) then Windows (1478 LOC) in two atomic PRs to halve review burden.

## Acceptance criteria

- [ ] `setup-linux.sh` ≤ 100 LOC after refactor.
- [ ] `setup-windows.ps1` ≤ 100 LOC after refactor.
- [ ] All phase modules exist under `scripts/setup/NN-*.{sh,ps1}`.
- [ ] `scripts/setup/README.md` documents phase boundary rationale.
- [ ] Healthcheck on a clean Linux VM passes pre- and post-refactor with zero diff in deployed files.
- [ ] Healthcheck on a clean Windows VM passes pre- and post-refactor with zero diff in deployed files.
- [ ] All existing bats tests pass; updated paths where references moved.
- [ ] `git diff main -- setup-linux.sh setup-windows.ps1 scripts/setup/` shows pure structural movement, no new logic.
- [ ] PR splits into two atomic PRs (Linux refactor, Windows refactor) — each independently reviewable.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → REFACTOR-006.
- Prior survey rejection of topical-discovery: `research/dotfiles-survey.md` § "Top 6 ideas a aplicar" #6.
- Inspiration: fmontes/dotfiles `01-xcode.sh`, ..., `07-post-install.sh`.
- AGENTS.md class-length rule: `AGENTS.md:222` (<250 lines).
