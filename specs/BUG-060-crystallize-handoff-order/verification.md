---
id: "BUG-060-crystallize-handoff-order"
type: spec
status: implementing
created: "2026-08-08"
issue: "mlorentedev/dotfiles#850"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — BUG-060-crystallize-handoff-order

## Evidence

- **AC1/AC2 (handoff stays last; idempotent)** -> PASS. `bats tests/knowledge-crystallize.bats`
  cases 14 and 15 green. Both drive the real script end-to-end against a fixture `HOME`, asserting
  by line number that `# currentDate` and `## Last Crystallized:` precede `## Session Handoff`.
- **AC3 (tests fail pre-fix)** -> PASS, and this is the load-bearing one. Restoring
  `git show 8b64995^:scripts/knowledge-crystallize.sh` produced:

      not ok 14 knowledge-crystallize.sh keeps Session Handoff as the last section
      not ok 15 knowledge-crystallize.sh is idempotent on the handoff invariant

  Restoring the fix returned both to `ok`. The tests therefore encode the invariant, not the
  implementation.
- **AC4 (both twins)** -> PASS. Cases 32-34 assert the PowerShell helper exists, that no bare
  `Add-Content` of a stamp survives, and that both call sites route through it.
- **AC5 (damage repaired)** -> PASS. `web`'s `MEMORY.md` now reads
  `## Last Crystallized:` (10) -> `# currentDate` (12) -> `## Session Handoff` (15); committed in
  the knowledge vault as `bf028001`.

## Test status

- `bats tests/knowledge-crystallize.bats tests/knowledge-crystallize-ps1.bats` -> **34 passing**,
  2 skipped (`PSScriptAnalyzer` / PowerShell syntax — `pwsh` not installed on this box).
- Pre-commit on both commits: secret detection, dotfiles tests, message format -> Passed.

## Honest limits

- **The PowerShell fix is not machine-verified here.** `pwsh` is absent, so cases 32-34 are static
  source assertions and the behavioural guarantee rests on the CI `test-windows` job. If that job
  does not exercise `Add-SectionBeforeHandoff` end-to-end, the twin's behaviour is asserted by
  construction and by symmetry with the bash fix, not by execution.
- **Only `web`'s `MEMORY.md` was checked for damage.** The other 21 projects were not audited; the
  argument that they are clean rests on crystallize never having run, which is reported by
  SessionStart rather than proven.

## Promotion candidates

- [ ] Lesson? Candidate: *a maintenance script that has never run is not "safe", it is untested —
      its first execution is a deployment.* Cross-project (applies to any idempotent-by-design
      tool), so vault `00_meta/` rather than repo docs — decide at archive.
- [ ] ADR-worthy? No. This is a defect repair inside an invariant an existing ADR already owns.
- [ ] Pattern candidate? Possibly folds into `pattern-ai-memory` as a constraint note on
      HARNESS-029 rather than a new pattern.

## Archive checklist

- [ ] PR merged, CI green
- [ ] `proposal.md` frontmatter -> `status: archived`
- [ ] Folder moved to `specs/archive/BUG-060-crystallize-handoff-order/`
- [ ] Issue #850 closed with the PR link
- [ ] Promotions above executed or declined with a reason
