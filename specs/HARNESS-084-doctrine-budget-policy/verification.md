---
tags: [spec, verification, templates]
created: "2026-09-05"
---

# Verification - HARNESS-084-doctrine-budget-policy

## Evidence

- [x] **AC1–AC3** — `bats tests/compile-harness.bats -f 'full-only'` → `1..1 ok`. The guard
      asserts the region is absent from `.gemini/GEMINI.md`, present in the full surface,
      that `rule two after the region` survives, and that no `full-only:` marker reaches
      either file.
- [x] **AC4** — `render_region` builds its output into a variable and checks the status before
      filtering. See below.
- [x] **AC5** — `bats tests/skills-pipeline.bats` → *"HARNESS-056: the compact doctrine payload
      carries it and stays under its cap"* ok. It was **12826 chars against a 12000 cap**
      before this change, 826 over.
- [x] **AC6** — `bats tests/compile-harness.bats` 73/73 ok. `shellcheck` reports **6 findings
      before and 6 after**, measured by stashing the change; none introduced.

## Test status

- `tests/compile-harness.bats`: 73/73.
- `tests/skills-pipeline.bats`: one failure, **`BUG-771`**, confirmed pre-existing by stashing
  the change and re-running on a clean tree — it fails there too. That is #1409, a fixture
  isolation issue that is green in CI.

## Decisions made during implementation

- **The region moves; it does not shrink.** Raising the cap was rejected on #1241 (12000 is the
  vendor's limit, so raising the assertion moves truncation to silent loss at the destination),
  and trimming was worse: the compact payload substitutes for the constitution on those
  surfaces, and shortening a prohibition is how it stops being one.
- **What goes behind the marker is a safety boundary before a budget one.** A capped rules file
  carries the prohibition; the exception, its four conditions and their reasoning are a decision
  a human makes on a specific PR, and an agent that knows an exception exists may try to qualify
  for it.
- **The guard asserts both directions on purpose.** A marker that dropped its region from every
  render would pass any cap check while deleting doctrine — from the cap's point of view,
  identical to success. Same reason it asserts the region closes: a missing end marker would
  swallow the rest of the record and only make the payload smaller.

## A defect committed while writing this

`render_region` was first written as `render_region_raw "$@" | grep -v ...`. That returns
**grep's** exit status, so the `return 1` for a missing source-of-record was swallowed while its
error message still printed. Three `HARNESS-072` coverage tests went red and are what caught it.

This is lesson 268 and the repo's own prohibited-pattern table — *"drop the pipe and use `$?`"* —
met while adding a feature to the script that generates those very rules. Recorded because the
near-miss is more useful than the fix.

## Review dispositions (round 1, agy/gemini-3.1-pro-high, FAIL)

- **Blocker, unclosed `full-only` region truncates silently: applied.** `render_region_compact`'s awk now ends with an `END` check that prints `a full-only:begin has no matching full-only:end` and exits 1, and `set -euo pipefail` carries that out of `deploy_doctrine`. Guard: `HARNESS-056: an unclosed full-only region fails the deploy instead of truncating the payload`. It was red before the fix (`[ "$status" -ne 0 ]` failed) and green after; `tests/compile-harness.bats` and `tests/skills-pipeline.bats` 110/110. This corrects the earlier claim that the guard "asserts the region closes": it asserted that a *closed* region ends, which is not the same thing.
- **Minor, `fold_to_ascii` and `migrate_legacy_preamble` undocumented: declined.** Neither is this spec's change. Both landed in #1685 (`ebc2e20c`), and they reached the reviewed diff because the review base (`4d0ffa99`) predates that merge.
- **Minor, command substitution strips trailing newlines in `render_region`: declined.** The full surface is a managed region that `replace_region` writes line by line, so trailing blank lines carry nothing. The reviewer marked it "no action required".

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the near-miss above is lesson 268's class, already recorded.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a render mechanism inside the existing harness compile contract.
- [x] New pattern candidate for `00_meta/patterns/`? no: the mechanism is repo-specific.
