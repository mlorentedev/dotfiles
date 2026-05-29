---
id: copilot-cli-v1-vs-v2-detection
type: troubleshooting
status: active
created: "2026-05-18"
---

# Copilot CLI v1 vs v2 detection in setup scripts

## Symptom

Running `setup-windows.ps1` or `setup-linux.sh` reports:

```text
[INFO] GitHub Copilot CLI extension not installed, skipping Copilot config
       (install with: gh extension install github/gh-copilot)
```

...despite having the (new) GitHub Copilot CLI installed and operative. Following the suggested install command fails with:

```text
"copilot" matches the name of a built-in command or alias
```

## Root cause

Two different products share the "Copilot CLI" name:

1. **Legacy: `gh-copilot` extension (v1, ~2023-2024)** — installed via `gh extension install github/gh-copilot`, invoked as `gh copilot suggest|explain "..."`. Wrapper around suggest/explain API.
2. **New: `copilot` standalone (v2, 2025+)** — installed via `winget install GitHub.Copilot` (Windows). Binary `copilot` directly on PATH. Agentic interface (UX closer to Claude Code than to v1).

`dotfiles` up to PR #48 (2026-05-18, BUG-003) only detected the v1 extension path. On machines with v2 installed, detection failed and `~/.copilot/copilot-instructions.md` was never deployed.

## Resolution

Pull `main` >= commit `49bb58e` (PR #48) and re-run `setup-windows.ps1` or `setup-linux.sh`. The new logic detects via `Get-Command copilot` / `command -v copilot` and deploys to `~/.copilot/` correctly.

Verification: setup log should now show:

```text
[INFO] GitHub Copilot CLI detected at <path>, deploying configuration...
[SUCCESS] copilot-instructions.md deployed successfully (verified pointer to AGENTS.md)
[SUCCESS] GitHub Copilot CLI configured (aliases cop/cops in profile.ps1)
```

Aliases renamed at the same time: `ghcs`/`ghce` (v1) -> `cop`/`cops` (v2). Old aliases no longer defined.

## Related

- BUG-003 PR: #48 (`fix(setup,aliases): detect new standalone Copilot CLI v2`)
- BUG-002 PR: #47 (sibling verify-string drift fix from same audit session)
- Lesson: [`../lessons.md`](../lessons.md) — 2026-05-18 entry on detect-and-act upstream drift
- ADR-010 (pending update post AI-017/AI-018): cross-agent harness parity matrix — v2 changes several cells
