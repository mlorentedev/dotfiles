---
tags: [spec, tasks, templates]
created: "2026-08-07"
---

# Tasks - BUG-049-test-sh-repo-root

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `fix/test-sh-repo-root`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

Diagnosis had to come first here: the symptom (`utils.sh not found`) points at a
missing file, and the actual defect is that the suite was looking in the wrong
tree entirely. The baseline capture in step 1 is what makes AC3 provable.

- [x] [AC3] Capture the Linux baseline: run the suite on untouched `main` in a container, record the exact failing checks
- [x] [AC1] [AC5] Introduce `REPO_DIR` (tree under test) and stop overwriting `DOTFILES_DIR` (deploy environment)
- [x] [AC1] Repoint `SCRIPTS_DIR`, `SENSITIVE_DIR`, the `setup-linux.sh` syntax check and the `gh_get_repo` `cd` at `REPO_DIR`
- [x] [AC2] [AC4] Add `IS_WINDOWS` detection and skip the POSIX-only assertions (symlink semantics, mode bits, `~/.zshrc` and friends) with stated reasons
- [x] [AC5] Convert the two now-honest environment assertions to explicit skips when the environment is un-provisioned
- [x] [AC3] Re-run the Linux container and diff the failing set against the baseline
- [x] [AC2] Confirm the `dotfiles-test` pre-commit hook reports `Passed` on Windows

## Closing

- [x] [AC6] Add `tests/test-sh-resolution.bats` and prove it red against the pre-fix script
- [x] Every acceptance criterion from `proposal.md` is covered by evidence
- [x] Lint passes (`bash -n`, `shellcheck -S warning`)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#799)

## Note on verification shape

The first pass verified this differentially — the same suite on both platforms,
before and after, failing sets compared — on the reasoning that `scripts/test.sh`
*is* the test suite and there is no harness that runs the runner.

That was weaker than the repo's own rule for `.sh` changes, and a review caught it.
`tests/test-sh-resolution.bats` now asserts the resolution contract directly by
running the script under a decoy `DOTFILES_DIR` shaped like the Windows deploy
mirror. It is red on the pre-fix script, so it is a genuine guard rather than a
restatement of the outcome. The differential evidence stays, as the wider
no-regression signal.
