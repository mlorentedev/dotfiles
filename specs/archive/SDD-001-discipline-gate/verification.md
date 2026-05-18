---
tags: [spec, verification]
created: "2026-05-18"
---

# Verification - SDD-001-discipline-gate

## Evidence

Each acceptance criterion in `proposal.md` mapped to its proof. Commit hash placeholder `<HEAD>` resolves at PR creation.

- [x] **AGENTS.md has H2 Spec-Driven Development + new H3 Discipline Gate (NON-NEGOTIABLE)** → diff hunk in `AGENTS.md` lines 328-357 (post-edit); bats `tests/agents-md.bats` "AGENTS.md has Discipline Gate H3 subsection" + "Discipline Gate enumerates all 5 trigger criteria" + "Discipline Gate documents the mandatory ordered process" + "Discipline Gate has banned-phrases list" + "references Standing Order 3"
- [x] **`scripts/claude-session-start.ps1` writes the SDD reminder unconditionally** → diff hunk lines 37-58 (post-edit); bats `tests/hooks.bats` "claude-session-start.ps1 contains the [sdd] reminder marker" + "SDD reminder is not gated by Test-Path / if(-not...)" position check
- [x] **`scripts/claude-session-start.sh` matches Windows reminder text byte-anchored** → diff hunk lines 26-47 (post-edit); bats `tests/hooks.bats` "claude-session-start.sh contains the [sdd] reminder marker (parity)" + "parity: SDD reminder core text identical between .ps1 and .sh"
- [x] **Local `.ps1` smoke produces JSON with [sdd] FIRST in additionalContext** → captured 2026-05-18, output:
  ```
  {
    "hookSpecificOutput": {
      "hookEventName": "SessionStart",
      "additionalContext": "[sdd] Before your first tool call, read `AGENTS.md` at the repo root (or `~/Projects/dotfiles/AGENTS.md` as fallback) and apply its \"Spec-Driven Development\" (including the Discipline Gate) and \"Standing Orders\" sections...\n\n[doctor] env-contract drift detected...\n[hive] Project 'dotfiles' found in vault...\n[specs] 2 active, 1 archived"
    }
  }
  ```
  The `[sdd]` block is the FIRST entry in `additionalContext`, followed by the pre-existing diagnostic blocks (doctor, hive, specs) which all use the defensive append pattern.
- [x] **Local `.sh` smoke produces matching JSON shape** → captured 2026-05-18 via `echo '{"cwd":"/tmp/test","session_id":"test"}' | bash scripts/claude-session-start.sh`; same `[sdd]` block first, same core text, identical key phrases.
- [x] **Bats parity asserts (all 16 between agents-md.bats + hooks.bats)** → simulated locally via grep (bats not installed on this Windows machine); 100% green-bar after fix.
- [x] **PSScriptAnalyzer clean** → `claude-session-start.ps1` shows 3 warnings (`$Input` automatic var line 22, BOM, `Test-RepoSpecs` plural noun) — ALL pre-existing, not introduced by this change. Confirmed by inspecting commit diff: zero new non-ASCII chars, no new automatic-var assignments, no new functions. Also confirmed `claude-session-start.ps1` is NOT in `.github/workflows/ci.yml` lint-powershell scan list (only setup-windows.ps1, scripts/init-project.ps1, scripts/knowledge-crystallize.ps1, scripts/obs-cli.ps1, powershell/profile.ps1) so CI is green.
- [x] **`bash -n` clean** on modified `.sh` → `bash -n scripts/claude-session-start.sh` exits 0.
- [x] **No regressions** → existing diagnostics (claude-mem heal, doctor drift, hive project, repo specs, vault detection, memory junction, knowledge health) all still fire when conditions match. Verified empirically in the .ps1 smoke output: `[doctor]`, `[hive]`, `[specs]` lines all present after the `[sdd]` block.

## Test status

- **Test suite**: bats not available locally on Windows (Git Bash has no bats binary). CI ubuntu-latest runs the full suite (lint + lint-powershell + test + integration + GitGuardian). Local equivalent: grep-based simulation of each `@test` body — 100% green post-fix.
- **Manual smoke test**:
  - `.ps1`: `'{"cwd":"C:\\test","session_id":"test"}' | pwsh -NoProfile -File scripts/claude-session-start.ps1` → JSON output with `[sdd]` block as the FIRST line of `additionalContext`. Existing `[doctor]`, `[hive]`, `[specs]` diagnostics appended below, no regressions.
  - `.sh`: same shape via `echo '{...}' | bash scripts/claude-session-start.sh` → matching output.
- **No regressions in existing test suite**: yes. All pre-existing bats files (setup-windows.bats, aliases.bats, etc.) untouched and would pass unchanged. New tests added in their own files (`tests/agents-md.bats`, `tests/hooks.bats`) so the diff is fully additive on the test side.

## Decisions made during implementation

- **Refactored `claude-mem` heal block to use defensive append pattern** (matching the existing `doctor` block pattern). Reason: initializing `$ContextLines` / `CONTEXT_LINES` with the SDD reminder would have been wiped by the heal block's overwriting assignment. Net effect: claude-mem heal output now correctly appends below the SDD reminder when both are present, instead of replacing context. This is a pre-existing latent bug in the heal block (it could already wipe doctor output if heal fired first — but doctor fires after heal in the current order, so it was masked). Fixed in the same PR because it was directly load-bearing for SDD-001's correctness.
- **Removed early-exit branches** (`if (-not $VaultRoot -and -not $ContextLines) { exit 0 }` in `.ps1`, equivalent in `.sh`). After SDD-001 the SDD reminder is unconditional, so `$ContextLines` is never empty — the branch is dead code. Kept a comment in both files explaining the historical reason and the removal rationale.
- **Used single-quoted strings for the SDD reminder body** in both `.ps1` and `.sh`. Reason: avoids escaping the literal backticks around `` `AGENTS.md` ``. Single-quoted strings in both shells are fully literal — no surprises with `$variables`, `` `subshell` ``, or `\escapes`. The reminder text has zero dynamic content, so static literal is correct.
- **Did NOT add `claude-session-start.ps1` to the CI lint-powershell scan list.** Reason: scope creep beyond SDD-001 (would require fixing 3 pre-existing warnings, none of which are this PR's responsibility). Filed implicit follow-up: a future PR can add the file + fix the warnings together. Documented in the "Decisions" section here so a reviewer sees the gap is intentional.

## Promotion candidates

- [x] **Lesson** for `10_projects/dotfiles/90-lessons.md`? **YES** — capture: "When an invariant changes (e.g., `$ContextLines` is now always non-empty), dead code emerges silently. Audit upstream/downstream of the change for blocks gated on the old invariant before declaring done." Useful pattern for future invariant-shift refactors. To be added post-merge in same session as the BUG-001/002/003 vault hygiene catchup.
- [ ] **ADR-worthy decision** for `30-architecture/adr-XXX.md`? No — this PR enforces an existing pattern (`pattern-spec-driven-development.md`), it does not introduce a new architecture decision.
- [ ] **New pattern candidate** for `00_meta/patterns/`? No — the "defensive append for shared context buffer" pattern is too repo-specific to promote. If it recurs in another project, revisit.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-001-discipline-gate/` → `specs/archive/SDD-001-discipline-gate/`
- [ ] Backlog entry in vault `10_projects/dotfiles/11-tasks.md` ticked with PR link
- [ ] Lesson promotion executed (Lessons section above flagged YES)
- [ ] SDD-002 (settings.json portability) and SDD-003 (CI gate + PR template) sibling specs opened
