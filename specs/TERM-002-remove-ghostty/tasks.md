---
tags: [spec, tasks]
created: "2026-06-10"
---

# Tasks - TERM-002-remove-ghostty

> One task = one focused concern. Ticked as completed.

## Setup

- [x] Branch `refactor/remove-ghostty` (worktree); issue #281 self-assigned
- [x] proposal.md complete

## Implementation

- [x] Remove ghostty from setup-linux.sh, healthcheck.{sh,ps1} (renumber 13 -> 12), tmux.conf, versions.conf
- [x] Delete terminals/, tests/ghostty.bats, docs/runbooks/guide-ghostty-setup.md
- [x] Update tests: healthcheck.bats, healthcheck-ps1.bats (section counts), verify-setup.bats, tmux.bats, powershell-profile.bats
- [x] Clean living docs (README, ai/opencode/README, guide-opencode-go-setup, architecture map) + active specs (AI-021, REFACTOR-006, REFACTOR-010)
- [x] Abandon DX-003 -> specs/archive/_abandoned/
- [x] Companion: yarn pinned install + drift reconcile (setup-{linux,windows}) + healthcheck version match (sh, ps1) + bats

## Closing

- [x] shellcheck clean; full bats suite green
- [x] Zero ghostty refs outside archive/CHANGELOG (grep-verified)
- [x] verification.md filled in
- [ ] PR opened closing #281
