---
id: "TOOL-030-forge-write-surface"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-10-16"
issue: "mlorentedev/dotfiles#2635"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "owner authorised 2026-10-16: one spec closes BUG-152, the Gitea classifier denials and the per-harness permission guard together (12 active, limit 10, 2026-10-16)"
---

# TOOL-030-forge-write-surface

> **Naming**: file lives at `<repo>/specs/TOOL-030-forge-write-surface/proposal.md`. `TOOL-030-forge-write-surface` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #2635: TOOL-030: versioned Gitea CLI plus scoped allow rules and a per-harness permission guard -->

Agents write to forges (issues, comments, PRs, board fields) through whatever is at hand: `gh` with a prose body on the command line, or ad-hoc Python against a REST API. Three failures share that root, measured 2026-10-16: (1) `permissions.deny` globs such as `Bash(*dotf secrets show*)` match prose inside a commit or issue body and refuse a harmless call (BUG-152, #2634; four calls in one session, and it made #2629 a fix for the wrong cause); (2) the auto-mode classifier refuses writes to a private Gitea as "External System Writes" because they arrive as Python plus a token to an unknown host (imagesensortool session); (3) each harness speaks its own permission dialect (Claude `permissions.allow`, agy `command(...)`, opencode `permission`, Copilot trusted folders, pi none), so every new tool needs N rules and nothing checks that they agree. One narrow, versioned write surface in `dotf` gives all three a single answer, in the repo-wide direction of ADR-050 (one contract for every repo, agent, forge and OS).

## What

A `dotf forge` write surface for agents, forge-neutral:

- Verbs: `dotf forge issue create|comment|edit`, `dotf forge pr create|comment|edit`, and `--board` to set the Project fields `new-ticket` sets by hand today. No `merge`, no `close`: merging stays `dotf pr land` behind the merge grant, closing stays with the owner.
- A body is read only from `--body-file` or stdin, never from an argument, so prose never reaches a permission glob.
- The forge is detected from `origin` (the `initrepo.ClassifyOrigin` classification is the starting point). GitHub is served through the `gh` REST endpoint, as `lane.REST` already does; Gitea through a REST client inside `dotf`. The token comes from `dotf secrets` at call time and is never printed or kept as login state. No `tea` dependency.
- One narrow allow set replaces per-tool rules: `Bash(dotf forge issue:*)`, `Bash(dotf forge pr create:*)`, `Bash(dotf forge pr comment:*)`, `Bash(dotf forge pr edit:*)`. The settings PR that adds it also reverts #2629 (wrong premise) and is landed by the owner (#2178).
- A per-harness guard: one bats `@test` per harness config (claude, agy, opencode, copilot, pi) asserting `dotf` is not denied in that harness's dialect, so the dialects cannot drift apart silently.
- The `new-ticket`, `pr-review-triage` and `autopilot` skills call `dotf forge` instead of carrying `gh` recipes.

## Out of scope

- Merge, close, branch protection (`dotf forge protection` and `dotf pr land` keep those).
- Narrowing the `permissions.deny` globs themselves (BUG-152 keeps that option; this spec removes the need by never putting prose on the command line).
- Forges other than GitHub and Gitea; a third backend plugs into the same interface later.
- Reading verbs (`gh`/`dotf pr` reads already pass).

## Risks / open questions

- **The classifier may still refuse Gitea writes through `dotf forge`.** The hypothesis (it flags raw HTTP plus token to a non-GitHub host, not the write itself) is unmeasured. PR 1 ships one read verb and one Gitea comment, and records the classifier's answer on #2635 before any allow rule is added. If it refuses, the fallback is `autoMode.environment` naming the Gitea host, which is prose and still passes the classifier.
- Gitea REST differs from GitHub on labels, projects and draft state; `--board` is GitHub-only at first and degrades with a stated message, never silently.
- The allow set widens agent authority: prepared by the agent, landed by the owner.
- Shell portability: nothing here is shell; it is Go in `cli/`, and the guard bats must pass under bash and zsh and macOS bash 3.2.

## Acceptance criteria

- [ ] `dotf forge issue create --body-file F` opens an issue on GitHub and on the private Gitea, and the command line contains no body text.
- [ ] `issue comment`, `issue edit`, `pr create`, `pr comment`, `pr edit` work on both forges; there is no `merge` or `close` verb, asserted by a test.
- [ ] The forge is detected from `origin`; an undetectable one fails with an ERROR/WHY/FIX message, never a guess.
- [ ] The Gitea token is read from `dotf secrets` at call time and appears in no output, log or argument (a test greps for it).
- [ ] A bats guard has one `@test` per harness config and fails when `dotf` is denied in that harness's dialect.
- [ ] The measured classifier result for a Gitea write is recorded on #2635 before the allow set lands.
- [ ] The three skills use `dotf forge`; #2629 is reverted in the owner-landed settings PR.

## References

- Bitácora board: mlorentedev/dotfiles#2635 (gate), #2634 (BUG-152, the deny-glob cause), #2631/#2633 (BUG-151, fixed on the way)
- Related ADR: ADR-050 (lane contract for every repo, agent, forge and OS), ADR-030 (checkout-first reads), ADR-049 (gitea-mq)
- Existing code: `cli/internal/forge` (protection), `cli/internal/lane/forge.go` (GitHub REST via gh), `cli/internal/initrepo/forge.go` (origin classification)
