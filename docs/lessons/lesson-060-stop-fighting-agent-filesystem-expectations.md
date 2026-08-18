---
id: lesson-060-stop-fighting-agent-filesystem-expectations
type: lesson
status: active
created: "2026-05-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 060: Stop fighting agent filesystem expectations

**Context:** SDD-007 was triggered by BUG-100 — `agy` (Antigravity CLI) v1.0.2 collided with our deploy strategy: agy writes in-place to `~/.gemini/...` paths, but our `setup-linux.sh` had placed symlinks at those paths pointing back to the repo. Symptoms: EEXIST errors (forum #145851), circular link traversal (gemini-cli issue #10960), silent state corruption. The reflex fix would have been "patch each collision case as it surfaces". The root-cause fix was different.

**Problem:** Our deploy strategy was *fighting the agent's expected filesystem layout*, not the agent being buggy. Every new agent we adopt (today: agy, opencode, claude-mem; tomorrow: Cursor, Codex, Devin, whatever) has its own conventions for what it writes-in-place, what it expects to be a regular file, what it tolerates as a symlink. If our deploy strategy hardcodes a single mechanism (symlinks pointing back to the repo), we will hit BUG-100-class incidents N times — once per agent that disagrees with that mechanism.

**Solution:** Default deploy mechanism = **copy** (atomic via `deploy_file` helper, idempotent via `cmp -s`). Symlinks become the *exception*, reserved for: (a) intentional vault↔home bindings (Obsidian vault content → memory paths), (b) secret files needing canonical absolute paths (`~/.ssh/id_ed25519`, `~/.config/age/key.txt`). All other config paths get copies. The "edit-in-`~`-silently-loses-change" failure mode that copies introduce gets neutralized by a `check_deployed` drift assertion in `scripts/healthcheck.sh` — drift surfaces loud red. ADR-012 captures the decision.

**Why:** Symlinks express "this path IS the repo file" — that's a strong claim about filesystem identity that not every consumer of the path respects. Copies express "this path is *a copy of* the repo file at deploy time" — a weaker, more universal claim. The weaker claim is portable across N agents; the stronger one only works as long as all consumers cooperate. SDD-008 will extend this discipline to skills (vault `00_meta/skills/` → render via copy to each agent's native path, no symlinks).

**How to apply:** When adding a new deploy target (any path under `~/.*/` that the repo owns), default to `deploy_file` (copy). Only use symlinks when there's a SPECIFIC, documented reason (cross-system binding that must stay live; secret with hard-coded absolute path requirement). Update the `DEPLOYED_FILES` registry so the drift assertion covers the new path. If the target is consumed by a CLI tool that writes-in-place to its own configuration directory, *always* use copy — don't even debate it.

**Tags:** `#deploy` `#cross-agent` `#filesystem` `#sdd-007` `#bug-100` `#adr-012` `#dotfiles`
