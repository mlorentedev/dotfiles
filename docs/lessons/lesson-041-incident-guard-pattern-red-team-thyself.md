---
id: lesson-041-incident-guard-pattern-red-team-thyself
type: lesson
status: active
created: "2026-05-19"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 041: Incident → guard pattern (red-team thyself)

**Context:** During a single 2026-05-19 session, Hive's vault_patch MCP wrote the literal 2-character sequence backslash-n into dotfiles/11-tasks.md four separate times instead of interpreting it as a newline. Each occurrence corrupted a markdown bullet list by merging two items into one physical line — invisible in rendered markdown but breaking init-spec.sh's vault-gate grep (which anchors on `^- [ ] **<id>**`) and any downstream line-based parser. The user surfaced the meta-issue: "la red de seguridad tiene que ir mejorándose a sí misma" — the safety net must keep improving itself.</context>
<parameter name="problem">Each corruption was fixed manually with the Edit tool. No guard was added until the 4th occurrence. By then, the bug class had already burned ~10 minutes of cumulative friction. The general failure: when a bug class hits, we tend to fix the immediate symptom and move on instead of adding a CI assertion / health check / parity test that prevents the next occurrence. Three sibling instances in the same session reinforce the pattern: (1) AI-019 missed `.github/copilot-instructions.md` Model Tier section — fixed in SDD-005 with `tests/docs-drift.bats`; (2) BUG-001 + BUG-002 verify-string drift between setup-linux.sh and setup-windows.ps1 — fixed earlier in PR #40 + #47 with bats parity asserts; (3) Hive vault_patch literal `[BS-n]` — fixed in SDD-006 with `scripts/check-md-escapes.sh` + bats. All three are the same meta-pattern: a class of failure recurs because each occurrence was patched without adding the structural guard.
**Problem:** 
**Solution:** **General rule (incident → guard, red-team thyself):** every bug class encountered MUST emit a CI assertion or health check in the SAME PR that fixes the symptom. Three signals that the rule is being violated: (a) you hit the same bug class twice in one session, (b) the fix is "I'll edit the file manually" with no test added, (c) the PR body says "I'll add the guard later". When you hit a bug class for the 2nd time, STOP. Don't fix the symptom — add the guard. The guard prevents the 3rd through Nth occurrence. **Concrete artefacts shipped under this rule in dotfiles:** `tests/docs-drift.bats` (mirror-file parity, SDD-005), `tests/check-md-escapes.bats` + `scripts/check-md-escapes.sh` (vault_patch corruption, SDD-006), `tests/setup-windows.bats` parity asserts (verify-string drift, BUG-002). **Pre-promotion to 00_meta/patterns/:** the pattern is dotfiles-specific for now; promote to a global pattern when a second project applies it.
**Tags:** `#safety-net` `#ci` `#testing` `#vault-patch` `#incident-driven` `#meta-pattern`
