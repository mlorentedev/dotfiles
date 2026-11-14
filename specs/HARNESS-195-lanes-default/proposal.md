---
id: "HARNESS-195-lanes-default"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-13"
issue: "mlorentedev/dotfiles#3432"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# HARNESS-195-lanes-default

> **Naming**: file lives at `<repo>/specs/HARNESS-195-lanes-default/proposal.md`. `HARNESS-195-lanes-default` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3432: Lanes as the default for every session, agent and machine -->

ADR-060 decided on persistent named lanes on 2026-11-07, but no session uses them. On 2026-11-12, no repository carried `.dotf/lanes.json`, no machine had a lane directory, and dotfiles alone had 41 branch-per-directory worktrees on the Mac. A session's working directory is fixed when it starts, so nothing reaches a lane unless the command that starts the session routes it there. Today that path:
- launches only claude, pi and agy;
- needs herdr;
- arms machine-wide from an uncommitted file;
- is invisible to the session that ends up in it.

The ADR-060 amendment of 2026-11-12 (#3435) settled the policy. This spec builds the path, so that any agent, on any OS and machine, works in a lane by default.

## What

After this spec, in a repository that carries the committed `.dotf/lanes.json`:

1. **Each machine activates through converge.** A `lanes` converge step reads the committed marker on the default branch, creates the three lanes (`feat-lane-primary`, `feat-lane-secondary`, `fast-track-lane`), and writes an activation record. A re-run reports `changed=0`.
   - **Decision: where the record lives.** The record is a file named `dotf-lanes-active` in the repository's git common dir. Only converge writes it. It is not committed and not in any working tree, so `git clean -fdx` cannot remove it (lesson 597).
   - **Decision: which repositories converge activates.** Only the repositories it manages, and the deploy clone (`DOTFILES_DIR`) is never one of them. The exemption therefore needs no match on the clone's name or path.
   - Nothing else activates a machine: not `dotf lane init` run by hand, not a marker in any working tree, not the deploy clone.
2. **The guard arms from the activation record and from nothing else** (#3437). The guard drops both marker lookups, `$top/.dotf/lanes.json` and `$main_root/.dotf/lanes.json`. Otherwise any checkout of main that carries the committed marker would arm itself. The deploy clone (`DOTFILES_DIR`) is never armed. An expired lease and a branch worktree with no lease are reported, not ignored (#3438).
3. **Every session is told where it stands.** `dotf mem resume` and `dotf mem session-start` print one line, and the line reaches each harness through the surface it has:
   - "lane `<name>` ACTIVE on `<branch>`", when the session is in a lane;
   - "this repository runs lanes: start with `dotf lane switch <branch>`", when it is outside them.

   - **claude:** through the `mem session-start` hook already bound in `harness/manifest.json`.
   - **pi and opencode:** the manifest declares a `dotfiles-gate.ts` session-start extension for each, but nothing writes either file (`emit: false`; `docs/runbooks/guide-agent-provisioning.md` says "not written by bind"). Neither exists on the Mac. This spec builds one small extension per harness that runs `dotf mem session-start` at session start and adds its output to the session's context: pi's `session_start` event, and opencode's `chat.system.transform`, which injects into the system prompt rather than printing. dotf deploys both.
   - **agy:** its hooks document has no session-start event, only `PreInvocation`, which fires before each turn. The spec binds a `PreInvocation` hook that prints the line once per session. Whether agy shows that hook's output to the model is unverified; the observed launch (AC9) decides. If it does not, agy joins copilot and codex below.
   - **copilot and codex:** they have no hook surface yet (#803). They get the line from `dotf mem resume`, which AGENTS.md tells every session to run first. This is the one non-mechanical path, stated as such, and it closes when they gain hooks.

4. **`dotf worktree add` coexists with lanes.** In a repository with active lanes it still creates the worktree, and prints the lane command to use instead. Detached worktrees, and those `spec review` creates, print nothing.
5. **`dotf lane next` works without herdr.** With no backend, the printed commands enter the lane before they start the agent. The same commands run under zsh, bash and PowerShell.
6. **More agents can be launched.** copilot and opencode gain launchers after an observed launch (the CLI-119 rule). codex and gemini gain theirs when they are installed and observed. An npm `.cmd` shim stays refused on Windows (#3442), and `dotf secrets run` stops reaching one (#3440).
7. **`dotf lane doctor` reports what blocks a commit:**
   - detached worktrees with no lease (the patch preserved on #3327);
   - expired leases;
   - a deployed `lane-guard.sh` that lags the repository's copy.
8. **The doctrine names one way in.** In a repository with lanes, AGENTS.md and the `using-git-worktrees`, `handoff` and `autopilot` skills say that work starts with `dotf lane switch` or `dotf lane next`. AGENTS.md's implementation tiers stop being called "lanes", because the name collides with ADR-060's lanes.

## Out of scope

- Rolling out to kubelab and the other repositories (#3154 and #3318). This spec ends with dotfiles active on the Mac, the Linux guest and the Windows guest.
- Removing the in-tree lease mirror (#3443, HARNESS-194).
- A tmux backend. The no-backend path (item 5) is the agnostic floor; tmux is an optimisation for later.
- Making the lane count configurable. The ADR-060 amendment fixes it at three on purpose.
- Re-rendering the harness from the vault (#3456). It is a prerequisite for the doctrine PR and gets its own PR.

## Risks / open questions

These are true unknowns. The design decisions are recorded in **What**.

- **Unverified on Windows:** the guard under Git for Windows `sh`, and lane init, switch and park. Acceptance runs on the Windows guest (`dotf vm windows`).
- **opencode `--prompt` may pre-fill the input box without submitting it.** The observed launch decides. If it does not submit, opencode gets no launcher and that is recorded, not guessed around.
- **The sweep queue.** Activating dotfiles turns the existing worktrees, 41 on the Mac on 2026-11-12, into the sweep's queue. The sweep acts only on merged branches. Unmerged worktrees stay until their PRs land, and the guard lets them through because a branch worktree passes.

## Acceptance criteria

- [ ] AC1. On a machine whose default branch carries `.dotf/lanes.json`, `dotf converge --only lanes` creates the three lanes and the activation record, and a second run reports `changed=0`. A marker that exists only in a working tree activates nothing.
- [ ] AC2. With an activation record, the guard blocks a commit on `main` in the main checkout and lets a commit through on a branch worktree. Without one, it blocks nothing. In the deploy clone it never blocks. These are tested on Linux, macOS and Windows CI legs.
- [ ] AC3. `dotf mem resume` and `dotf mem session-start` print the lane line from a lane and from a non-lane worktree of an active repository, and print nothing for a repository without lanes. claude's hook, the pi and opencode extensions and agy's `PreInvocation` hook are deployed by dotf. A test fails if a harness whose manifest entry declares a session-start presence has nothing deployed to carry the line.
- [ ] AC4. `dotf worktree add` in an active repository prints the `dotf lane switch` hint and still creates the worktree. `AddDetached` prints no hint.
- [ ] AC5. `dotf lane next --dry-run` with no backend prints commands that end inside the lane directory before the agent command. A golden test covers the zsh, bash and PowerShell forms.
- [ ] AC6. `launchers` holds an entry for each agent with a recorded observed launch (`LANE-READY`) in `verification.md`. `dotf secrets run` refuses an npm `.cmd` shim on Windows the same way `lane exec` does.
- [ ] AC7. `dotf lane doctor` reports a detached worktree with no lease, an expired lease, and a drifted deployed guard, each with its fix. Exit status is unchanged for warnings.
- [ ] AC8. AGENTS.md, the `dotf init` template that seeds other repositories' AGENTS.md (`cli/internal/initrepo/templates/agents-spec-section.md`) and the three skills name the lane entrances. Neither AGENTS.md nor the template calls the implementation tiers "lanes", and a test fails if either does again.
- [ ] AC9. dotfiles is active on the Mac, the Linux guest and the Windows guest. A real session of each launched agent starts in a lane and reports it through AC3's line. The evidence is recorded in `verification.md`.

## References

- Bitácora board: #3432 (epic), with #3327, #3437, #3438, #3440, #3430 and #3436 as its children
- Related ADRs: `docs/adr/adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md` (Amendment 2026-11-12), `docs/adr/adr-061-lane-identity-in-the-worktree-git-dir.md`, `docs/adr/adr-062-the-lease-lives-in-the-worktree-git-dir.md`
- Related specs: `specs/archive/CLI-119-lane-next/` (the observed-launch rule), `specs/archive/HARNESS-193-lease-in-git-dir/`
- Lessons: 568 (key activation on the signal the feature writes), 594 (npm `.cmd` shims and cmd.exe), 597 (excluded is not safe from `git clean`)
