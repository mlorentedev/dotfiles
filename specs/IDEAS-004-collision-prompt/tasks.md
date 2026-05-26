---
tags: [spec, tasks, ideas-004]
created: "2026-05-25"
---

# Tasks - IDEAS-004-collision-prompt

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [ ] Branch created from main: `feat/IDEAS-004-collision-prompt`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] R5 resolved: default-under-force is `backup` (or whatever the final decision is) — recorded in proposal.md
- [ ] No remaining open questions in "Risks / open questions"

## Implementation

> TDD order. Test stdin-mocking first (proves the contract), then integration, then CI guards.

- [ ] Write failing bats `tests/collision-prompt.bats` #1: `prompt_collision <dst> <src>` with stdin `s\n` returns `skip` and leaves destination untouched.
- [ ] Implement skeleton `prompt_collision()` in `scripts/utils.sh`. #1 passes.
- [ ] Add tests #2-#6 for `S`, `o`, `O`, `b`, `B` paths. Implement the state machine + `__DOTFILES_COLLISION_MODE` global.
- [ ] Test #7: backup action creates file at `<dst>.bak.<timestamp>` with original content preserved; original gets replaced with symlink to source.
- [ ] Test #8: timestamp collision protection — back-to-back invocations produce two distinct backup files (use `$$` or nanoseconds).
- [ ] Test #9: `DOTFILES_SETUP_FORCE=1` short-circuits the prompt and applies the chosen default action.
- [ ] Refactor `link_file()` (or equivalent in `setup-linux.sh`) to call `prompt_collision()` on regular-file destinations.
- [ ] Integration test: run `setup-linux.sh` in a sandbox with a pre-existing `~/.zshrc` regular file + `DOTFILES_SETUP_FORCE=1` → assert backup created, symlink in place.
- [ ] Cross-shell test: same suite under bash and zsh matrix.
- [ ] Update `.github/workflows/*.yml` to export `DOTFILES_SETUP_FORCE=1` before any `setup-linux.sh` invocation.
- [ ] Update README "First-time setup" section explaining the prompt + force-mode env var.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] features.json contains a row per criterion
- [ ] Lint: `shellcheck scripts/utils.sh` exits 0
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

Drop the following into `<repo>/specs/IDEAS-004-collision-prompt/features.json`:

```json
[
  {
    "id": "IDEAS-004-collision-prompt-f1",
    "behavior": "prompt_collision honors all 6 input paths",
    "verification": "bats tests/collision-prompt.bats --filter 'six paths'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-004-collision-prompt-f2",
    "behavior": "DOTFILES_SETUP_FORCE=1 skips prompt and applies default action",
    "verification": "bats tests/collision-prompt.bats --filter 'force mode'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-004-collision-prompt-f3",
    "behavior": "backup files use collision-resistant timestamp",
    "verification": "bats tests/collision-prompt.bats --filter 'timestamp collision'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-004-collision-prompt-f4",
    "behavior": "link_file integration: collision triggers prompt path",
    "verification": "bats tests/collision-prompt.bats --filter 'integration'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-004-collision-prompt-f5",
    "behavior": "Cross-shell: bash and zsh both pass the suite",
    "verification": "BATS_MATRIX_SHELL=bash bats tests/collision-prompt.bats && BATS_MATRIX_SHELL=zsh bats tests/collision-prompt.bats",
    "state": "pending",
    "evidence": ""
  }
]
```
