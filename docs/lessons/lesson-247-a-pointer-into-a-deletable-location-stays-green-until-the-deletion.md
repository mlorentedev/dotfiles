# Lesson 247 — A pointer into a deletable location keeps every check green until the deletion, and then the cleanup gets blamed

**Date:** 2026-08-31
**Context:** A routine worktree cleanup on msi. `dotfiles-wt-gentleai` held `feat/HARNESS-105`, merged as #1404; the worktree was clean, unoccupied, and its branch content verified byte-identical to the squash. Removing it immediately broke `pre-commit` for every session on the box: `Run dotfiles tests` failed 77/78 on `FAIL: DOTFILES_REPO_DIR: directory missing`, and no commit could land.
**Category:** paths, guards, adr-025, shared-surface

## What happened

`~/.config/dotfiles/machine.json` — the ADR-025 SSOT for per-machine path
resolution — read:

```json
{ "paths": { "DOTFILES_REPO_DIR": "/home/manu/Projects/dotfiles-wt-gentleai" } }
```

Some earlier session had persisted its **current working directory** as the
canonical repo dir, and that directory was a throwaway worktree. The value was
wrong the moment it was written: a worktree for a feature branch is a temporary
view of the repo, never the repo. Nothing detected it for as long as the
directory happened to exist.

The tempting reading of the incident is "the cleanup broke the build". The
accurate one is the reverse. `dotf doctor` already carries the right check, and
it fired the instant it could:

```
[FAIL] DOTFILES_REPO_DIR=/home/manu/Projects/dotfiles-wt-gentleai (path does not exist)
```

The guard was never broken. It was **unable to fire**, because the only symptom
it can observe is absence, and the wrong path was present. Deleting the worktree
did not introduce the defect; it converted a silent one into a loud one, which is
the direction this repo wants and the direction that gets misattributed.

## The rule

**A check that passes because of an accident and a check that passes because of
correctness are indistinguishable, and removing the accident is usually how you
find out which one you had.** So when a deletion breaks something, the first
question is not "what did I break" but "what was depending on this that should
never have depended on it".

Two corollaries earned the hard way in the same session:

- **A write-side that accepts the caller's cwd will eventually record a
  disposable location.** `dotf env set` takes an explicit path for exactly this
  reason; whatever wrote the worktree path did not. A path SSOT should refuse, or
  at least warn on, a value inside `.git/worktrees` — the repo has a canonical
  checkout and a worktree is never it.
- **Fix the SSOT and you have still not fixed the running processes.** The
  resolution cascade prefers the environment variable over `machine.json`, so
  after `dotf env set` + `dotf env generate` the repaired value was invisible to
  every shell already started:

  ```
  dotf env path DOTFILES_REPO_DIR                     # -> the deleted worktree
  env -u DOTFILES_REPO_DIR dotf env path DOTFILES_REPO_DIR   # -> /home/manu/Projects/dotfiles
  ```

  Only the controlled environment shows the truth on disk. This is lesson 235's
  claim again in a new place: *"I cannot reproduce it" is a statement about the
  instrument before it is a statement about the system* — and its mirror, *"I
  fixed it" is a statement about the instrument too* until you re-measure without
  the inherited state.

## Second instance, same shape, same session

Before the cleanup, deciding whether a peer session's worktree was free was done
by asking whether its working tree was clean. It was clean, and it was **not**
free: a `claude` session had started there two minutes earlier and had not yet
written anything. A rebase and force-push were one step away.

Cleanliness is a property of the tree; occupancy is a property of the processes.
The instrument that answers the real question is:

```sh
for p in $(pgrep -f claude); do printf '%s -> %s\n' "$p" "$(readlink /proc/$p/cwd)"; done
```

Same failure as the pointer above: a signal that reads green for as long as an
unrelated accident holds — here, that the occupying session had been idle just
long enough to leave no trace.

## How it was caught

By a hook refusing a commit, which is the cheap end of this. The expensive end
was already visible in the same session and is worth stating: `dotf doctor` had
been reporting this box healthy on that key for as long as the stale worktree
survived, and any machine whose `machine.json` names a still-existing wrong path
is being reported healthy right now.
