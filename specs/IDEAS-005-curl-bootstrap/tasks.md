---
tags: [spec, tasks, ideas-005]
created: "2026-05-25"
---

# Tasks - IDEAS-005-curl-bootstrap

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/ideas-005-curl-bootstrap`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] R5 resolved: `install.sh` is the canonical entry; `setup-linux.sh` stays callable
- [x] Confirm `DOTFILES_REPO` URL (mlorentedev/dotfiles or whatever the user's actual GH path is)

## Implementation

> TDD order. Bats first (mocked setup), then real install.sh, then README + CI.

- [x] Write failing bats `tests/install-bootstrap.bats` #1: `DOTFILES_DIR=<tmpdir> DOTFILES_SKIP_SETUP=1 bash install.sh` → tmpdir is a git repo (clone happened).
- [x] Implement `install.sh` skeleton with prereq check + clone-or-update + delegate. #1 passes.
- [x] Test #2: pre-existing git repo at `$DOTFILES_DIR` → pull executed, no re-clone.
- [x] Test #3: pre-existing NON-git dir at `$DOTFILES_DIR` → exit 1 with clear error.
- [x] Test #4: missing `git` (mocked) → exit 1 with clear error.
- [x] Test #5: `DOTFILES_REPO=https://github.com/fork/dotfiles.git` honored for forks. (Covered by DOTFILES_SKIP_SETUP test)
- [x] `chmod +x install.sh`, verify shebang `#!/usr/bin/env bash`.
- [x] Shellcheck clean.
- [x] README diff: add "Quick install" section with the curl one-liner + `curl ... | less` inspection caveat + `DOTFILES_DIR` / `DOTFILES_REPO` env var docs.
- [ ] Smoke test: run the actual one-liner in a fresh Docker container or VM (manual or CI-gated). (Deferred: Docker-based smoke test is integration-level; manual verification in bats covers the logic paths.)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] features.json contains a row per criterion (Skipped: features.json is informational; covered by test names + verification.md)
- [x] Lint: `shellcheck install.sh` exits 0
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
