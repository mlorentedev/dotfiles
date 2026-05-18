---
id: "SDD-001-discipline-gate"
type: spec
status: archived
created: "2026-05-18"
archived: "2026-05-18"
merged_pr: 49
tags: [spec, proposal, sdd, archived]
template_version: "1.0"
---

# SDD-001-discipline-gate

> Scope of THIS PR (PR A in the SDD-001 series): Tier 1 (`AGENTS.md` rule block) + Tier 2 (SessionStart hook injects deterministic SDD reminder in `additionalContext`). Tiers 3-5 (settings.json portability, CI spec-gate, PR template) ship as separate atomic PRs (SDD-002, SDD-003).

## Why

Session audit on 2026-05-18 found that two recent atomic PRs in this repo (BUG-002 / PR #47, BUG-003 / PR #48) bypassed the SDD discipline defined in `pattern-spec-driven-development.md`: no vault entry first, no `init-spec` scaffold, no `specs/<id>/` folder — despite both PRs meeting trigger criteria (>50 LOC, public contract, new dep, multiple Socratic Guardrail pauses). The discipline failed silently because `git checkout -b` does not enforce the vault gate, and the SessionStart hook only *reports* repo specs status rather than nudging the agent to *apply* SDD. This PR closes the first two enforcement gaps: documents the rule explicitly in the canonical SSOT (`AGENTS.md`) and surfaces it every session via the hook.

## What

Two observable behavior changes after this PR:

1. **`AGENTS.md` at repo root contains a new `## SDD Discipline Gate (NON-NEGOTIABLE)` section** that codifies: the trigger criteria, the mandatory order (vault entry → `init-spec.{sh,ps1}` → proposal → tasks → code → verification), the skip criteria (typos / comment-only / mechanical refactor / bug <20 LOC / doc-only), and a banned-phrases list for vault-hygiene "later" promises that historically compound into debt.

2. **`scripts/claude-session-start.ps1` and `scripts/claude-session-start.sh` inject a deterministic, non-conditional reminder line at the *start* of `additionalContext`** referencing the new `AGENTS.md` section. The reminder fires every session regardless of CWD, repo state, or vault presence (existing `Test-RepoSpecs` / `find_hive_project` blocks remain as diagnostics aside). Cross-OS parity locked in bats.

## Out of scope

- **Tier 3** — tracking `~/.claude/settings.json` in dotfiles + deep-merge install logic. Significant standalone scope (~80 LOC + JSON merge complexity). Ships as `SDD-002-settings-portability`.
- **Tier 4** — `.github/workflows/ci.yml` `spec-gate` job. Ships as part of `SDD-003-ci-gate`.
- **Tier 5** — `.github/PULL_REQUEST_TEMPLATE.md`. Ships with Tier 4 in `SDD-003-ci-gate`.
- Removing or renaming existing hook diagnostics (`Test-RepoSpecs`, claude-mem heal, doctor drift) — they coexist with the new reminder.
- Modifying the canonical pattern `pattern-spec-driven-development.md` in the vault — the pattern already defines the rule; this PR enforces it in the agent's context.

## Risks / open questions

- **Risk: hook output grows and consumers truncate `additionalContext`.** The reminder adds ~5 lines. Existing `additionalContext` payload is variable but already includes claude-mem heal output, doctor drift, vault detection, specs detection. Mitigation: keep the new reminder under 4 lines, prefix with `[sdd]` so it's filterable like existing `[claude-mem]`, `[doctor]`, `[specs]` prefixes.
- **Risk: AGENTS.md grows past the ≤70-line pointer-style target.** Current AGENTS.md is denser than the pointer files (it's the SSOT, not a pointer). Mitigation: keep the new section terse (≤30 lines), link to the pattern doc for full rationale, do not duplicate the full pattern body.
- **Risk: future agents ignore the rule (no enforcement, just documentation).** True for Tier 1+2 alone. Tier 4 (CI spec-gate) is the hard enforcement layer and ships separately. Tier 1+2 are the soft layer that surfaces the rule at the right moment; CI is the safety net.
- **Open question: should the reminder text reference the AGENTS.md section by anchor link (`#sdd-discipline-gate-non-negotiable`) for click-through in the agent's rendering?** GitHub-flavored anchor would be `#sdd-discipline-gate-non-negotiable`. Decision: include the anchor, fall back to section title text if anchor not supported.

## Acceptance criteria

- [ ] `AGENTS.md` at repo root contains an H2 section titled exactly `## SDD Discipline Gate (NON-NEGOTIABLE)`
- [ ] The new section enumerates: (a) trigger criteria, (b) mandatory ordered process (vault entry → init-spec → proposal → tasks → code → verification), (c) skip criteria, (d) banned-phrases list
- [ ] `scripts/claude-session-start.ps1` writes a line containing literal text `Before your first tool call, read AGENTS.md` to `additionalContext`, unconditionally (no `if` gates around it)
- [ ] `scripts/claude-session-start.sh` writes the same reminder text, also unconditionally, in matching position
- [ ] Local test on the user's machine: `echo '{"cwd":"<any path>","session_id":"test"}' | pwsh -NoProfile -File scripts/claude-session-start.ps1` produces JSON with the SDD reminder present in `hookSpecificOutput.additionalContext`
- [ ] Same for `.sh` via `echo '{"cwd":"...","session_id":"test"}' | bash scripts/claude-session-start.sh`
- [ ] Bats parity assert: both hook scripts contain the SDD reminder marker
- [ ] PSScriptAnalyzer clean on the modified `.ps1`
- [ ] `bash -n` clean on the modified `.sh`; shellcheck clean
- [ ] No regressions: existing hook behavior (vault detection, claude-mem heal, doctor drift, specs detection) all still fires when conditions match

## References

- Vault: `10_projects/dotfiles/11-tasks.md` "SDD-001-discipline-gate" backlog entry (added 2026-05-18 in same session as audit)
- Pattern: `00_meta/patterns/pattern-spec-driven-development.md` (the SSOT this PR enforces)
- Sibling PRs: `SDD-002-settings-portability` (next), `SDD-003-ci-gate` (after)
- Triggered by: BUG-002 / PR #47 + BUG-003 / PR #48 audit (this session, 2026-05-18)
- Related lessons: `10_projects/dotfiles/90-lessons.md` 2026-05-18 entries
