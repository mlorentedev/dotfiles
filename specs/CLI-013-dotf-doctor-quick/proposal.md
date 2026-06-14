---
id: "CLI-013-dotf-doctor-quick"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-14"
issue: "dotfiles#380"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-013-dotf-doctor-quick

> **Naming**: file lives at `<repo>/specs/CLI-013-dotf-doctor-quick/proposal.md`. `CLI-013-dotf-doctor-quick` is `AREA-NNN-slug`.

## Why

<!-- from issue #380: CLI-012 follow-ups: Windows dotf doctor wiring + dotf doctor --quick for the SessionStart hook -->

CLI-012 (#376) retired the per-session env-contract drift check from `scripts/claude-session-start.sh`: the full `dotf doctor` sweep is ~2.8s (dominated by the `compile-harness.sh --check` re-render) — too heavy to fork on every Claude session start, and a PATH-command call also broke the hook's hermetic test isolation. That removed a real capability (Claude was told, at session start, when the env-contract had drifted). This is the debt-paydown half of #380: a focused `dotf doctor --quick` that runs only the env-contract sweep, fast enough to wire back into the hook.

## What

A `--quick` flag on `dotf doctor` that runs **only** the env-contract sections (environment variables, PATH entries, required binaries) — the old `doctor.sh` scope — and skips the heavy healthcheck sweep (core/versioned/symlinks/vault/secrets/tmux/opencode/**harness-drift**/antigravity). Same exit contract (0 = pass, 1 = any FAIL) over the reduced check set. Sub-100ms, no `compile-harness` fork.

`scripts/claude-session-start.sh` is re-wired to surface drift via `dotf doctor --quick`, guarded so it stays a no-op in the hermetic test: it runs only when `dotf` is on PATH **and** a deployed `env-contract.json` exists under `DOTFILES_DIR` (the test's isolated `$HOME` has neither the contract nor the file, so the block skips — preserving `session-start-false-positives.bats`).

## Out of scope

- The Windows wiring of `dotf doctor` (#380 item 1) — gated on a Windows `install-dotf`.
- Any change to which checks the full (non-`--quick`) sweep runs.
- A `--fix` combined with `--quick` heal flow — `--quick` is report-only (the hook never mutates).

## Risks / open questions

- **Hermetic test breakage.** Repointing the hook to a PATH command (`dotf`) risks running the real sweep inside `session-start-false-positives.bats`. Mitigation: gate on the deployed `env-contract.json` existing, which the isolated test `$HOME` lacks → the block skips, exactly as the old sibling-file-absent guard did.
- **Output noise.** `--quick` still prints its sections; the hook greps only `[WARN]`/`[FAIL]` lines (as `doctor.sh` did), so a clean run surfaces nothing to Claude.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `dotf doctor --quick` runs only the contract sections (env vars, PATH, required binaries); the harness-drift / version / vault / secrets / tmux / opencode / antigravity sections do NOT run.
- [ ] `dotf doctor --quick` keeps the exit contract: 0 when the contract checks pass, 1 on any contract FAIL.
- [ ] `dotf doctor --quick` does not fork `compile-harness.sh` (runs in well under the full sweep's ~2.8s).
- [ ] `scripts/claude-session-start.sh` surfaces env-contract drift to Claude via `dotf doctor --quick`, gated so `session-start-false-positives.bats` stays green (the block skips when no deployed contract is present).
- [ ] `go test ./...` covers the quick-mode section selection.

## References

- GitHub issue: `dotfiles#380` (work-gate)
- Predecessor: CLI-012 (`specs/archive/CLI-012-dotf-doctor/`), which removed the per-session check
- Roadmap: `docs/adr/adr-021-cli-orchestration-roadmap.md` (SessionStart hook port = step 6)
