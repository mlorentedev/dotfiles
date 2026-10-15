---
id: "lesson-442-a-declaration-read-from-a-checkout-is-only-as-current-as-the-checkout"
type: lesson
status: active
title: "A declaration read from a checkout is only as current as the checkout"
created: "2026-10-15"
owner: manu
tags: [forge, branch-protection, iac, staleness]
---

# A declaration read from a checkout is only as current as the checkout

**Context:** `dotf forge protection check` and `apply` read `forge/branch-protection.json` from the checkout that contains the cwd (GUARD-017). On 2026-10-15 #2545 merged, changing dotfiles' `required_status_checks.strict` from true to false, and the owner ran `apply` then `check` from the main checkout.

**Problem:** The main checkout had not been pulled since before #2545. `apply` reported `UNCHANGED` and `check` reported `OK` for every repository, while the live forge still had `strict: true` and main declared false. Both commands were right about the file they read and wrong about the question asked. The same stale checkout pointed the other way is worse: run after someone else's declaration change was applied, `apply` would have written the old declaration back to the forge, a revert nobody merged.

**Solution:** `forge.CheckFresh` fetches origin and byte-compares the checkout's declaration with the one origin's default branch holds (`origin/HEAD`, else `origin/main`, else `origin/master`; never a local branch, which would compare the file with itself). `check` and `apply` refuse on a difference, and on an unanswerable one (fetch failed, no default branch): a write needs the network anyway, so there is no offline case to keep. `--unmerged` uses the checkout's file, and `apply` takes it only with `--dry-run`: a preview of a declaration change on its branch, never a write. `dotf doctor` compares with origin as last fetched, without fetching, and WARNs on a difference. Content, not commits: a checkout behind on other files is fresh.

**Why:** Declaration-as-code moves the SSOT into git, but a command that reads a working tree reads one copy of it, of unknown age. When the action is a write to shared state, "what has landed" must be the input, and the command has to establish that itself; "remember to pull" is a convention, and the owner, running it by hand, was the one it failed. Writing a test for this, a local `main` fallback survived a mutation because the no-remote row failed at the fetch first: the row that kills it needs a reachable origin with no default branch, and Git ≥2.48 creates `origin/HEAD` on fetch, so that origin's HEAD must name a branch that does not exist.
