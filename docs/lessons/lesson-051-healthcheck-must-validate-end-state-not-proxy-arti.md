---
id: lesson-051-healthcheck-must-validate-end-state-not-proxy-arti
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 051: Healthcheck must validate end-state, not proxy artifacts

**Context:** BUG-014 (PR #75, today). The pre-existing healthcheck assertion for claude-mem (BUG-012 era) checked whether the filesystem JUNCTION existed at `~/.claude/plugins/marketplaces/thedotmack/`. The junction exists if BUG-012's heal ran. But the heal only runs if the marketplace dir is present, and the marketplace dir presence does NOT imply the plugin is installed in `installed_plugins.json` (Claude Code's canonical install record). Result: healthcheck reported `PASS: claude-mem marketplace legacy junction present` while `/mem-search` was unavailable, session-start hook never fired, and `installed_plugins.json` had zero `@thedotmack` entries. False positive that hid the real bug for days.
**Problem:** Asserting a proxy artifact (filesystem junction = consequence of heal) instead of canonical state (installed_plugins.json = source of truth for "is the plugin actually installed") makes the healthcheck unable to detect a whole class of failures. The asymmetry is dangerous: proxy artifacts can exist WITHOUT the canonical state being correct.
**Solution:** Every healthcheck assertion should validate the END-STATE that the user cares about, not a proxy. For "is plugin X installed?" → grep `installed_plugins.json`. For "is service Y running?" → query the service status, not "config file exists". For "is alias Z configured?" → invoke the alias and check exit, not "alias line present in profile". When a proxy is the only available signal, the assertion message should explicitly say "proxy" — and a PRIMARY canonical assertion should come first.
**Tags:** `#healthcheck` `#observability` `#false-positive` `#end-state-vs-proxy`
