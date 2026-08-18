---
id: lesson-074-a-cross-environment-ssot-validator-must-split-cont
type: lesson
status: active
created: "2026-05-31"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 074: A cross-environment SSOT validator must split "content drift" (fail) from "runtime absent off-box" (warn)

**Context:** HERMES-001 Track B. `80_agents/hermes-nan/scripts/validate.sh` checks the vault SSOT for the Hermes agent; AC6 is "vault SSOT consistent, validate.sh green". The script also checked box-only runtime facts (post-commit hook, cron entry, `uvx`).
**Problem:** Those runtime checks only pass on the provisioned Hermes box. As hard failures they made AC6 unprovable anywhere else — the script could never be green from a dev checkout or CI, so "green" had no portable meaning.
**Solution:** Split the exit policy by what the artifact is SSOT for. `fail` (exit 1) = vault SSOT inconsistency (a required file missing/malformed), content the vault owns and that holds everywhere. `warn` (exit 0) = box-runtime advisory (hook/cron/tool), environmental and absent off-box. A green run then means "SSOT internally consistent" regardless of where it runs, and the same script still does the full check on the box. Generalize: a validator that mixes content invariants with environment state must rank them, or it asserts nothing portable.
**Tags:** `#validation` `#ssot` `#hermes` `#exit-codes` `#cross-environment`
