---
id: "HARNESS-070-deploy-convergence"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-12"
issue: "mlorentedev/dotfiles#843"   # anchor issue; also covers #869, #828 (see References)
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-070: Deploy convergence

> **Naming**: file lives at `<repo>/specs/<feature-id>/proposal.md`. `<feature-id>` is `AREA-NNN-slug`.

## Why

<!-- from issue #843: BUG-058: the harness deploy never prunes, so records deleted from the repo keep failing doctor from the mirror -->

Three open issues share one root shape: the harness deploy engine converges forward (repo → mirror → agent) but never converges backward (deleted-in-repo → gone-everywhere-else), and nothing forces the forward direction to actually run. **#843** — deleted `harness/` records survive forever in the `~/.dotfiles` mirror because the repo→mirror copy is copy-only. **#869** — a merged guard can sit uninstalled indefinitely because self-deploy is opt-in and no check reports "the binary you're running predates the guard you think protects you." **#828** — doctrine (AGENTS.md / CLAUDE.md content) reaches `agy`/`codex` through `compile-harness.sh --deploy` but reaches `claude`/`opencode`/`pi`/`copilot` only through a full `setup-linux.sh` run, so a lightweight `--deploy` silently under-serves four of six surfaces. Reproduced live on this machine today: `dotf doctor` shows 6 files drifted repo↔mirror after 9 merged PRs + a release cut, `dotf` itself one version behind its own pin, and (initially misread as a fourth instance of the same class, corrected below) 4 skills reported as symlinks.

## What

1. `dotf doctor` detects orphan records under `harness/{skills,agents}` in the deploy mirror (present in `$DOTFILES_DIR`, absent from the repo) and `dotf doctor --fix` prunes them.
2. `dotf doctor` FAILs (not WARNs) when the installed `dotf` version differs from the `versions.conf` pin — a stale `dotf` means whatever guards shipped after it was built are not running at all, which is a harder failure than an ordinary tool being one version behind.
3. `compile-harness.sh --deploy` copies the full doctrine/instruction files (`ai/claude/CLAUDE.md`, `AGENTS.md`, `ai/copilot/copilot-instructions.md`) to their per-agent `$HOME` paths itself, so a standalone `--deploy` run — without a full `setup-linux.sh` pass — brings all six surfaces (agy, codex, claude, opencode, pi, copilot) current in one command.
4. `dotf doctor` reports (never silently tolerates) a deployed instruction file that has drifted from its repo source.
5. **Correction to the framing this spec was scoped under**: the 4 "symlinked skills" evidence (`computer-use`, `find-skills`, `orca-cli`, `orchestration`) is investigated and found to be a false positive in `checkDeployedSkillSymlinks`, not a BUG-100 regression — those names have no `harness/skills/` record; they are Orca's own `~/.agents/skills/` symlink mechanism, deliberately excluded from the strict symlink sweep by prior art (AI-022 spec, for the `pi` case). The check is narrowed to only flag symlinks at names the harness actually manages, matching the policy `warn_unmanaged_output` already applies on the deploy side.

## Out of scope

- The `scripts/`, `sensitive/`, `.zsh/`, `ssh/`, `secrets/` mirror-orphan surfaces from #802 (36 shell twins, secret-file pruning). #843's fix covers `harness/` only; #802 stays open as the umbrella for the other five surfaces and is referenced, not closed.
- `setup-linux.sh` / `setup-windows.ps1` changes. This session's surface is `scripts/compile-harness.sh`, `harness/`, and the doctor package; the redundant instruction-file copies already in `setup-linux.sh` are left in place (harmless — same content, now also written earlier by `--deploy`).
- `DOTFILES_AUTODEPLOY` disposition (#869 remedy (a)) — a per-machine judgment call for the user, not a code change. Surfaced as an open decision in the PR body.
- bats coverage for the shell-side `--deploy` change (`tests/*.bats` is out of scope for this session per explicit multi-session coordination) — the Go-side doctor checks get Go tests; the shell change is verified manually and by the existing `--check` gate, with a bats follow-up proposed, not filed, in the PR body.

## Risks / open questions

- **Sequencing risk in `--deploy`**: the new instruction-file copy must run before `deploy_skills` (copilot catalog injection) and `deploy_agents` (AGENT-PRESENCE injection) in the same file, or it clobbers what those steps just wrote. Resolved by ordering: copy → skills/catalog → agents/presence → doctrine.
- **Staleness-check false positive**: a deployed instruction file legitimately differs from its repo source by the injected AGENT-PRESENCE region (and, for copilot, the skill-catalog region). The new drift check must strip marked regions from both sides before comparing, or it false-fails immediately after a clean deploy.
- **Windows parity gap**: the new manifest-driven instruction copy is consumed only by `compile-harness.sh` (Linux). `setup-windows.ps1` keeps its own existing copy logic unchanged, so the "single command converges all six surfaces" property is Linux-only for now. Named as a known gap for the Windows-batch session, not fixed here.

## Acceptance criteria

- [ ] AC1: `dotf doctor` reports a `harness/{skills,agents}` record present in the mirror but absent from the repo as a FAIL; `dotf doctor --fix` removes it and is idempotent (#843).
- [ ] AC2: `dotf doctor` reports `dotf` version drift against `versions.conf` as a FAIL, not a WARN (#869 remedy b).
- [ ] AC3: running `scripts/compile-harness.sh --deploy` alone (no `setup-linux.sh`) updates `~/.claude/CLAUDE.md`, `~/.config/opencode/AGENTS.md`, `~/.pi/agent/AGENTS.md`, and (when `copilot` is on PATH) `~/.copilot/copilot-instructions.md` to match the current repo source, and injects AGENT-PRESENCE/catalog regions afterward without clobbering the copy (#828).
- [ ] AC4: `dotf doctor` reports a deployed instruction file that has drifted from its repo source (region-stripped comparison), never silently (#828 AC2).
- [ ] AC5: `checkDeployedSkillSymlinks` no longer FAILs on a symlink whose name has no `harness/skills/` record (Orca/pi-managed foreign skills), while still FAILing on a symlink at a managed name.

## References

- Bitácora board: mlorentedev/dotfiles#843 (BUG-058, anchor), #869 (OPS-025), #828 (HARNESS-058) — all covered by this spec; `Closes` all three in the PR.
- Related, not closed: mlorentedev/dotfiles#802 (BUG-051, the umbrella prune-on-deploy design across all six surfaces).
- Precedent for the symlink-narrowing policy: `specs/archive/AI-022-pi-harness-slot/proposal.md` (deliberate exclusion of `~/.agents/skills`-sourced links), `warn_unmanaged_output` in `scripts/compile-harness.sh`.
- Related ADR: `docs/adr/adr-012-deploy-strategy-copy-with-drift-assertion.md` (copy-with-drift-assertion, BUG-100 origin).
