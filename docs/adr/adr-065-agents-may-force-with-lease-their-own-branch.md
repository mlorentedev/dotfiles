---
id: "ADR-065-agents-may-force-with-lease-their-own-branch"
type: adr
status: accepted
owner: manu
date: "2026-11-19"
issue: "mlorentedev/dotfiles#3607"
tags: [architecture, decision, harness, security, hooks]
created: "2026-11-19"
---

# ADR-065: agents may force-with-lease their own branch, and a hook, not a glob, decides

## Status

Accepted 2026-11-19. The owner decided in session ("si, permitelo en todos es lo mejor"): the agent may
push `--force-with-lease` in every repository, under the limits below. The spec is
`specs/HARNESS-201-force-with-lease-own-branch/`. This relaxes a standing barrier, which is why it is
recorded here.

## Context

`ai/claude/settings.json` denied `Bash(git push *--force*)`, `Bash(git push -f*)` and
`Bash(git push * -f*)`. The first glob also matches `--force-with-lease=<branch>:<sha>`, the safe form an
agent needs after rebasing its own feature branch. The cost was a half-finished rebase, or a second
branch and a second pull request.

The globs were also weaker than they read. A permission rule matches the command text, not the
destination, so `git push origin +main` (a `+` refspec forces without any flag) and
`git -C <dir> push --force` passed. Server-side branch protection is not confirmed on every repository
the harness deploys to, so on some of them the local rule is the only barrier.

A glob cannot say "the lease form is allowed, a plain force is refused, the default branch is refused".
Deny rules also win over a hook's decision, so the globs had to narrow before anything could be allowed.

## Decision

1. **One force form is allowed:**

   ```
   git push <remote> --force-with-lease=<branch>:<full sha> <refspec>
   ```

   with exactly one lease and one refspec (`<b>`, `<b>:<b>`, `HEAD`, `HEAD:<b>`, or the `refs/heads/`
   spelling). The lease branch, the destination and the branch checked out in the resolved working
   directory are one name. The sha is 40 or 64 hex characters. The destination is not the remote's
   default branch (`refs/remotes/<remote>/HEAD`) and is never `main` or `master`, resolved or not.
   `--force-if-includes` may accompany it. Options may sit before or after the remote.

2. **Everything else that forces is refused:** `--force`, `-f` (also in a cluster such as `-uf`), a bare
   `--force-with-lease`, a lease with no sha or a short or symbolic one, `--force-if-includes` alone,
   `--mirror`, any `+<refspec>`, a second lease, a deletion (`--delete`, `-d`, `:<branch>`), `--all`,
   `--tags`, `--prune` or any other option beside a lease, an implicit remote, a detached HEAD, a working
   directory that cannot be resolved (`cd $X`, `--git-dir`, `GIT_DIR=`), a variable or substitution inside
   a force-like push, and any abbreviation of a force option that git would accept.

3. **A hook decides, parsed from the command.** `dotf harness push-guard` is bound by `dotf harness bind`
   as a second `PreToolUse` hook (id `push-guard`) beside the persona gate. It reads `tool_name`,
   `tool_input.command` and `cwd`, splits the string into simple commands (`&&`, `||`, `;`, `|`, `&`,
   newlines, subshells, `$(...)`, backticks, `bash -c`, `eval`), follows `cd` and `git -C` to find the
   directory, and asks real git for the checked-out branch and the default branch. It answers exit 0 or
   exit 2 and nothing else, because Claude reads any other status as a non-blocking error. The decision
   logic is harness-neutral Go in `cli/internal/harness/pushguard.go`.

4. **The parser is asymmetric.** A push that does not force passes untouched, exactly as before; a
   force-like push passes only in the one literal shape, and anything force-like the parser cannot read to
   the end is refused. A parser bug therefore fails toward refusing a force, never toward allowing one and
   never toward blocking an ordinary push.

5. **The hook never grants.** It refuses or stays silent. No `git push` rule enters `permissions.allow`
   (a test pins that), so whether a verified lease push prompts is the permission system's call. Skipping
   the prompt is a wider grant than the owner's "allow" and stays an open question.

6. **The globs narrow to what stays exact** and remain the first layer, and the only one when the hook is
   down: `--force` followed by a space or at the end, `-f`, the bare lease, `--mirror`, the `git * push`
   variants, and a lease naming `main:` or `master:` followed by more arguments. A test pins the list and
   runs a command table through an emulation of the rule syntax, so a later edit cannot widen it silently
   or catch the lease form again.

7. **`npm run land:push` is refused** in the hook (also through `pnpm`, `yarn`, `bun`, `run-script`, a
   chain or `bash -c`) and in the deny list. That mode is the owner's.

## Consequences

**Rollout order matters.** The hook ships in a released binary; settings and manifest ship from the repo.
Deploy after the release that contains `push-guard`. Before it, an older binary binds a hook calling a
subcommand it lacks (an error per tool call, not a block) and the narrowed globs leave the lease form
unguarded.

**The old broad glob leaves an existing box only through the deploy record.** Permission lists union on
merge, so narrowing the template does not remove `Bash(git push *--force*)` from a deployed
`settings.json`. The merge's retire step removes it when the previous deploy recorded it, which a test
drives through the real template. Without a record (a box that never deployed with records, or whose
record was lost) the old glob stays and the lease form stays denied there: no weaker, only not yet
relaxed. A second test names that state.

**The hook fails open.** A Claude hook that crashes, times out (5 s, bounded; lesson 264) or whose binary
is missing allows the call. Then the retained globs are the only barrier: they refuse plain `--force`,
`-f` at the start of the options, the bare lease and a lease on `main`/`master` followed by arguments;
they do not see `+<refspec>`, a `-uf` cluster, a lease last on the line or a lease on another default
branch. These were not seen by the old globs either.

**Stricter than the owner's list.** The issue also pins the destination to the checked-out branch; the
owner's list does not mention it. The stricter reading applies. Relaxing it is a one-line change in
`refuseOtherBranch` and a decision for the owner.

## Limits

- **Other surfaces.** Claude Code is the only surface with a pre-tool hook that carries the command and a
  deploy path for it here. agy has a `PreToolUse` bind, but its shell tool name and argument shape are not
  measured in this repository, and its settings allow `command(*)` and deny only `rm`; no force rule
  exists there to relax, and none is added until the payload is measured. pi and opencode wrappers are
  declared `emit: false`. Copilot is instruction-only. Gemini CLI and Codex are instruction-only: AGENTS.md
  is the only barrier there.
- **Text, not behaviour.** The hook reads the command string. Git aliases, `push.*` and `remote.*.push`
  from config on disk, scripts and `make` targets that run git, and other tools that push (API clients)
  are outside what a string can show. `-c remote.<n>.push=+...` on the command line is refused.
- **A parser, not a sandbox.** The guard reads shell text with its own lexer. Two independent review
  rounds each found further spellings that reached a forced push (heredoc forms, wrapper options,
  subshell scope, attached git options, `$'...'`); every one was fixed test-first, and the next round
  will probably find another. It is a guard against an agent that mistypes or reaches for the wrong
  form, not against one that obfuscates on purpose. Anything a shell can express that the lexer cannot
  follow should fail closed (an unparseable text that mentions a push is refused); a new bypass is a bug
  to fix with a row in `TestGuardCommand_*`, not a reason to widen the allowed form. Five rounds in,
  three heredoc-feeder spellings stay open; #3611 tracks a fail-closed text backstop for them.
- **Plain pushes are out of scope.** A non-forced `git push origin HEAD:refs/heads/main` stays as it is;
  branch protection on the forge is the barrier for it.
- **No forge lookup.** The default branch is read from `refs/remotes/<remote>/HEAD` and backstopped by
  `main` and `master` by name. A repository whose default is another name and whose `origin/HEAD` is unset
  is not protected by resolution (it is by the lease needing to match the checked-out branch).
- **PowerShell tool.** The hook parses its commands, but the deny globs are written for the `Bash` tool.
- **The glob emulation is not Claude's matcher.** The pin test proves what the rule text says under the
  documented syntax, not how Claude reads it.
- **Windows.** A POSIX shell on Windows sends `cd /c/Users/...`; the hook converts the MSYS drive
  spelling for the native git it runs. Directory handling is tested with real git, on the platform it runs.
- **A refused call leaves no durable record** beyond the model's transcript; the hook does not journal.

## Reversal

Remove the `push-guard` entry from `harness/manifest.json`, restore the three original deny globs and run
`dotf harness bind` and `dotf deploy`. The retire step then removes the narrowed globs as it removed the
old one.
