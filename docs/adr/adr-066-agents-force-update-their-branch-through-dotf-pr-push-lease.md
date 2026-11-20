---
id: "ADR-066-agents-force-update-their-branch-through-dotf-pr-push-lease"
type: adr
status: accepted
owner: manu
date: "2026-11-20"
issue: "mlorentedev/dotfiles#3612"
tags: [architecture, decision, harness, permissions, git]
created: "2026-11-20"
---

# ADR-066: agents force-update their own branch through `dotf pr push-lease`

## Status

Accepted 2026-11-19. The owner chose the dotf command on #3612 over the approach of #3608 (HARNESS-201),
whose ADR-065 exists only on that branch; this ADR takes the next number so the two never collide. The
spec is `specs/HARNESS-203-lease-push-command/`.

## Context

After rebasing its own feature branch an agent has to replace the branch on the remote. The Claude Code
template denies `Bash(git push *--force*)`, `Bash(git push -f*)` and `Bash(git push * -f*)`, which
refuse every spelling of a forced push, `--force-with-lease` included. The rebase then ends half done,
or the agent routes around it with a second branch and a second pull request (#3607).

#3608 allowed the push by narrowing those globs to let `--force-with-lease=<branch>:<sha>` through and
by parsing the shell command in a `PreToolUse` hook to refuse everything else. Six independent review
rounds failed it, each with spellings the hook's lexer read differently from the shell that would run
them; about 11 of their 24 Major findings touch the lease grant. Measured with an emulated matcher on
the 188 rows of that branch's own refusal table, 28 to 40 commands that main denies by glob would be
refused only by the hook, and all of them carry a lease: that exposure exists only because the glob was
narrowed. (Another 68 to 112 rows are holes main already has; they are not this decision's.)

## Options considered

1. **Narrow the deny globs and parse the command in a hook (#3608).** Rejected. A deny-list lexer for
   shell text has to agree with bash on every quoting, expansion and separator rule, and six rounds
   showed it does not converge. Every narrowed glob turns a static refusal into the parser's judgment.
2. **A raw-text rule in the hook, no lexer.** Rejected. A prototype missed nothing in its own sample,
   but on 362 push commands from 23 session transcripts it refused 50 it should have let through, and it
   still needs the narrowed glob to let the allowed shape reach it.
3. **A dotf subcommand that builds git's arguments itself.** Chosen. No shell text is parsed and no
   deny rule changes.

## Decision

1. `dotf pr push-lease --expect <sha> [--remote <name>]` replaces the branch of the same name on the
   remote with the checked-out commit: `git push --force-with-lease=refs/heads/<b>:<expect> <remote>
   <head sha>:refs/heads/<b>`, run through `exec` with no shell. HEAD is read once, so the push and the
   printed `<branch> <old>..<new>` line name the same commit.
2. It refuses before git pushes, each with a stable code: a detached HEAD (`detached-head`); `main` or
   `master` in any case (`protected-branch`); an `--expect` that is not a full 40 or 64 hex sha
   (`bad-expect`); a `--remote` that is not a configured remote name, such as a URL, a path or a
   leading `-` (`unknown-remote`); a remote whose push URL differs from its fetch URL, by `pushurl` or
   `pushInsteadOf`, since every check reads the fetch URL (`split-remote`); a remote that cannot be
   read (`remote-unreadable`); the remote's
   default branch, by what `ls-remote --symref` reports or what `refs/remotes/<remote>/HEAD` recorded
   (`default-branch`), and neither known (`default-unresolved`); a branch whose remote sha is not
   `--expect` (`stale-lease`).
3. `--expect` comes from the caller and is never derived. A lease read from the remote-tracking ref
   holds after any fetch, which makes it a plain force.
4. The stale-lease check reads the remote before pushing because git does not check a lease on a ref
   that already equals the pushed commit: measured 2026-11-20, a wrong lease against such a ref printed
   `Everything up-to-date` and exited 0. The lease itself still covers a push landing after that read.
5. Pre-push hooks run; there is no `--no-verify`. Every git child runs with `GIT_TERMINAL_PROMPT=0`.
6. The deny rules stay as on main, and no allow or ask rule names the command, so it runs under the
   ordinary permission prompt. `cli/internal/claude/push_lease_rules_test.go` pins both.
7. `AGENTS.md` names the command as the way an agent force-updates its own branch.

## Consequences

- No permission rule is relaxed and no shell text is parsed. The capability is a CLI, so every agent
  harness gets it, not only the one with the hook.
- A lease protects only against changes made after `--expect` was read. An agent that reads the remote
  sha fresh and pushes without integrating what it finds overwrites those commits. Accepted: the branch
  is the agent's own, and the command cannot know who else pushed.
- Only `main`, `master` and the remote's default are refused. Other long-lived shared branches
  (`release/*`, `develop`, `gh-pages`) can be force-updated by a caller who passes their current sha
  and rely on the forge's branch protection. Proposed as accepted; pending the owner's confirmation.
- Every push reads the remote first; offline, the command refuses instead of pushing. There is no
  timeout: a remote that accepts the connection and then stalls holds the turn (#3617).
- A remote that fetches over one URL and pushes over another (https fetch, ssh push) is refused, a
  false refusal for the same repository reached two ways. Reading the push URL instead would admit it,
  at the cost of handling several push URLs; refusing keeps every check about the push target.
- The command ships in a released binary, the `AGENTS.md` line with the repository. Before the release
  an agent that follows the line gets an unknown-command error: loud, and nothing is pushed.
- The `AGENTS.md` line reaches other repositories only where they fall back to this file. Carrying it
  into the harness-generated instructions every repository receives is #3613.
- #3608 is paused. What remains of it as a refuse-only guard, with #3611 and ADR-065, is decided after
  this lands.

## References

- #3612 (this decision, spec HARNESS-203), #3607 (the need), #3608 and #3611 (the replaced approach).
- `cli/internal/pushlease/`, `cli/internal/cmd/pr_push_lease.go`.
- Precedent for a lease push built as argv: `cli/internal/lane/claim.go` (`Claimer.push`).
