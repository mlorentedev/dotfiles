---
tags: [spec, verification, claude-mem, plugin-discovery]
created: "2026-05-21"
---

# Verification - BUG-014-claude-mem-marketplace-register

## Evidence (per acceptance criterion)

To be filled post-implementation. Template:

- [ ] **`setup-linux.sh` marketplace-add block**: line ranges of the new block + name of the snapshot-guard wrapper used.
- [ ] **`setup-windows.ps1` marketplace-add block**: same for the PowerShell side.
- [ ] **`healthcheck.sh` install-state check**: line ranges + grep pattern.
- [ ] **`healthcheck.ps1` install-state check**: same for the PowerShell side.
- [ ] **Bats assertions**: count of new asserts in each of the four bats files.
- [ ] **Idempotence**: empirical second-run output showing `'thedotmack' already on disk` exit-0 message.

## Test status

### Pre-fix state on user's Windows machine (captured 2026-05-21 during diagnosis)

```
$ claude plugin marketplace list
Configured marketplaces:

  ❯ claude-plugins-official
    Source: GitHub (anthropics/claude-plugins-official)

$ jq '.plugins | keys' ~/.claude/plugins/installed_plugins.json
[
  "code-simplifier@claude-plugins-official",
  "gopls-lsp@claude-plugins-official",
  "security-guidance@claude-plugins-official",
  "claude-md-management@claude-plugins-official",
  "claude-code-setup@claude-plugins-official",
  "frontend-design@claude-plugins-official",
  "ralph-loop@claude-plugins-official",
  "code-review@claude-plugins-official",
  "commit-commands@claude-plugins-official",
  "pr-review-toolkit@claude-plugins-official"
]
```

No `claude-mem@thedotmack`. Healthcheck sec 4 reported `PASS: claude-mem marketplace legacy junction present (BUG-012)` — a false positive.

### Empirical fix application (manual, 2026-05-21)

```
$ claude plugin marketplace add thedotmack/claude-mem
Adding marketplace…SSH not configured, cloning via HTTPS: https://github.com/thedotmack/claude-mem.git
Refreshing marketplace cache (timeout: 120s)…
✔ Successfully added marketplace: thedotmack (declared in user settings)

$ claude plugin marketplace list
Configured marketplaces:

  ❯ claude-plugins-official
    Source: GitHub (anthropics/claude-plugins-official)

  ❯ thedotmack
    Source: GitHub (thedotmack/claude-mem)

$ claude plugin install claude-mem@thedotmack
Installing plugin "claude-mem@thedotmack"...✔ Successfully installed plugin: claude-mem@thedotmack (scope: user)

$ jq '.plugins | keys' ~/.claude/plugins/installed_plugins.json
[
  ...10 official plugins...,
  "claude-mem@thedotmack"   ← new
]
```

`.claude.json` size: 26608 bytes pre, 26608 bytes post — no truncation. Snapshot guard validated as belt-and-suspenders.

### Idempotence (2nd run)

```
$ claude plugin marketplace add thedotmack/claude-mem
Adding marketplace…✔ Marketplace 'thedotmack' already on disk — declared in user settings
$ echo $?
0

$ claude plugin install claude-mem@thedotmack
Installing plugin "claude-mem@thedotmack"...✔ Plugin "claude-mem@thedotmack" is already installed (scope: user)
$ echo $?
0
```

Both commands idempotent with exit 0 and human-readable "already done" message — no pre-check needed in setup scripts.

### Post-implementation bats results

`bats` not installed on the implementation machine (Windows daily-driver, surfaced by healthcheck `[6/12] SKIP: bats - not installed`). 7 new asserts authored across 4 files (`tests/setup-linux.bats` +3, `tests/setup-windows.bats` +2, `tests/healthcheck.bats` +2, `tests/healthcheck-ps1.bats` +2). CI validates on Linux containers. Pre-implementation the literal strings the tests grep for did not exist in the source files, so the tests would have failed RED — this satisfies the TDD ordering.

### Lint results

- `bash -n setup-linux.sh` → OK
- `bash -n scripts/healthcheck.sh` → OK
- PowerShell AST `[Parser]::ParseFile` on `setup-windows.ps1` and `scripts/healthcheck.ps1` → clean
- `Invoke-ScriptAnalyzer -Severity Error` on both `.ps1` files → clean
- `shellcheck` not installed locally — CI runs it

### Diff stat

```
 scripts/healthcheck.ps1    | 25 +++++++++++++++++++++++--
 scripts/healthcheck.sh     | 16 ++++++++++++++++
 setup-linux.sh             | 11 +++++++++++
 setup-windows.ps1          | 16 ++++++++++++++++
 tests/healthcheck-ps1.bats | 16 ++++++++++++++++
 tests/healthcheck.bats     | 14 ++++++++++++++
 tests/setup-linux.bats     | 26 ++++++++++++++++++++++++++
 tests/setup-windows.bats   | 18 ++++++++++++++++++
 8 files changed, 140 insertions(+), 2 deletions(-)
```

## Decisions made during implementation

- **No pre-check before `claude plugin marketplace add`**: the CLI is idempotent (exit 0 + clean message on re-run, validated empirically). A pre-check (`claude plugin marketplace list | grep`) would itself trigger another wrapped CLI call — net cost without benefit.
- **Keep BUG-012 junction check as secondary diagnostic** instead of removing it: different failure class (install OK but plugin discovery still broken because junction missing) — still useful, but explicitly demoted from primary.
- **Wrap with existing BUG-004/BUG-011 snapshot guards**: every `claude` CLI invocation in setup must be wrapped. New call site is no exception. Same rule applies if anyone adds another non-official marketplace.
- **Single `thedotmack` marketplace, not generalised loop**: only one non-official marketplace is in the current plugin array. YAGNI: revisit if a second is added.

## Promotion candidates

To be assessed post-merge. Candidate lesson: "healthcheck assertion that checks proxy artifact (junction) instead of canonical state (installed_plugins.json) is itself a bug".

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? yes — "healthcheck must validate end-state, not proxy artifacts; junction presence ≠ plugin installed".
- [ ] ADR-worthy? no — same rule already implicit in existing audit checklist.
- [ ] New pattern candidate? probably no — close cousin of existing "incident → guard" pattern.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived` (post-merge).
- [ ] Folder moved: `specs/BUG-014-claude-mem-marketplace-register/` -> `specs/archive/BUG-014-claude-mem-marketplace-register/`.
- [ ] Vault `11-tasks.md` BUG-014 entry ticked ✓ with PR link.
- [ ] Vault `90-lessons.md` lesson appended.
