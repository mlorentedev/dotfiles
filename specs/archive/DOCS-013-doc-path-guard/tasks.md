---
tags: [spec, tasks, templates]
created: "2026-08-11"
---

# Tasks - DOCS-013-doc-path-guard

> TDD order. One task = one focused commit. Tick as you go.
>
> Written after the fact, honestly: the work was done in one pass on
> `fix/claude-md-staleness` before the spec-gate flagged it at 293 production
> LOC. The order below is the order it actually happened — the guard first,
> run against the unfixed file, then the corrections until it went green. The
> spec is retrofitted; the sequence is not invented.

## Setup

- [x] Branch (worktree) created from main: `fix/claude-md-staleness`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Work-gate: issue #916 open and linked

## Implementation

- [x] `scripts/check-doc-paths.sh`: extract backticked tokens, filter to
      repo-rooted paths, assert each resolves. Run against the **unfixed**
      `.claude/CLAUDE.md` first — 8 genuine findings. [AC5]
- [x] Tighten the matcher after the first revision flagged 26 tokens including
      `&>/dev/null`, `ai/<agent>/` and the model id `opencode-go/qwen3.6-plus`.
      First segment must be a real top-level repo entry, computed from disk. [AC6]
- [x] Control-run on `AGENTS.md`: 13 false positives from resolving bare
      filenames by basename (vault patterns, `machine.json`, `review.md`).
      Drop bare-filename resolution entirely. `AGENTS.md` now clean. [AC6]
- [x] Fix the false negative found by the control run: the ALL-CAPS placeholder
      rule swallowed `SKILL.md`, hiding the dead `ai/skills/*/SKILL.md` glob.
      Placeholder reading now applies only when the token lacks a known
      extension. [AC5]
- [x] `tests/check-doc-paths.bats`: 8 cases — catches a dead path, catches an
      empty glob, accepts a live path, ignores nine pathish non-paths, keeps
      ALL-CAPS-with-extension checked, usage error, and applies the guard to all
      six instruction files. [AC5][AC6][AC7]
- [x] `.claude/CLAUDE.md` Key Files: drop/repoint dead rows, add `cli/`,
      `secrets/registry.yaml`, harness engine + manifest. [AC1]
- [x] `.claude/CLAUDE.md` Secrets System + "adding a secret" → ADR-028. [AC2]
- [x] `.claude/CLAUDE.md` Verification Commands → both layers, pinned linter,
      corrected test count. [AC3]
- [x] Health-check recipes → `dotf doctor`, with a self-refuting note that the
      twins are retired. [AC1]
- [x] Vault literals → `$VAULT_PATH` (ADR-025). [AC4]
- [x] Lessons pointer → `docs/lessons.md` (Standing Order #2). [AC1]
- [x] Remove the aider section (sunset by opencode). [AC1]
- [x] `README.md`: same dead `load-secrets.sh` entrypoint in the tree and the
      human-entrypoints table → `dotf secrets`. [AC5]
- [x] Convention documented in the script header: a backticked repo path is a
      live claim; name retired paths in plain text.

## Verification

- [x] `shellcheck` clean; `bash -n` + `zsh -n`; guard executed under zsh
- [x] `bats tests/check-doc-paths.bats` → 8/8
- [x] Guard clean on all six instruction files
- [x] Full bats suite for regressions
- [x] `check-bats-names.sh` clean

## Out

- [ ] `<claude-mem-context>` block — #832 owns it, deliberately untouched
