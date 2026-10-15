---
tags: [spec, verification, templates]
created: "2026-10-14"
---

# Verification - CLI-118-worktree-one-root

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] AC1 root declared and rendered: declared in `env-contract.json` (PR 1), kept equal to herdr's directory by `tests/herdr-config.bats` "WORKTREE_ROOT and herdr's worktrees.directory name the same root" (mutation: changing herdr's directory fails it). Rendering into the three configs is PR 7.
- [x] AC2 path: `TestResolveWorktreePath` (four herdr 0.9.3 rows measured on #2503), `TestAddUnderWorktreeRoot`, `TestAddUnderWorktreeRootFollowsBranch` (PR 1).
- [ ] AC3 branch validation: PR 2.
- [ ] AC4 `--agent`: PR 6.
- [ ] AC5 adoption: PR 4.
- [ ] AC6 done: slug resolution in PR 1 (`TestWorktreeDoneTakesTheSlugAddWasGiven`, `TestWorktreeDoneTakesTheSlugOfALegacySibling`, `TestWorktreeDoneRefusesASlugInBothLayouts`, and `TestWorktreeDoneTakesTheBranchGivenToAdd` for `add <slug> -b <branch>`, whose directory follows the branch: done takes the branch or its slug, and add prints which); branch deletion in PR 4.
- [ ] AC7 live session: gate f cwd check fixed in PR 1 (`TestGateFRefusesACwdInsideTheWorktreeWithoutTheProcessScan`); transcript check in PR 3.
- [ ] AC8 migrate: PR 5.
- [ ] AC9, AC10 docs, skills, ADR: PR 7.
- [ ] AC11 cross-shell / cross-OS: per PR.

## Test status

- PR 1, macOS: `cd cli && go build ./... && go vet ./... && go test ./...` all ok; `GOOS=windows go vet ./internal/worktree/ ./internal/cmd/` clean; `golangci-lint run` (pinned 2.12.2) 0 issues; `bats tests/herdr-config.bats` 6 ok; env-contract consumers (`tests/env-contract.bats`, `guard-setup-preexports-present.bats`, `opencode.bats`, `setup-linux.bats`) 131 ok.
- No regressions in existing test suite: yes.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- `done <slug>` resolution moved from PR 4 into PR 1: once `add` writes under the root, a sibling-only `done <slug>` would have broken in between.
- `list --all` needed no change: it finds worktrees through `git worktree list`, not by directory layout.
- `add` refuses when WORKTREE_ROOT resolves empty and no `--path` is given (`TestAddRefusesWithoutARoot`), and `done` says the root was not searched (`TestWorktreeDoneSaysWhenTheRootWasNotSearched`). `env.ResolvePath` returns "" when env-contract.json is neither deployed nor reachable from the cwd; falling back to the sibling there would pick the layout by accident. Measured on the Mac the same day: `DOTFILES_REPO_DIR` names the main checkout, which reads its own env-contract.json first, and until PR 1 lands that file has no WORKTREE_ROOT. The dev build refused (`WORKTREE_ROOT could not be resolved`) where the previous code would have made `dotfiles-wt-<slug>` silently.
- dotf creates the worktree and herdr only opens it (owner, 2026-10-14), so the lease, branch check and isolation gate hold on every path, Windows included.

- `done` identifies a worktree under the root by its branch slug, the one name herdr's layout gives it, not by a second slug recorded in the lease: a herdr-made or adopted worktree would never carry that slug (#2527 review, PR-Agent finding 1).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-118-worktree-one-root/` -> `specs/archive/CLI-118-worktree-one-root/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
