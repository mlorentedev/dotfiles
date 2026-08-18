---
id: lesson-113-windows-winget-jq-emits-crlf-breaks-jq-read
type: lesson
status: active
created: "2026-06-20"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 113: Windows winget jq emits CRLF — breaks `< <(jq)` + read

**Context**: `compile-harness.sh --refresh` (the harness deploy engine) aborted on Windows with `section "6-attribution-policy" not found`, even though the slug matched a real heading. The deployed agent skills had silently drifted from the vault SSOT for ~16 skills.

**Problem**: The winget build of jq (1.8.1) emits CRLF on every output line. MSYS/Git-Bash command-substitution `$(jq …)` strips the trailing CRLF, so single-value reads look clean — but `read`/`mapfile`/`for` fed via `< <(jq …)` keep the bare `\r` in the last field, so `slug == want` comparisons (and any path built from jq output) silently fail. Because Windows `setup` only runs the deploy half (refresh is Linux-only), nothing caught the resulting vault→records drift.

**Solution**: Shadow `jq` with a CR-stripping wrapper that preserves the real exit status (`return ${PIPESTATUS[0]}`, so `jq -e` truthiness still works); verify the binary with `type -P jq`, not the function. (PR #511; to be superseded by the Go port CLI-026.)

**Rule**: Shell engines that read tool output into loops via process substitution must strip `\r` — the `$(...)` CRLF-strip is a Git-Bash illusion that does NOT extend to `< <(...)`+read. And: a `vault → records → deploy` pipeline has two drift axes — cover BOTH (records↔deploy = CLI-019/#488; vault↔records = CLI-026 `check --against-vault`), or one half drifts silently.
