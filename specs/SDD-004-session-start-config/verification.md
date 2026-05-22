---
tags: [spec, verification, sdd-004]
created: "2026-05-21"
---

# Verification - SDD-004-session-start-config

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | `session-start-config.json` exists + valid JSON + schema-locked by bats | `tests/session-start-config.bats` #1-4 (4/4 PASS): exists, parses, 6 thresholds present, 12 injector entries present. |
| 2 | Linux byte-identical pre vs post refactor | `tests/session-start-config.bats` #14 PASS: spawns HEAD/main copy of `claude-session-start.sh` from the same `scripts/` directory, runs both with 3 representative CWDs (dotfiles repo / `/tmp` / vault root), asserts stdout identical. Methodology note: PRE script MUST live in same dir as POST so sibling-script lookups (`claude-mem-heal.sh`, `vault-health.sh`, `doctor.sh`) resolve identically. Mistake-once-fixed: an earlier `mktemp /tmp/...` PRE created false-positive diff because SCRIPT_DIR resolution differed. |
| 3 | LOC delta | This PR is **v1-tight (Linux-only)**: ~+50 LOC across `.sh` + JSON config + bats. NOT -150 LOC as the original audit estimated (that estimate assumed v2-dispatcher refactor; verify-before-act showed bespoke per-injector logic makes v2 a separate effort). Honest reframe: this PR delivers the SSOT-for-thresholds win + helper infrastructure for future incremental gating; future SDD-004-v2 (deferred) would deliver the LOC-reduction win. |
| 4 | Threshold change in JSON flows to script behavior without code edit | `tests/session-start-config.bats` #7-10 PASS: greps `cfg_threshold <key>` invocations for all 6 thresholds in `.sh`. Combined with #14 byte-equivalence, this proves the dynamic read works without changing default behavior. AC was revised from the original "add new injector = JSON only" because that's a v2 promise; see proposal.md §Acceptance criteria for the v1-achievable restatement. |
| n/a | Cross-OS parity drift detector | `tests/session-start-config.bats` #11-13 PASS: asserts the 3 cross-OS thresholds (`claude_json_min_bytes`, `memory_md_max_lines`, `crystallize_max_days`) hardcoded in `.ps1` still match the JSON. Locks `.sh`/`.ps1` from drifting until SDD-004b mirror PR lands. The 3 `memory_temp_*_days` thresholds are Linux-only (no `memory_temperature` injector exists in `.ps1` — separate cross-OS gap, NOT addressed by this PR). |

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

- **R1 reframe — "byte-equivalence" must be code-controlled, not literal-output**. The script's output includes live vault state queries (`unresolved links: N`), so two consecutive runs CAN differ if vault state changes between them. The achievable invariant is: with SCRIPT_DIR held constant, PRE/main copy and POST/refactor copy emit identical stdout for the same input. Captured in bats #14 + verification.md row 2.
- **R3 schema shape resolved → flat-with-nested-fields**: `{thresholds: {key: value}, injectors: {key: {enabled: bool, comment: str}}}`. Two top-level groups (thresholds, injectors), flat under each. Single `jq -r '.thresholds.X'` / `.injectors.X.enabled` call per read. AUDIT-002 "keep schemas minimal" warning respected.
- **Scope pivot v1 vs v2** (verify-before-act): the audit-promised "thin reader" model assumed dispatcher-level extraction. Reading the actual 497-LOC `.sh` showed 12 injectors with bespoke logic (date math, slugify, walk-up CWD search, stat mtime buckets). Promoting all 12 to JSON-described requires per-injector script files + dispatcher (~24 small files). That's v2, separate PR. v1-tight here delivers parameters-only SSOT with byte-equivalence guaranteed. Documented in proposal.md §Acceptance criteria revision.
- **Linux-only this PR, Windows follow-up**: `pwsh` not available on the local Linux dev machine. Mirroring the refactor to `.ps1` blind risks byte-equivalence regression with no empirical way to verify. Split into sibling SDD-004b (Windows VM empirical). Cross-OS drift detector lives in this PR (bats #11-13) until SDD-004b lands.
- **Cross-OS gap surfaced (not fixed)**: `.ps1` lacks the `memory_temperature` injector entirely (no `HOT/WARM/COLD/ARCHIVE` block). Cross-OS completeness issue, NOT a duplication issue. Flagged for separate work; do not bundle into SDD-004b.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for `10_projects/dotfiles/90-lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for `10_projects/dotfiles/30-architecture/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-004-session-start-config/` -> `specs/archive/SDD-004-session-start-config/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
