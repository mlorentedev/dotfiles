---
id: "DOCS-013-doc-path-guard"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-11"
issue: "mlorentedev/dotfiles#916"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# DOCS-013-doc-path-guard

## Why

`.claude/CLAUDE.md` was gitignored until #907 tracked it so its mandatory shell
rules would stop being per-machine and invisible. Tracking made the file
authoritative for every agent that opens this repo; it did not make it true. On
the day it landed it named seven files that no longer exist, and two sessions in
two days acted on one of them — `./scripts/healthcheck.sh`, retired into
`dotf doctor` — before noticing. The repo has guards for content sync
(`docs-drift.bats`), markdown corruption (`check-md-escapes.sh`) and harness
drift (`compile-harness.sh --check`), and none that asks whether a path named in
prose still resolves.

## What

Two things, and the second is why this is a spec rather than a doc edit.

1. **The instructions match reality.** Dead Key Files rows dropped or
   repointed; the Secrets System section describes ADR-028 (child-process
   injection via `dotf secrets`) instead of the retired login-time loader it was
   written to abolish; Verification Commands names the Go layer and the pinned
   linter; health-check recipes point at `dotf doctor`; hardcoded
   `~/Projects/knowledge` literals become `$VAULT_PATH` per ADR-025; the lessons
   pointer moves from the vault to `docs/lessons.md` per Standing Order #2.

2. **A convention, enforced.** `scripts/check-doc-paths.sh` establishes that
   **a backticked repo path in an instruction file is a live claim**, and fails
   CI when one does not resolve. Naming a retired path is still allowed — in
   plain text, not backticks. That single rule is what makes the guard
   maintainable without a per-file exception list.

## Out of scope

- The dead `<claude-mem-context>` block at the end of `.claude/CLAUDE.md` —
  #832 (MEMORY-005) owns claude-mem's retirement.
- `docs/lessons.md` and other historical records. They name retired scripts on
  purpose; running the guard against them would report a dozen "failures" that
  are all correct as history. The guard is for files that tell someone what to
  do.
- Broadening the guard to link-checking (URLs, anchors) or to the vault.

## Acceptance

- [x] Every repo path named in `.claude/CLAUDE.md` resolves
- [x] Secrets documentation describes the ADR-028 model, with no reference to
      `env-mapping.conf` syntax as if it were live
- [x] Verification Commands covers the Go layer, naming the pinned
      `GOLANGCI_LINT_VERSION` and why an unpinned local run proves nothing
- [x] No hardcoded vault literal remains in the file
- [x] `scripts/check-doc-paths.sh` exits 0 on all six instruction files and 1 on
      a seeded dead path
- [x] The guard produces **zero** false positives on `AGENTS.md`, the largest
      instruction file
- [x] A bats suite pins both the catching and the not-crying-wolf behaviour

## Risks

**A guard with false positives is worse than none** — it gets bypassed, and then
the real regression sails through. Mitigated by making the matcher conservative
by construction (only backticked, repo-rooted tokens whose first segment is a
real top-level entry, computed from disk) rather than by an exclusion list that
would rot exactly as the docs did. Measured: an earlier revision produced 13
false positives on `AGENTS.md`; the shipped one produces zero.

**A guard with false negatives is a placebo.** Found one while building it: an
ALL-CAPS "placeholder" rule swallowed `SKILL.md` and so skipped the dead
`ai/skills/*/SKILL.md` glob the guard existed to catch. Both that case and the
false-positive cases are pinned as tests, so the next tightening cannot silently
re-open either.

## Alternatives rejected

- **Fix the docs, no guard.** This is the second time this file class has misled
  a session; the incident-to-guard rule exists for exactly this repetition.
- **Resolve bare filenames by basename.** Would catch a few more stale mentions,
  but flagged vault patterns, `machine.json` and `review.md` on `AGENTS.md` —
  files that legitimately live elsewhere. Rejected on the false-positive risk
  above.
- **A per-file exclusion list.** Rots the same way the docs rot, and hides the
  rule it is meant to express.
