---
tags: [spec, verification, opencode, multi-agent, cross-os-parity]
created: "2026-05-21"
---

# Verification - AI-014-opencode-windows-bootstrap

## Evidence (per acceptance criterion)

To be filled post-implementation. Template:

- [ ] **`SST.opencode` in `$tools`**: line range + winget output transcript.
- [ ] **`opencode.jsonc` reconcile-not-skip**: line range + manual SHA256 equality before/after.
- [ ] **Commands sync**: line range + before/after file count in destination.
- [ ] **Healthcheck transition**: pre/post-fix healthcheck sec 10 output (SKIP → PASS).
- [ ] **Bats**: count of new asserts + CI green.
- [ ] **Lint**: PSScriptAnalyzer + AST clean.

## Test status

### Pre-fix state on user's Windows machine (captured 2026-05-21)

```
PS> winget list opencode 2>&1
(no installed package matched the query)

PS> Get-Command opencode -ErrorAction SilentlyContinue
(empty)

PS> Test-Path "$env:USERPROFILE\.config\opencode\opencode.jsonc"
False

Healthcheck sec 10/12:
  SKIP: opencode binary - not installed (AI-014 admin-conditional; deploy via setup-windows.ps1 when ready)
  SKIP: opencode.jsonc - not deployed at C:\Users\Manu\.config\opencode\opencode.jsonc (AI-014 pending)
```

### Empirical pre-flight (2026-05-21 in this session)

```
PS> winget search opencode --source winget
Name                   Id                                   Version Match
-----------------------------------------------------------------------------
OpenCode               SST.OpenCodeDesktop                  1.15.6  ProductCode: opencode
opencode               SST.opencode                         1.15.6
...

PS> Invoke-WebRequest https://opencode.ai/install.ps1 -Method Head
404 Not Found            # confirms no PowerShell installer upstream

PS> Invoke-RestMethod https://registry.npmjs.org/opencode-ai | %{ $_.'dist-tags'.latest }
1.15.7                   # npm fallback available, not chosen
```

Winget package `SST.opencode` (NOT `SST.OpenCodeDesktop` — that is the GUI). User-scope installable (no admin), winget manages PATH.

### Post-implementation results

To be filled.

### Lint results

To be filled.

## Decisions made during implementation

- **Winget over curl-bash or npm**: empirical probe showed `SST.opencode` v1.15.6 is published natively. winget handles PATH, user-scope, no admin, no shell-script execution policy concerns. The original AI-014 vault entry's "admin-conditional curl-bash equivalent" hypothesis was based on the Linux pattern; the actual Windows ecosystem is cleaner.
- **No skills-to-opencode.ps1 port**: regeneration of `ai/opencode/commands/` from `ai/skills/` stays a Linux/CI concern (the committed files are the contract Windows reads).
- **Single canonical `opencode.jsonc`**: same file shipped to both OSes via reconcile-not-skip. If a Windows-specific divergence appears later, it gets its own ticket — premature splitting now would create a 2-file SSOT.
- **Healthcheck wording**: minimal change — only drop the `AI-014 pending` parenthetical. The SKIP-when-missing path remains for the case where the user explicitly skips OpenCode install (e.g. `winget` absent).

## Promotion candidates

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? **yes** — "When porting a Linux feature to Windows, probe the native package manager (`winget`) FIRST before assuming the Linux install path (curl-bash, npm) is the only option. The cleaner channel often exists but is invisible to a Linux-first design."
- [ ] ADR-worthy decision? **no** — this is a tactical mirror of an existing ADR (ADR-009 already covers the Multi-Agent Runtime intent).
- [ ] New pattern candidate? **no** — generalisation already implied by existing `pattern-setup-script-idempotence`.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`.
- [ ] Folder moved: `specs/AI-014-opencode-windows-bootstrap/` → `specs/archive/AI-014-opencode-windows-bootstrap/`.
- [ ] Vault `11-tasks.md` AI-014 entry ticked ✓ with PR link.
- [ ] Vault `90-lessons.md` lesson appended.
- [ ] `40-runbooks/guide-opencode-go-setup.md` updated with Windows install delta.
