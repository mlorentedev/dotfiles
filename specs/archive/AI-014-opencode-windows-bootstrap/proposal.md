---
id: "AI-014-opencode-windows-bootstrap"
type: spec
status: implementing
created: "2026-05-21"
tags: [spec, proposal, opencode, multi-agent, cross-os-parity]
template_version: "1.0"
---

# AI-014-opencode-windows-bootstrap

## Why

OpenCode is the secondary AI coding agent (ADR-009 Multi-Agent Runtime). The Linux side shipped 2026-05-16 via AI-011 (PRs #34/#36/#37): `setup-linux.sh` installs the binary via `curl -fsSL https://opencode.ai/install | bash`, deploys `opencode.jsonc`, deploys all command files from `ai/opencode/commands/`. The Windows side has been a known gap — `healthcheck.ps1` sec 10/12 emits two SKIPs (`opencode binary - not installed (AI-014 admin-conditional)`, `opencode.jsonc - not deployed at ... (AI-014 pending)`) on every run, and the user has been explicit (2026-05-21 session) that they want OpenCode running on Windows soon. The original AI-014 backlog entry assumed Windows would need an "admin-conditional curl-bash equivalent or `npm i -g opencode-ai`" with graceful-skip logic for non-admin machines — empirical probe 2026-05-21 against winget revealed `SST.opencode` (v1.15.6) is published natively, user-scope installable, with PATH integration handled. That is a substantially cleaner path than either of the originally hypothesised options and reduces this spec from ~150 LOC of admin-detection logic to ~50 LOC mirroring the existing winget tools array pattern.

## What

After this PR, a fresh `setup-windows.ps1` run on a Windows machine with `winget` available will:

1. Install the OpenCode binary via `winget install SST.opencode` (joins the existing `$tools` array in section 1c). User-scope, no admin, idempotent — winget itself skips when the package is already installed at the requested version.
2. Deploy `ai/opencode/opencode.jsonc` to `$env:USERPROFILE\.config\opencode\opencode.jsonc` (reconcile-not-skip pattern: byte-compare against source, copy on drift).
3. Sync `ai/opencode/commands/*.md` to `$env:USERPROFILE\.config\opencode\commands\` (add new, leave unchanged, remove orphans) so `/audit`, `/test`, `/writing-plans`, etc. are available from any cwd where the user launches `oc`.
4. Healthcheck sec 10/12 transitions from SKIP to PASS on both checks (binary in PATH, jsonc deployed with `$schema`).

The `/connect` flow (interactive auth to the Go subscription) stays manual, identical to the Linux runbook. Profile alias `oc` is already conditional on `Get-Command opencode` since 2026-05-16, so it activates the moment the binary is on PATH.

## Out of scope

- Porting `scripts/skills-to-opencode.sh` (the regenerator) to PowerShell — Windows reads the committed `ai/opencode/commands/*.md` directly; regeneration is a Linux/CI-side concern.
- Automating the `/connect` interactive flow — requires pasting an API key, intentionally manual per the Linux runbook.
- A separate Windows runbook — the `guide-opencode-go-setup.md` runbook is OS-agnostic for the `/connect`/PAYG-guardrail/troubleshooting sections; a small Windows install delta gets folded as a section in that same runbook post-merge, not a parallel file.
- Provider configuration variations (`opencode.jsonc` overrides per-OS) — the same canonical config ships to both OSes; if a Windows-specific divergence becomes necessary it gets its own ticket.
- Ghostty Windows (TERM-002) — orthogonal terminal emulator concern.

## Risks / open questions

- **Risk: `winget` not on PATH on a fresh Windows install.** Mitigation: the entire developer-tools block is already gated on `Get-Command winget` (setup-windows.ps1:260). If absent, every winget tool — including OpenCode — gracefully skips with a clear `[WARNING] winget not found` line. Same blast radius as `gh`/`zoxide`/`age` install.
- **Risk: winget package `SST.opencode` becomes stale relative to the upstream `curl|bash` installer.** Mitigation: the package is maintained by the OpenCode team (publisher `SST`); historical release cadence on winget tracks upstream within hours. If divergence becomes chronic, future ticket switches to the `npm install -g opencode-ai` fallback (also empirically validated 2026-05-21: registry has v1.15.7).
- **Risk: opencode.jsonc deployment overwrites user-side experiments.** Mitigation: the reconcile-not-skip pattern only copies when source and destination differ (per-byte `Compare-Object` or `cmp` equivalent). Identical to the Linux block's `cmp -s` behaviour. Users editing the deployed file should commit changes back to the repo source-of-truth or accept they will be reconciled away — same contract as on Linux.
- **Risk: commands directory ends up with stale files when a skill is removed from `ai/skills/`.** Mitigation: orphan removal walks the destination and deletes any `*.md` not present in the source (mirror of `setup-linux.sh:455-461`).

## Acceptance criteria

- [ ] `setup-windows.ps1` adds `SST.opencode` to the `$tools` array in section 1c. Idempotent via the existing per-tool `Get-Command` check.
- [ ] `setup-windows.ps1` deploys `ai/opencode/opencode.jsonc` to `$env:USERPROFILE\.config\opencode\` using a reconcile-not-skip block (byte-equality check via `Get-FileHash` or `Compare-Object` before copy).
- [ ] `setup-windows.ps1` syncs `ai/opencode/commands/*.md` to `$env:USERPROFILE\.config\opencode\commands\` (add new, leave unchanged, remove orphans).
- [ ] `scripts/healthcheck.ps1` sec 10/12 wording for the SKIP-when-absent branches updated to remove the "AI-014 pending" qualifier (the present-but-not-installed case still SKIPs with the original "not installed" reason, just without the parenthetical pointer).
- [ ] `tests/setup-windows.bats`: bats asserts lock the `SST.opencode` entry in `$tools`, the reconcile-not-skip block for `opencode.jsonc`, and the commands sync block.
- [ ] `Invoke-ScriptAnalyzer -Settings .PSScriptAnalyzerSettings.psd1 -Severity Error,Warning setup-windows.ps1` → clean (no em-dash regression like BUG-014).
- [ ] PowerShell AST parse of `setup-windows.ps1` clean.
- [ ] Empirical run on user's Windows: `winget install SST.opencode` succeeds non-elevated, `opencode --version` reports a version, `~/.config/opencode/opencode.jsonc` matches `ai/opencode/opencode.jsonc` byte-for-byte, `~/.config/opencode/commands/` contains all 12 `.md` files from `ai/opencode/commands/`.
- [ ] verification.md ships with the empirical evidence above.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § **AI-014-opencode-windows-bootstrap** entry.
- Predecessor: AI-011-opencode-bootstrap (Linux side, PRs #34/#36/#37, 2026-05-16). This PR mirrors that pattern adapting to Windows tooling.
- Runbook: `40-runbooks/guide-opencode-go-setup.md` — OS-agnostic for `/connect`/PAYG/troubleshooting; Windows install delta folded into it post-merge.
- ADR: [[adr-009-multi-agent-runtime]] — strategic intent for Claude Code primary + OpenCode secondary.
- Pattern: [[pattern-setup-script-idempotence]] — reconcile-not-skip rationale.
- Upstream: `SST.opencode` on winget (v1.15.6 confirmed 2026-05-21); fallback `npm install -g opencode-ai` (v1.15.7) and `curl -fsSL https://opencode.ai/install | bash` (Linux path) documented in proposal for future-ticket consideration.
