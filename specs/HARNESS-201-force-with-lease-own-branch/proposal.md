---
id: "HARNESS-201-force-with-lease-own-branch"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-11-19"
issue: "mlorentedev/dotfiles#3607"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
risk: high
wip_override: "owner-approved change dispatched by the lead (2026-11-19); the 11 active specs belong to other work and are not mine to archive (11 active, limit 10, 2026-11-19)"
---

# HARNESS-201-force-with-lease-own-branch

## Why

`ai/claude/settings.json` denies `Bash(git push *--force*)`, which also blocks `--force-with-lease=<branch>:<sha>`, the safe form an agent needs after rebasing its own feature branch; the rebase ends half done or in a second branch and a second pull request. The same globs are a weaker barrier than they look: permission rules match the command text, so `git push origin +main` and `git -C <dir> push --force` pass, and server-side branch protection is not confirmed on every repository the harness deploys to. Owner decision, 2026-11-19: allow the agent force-with-lease pushes in every repository under fixed limits (below). A glob cannot say "lease allowed, force refused, default branch refused", so the decision logic moves into a hook that parses the command.

## What

- A new `dotf harness push-guard` command, bound by `dotf harness bind` as a second `PreToolUse` hook (id `push-guard`) on Claude. It reads the hook payload (`tool_name`, `tool_input.command`, `cwd`), parses every simple command in the string (`&&`, `||`, `;`, `|`, newline, `$(...)`, backticks, `bash -c '...'`, `eval`), and answers exit 2 with a reason for a refused push, exit 0 otherwise. It never exits 1 and never emits an allow decision: it refuses or stays silent.
- The parser is asymmetric by design. A push is *force-like* when it carries `--force`, `-f` (also inside a short cluster), any `--force-with-lease` or `--force-if-includes` spelling (including git's unique-prefix abbreviations), `--mirror`, or a `+<refspec>`. A push that is not force-like is passed untouched, exactly as today. A force-like push is allowed only in one fully literal shape; anything force-like the parser cannot fully read is refused.
- The one allowed shape: `git [-C <dir>] push <remote> --force-with-lease=<branch>:<full-sha> <refspec>` (options may sit before or after the remote, `--force-if-includes` may accompany it), where exactly one lease and exactly one refspec are given, the refspec is `<b>`, `<b>:<b>`, `HEAD`, `HEAD:<b>` or the `refs/heads/` spelling of those, the lease branch, the destination and the branch checked out in the resolved working directory are the same name, the sha is 40 or 64 hex characters, and the destination is neither the remote's default branch (`refs/remotes/<remote>/HEAD`) nor, unconditionally, `main` or `master`.
- Refused with a reason: bare `--force-with-lease`, `--force-with-lease=<branch>` without a sha, `--force`, `-f`, `--force-if-includes` with no valid lease, `--mirror`, `+<refspec>`, any deletion (`--delete`, `-d`, `:<branch>`, `--prune`) or `--all`/`--tags` combined with a force-like option, a lease naming another branch than the destination or the checked-out branch, a detached HEAD, an unresolvable directory (`cd` to a variable, `--git-dir`, `GIT_DIR=`), a variable or substitution inside a force-like push, a remote left implicit.
- `npm run land:push` (any arguments, also through `pnpm`, `yarn`, `bun` and `run-script`) is refused: that mode is the owner's.
- `ai/claude/settings.json` keeps deny globs only where they stay exact (`--force` followed by a space or at the end, `-f`, `--mirror`, `git * push` variants, the `main:` and `master:` leases) so they remain the first layer, and adds `Bash(npm run land:push*)`. No `git push` rule enters `permissions.allow`: the lease form is no longer denied, and the normal permission flow decides about the prompt.
- The deploy merge unions lists, so the narrowed template does not remove the old broad glob from an already-deployed settings file by itself; it is removed through the deploy record's retire step, which a test exercises.
- An ADR records the decision, the allowed and refused forms and the hook that enforces them.

## Out of scope

- A non-force push, including `git push origin HEAD:refs/heads/main`, stays as it is today: the hook does not widen into a general push policy. Branch protection on the forge remains the barrier for those.
- A forge lookup for the default branch inside the hook (slow, needs a credential, fails open). `refs/remotes/<remote>/HEAD` plus the `main`/`master` backstop is used instead.
- Emitting an allow decision that skips the permission prompt. A wider grant than "stop refusing"; listed as an open question.
- Aliases, `push.*` and `remote.*.push` config read from disk, scripts and wrappers that run git internally (`make`, `sh script.sh`), and other tools that push (API clients). The hook reads the command text it is given.

## Surfaces (Limits)

- **Claude Code:** hook plus narrowed deny globs, as above. The only surface with both a pre-tool hook that carries the command and a deploy path for it here.
- **agy:** has a `PreToolUse` bind, but its shell tool name and argument shape are not measured in this repository (the gate there is "measurement plumbing"), and `ai/agy/settings.json` allows `command(*)` and denies only `rm`. No force-push rule exists there today, so nothing is relaxed; nothing is added until the payload is measured.
- **pi, opencode:** the TypeScript wrappers are declared `emit: false`; not emitted.
- **Copilot:** instruction-only deployment, no pre-tool hook here.
- **Gemini CLI, Codex:** instruction-only. The doctrine line in AGENTS.md is the only barrier.
- **Claude's PowerShell tool:** the hook parses its commands too, but the deny globs are written for the `Bash` tool; the permission rule syntax for the PowerShell tool is not pinned here.
- **Fail-open:** a Claude hook that crashes, times out or whose binary is missing allows the call (lesson 264). The retained deny globs are the backstop for the plain `--force` and bare-lease forms in that case; a lease naming the wrong branch is then unguarded.

## Risks / open questions

- **Rollout order.** The hook ships in a released binary, the settings and manifest ship from the repo. Settings deployed before the binary with the guard leave the narrowed globs without the hook, and the new hook entry would call a subcommand the old binary lacks (a non-blocking error per call). Deploy after the release that contains `push-guard`; the PR says so.
- **Stricter than the owner's list.** The issue also pins the destination to the checked-out branch; the owner's list does not mention it. The stricter reading is implemented. Open question for the owner: relax it?
- **Prompt skipping.** Whether a verified lease push should also skip the permission prompt (hook allow decision) is left to the owner; this change does not.
- **Windows path forms.** `cd /c/Users/...` arrives from subagents; a native binary cannot resolve it. The guard normalises `/c/x`, `C:\x`, `C:/x` and relative forms, and refuses the lease when it still cannot resolve a directory.

## Acceptance criteria

- [ ] AC1 A table test of command strings with an allow/refuse verdict covers every form listed under What (allowed shape, each refused form, `git -C`, options before and after the remote, chained commands, `HEAD:refs/heads/main`-style refspecs, `bash -c`, `$(...)`, short clusters, abbreviated long options) and normal pushes that stay allowed; it failed before the guard existed.
- [ ] AC2 `dotf harness push-guard` fed a real Claude-shaped payload on stdin exits 2 with a reason on stderr for a refused push and 0 for an allowed one, and never exits 1 (garbage input, empty input, a non-shell tool).
- [ ] AC3 The checked-out branch and the default branch are resolved from a real temporary git repository (`origin/HEAD` set and unset), not stubbed, on this platform including `/c/...` and `C:\...` directory spellings.
- [ ] AC4 `dotf harness bind` emits the `push-guard` hook beside the `gate` hook on `PreToolUse` without either evicting the other, and the manifest command resolves to a real subcommand.
- [ ] AC5 A test pins the deny list and the allow list of `ai/claude/settings.json`: no `git push` rule in allow, `land:push` in deny, the exact set of force-related deny globs, and every deny glob matched against the command table by the same matcher semantics.
- [ ] AC6 A deploy test shows the old broad glob is retired from a settings file deployed from the previous template, through `mergeIntoWith` with a prior record; the no-record case is stated by a test.
- [ ] AC7 The ADR names the allowed form, the refused forms and the hook; the repository's full gates pass.
