---
tags: [spec, tasks, terminal, ghostty]
created: "2026-05-18"
---

# Tasks - TERM-001-ghostty-bootstrap

> Retrospective: the implementation shipped piecemeal between 2026-05-17 and the present day across several PRs (PR #38 tmux truecolor passthrough, then the setup + healthcheck + bats + runbook commits on main). This file closes the spec lifecycle by mapping the existing artefacts back to the proposal's TDD plan.

## Setup

- [x] Branch: `chore/TERM-001-close-spec-lifecycle` (housekeeping only — implementation is already on main).
- [x] `proposal.md` complete and AC testable.
- [x] No open questions blocking close-out.

## Implementation (already shipped on main)

- [x] `versions.conf`: `GHOSTTY_VERSION=1.3.0` pinned.
- [x] `terminal/ghostty/config`: 23-line canonical config (Catppuccin Mocha, JetBrainsMono Nerd Font Mono, `confirm-close-surface = true`, `bell-features = no-system`, `window-vsync = true`).
- [x] `setup-linux.sh`: detect-and-act ghostty install (warn-not-fail if missing) + reconcile-not-skip config deploy + version pin check.
- [x] `scripts/healthcheck.sh`: Section 11/12 "Ghostty" with 3 checks (binary present, version matches versions.conf, config deployed with theme + font-family). Section 12/12 added simultaneously for "Repo ↔ Deploy-Dir Drift".
- [x] `tests/ghostty.bats`: 10 assertions (config presence, font-family, theme, confirm-close-surface, kebab-case typo guard, `ghostty +validate-config` smoke, semver pin, version drift check, setup-linux block presence, repo-path canonical reference).
- [x] Vault runbook `40-runbooks/guide-ghostty-setup.md`: GUI-only steps (Nerd Font install, set-as-default), SSH terminfo workaround, recommended workflow alongside tmux.
- [x] tmux truecolor passthrough (PR [#38](https://github.com/mlorentedev/dotfiles/pull/38)): `xterm-ghostty:Tc` override so claude / shell colors render correctly under tmux+ghostty.

## Lifecycle close (this PR)

- [ ] `features.json` written with 10 features mapping to proposal AC.
- [ ] `verification.md` filled with evidence + commit hashes.
- [ ] `proposal.md` frontmatter `status: archived`.
- [ ] Folder moved: `specs/TERM-001-ghostty-bootstrap/` → `specs/archive/TERM-001-ghostty-bootstrap/`.
- [ ] Vault `11-tasks.md` ticked with PR link to this housekeeping PR.

## Lessons

The `chore: close spec lifecycle` pattern is a useful housekeeping shape when a feature ships piecemeal before the spec was formally closed. It is NOT a workaround for the SDD discipline (the proposal was filled BEFORE the work shipped); it is the final step that turns "shipped artefacts" into "archived spec + audit trail".

This pattern is mild scope creep risk — be careful not to introduce new implementation in a "close spec" PR. This PR introduces ZERO production code changes.
