---
id: "HARNESS-203-lease-push-command"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-11-19"
issue: "mlorentedev/dotfiles#3612"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
risk: high
wip_override: "owner decision 2026-11-19 replaces the approach of HARNESS-201 (#3608); the 11 active specs on main belong to other work and are not mine to archive (11 active, limit 10, 2026-11-19)"
---

# HARNESS-203-lease-push-command

## Why

<!-- from issue #3612: HARNESS-203: dotf command that force-with-leases the agent's own branch -->

After rebasing its own feature branch an agent must force-with-lease it, and today `Bash(git push *--force*)` denies every spelling of that: the rebase ends half done, or in a second branch and a second pull request (#3607). HARNESS-201 (#3608) allowed it by narrowing the deny glob and parsing the shell command in a hook; six independent reviews failed it, each with new spellings, and on its own refusal table 28 to 40 cases that main denies would be held only by the hook, all of them lease-spelled. A dotf command that builds git's arguments itself gives the agent the lease push without relaxing any deny rule and without parsing shell text.

## What

- `dotf pr push-lease --expect <sha> [--remote <name>]` pushes the branch checked out in the current directory to the same name on the remote: `git push --force-with-lease=refs/heads/<b>:<sha> <remote> <head sha>:refs/heads/<b>`, built as argv, no shell. HEAD is read once, so the push and the printed line name the same commit.
- Refused before git pushes, with a stable code, a reason and exit 1: detached HEAD; branch `main` or `master` (any case); `--expect` missing or not 40/64 hex; `--remote` not a configured remote name (a URL, a path or a leading `-` is refused); a remote whose push URL differs from its fetch URL (`remote.<r>.pushurl`, `url.<base>.pushInsteadOf`), or that has more than one URL (a second `remote.<r>.url`), or that sets `remote.<r>.receivepack` or `remote.<r>.uploadpack` (the program run on the remote for a push or a read, which no URL shows), because every check reads one repository (`ls-remote` takes the first URL) and the push would write elsewhere (git pushes to every URL a remote has); the remote unreadable; the branch equal to the remote's default by either account, what `git ls-remote --symref <remote> HEAD` reports now or what `refs/remotes/<remote>/HEAD` recorded (neither known is a refusal). Local checks run before any network call; one `ls-remote` reads the default and the branch together.
- `--remote` defaults to the branch's push remote (`branch.<b>.pushRemote`, then `remote.pushDefault`), else its upstream remote (`branch.<b>.remote`), else `origin`.
- An `--expect` that differs from the remote's current sha for the branch (`git ls-remote`) is refused before pushing, also when the remote already equals the local HEAD; the lease still covers a change between that read and the push.
- Pre-push hooks run (no `--no-verify`). A stale lease is git's rejection, passed through with a non-zero exit.
- Every git child runs with `GIT_TERMINAL_PROMPT=0`: a remote asking for credentials fails instead of waiting on a terminal.
- On success it prints one line, `<branch> <old>..<new>`, the remote sha before and after, as evidence for the pull request. When the remote already held the local HEAD, git pushes nothing and the line ends `(already there, nothing pushed)`.
- Deny globs and `permissions.allow` unchanged. One line in `AGENTS.md` names the command as the way an agent force-updates its own branch.

## Out of scope

- What remains of #3608: a refuse-only guard (the lexer or a raw-text rule), its narrowed deny globs, #3611 and ADR-065. A separate decision after this lands.
- Skipping the permission prompt: no `permissions.allow` rule names the command; whether a lease push may run unprompted stays the owner's open question.
- Any push other than the checked-out branch to its own name: no refspec argument, no deletion, no tags, no `--mirror`, no second lease.

## Risks / open questions

- A lease protects only against changes made after the agent read `--expect`. An agent that fetches, reads the remote sha and pushes without integrating someone else's commits overwrites them. Accepted: the branch is the agent's own; the command cannot know who else pushed.
- Rollout: the command ships in a released binary, the `AGENTS.md` line from the repo. Deployed before the release, an agent calls a missing subcommand: a loud error, no push. Deploy after the release.
- Every push reads the remote first (`ls-remote`, for the default branch and the stale-lease check); a failure is a refusal, never a push. The `main`/`master` backstop holds regardless.
- Checked: no deny, allow or ask rule in `ai/claude/settings.json` matches `dotf pr push-lease` (emulated matcher), so the normal prompt applies.
- Only `main`, `master` and the remote's default are refused. Other long-lived shared branches (`release/*`, `develop`, `gh-pages`) can be force-updated by a caller who passes their current sha. `dotf` does not protect them; they are covered only where the forge has a rule for them. Measured 2026-11-23 over the 25 personal repositories pushed in the last two months and the work repository: no `release/*` or `develop`, and one `gh-pages` (boilerplates), a branch a deploy job normally rewrites anyway. Accepted by the owner, 2026-11-23, with no configurable list: one would duplicate what the forge owns, and if ever built it must only refuse, never grant by branch name. Revisit when a repository where agents push gains a shared `release/*` or `develop`.
- No timeout on `ls-remote` or `push`: a remote that accepts the connection and then stalls holds the turn. Ticketed as #3617 rather than built here.
- Configured code that git runs for both directions alike is not refused: `core.sshCommand`, `GIT_SSH_COMMAND`, `core.gitProxy`, a `remote.<r>.vcs` helper. Such a program could still route a read and a push apart, but whoever sets it already runs code at push time, as a pre-push hook does, and refusing `core.sshCommand` would refuse ordinary key setups. The split guard covers configuration that names a different target, not code written to deceive it. `receivepack` and `uploadpack` are refused because each applies to one direction only.
- With `push.recurseSubmodules=on-demand` git pushes a submodule first with the same refspec; it fails closed today only because the source of the refspec is a sha, not a ref name. Probed 2026-11-23, nothing written.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1 Against a real temporary repository with a bare remote: after a rebase, `dotf pr push-lease --expect <remote sha>` moves the remote branch to the local HEAD and prints `<branch> <old>..<new>`; with a stale `--expect` the remote ref is unchanged and the exit is non-zero; with the remote already at HEAD the line ends `(already there, nothing pushed)`.
- [ ] AC2 A table test refuses with exit 1 and a reason on stderr, the remote refs unchanged: detached HEAD; `main`; `master`; the remote default (`origin/HEAD` set; unset and found by `ls-remote`; changed on the remote since `origin/HEAD` was recorded; known by neither); the remote unreadable; `--expect` missing, short, symbolic; `--remote` unknown, a URL or path, option-like; a remote whose push URL differs from its fetch URL (`pushurl`, `pushInsteadOf`), that has a second `url`, or whose `receivepack` or `uploadpack` sends the push or the read to another repository, with the push target's refs unchanged too.
- [ ] AC3 A pre-push hook in the temporary repository runs on a lease push, and its refusal stops the push.
- [ ] AC4 A test pins `ai/claude/settings.json`: deny still holds `Bash(git push *--force*)`, no rule matches `dotf pr push-lease`. `AGENTS.md` names the command; an ADR records the decision with an alternatives section (glob plus hook, raw text, this command); the repository's full gates pass.

## References

- Bitácora board: #3612 (this spec); the need is #3607.
- The approach this replaces: #3608 (HARNESS-201, narrowed globs plus a parsing hook) and #3611 (its heredoc backstop); its ADR-065 lives on that branch only. This decision is recorded as ADR-066 (`docs/adr/adr-066-*.md`).
- Precedent for a lease push built as argv: `cli/internal/lane/claim.go` (`Claimer.push`).
