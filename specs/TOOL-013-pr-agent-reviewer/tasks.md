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
- [x] [AC1] **Empirical acceptance: open a PR carrying a known defect and confirm
      the reviewer finds it.** Cannot run before merge — the workflow is not on
      the default branch yet. This is the real test and no fixture substitutes
      for it. Met by defects nobody planted, which is the stronger form: see
      `verification.md`, "Evidence for the archive" (#2039, #2211)

## Formerly blocked, now resolved

- [x] Deliver the key: `dotf secrets sync ci`. Was blocked by #983 (now closed): the command refused
      the whole batch when any entry failed its liveness check, and a dead `BITACORA_PAT` kept
      `NAN_API_KEY` out of GitHub Actions secrets, so the workflow authenticated with an empty
      string. AI-045 `tasks.md` records the sync delivering CI keys on 2026-10-08, and every NaN
      review since authenticates with it. The shape stays worth naming: one dead credential
      blocked delivery of every other, #1004's batch-abort defect in a second command.
- [x] Verify the registry change end to end. Was blocked by #939 (now closed): `dotf` resolved
      `DOTFILES_REPO_DIR` ahead of the cwd, so `secrets sync ci` read the registry in the main
      checkout rather than the worktree.

## Closing

- [x] Every acceptance criterion is covered by at least one feature with a
      non-vacuous verification command
- [x] Every acceptance criterion has a matching entry in `features.json`
- [x] Config and workflow parse (`tomllib`, `yaml.safe_load`)
- [x] Tests pass (`bats tests/pr-agent-config.bats` — 11/11)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
- [x] Archive review findings F1-F3 addressed (`review.md`, FAIL): `[pr_reviewer] inline_key_issues = true`
      publishes the automatic review's located findings inline from the same `/review` call (F1);
      the guard reads that setting and checks `review` runs automatically, and the config comments
      no longer claim `/improve` carries the inline path (F2); `vendor/**` is carried in the ignore
      list (F3). The registry and classifier say inline findings attest through `reviews[]`, and the
      triage skill reads `pulls/N/comments`. Evidence on 2026-10-11 (#2325): `bats
      tests/pr-agent-config.bats` 73/73 (one network-leg skip, which passes under
      `DOTF_TEST_NETWORK=1`) and `tests/check-review-attestation.bats` 62/62; mutations of
      `inline_key_issues`, `auto_review`, `vendor/**` and the declared login each turn their test red.
      The `11/11` above is the count when this spec first closed.
- [x] AC1 live evidence: the first PR reviewed after the fix merges carries a COMMENTED review by
      `github-actions` with inline comments; then `verification.md` AC1 is rewritten from it
      (#2329 at 03:18Z was the first, then #2332 and #2334; recorded on #786)
- [ ] Adversarial review passes before archive (`dotf spec review TOOL-013-pr-agent-reviewer`)

## Follow-on, after the first production runs (#1044)

The first live executions exposed two things inspection had not, both fixed in
the same PR:

- [x] The reviewer is told what the harness requires, not merely handed it. Every
      review opens with a HARNESS COMPLIANCE pass over `AGENTS.md` and
      `.claude/CLAUDE.md`, reported per item even when everything passes. The
      config already loaded `AGENTS.md` but nothing asked the reviewer to use it,
      so compliance was checked when the model happened to notice.
- [x] `.claude/CLAUDE.md` joins `repo_context_files`. `extra_instructions` sent
      the reviewer to the prohibited-pattern table in a file that was not in
      context — an instruction naming a source it could not read.
- [x] Inline suggestions actually enabled. The `[pr_reviewer]` comment has
      claimed since this shipped that inline comments are "the reason this work
      exists"; `review` does not post inline, `improve` does, and there was no
      `[pr_code_suggestions]` section, so dual publishing sat at its default of
      -1. Measured across #1042, #1047 and #1051: 0 inline comments.
- [x] Four guards pin these as decisions, each observed failing on its own
      mutation, with the mutation's arrival verified by checksum rather than
      against HEAD — a dirty tree makes the latter report an invalid mutation as
      applied.

## Upstream update contract (#2010)

- [x] [AC8] Add a failing BATS test in `tests/pr-agent-config.bats` proving
      the workflow pin is checked against approved upstream source identities
      instead of a manually copied version string; run it before adding the
      approved manifest.
- [x] [AC8] Add the approved source identities and remove stale version claims
      from `scripts/pr-agent-push-gate.sh`; derive the filter's upstream ref
      from the executing workflow. Verify an unchanged upstream file passes
      and a changed file or failed API read fails closed.
- [x] [AC9] Add a failing BATS test proving `pr-agent.yml` explicitly fails on
      tool errors, executes at most one Action, and retains its internal model
      fallback; run it before changing the workflow.
- [x] [AC9] Remove the unconditional second Action and update the publication
      guard and affected tests without weakening ADR-040's three-commit gate.
- [x] [AC8] [AC9] Run targeted BATS tests and lint; record results in
      `verification.md` and disposition the two #2010 reviewer findings.

## Deliberately not done here

Rollout to other repos, retiring CodeRabbit, the parallel-measurement window,
and the Gitea leg (#1005). #786's full scope is a fleet rollout; this slice
proves the shape on one repo, and the standing rule caps autonomous reviewers at
two, so both run in parallel until evidence retires one.

## Machine-readable features

`features.json` sits beside this file, one feature per acceptance criterion
(f1–f7). **Pass-state gating:** the agent CANNOT write `"state": "passing"` —
only the harness, after running `verification` and capturing exit code 0.
