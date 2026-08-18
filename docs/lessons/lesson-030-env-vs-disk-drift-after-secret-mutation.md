---
id: lesson-030-env-vs-disk-drift-after-secret-mutation
type: lesson
status: active
created: "2026-05-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 030: Env-vs-disk drift after secret mutation

**Context:** While diagnosing dotfiles issue #7 (https://github.com/mlorentedev/dotfiles/issues/7) where `secrets_rotate` appeared to silently fail to update the encrypted file. Investigation showed the encrypted .age file WAS being updated correctly on disk (mtime + SHA256 confirmed change after rotate), but every consumer of the secret saw the old value.
**Problem:** load-secrets.sh exports `$VAR` once at shell startup. `secrets_rotate` updated the on-disk .age file but did NOT re-export `$VAR` in the current shell. Any subsequent read of `$VAR` (gh CLI, curl, scripts, even `secrets_show` without `--raw`) returned the cached old value, indistinguishable from a real failure. This wasted ~30min of debugging adding instrumentation to age_encrypt before realizing the encryption was fine. The issue was env-vs-disk drift between shell state and disk state, with no automatic reconciliation. Compounded by: `_secrets_sync_to_repo` silently no-op'd when `DOTFILES_REPO_DIR` was unset, so the user's primary verification step (git status in the repo) could miss real updates. Also the project lacked a `secrets_remove` function, so deletion was a manual three-step (edit mapping + rm .age + manual sync) that bypassed audit logging.
**Solution:** After any mutating secret operation (add/rotate/remove/add_file), auto-update the current shell so `$VAR` matches disk immediately — eliminate the manual `secrets_refresh` step that users forget. Concretely: rotate/add do `export_var "$var" "$value"`; remove does `unset_var`; add_file calls `secrets_refresh` (file deployment is non-trivial). Make `_secrets_sync_to_repo` warn loudly to stderr when no repo is resolved, and add an auto-detect default of `$HOME/Projects/dotfiles` when `DOTFILES_REPO_DIR` is unset. Add a missing `secrets_remove VAR_NAME [--yes]` that handles plain + file secrets, syncs deletions to repo, and updates audit log. Generalizes to: any system where the source of truth (disk) and the working copy (env, cache, deployed file) can diverge MUST either auto-reconcile after mutation OR loudly warn — silent drift is the worst failure mode because it produces false bug reports and wastes investigation time.
**Tags:** `#secrets` `#shell` `#env-drift` `#ssot` `#false-positive-debugging`
