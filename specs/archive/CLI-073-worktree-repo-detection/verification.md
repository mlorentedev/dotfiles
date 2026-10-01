---
tags: [spec, verification, templates]
created: "2026-09-29"
---

# Verification - CLI-073-worktree-repo-detection

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `TestCheckRepoDirResolves/worktree checkout -> pass`
- [x] Criterion 2 -> `TestCheckRepoDirResolves/normal checkout -> pass`
- [x] Criterion 3 -> the missing-path and non-checkout rows in
  `TestCheckRepoDirResolves`
- [x] Criterion 4 -> `TestCheckRepoDirResolves/checkout subdirectory -> fail`
- [x] Criterion 5 ->
  `TestClaudeContextRecognizesLinkedWorktreeFromRootAndSubdirectory`

## Test status

- Fail-first: `go test ./internal/doctor -run '^TestCheckRepoDirResolves$' -count=1`
  failed only for the linked-worktree row with `not a git checkout`.
- Targeted suite: the same command passes after resolving and comparing Git's
  top-level path.
- Checkout-root fail-first: the subdirectory row initially passed because
  `--is-inside-work-tree` cannot distinguish a root from a descendant; it now
  fails unless the configured path equals `--show-toplevel`.
- Session-start fail-first: both the worktree root and `cli/` lacked all four
  blocks. The focused mem test now passes with the main project name.
- Static checks: `go build ./...`, `go vet ./internal/doctor`,
  `golangci-lint run --new-from-rev=main`, `git diff --check`, and
  `jq empty specs/CLI-073-worktree-repo-detection/features.json` -> pass.
- Manual smoke test: source-built `dotf doctor` with `DOTFILES_REPO_DIR` pointed
  at this linked worktree no longer emits `not a git checkout`.
- No regressions: normal checkout, missing path, and existing non-checkout rows
  remain green.
- Full doctor package: all tests reach completion except the pre-existing
  Windows symlink privilege failure tracked by #1804.
- Adversarial review round 1 dispositions:
  - **Applied:** checkout-root comparison now canonicalizes both paths with
    `filepath.EvalSymlinks`; a symlinked checkout path compares equal to Git's
    physical top-level path.
  - **Applied:** submodule `.git/modules/...` pointers resolve to the submodule
    project name instead of the superproject.
  - **Applied:** bare-repository worktree pointers derive the project name from
    the common `project.git` directory.
- Adversarial review round 2 dispositions:
  - **Applied:** submodule worktree pointers now anchor on the exact
    `.git/modules/` sequence and select the component immediately before
    `worktrees/`, rather than returning the worktree name.
  - **Applied:** paths containing an unrelated parent directory named
    `modules` are not classified as submodules.
  - Named regressions:
    `TestCheckoutProjectNameHandlesSubmoduleAndBareWorktreePointers/submodule_worktree`
    and `/ordinary_worktree_below_a_parent_named_modules`.
- Post-archive CodeRabbit review dispositions:
  - **Applied:** the doctor Git probe now strips repository-local Git variables
    from only its subprocess environment; the parent process is unchanged.
  - **Applied:** checkout discovery resolves symlinked working directories before
    walking ancestors.
  - **Applied:** generic `--separate-git-dir` layouts keep the checkout directory
    name instead of treating the metadata directory as project identity.
  - **Skipped:** CodeRabbit's docstring coverage warning is not a repository gate
    and adding comments to self-explanatory unexported helpers would conflict with
    the repository's comment policy.
  - **Skipped:** `features.json` remains `pending` with empty evidence by design;
    only the harness may write terminal feature state, and recent archived specs
    such as WIN-014 and HARNESS-168 retain the same harness-owned state.
- Post-review validation:
  - `go test ./internal/doctor ./internal/mem ./internal/cmd -count=1` -> pass.
  - `go vet ./internal/doctor ./internal/mem ./internal/cmd` -> pass.
  - `git diff --check` -> pass.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Ask Git for `--show-toplevel` instead of parsing `.git` pointer files in the
  doctor check; Git remains the authority and the existing seam keeps tests
  hermetic.
- Keep the cascade-only behavior: this check must validate the configured path,
  not hide a bad default by discovering doctor's current checkout.
- Session-start is intentionally filesystem-only: it walks to the `.git` entry
  and resolves a linked worktree's main project name from the pointer, avoiding
  a subprocess in every session-start hook.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: lesson 161 already records the
  linked-worktree `.git` file invariant.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: this aligns
  doctor with an existing repository-location decision.
- [x] New pattern candidate for `00_meta/patterns/`? no: the reusable rule is
  already captured by the worktree safety material.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-073-worktree-repo-detection/` -> `specs/archive/CLI-073-worktree-repo-detection/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

The issue remains open until PR #1835 merges.
