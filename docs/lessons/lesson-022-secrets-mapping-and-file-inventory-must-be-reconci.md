---
id: lesson-022-secrets-mapping-and-file-inventory-must-be-reconci
type: lesson
status: active
created: "2026-03-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 022: Secrets mapping and file inventory must be reconciled automatically

**Context**: `sensitive/` contained 35 encrypted `.secret.age` files but only 17 had entries in `env-mapping.conf`. 14 were app-specific secrets (mlorentedev) that didn't belong in dotfiles at all.

**Problem**: `load-secrets.sh` silently skipped missing files (`file_exists || return 1`) and never checked for orphans. No automated way to detect mapping↔file drift. Secrets accumulated without cleanup.

**Solution**: (1) Added `log_warning` on missing files and orphan detection to `secrets_load()` — runs passively at every shell startup. (2) Added healthcheck section 8/8 that validates every mapping entry has a file and every file has a mapping. (3) Classified secrets: personal cross-machine credentials stay, app-specific envs move to project SOPS.

**Rule**: Any system with a mapping file (config ↔ resource) needs bidirectional reconciliation: mapping→resource (missing?) and resource→mapping (orphan?). Run both checks automatically at load time (passive warning) and in CI/healthcheck (active audit). Silent failure on missing resources is a bug, not a feature.
