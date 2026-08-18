---
id: lesson-053-heal-scripts-versioned-against-the-upstream-bug-cl
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 053: Heal scripts versioned against the upstream bug class they paper over

**Context:** BUG-016 (PR #83, today). `claude-mem-heal.{sh,ps1}::Repair-McpJson` was authored against v12.7.4's broken `${_R%/}` literal in `.mcp.json` (PR #57, 2026-05-19). v13.0.0+ shipped a different broken pattern: cascading-printf via `sh -c` triggering the EPIPE race. The heal silently no-oped against v13.3.0 installs because the v12.7.4 signature was absent. User hit the upstream MCP failure repeatedly while the heal kept exiting clean — `[claude-mem-heal] .mcp.json already healthy: ...` was a false claim.
**Problem:** Heal scripts are bug-class-specific by design (they paper over a SPECIFIC broken upstream pattern). When the upstream changes its bug pattern (intentionally or accidentally — v12 → v13 shipped different brokenness), the heal's detection regex no longer matches, and the heal becomes a no-op. The script reports "healthy" while the install is broken. Worst: silent failure.
**Solution:** When the upstream version changes AND a bug class is still being papered over, the heal's detection MUST be refreshed in the same investigation that discovers the new pattern. (a) Detect each known broken signature with an OR (don't replace the v12 detection with v13 — keep both, since rollbacks happen). (b) Log which signature was patched so future audits can map heal output → upstream version. (c) Bats parity asserts must lock BOTH detection signatures (each on its own assert) so regressions surface in CI. (d) Add a stronger assertion in healthcheck that validates the canonical fix actually landed (the `head -n1` substring in the patched file), not just that the heal "ran". Companion to BUG-014's end-state-not-proxy lesson.
**Tags:** `#heal-pattern` `#upstream-versioning` `#defensive-scripting` `#claude-mem`
