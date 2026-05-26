---
tags: [spec, tasks, ideas-005]
created: "2026-05-25"
---

# Tasks - IDEAS-005-curl-bootstrap

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [ ] Branch created from main: `feat/IDEAS-005-curl-bootstrap`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] R5 resolved: `install.sh` is the canonical entry; `setup-linux.sh` stays callable
- [ ] Confirm `DOTFILES_REPO` URL (mlorentedev/dotfiles or whatever the user's actual GH path is)

## Implementation

> TDD order. Bats first (mocked setup), then real install.sh, then README + CI.

- [ ] Write failing bats `tests/install-bootstrap.bats` #1: `DOTFILES_DIR=<tmpdir> DOTFILES_SKIP_SETUP=1 bash install.sh` → tmpdir is a git repo (clone happened).
- [ ] Implement `install.sh` skeleton with prereq check + clone-or-update + delegate. #1 passes.
- [ ] Test #2: pre-existing git repo at `$DOTFILES_DIR` → pull executed, no re-clone.
- [ ] Test #3: pre-existing NON-git dir at `$DOTFILES_DIR` → exit 1 with clear error.
- [ ] Test #4: missing `git` (mocked) → exit 1 with clear error.
- [ ] Test #5: `DOTFILES_REPO=https://github.com/fork/dotfiles.git` honored for forks.
- [ ] `chmod +x install.sh`, verify shebang `#!/usr/bin/env bash`.
- [ ] Shellcheck clean.
- [ ] README diff: add "Quick install" section with the curl one-liner + `curl ... | less` inspection caveat + `DOTFILES_DIR` / `DOTFILES_REPO` env var docs.
- [ ] Smoke test: run the actual one-liner in a fresh Docker container or VM (manual or CI-gated).

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] features.json contains a row per criterion
- [ ] Lint: `shellcheck install.sh` exits 0
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

Drop the following into `<repo>/specs/IDEAS-005-curl-bootstrap/features.json`:

```json
[
  {
    "id": "IDEAS-005-curl-bootstrap-f1",
    "behavior": "install.sh clones to DOTFILES_DIR when absent",
    "verification": "bats tests/install-bootstrap.bats --filter 'clone'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-005-curl-bootstrap-f2",
    "behavior": "install.sh pulls when DOTFILES_DIR/.git exists",
    "verification": "bats tests/install-bootstrap.bats --filter 'pull'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-005-curl-bootstrap-f3",
    "behavior": "install.sh fails fast on non-git pre-existing dir",
    "verification": "bats tests/install-bootstrap.bats --filter 'non-git collision'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-005-curl-bootstrap-f4",
    "behavior": "install.sh fails fast when git missing",
    "verification": "bats tests/install-bootstrap.bats --filter 'no git'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-005-curl-bootstrap-f5",
    "behavior": "Shellcheck passes on install.sh",
    "verification": "shellcheck install.sh",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-005-curl-bootstrap-f6",
    "behavior": "README documents one-liner + verify-before-pipe",
    "verification": "grep -q 'curl -fsSL.*install.sh' README.md && grep -q 'verify' README.md",
    "state": "pending",
    "evidence": ""
  }
]
```
