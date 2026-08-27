---
id: "CLI-050-knowledge-crystallize-cutover"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#1269"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-050-knowledge-crystallize-cutover

## Why

CLI-021 (#490) built `dotf vault crystallize` beside `scripts/knowledge-crystallize.{sh,ps1}`
with byte-identical golden parity, but deliberately left every caller pointed at the shell —
the cutover was scoped to CLI-023 (#492). #492 turned out to be all-or-nothing: it bundles the
crystallize flip with `vault-health`/`vault-maintain` (increments 2-3 of #490, unbuilt) and the
entire spec-gate cluster (CLI-022 / #491, unbuilt) into one 9-script deletion. That is too large
a unit to land now. This ticket carves out the one slice that is fully ready — crystallize, with
13/13 golden byte-parity already proven — so the twin gets deleted on the schedule ADR-020 §5
actually calls for (port on contact) instead of waiting on unrelated, unstarted work.

## What

`scripts/knowledge-crystallize.{sh,ps1}` are deleted. `dotf vault crystallize` is the sole
implementation, reached from every former caller: the weekly maintenance script, `setup-linux.sh`
/ `setup-windows.ps1` deploy steps, the SessionStart hook's staleness message
(`cli/internal/mem/session_start_injectors.go`), and the docs/comments that told a human to run
the shell script directly.

## Out of scope

- `vault-health.sh`, `vault-maintenance-weekly.{sh,ps1}` themselves, `obs-cli.{sh,ps1}`,
  `check-md-escapes.sh`, and the spec-gate cluster — those stay in CLI-023 (#492), still blocked
  on CLI-022 (#491) and CLI-021 increments 2-3.
- The vault-side skill SSOT (`00_meta/skills/crystallize/SKILL.md`, which already says "or
  `dotf vault crystallize`") — edited + `compile-harness.sh --refresh`'d as a direct-to-master
  vault commit sequenced after this PR merges, per this repo's standing vault-commit convention
  and CLI-021's own sequencing note (editing it before the flip lands would point agents at a
  command that wasn't canonical yet; it is now).
- Two pre-existing, unrelated doc staleness items noticed while editing
  `docs/runbooks/guide-knowledge-distillation.md` (a "Session hook" row naming retired
  `claude-session-start.{sh,ps1}` scripts per CLI-025, and a Windows path-encoding example that
  still describes the pre-#689 delete-colon bug) — ticketed separately (DOCS-014 / #1271), not
  fixed here, to keep this diff to the crystallize cutover only.

## Risks / open questions

- **Test surgery, not just a caller flip.** Deleting the twin orphans five bats files that invoke
  it directly (`knowledge-crystallize.bats`, `-golden.bats`, `-ps1.bats`, `-yaml-guard.bats`) or
  regenerate goldens from it (`tests/golden/crystallize/capture.sh`). Resolved by checking
  equivalent coverage exists in the Go-native suite before deleting each one — not by assuming.
  `TestIsYAMLBlockScalar` (four/six-space indent) and `TestProcessProjectRefusesYAMLAndLeavesFileIntact`
  in `cli/internal/vault/crystallize_test.go` already cover what `-yaml-guard.bats` asserted
  against the shell; the go-parity suite's `yaml-wrapped` golden covers the CLI-level behavior.
  No coverage gap opened by the deletion.
- **The golden corpus's meaning changes, not just its runner.** `tests/golden/crystallize/`
  was captured to prove Go matched a live shell oracle. With the shell gone, those fixtures
  become the port's frozen characterization contract instead — documented in the parity suite's
  header and in `crystallize.go`'s own doc comment, so a future reader does not assume there is
  still something to regenerate from.
- **Two-tier deploy verification (lesson-150).** Before deleting, confirmed the *installed*
  `dotf` binary (`~/.local/bin/dotf`, not just the repo tree) already serves
  `vault crystallize --all` — it does; increment 1 shipped in an already-merged CLI-021 PR.

## Acceptance criteria

- [x] `scripts/knowledge-crystallize.{sh,ps1}` deleted.
- [x] Every caller (`vault-maintenance-weekly.{sh,ps1}`, `setup-linux.sh`, `setup-windows.ps1`,
      the SessionStart hook message, `.claude/CLAUDE.md`, the distillation runbook,
      `scripts/vault.sh`, `scripts/utils.ps1`) repoints at `dotf vault crystallize`.
- [x] `git diff --stat` touches only callers/docs/comments plus the deleted scripts and their
      superseded test files — no unrelated changes.
- [x] Go build/vet/test green (`cli/`); full bats suite green; `GOOS=windows go vet ./...` clean.
- [x] No coverage gap: every behavioral assertion the deleted bats files made has an equivalent
      passing Go-native test or golden case, verified by inspection before deletion (not assumed).

## References

- Bitácora board: mlorentedev/dotfiles#1269
- Parent tickets: #490 (CLI-021, built the Go port), #492 (CLI-023, the full 9-script cutover
  this slices out of), #491 (CLI-022, spec gate, still blocks #492)
- ADR-020 §5 — strangler-fig on contact
