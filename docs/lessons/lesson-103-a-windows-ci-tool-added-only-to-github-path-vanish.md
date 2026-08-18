---
id: lesson-103-a-windows-ci-tool-added-only-to-github-path-vanish
type: lesson
status: active
created: "2026-06-17"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 103: A Windows CI tool added only to $GITHUB_PATH vanishes when setup-windows rebuilds PATH from the registry

**Context**: BUG-025 (#425) replaced `choco install age.portable` in the `test-windows` job with a deterministic download of the pinned `age` release, wired onto PATH via `$GITHUB_PATH`, to kill the chocolatey-shim PATH-propagation flake (sibling of the `eza`/`zoxide` flake BUG-024).

**Problem**: `test-windows` went red on one deterministic failure — `pi`'s deployed `models.json` kept an unresolved `{env:NAN_API_KEY}` placeholder. The secrets sandbox encrypts a throwaway `nan.api-key.secret.age` and `setup-windows.ps1` decrypts it through `Substitute-EnvPlaceholders` → `& age --decrypt`. With choco, `age` lived on the **Machine registry PATH** and was always resolvable; the release-zip approach put it *only* on `$GITHUB_PATH` (a process-level PATH injection the runner applies to subsequent steps). But `setup-windows.ps1` rebuilds `$env:PATH` from the Machine+User registry mid-run (so freshly-installed tools are usable immediately), and that rebuild **discards** the process-only `$GITHUB_PATH` entry. So `& age` silently failed — its stderr was swallowed with `2>$null` — the secret never decrypted, and the placeholder survived into the deployed config. Two things hid the cause: the swallowed stderr masked the real error, and `test-windows` runs only on `pull_request` events, so every `push` to main shows it `skipped` — "main is green" was a false signal (the prior green came from PRs #413/#419/#421, which still used choco).

**Solution**: Persist `age` on the **User registry PATH** as well (`[Environment]::SetEnvironmentVariable('PATH', "$userPath;$ageDir", 'User')`) so it survives `setup-windows`'s registry-PATH rebuild — restoring the exact property the old choco Machine-PATH install had. The `eza`/`zoxide` failures in the same job were a *genuine* winget→PATH flake (BUG-024), proven by re-running the job (they passed on retry); the NAN failure reproduced across every re-run, which is what separated the real regression from the flake.

**Rule**: On Windows CI, any tool the *deploy script itself* must resolve at runtime has to be on the **registry PATH (Machine/User scope)**, not just `$GITHUB_PATH` — a script that rebuilds `$env:PATH` from the registry (a common pattern after installing tools) discards process-only additions. Never swallow a binary's stderr (`2>$null`) on a code path that can fail silently: a hidden `age --decrypt` error turned a one-line PATH bug into a cryptic downstream placeholder. And re-run a red job before declaring a regression — a failure that reproduces is a bug, one that clears is a flake; here a single job carried both at once.
