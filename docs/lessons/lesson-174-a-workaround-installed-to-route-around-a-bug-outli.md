---
id: lesson-174-a-workaround-installed-to-route-around-a-bug-outli
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 174: A workaround installed to route around a bug outlives the bug silently, because nothing re-examines it

**Context**: `dotf doctor` reported the GUARD-001 memory-sink guard healthy. It was not running in the dotfiles repo at all — the repo carried a local `core.hooksPath` override pointing at its own `.git/hooks`, and local scope beats the global dispatcher.

**Problem**: The override was almost certainly installed to route around the dispatcher bug later filed as BUG-055: with the fallback crashing, pointing `core.hooksPath` at the repo's own hooks was the only way to keep `git commit` working there. That also explains why dotfiles was the single repo immune to a machine-wide breakage. Once BUG-055 was fixed the workaround had no purpose, but nothing re-examines a config value after the reason for it disappears — so it kept quietly disabling a security guard in the repo that ships that guard. Removing it revealed a second casualty nobody had noticed: `validate-commit-msg` had never run in this repo either.

**Solution**: Probe before removing (`~/.dotfiles/git-hooks/commit-msg` and `pre-commit` both exit 0 now), then unset, then verify by effect rather than by path — stage a `memory/MEMORY.md` and confirm the commit is blocked with exit 1, and confirm a clean tree still exits 0.

**Rule**: A local override is a decision with no expiry date and no owner. When you fix a bug, search for the workarounds it caused — they are invisible precisely because they succeeded, and each one may be suppressing something unrelated. The tell that the *check* is also wrong: `dotf doctor` reported "all ok" both before and after the guard went from inert to active, because it read the global scope while git resolves the effective one. A guard check must ask *does this run here*, never *is the machine wired correctly* — the same distinction BUG-040 already ruled once.
