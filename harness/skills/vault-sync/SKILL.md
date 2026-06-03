---
name: vault-sync
description: "Vault git synchronization protocol: pull before read, pull --rebase before write, push after write. Handles divergence, conflict resolution, and recovery. Follow this whenever interacting with the knowledge vault."
---

# Vault Sync Protocol

## Overview

The knowledge vault is a git repository shared across multiple agents (Claude Code, OpenCode, Hermes Agent) and humans (Manu via Obsidian). Concurrent writes are the norm, not the exception. This protocol ensures safe synchronization without data loss or force pushes.

**Core principle:** The vault is always evolving. Never assume your local copy is up to date. Sync before every read, every write, and every push.

## The Iron Law

```
PULL BEFORE READ  |  PULL --REBASE BEFORE WRITE  |  PUSH AFTER WRITE
```

- **Skip pull before read?** You read stale data. Decision based on stale data.
- **Skip pull --rebase before write?** Push gets rejected. You have to unwind.
- **Skip push after write?** The vault diverges from remote. Next agent starts behind.

There is no shortcut. Do all three, every time.

## When to use

- **Every** vault interaction — whether reading, writing, or searching
- Before starting a session that will use vault data
- After any vault write (file create, edit, delete)
- When Hive MCP returns stale or inconsistent data
- When `vault_health` reports divergence between local and remote

## When to skip

- Read-only operations on the local vault that do NOT need to be current (rare — almost always sync first)
- Emergency operations where the vault is inaccessible and you're writing to /tmp/ for later manual sync

## The protocol

### Phase 1 — Pull before read

```bash
# Before ANY read, search, or query on the vault:
cd /tmp/hermes-vault && git pull --rebase
```

This ensures you're working with the latest state. If the vault has been modified by another agent or by Obsidian sync, you pick up those changes immediately.

**Why --rebase?** It keeps history linear. Merge commits in the vault create noise and complicate history bisection.

### Phase 2 — Pull --rebase before write

```bash
# Before ANY write, edit, or delete on the vault:
cd /tmp/hermes-vault && git pull --rebase
```

Even if you just pulled in Phase 1, another agent may have pushed since then. Pull again immediately before writing to minimize the chance of a non-fast-forward rejection.

### Phase 3 — Write using Hive MCP (preferred)

```python
# Use Hive tools for vault operations:
# vault_write, vault_patch, vault_create
# Hive auto-commits with message "vault: ..."
```

Hive's auto-commit integrates cleanly with git history. No manual `git add`/`git commit` needed.

### Phase 3b — Native write (fallback)

```bash
# Only when Hive MCP is unavailable:
# 1. Make changes via Write / Edit / patch
# 2. Stage and commit manually:
cd /tmp/hermes-vault && git add -A && git commit -m "vault: <description>"
```

Always use the `vault:` prefix to match Hive's convention.

### Phase 4 — Push after write

```bash
cd /tmp/hermes-vault && git push
```

If the push is rejected (non-fast-forward), proceed to Phase 5.

### Phase 5 — Divergence recovery

If `git push` fails with a non-fast-forward error:

```bash
# 1. Pull --rebase to reapply your commits on top of remote
cd /tmp/hermes-vault && git pull --rebase

# 2. Resolve any conflicts
# 3. Push again
git push
```

**Never use `git push --force` or `git push --force-with-lease` on the vault.** Force pushes rewrite history and corrupt the shared timeline. If rebase fails due to conflicts:

1. Use `git status` to identify conflicted files
2. Resolve each conflict manually
3. `git add <resolved-file>` for each
4. `git rebase --continue`
5. `git push`

If you cannot resolve a conflict (e.g., large auto-generated file), escalate to Manu.

## Error handling matrix

| Symptom | Cause | Action |
|---------|-------|--------|
| `push rejected (non-fast-forward)` | Remote has commits you don't | `git pull --rebase`, then push |
| `pull --rebase fails with conflicts` | Same file changed in both | Manual conflict resolution |
| `Could not resolve host` | Network down | Cache changes locally, retry later |
| `fatal: not a git repository` | Vault not cloned | Run setup.sh or git clone |
| `Permission denied (publickey)` | Wrong remote URL or no SSH key | Use HTTPS with token |
| `remote: Invalid username or password` | Token expired or invalid | Ask Manu for new token |
| `hint: diverged branches` | rebase detected divergence | Run `git pull --rebase` again, should auto-resolve |

## Conflict resolution

When `git pull --rebase` produces merge conflicts:

1. **Identify conflicted files:**
   ```bash
   git status
   # Look for files under "both modified" or "Unmerged paths"
   ```

2. **Resolve each conflict:**
   - Open the file and find `<<<<<<<`, `=======`, `>>>>>>>` markers
   - Decide which version to keep (or merge both)
   - Remove the conflict markers
   - Save the file

3. **Mark as resolved:**
   ```bash
   git add <file>
   ```

4. **Continue rebase:**
   ```bash
   git rebase --continue
   ```

5. **If stuck:**
   ```bash
   # Abort the rebase and restore original state
   git rebase --abort
   # Then escalate to Manu
   ```

## Automatic sync via Hive MCP

When using Hive MCP tools (`vault_write`, `vault_patch`, `vault_create`):

- Hive handles git commit automatically (message format: `vault: ...`)
- Hive does NOT auto-push. You must still run Phase 4 (push) after Hive operations.
- Hive does NOT auto-pull. You must still run Phase 1-2 before Hive operations.

**Reasoning:** Push and pull require network access and may fail. Hive defers those to you so you can handle retries and divergence.

## Pitfalls

- **Assuming your local copy is current.** It's not. Another agent or Obsidian sync may have pushed since your last pull.
- **Skipping pull before read.** Stale data leads to bad decisions.
- **Forgetting to push after write.** Your changes exist only on this machine. If the server dies, they're gone.
- **Using git push --force.** This is a shared repo. Force push destroys history. Never.
- **Multiple writes without intervening pushes.** Batch your writes, but push after each logical batch. Don't accumulate 10 commits and push once — that maximizes the chance of divergence.
- **Committing when Hive already auto-committed.** Double commits duplicate history. Check `git status` first.

## Verification

After completing the sync cycle:

1. Confirm remote is up-to-date:
   ```bash
   cd /tmp/hermes-vault && git status
   # Should say: "Your branch is up to date with 'origin/master'."
   ```

2. Confirm no uncommitted changes:
   ```bash
   git status --porcelain
   # Should be empty
   ```

3. Confirm last commit message matches vault convention:
   ```bash
   git log -1 --oneline
   # Should start with "vault:"
   ```

## Related

- `00_meta/patterns/pattern-hive-first-vault-access.md` — when to use Hive vs. native tools
- `00_meta/skills/verification-before-completion/SKILL.md` — verify before claiming completion
- `00_meta/skills/handoff/SKILL.md` — session handoff includes vault sync status
