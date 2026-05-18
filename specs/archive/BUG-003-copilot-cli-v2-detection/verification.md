---
tags: [spec, verification, archived]
created: "2026-05-18"
archived: "2026-05-18"
---

# Verification - BUG-003-copilot-cli-v2-detection

> Retroactive evidence file. Captures empirical results from PR #48 (merged 2026-05-18, commit `49bb58e`).

## Evidence

Each acceptance criterion from `proposal.md` mapped to its concrete proof.

- [x] **Setup logs "GitHub Copilot CLI already installed"** → captured 2026-05-18 in user's PowerShell after `pwsh -NoProfile -ExecutionPolicy Bypass -File setup-windows.ps1`. Exact line: `[INFO] GitHub Copilot CLI already installed`. winget version reported: v1.0.48.
- [x] **PATH refresh exposes `copilot.exe` to subsequent block** → setup output: `[INFO] GitHub Copilot CLI detected at C:\Users\mlorente\AppData\Local\Microsoft\WinGet\Packages\GitHub.Copilot_Microsoft.Winget.Source_8wekyb3d8bbwe\copilot.exe, deploying configuration...`
- [x] **`copilot-instructions.md` verified pointer** → `[SUCCESS] copilot-instructions.md deployed successfully (verified pointer to AGENTS.md)`
- [x] **Aliases cop/cops in profile.ps1, no ghcs/ghce references** → `[SUCCESS] GitHub Copilot CLI configured (aliases cop/cops in profile.ps1)`. Grep on the deployed `~/.claude/CLAUDE.md` + profile.ps1 → zero matches for `ghcs|ghce` definitions (only comment references in the deprecation note).
- [x] **BUG-002 regression-free** → same setup run output: `[SUCCESS] CLAUDE.md deployed successfully (verified pointer to AGENTS.md)` + `[SUCCESS] GEMINI.md deployed successfully (verified pointer to AGENTS.md)`. The two `[ERROR]` lines that triggered BUG-002 are gone.
- [x] **WIN-003 self-heal idempotent no-op** → `[INFO] SessionStart hook already correctly configured, skipping`. Re-running setup on a healed machine does not re-fix.
- [x] **CI 5/5 green on PR #48** → confirmed at `gh pr checks 48`: GitGuardian (pass 1s), integration (pass 40s), lint (pass 10s), lint-powershell (pass 34s), test/bats (pass 1m15s).

## Test status

- **CI on PR #48**: 5/5 green. Run URL: `https://github.com/mlorentedev/dotfiles/actions/runs/26048101176`.
- **Bats simulation (local, no bats binary on Windows)**: 9 new asserts across `tests/setup-windows.bats` + `tests/aliases.bats` validated by grep-by-grep emulation. 100% green.
- **Manual smoke (admin Windows machine, 2026-05-18)**: full setup re-run + filtered output produced the expected lines (above). Captured in session transcript.
- **PSScriptAnalyzer**: clean on `setup-windows.ps1` and `powershell/profile.ps1` (Severity Error+Warning).
- **bash -n + jq empty**: clean on `setup-linux.sh`, `.zsh/aliases.zsh`, `env-contract.json`.
- **No regressions**: verified that BUG-002 (#47) verify-strings still pass and WIN-003 (#21) hook self-heal still idempotent — same setup run.

## Decisions made during implementation

- **Aliases rename `ghcs`/`ghce` → `cop`/`cops`**: explicit user choice (Option C of three Socratic options). Documented in PR body + commit body. Breaking change accepted as the lesser evil vs cognitive trap of same-name-different-semantics.
- **Auto-install via winget (Windows only)**: explicit user opt-in. Existing winget block already handled 5 tools, pattern familiar.
- **Linux: detect-and-act, no auto-install**: distros vary too much; info message points to upstream docs.
- **AWS Copilot CLI collision not handled defensively**: <1% population, inline comment only.
- **`-SimpleMatch` on `Select-String -Pattern 'First, read \`AGENTS.md\`'`**: pattern with literal backticks; `-SimpleMatch` is the established convention from BUG-001 PR #40 (Copilot side). Re-applied for CLAUDE.md and GEMINI.md verify checks (which became BUG-002 in a sibling PR).
- **Stale `eval "$(gh copilot alias -- bash)"` cleanup in setup-linux.sh**: pre-existing line added by older setup runs; with v2 the `alias` subcommand does not exist, so the line errors silently on every shell startup. Idempotent `sed -i` removes it on the next setup run. Zero-cost on clean machines.

## Promotion candidates (all executed in same session)

- [x] **Lesson** → `10_projects/dotfiles/90-lessons.md` 2026-05-18 entry "detect-and-act scripts go silently inert when upstream products change their surface". Captured 3-layer mitigation (detect on binary not extension; quarterly upstream audit cadence; annotate detect-and-act blocks with last-validated-date).
- [x] **Troubleshooting** → `10_projects/dotfiles/50-troubleshooting/copilot-cli-v1-vs-v2-detection.md`. Symptom + root cause + resolution + cross-refs.
- [ ] **ADR** → not yet executed. Open as follow-up: ADR-010 (`30-architecture/adr-010-agent-harness-parity`) needs re-audit because v2 changes 3-4 cells of the Copilot column (`~/.copilot/skills/` exists; `~/.copilot/mcp-config.json` exists; permissions surface exists). Pending AI-017 + AI-018 empirical audits.
- [ ] **New pattern** → `00_meta/patterns/pattern-detect-and-act-upstream-audit.md` candidate if the lesson recurs in a 3rd project (recurrence rule: 2 = workaround, 3 = pattern). Currently 2 instances (this bug and the original BUG-001 + claude-mem heal pattern); under threshold.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder created directly at `specs/archive/BUG-003-copilot-cli-v2-detection/` (retroactive — no `specs/<id>/` to move from)
- [x] Backlog entry in vault `10_projects/dotfiles/11-tasks.md` ticked with PR #48 link (added 2026-05-18 in same session; pending user commit in vault repo)
- [x] Promotions: lesson + troubleshooting captured; ADR + pattern deferred with explicit rationale above

## Audit-trail note

This spec folder was created AFTER the PR merged, violating step-order of `pattern-spec-driven-development.md`. The violation directly triggered SDD-001 (PR #49) — the 5-layer enforcement stack that now makes this kind of slip much less likely. So while the retroactive nature is an embarrassment, the audit trail is honest and the fix-forward is in place.
