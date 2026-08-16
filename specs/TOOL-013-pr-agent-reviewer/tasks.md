---
tags: [spec, tasks, templates]
created: "2026-08-16"
---

# Tasks - TOOL-013-pr-agent-reviewer

> TDD order. One task = one focused commit. `[AC<n>]` maps a task to an acceptance criterion in `proposal.md`; `[P]` marks a task with no dependency on another unchecked one.

## Setup

- [x] Branch created from main: `feat/pr-agent-reviewer`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the
      token-budget question is a stated floor with a revision trigger, not an
      unknown; every other entry is a named dependency with an issue number

## Implementation

- [x] [AC5] Verify the action against reality before pinning it — the project has
      been renamed (`qodo-ai/pr-agent` now redirects to `The-PR-Agent/pr-agent`,
      which #786's body predates) and the first draft pinned `@v0.30`, a tag that
      does not exist. Pinned `v0.42.0`, confirmed to carry an `action.yaml`
- [x] [AC2] [AC3] [AC7] `.pr_agent.toml` — reasoning-class model, **no** cheap
      fallback, `sensitive/**` excluded, `AGENTS.md` as review context
- [x] [AC1] [AC6] `.github/workflows/pr-agent.yml` — `pull_request` +
      `issue_comment` for slash commands, drafts skipped
- [x] [AC4] Declare `NAN_API_KEY` as a CI consumer of this repo, so
      `dotf secrets sync ci` delivers exactly that key and no other
- [x] 11 bats cases asserting the DECISIONS rather than the syntax
- [ ] [AC1] **Empirical acceptance: open a PR carrying a known defect and confirm
      the reviewer finds it.** Cannot run before merge — the workflow is not on
      the default branch yet. This is the real test and no fixture substitutes
      for it

## Blocked, and not by anything in this change

- [ ] Deliver the key: `dotf secrets sync ci`. **Blocked by #983** — the command
      refuses the whole batch when any entry fails its liveness check, and
      `BITACORA_PAT` is a dead token (HTTP 401). Until that is resolved, or the
      operator passes `--skip-verify`, `NAN_API_KEY` cannot reach GitHub Actions
      secrets and the workflow authenticates with an empty string. Worth noting
      as a shape, not just an instance: one dead credential blocks delivery of
      every other, which is #1004's batch-abort defect in a second command
- [ ] Verify the registry change end to end. **Blocked by #939** — `dotf`
      resolves `DOTFILES_REPO_DIR` ahead of the cwd, so `secrets sync ci` reads
      the registry in the main checkout, not this worktree

## Closing

- [x] Every acceptance criterion is covered by at least one feature with a
      non-vacuous verification command
- [x] Every acceptance criterion has a matching entry in `features.json`
- [x] Config and workflow parse (`tomllib`, `yaml.safe_load`)
- [x] Tests pass (`bats tests/pr-agent-config.bats` — 11/11)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Adversarial review passes before archive (`dotf spec review TOOL-013-pr-agent-reviewer`)

## Deliberately not done here

Rollout to other repos, retiring CodeRabbit, the parallel-measurement window,
and the Gitea leg (#1005). #786's full scope is a fleet rollout; this slice
proves the shape on one repo, and the standing rule caps autonomous reviewers at
two, so both run in parallel until evidence retires one.

## Machine-readable features

`features.json` sits beside this file, one feature per acceptance criterion
(f1–f7). **Pass-state gating:** the agent CANNOT write `"state": "passing"` —
only the harness, after running `verification` and capturing exit code 0.
