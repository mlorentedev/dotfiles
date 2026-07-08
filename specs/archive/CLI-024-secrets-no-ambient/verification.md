---
tags: [spec, verification, secrets, shell]
created: "2026-06-25"
---

# Verification - CLI-024-secrets-no-ambient

> Phase 1b. Branch `feat/secrets-no-ambient-export` (stacked on `feat/secrets-run-jit`).

## AC1/AC2 — sourcing removed, wrappers added

- `.bashrc`/`.zshrc`: the `source load-secrets.sh` line is replaced by `opencode()/pi()/agy()` wrappers calling `dotf secrets run`, guarded by `command -v dotf`.
- `profile.ps1`: the `load-secrets.ps1` dot-source block is replaced by `function opencode/pi/agy { dotf secrets run -- … @args }`, guarded by `Get-Command dotf`.

## AC3 — RC files parse cleanly + ASCII

- `bash -n .bashrc` / `bash -n .zshrc` → OK.
- `profile.ps1` → PowerShell parser: 0 errors (850 tokens).
- No non-ASCII introduced in the shell RC (an em-dash + `→` I first added were replaced with ASCII).

## AC4 — bats green with the new contract

`bats tests/powershell-profile.bats` → **14/14 ok**, including the two rewritten assertions:

```
ok 7 profile.ps1 does NOT auto-source load-secrets and wraps AI CLIs via dotf secrets run (ADR-028)
ok 8 parity: neither .bashrc nor profile.ps1 auto-loads secrets; both wrap the AI CLIs
ok 14 profile.ps1 valid PowerShell syntax (if pwsh available)
```

No other test asserts the removed sourcing (`grep -rn` over `tests/`): `setup-windows.bats:76` asserts the **setup eager-load** (left intact — out of scope), and `load-secrets.bats` tests the **script** (untouched).

## Deferred (noted in proposal)

- setup-{linux,windows} eager-load migration; deleting the load-secrets twins; per-tool `--only` scoping.

## Merge gate

Stacked on #579 — merge **after** #579 and after a `dotf` redeploy shipping `dotf secrets run`.
