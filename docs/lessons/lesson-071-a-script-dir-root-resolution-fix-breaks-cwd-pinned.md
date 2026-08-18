---
id: lesson-071-a-script-dir-root-resolution-fix-breaks-cwd-pinned
type: lesson
status: active
created: "2026-05-31"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 071: A SCRIPT_DIR root-resolution fix breaks CWD-pinned fixture tests — add an env-override seam, not a code branch

**Context:** PR #192 changed compile-harness.sh root resolution from `git rev-parse --show-toplevel` (CWD-based) to SCRIPT_DIR-based (`cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.."`), to fix the section-12 healthcheck drift false-fail on the non-git ~/.dotfiles copy-deploy (ADR-012), where `git rev-parse` errors.
**Problem:** 26 compile-harness.bats tests run the REAL script against a throwaway fixture they `cd` into, relying on CWD-based (git-toplevel) root resolution to make the fixture the root. SCRIPT_DIR resolution instead pointed the script at the LIVE repo, so --refresh/--deploy operated on the wrong tree → mass failure. The two goals (robust resolution for the deploy copy vs. arbitrary fixture root for tests) are irreconcilable with a single hard-coded REPO_ROOT. A second, brittle test also broke: a guard that greps healthcheck.sh for an exact failure-message string, after #192 reworded the message.
**Solution:** Introduce an explicit env override: `REPO_ROOT="${HARNESS_REPO_ROOT:-$(SCRIPT_DIR-based)}"` — the SAME idiom the script already uses for VAULT_PATH. Tests `export HARNESS_REPO_ROOT="$REPO"` once in setup() to pin the fixture; production never sets it and keeps the SCRIPT_DIR default. One seam, two correct behaviors, zero production code branch. Generalize the existing override idiom rather than special-casing tests. Corollary: prefer grepping a stable SUBSTRING of a message (or asserting behavior) over an exact full-string match, so message rewording doesn't break the guard.
**Tags:** `#testing` `#bats` `#root-resolution` `#deploy-model` `#env-override` `#compile-harness`
