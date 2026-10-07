---
id: "CI-012-dedupe-ci-tests"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-10-07"
issue: "mlorentedev/dotfiles#2059"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "CI-012 rows are independent of the 19 open specs; three bounded rows, no shared files (19 active, limit 10, 2026-10-07)"
---

# CI-012-dedupe-ci-tests

> **Naming**: file lives at `<repo>/specs/CI-012-dedupe-ci-tests/proposal.md`. `CI-012-dedupe-ci-tests` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #2059: CI-012: the same behaviour is tested in several places, and the Windows leg re-runs 161 tests that only read the checkout -->

Governing rule of #2059: one behaviour, one test, at the cheapest layer that can prove it. This spec
tracks the rows of that issue one PR at a time; the first PR carries the Linux `test` job rows
(N4, N5, N7). Five bats files compiled their own `dotf` per file run, three of them with
`|| skip`, so a compile error in `cli/` read as skipped tests and a green job (BUG-055 / #807
class). Eleven cases in `dotf-agent-run.bats` re-asserted what `cli/internal/cmd` already tests, and
CI had no per-file bats timing for the #1744 budget guard to read.

## What

- **N4.** The Linux `test` job builds `dotf` once and exports `DOTF_BIN`. A shared helper
  (`tests/lib/dotf-bin.bash`) makes the five files use it. Without `DOTF_BIN`, a file builds once
  and a failed build FAILS (locally too); a missing Go toolchain skips on a developer machine and
  fails when `$CI` is set.
- **N5.** Eleven of the twelve `dotf-agent-run.bats` cases are deleted, each after its Go twin was
  shown to fail when the behaviour breaks; three twins that did not exist (the shipped top chain, the
  saturated pool through the command, the probe reaching a stub harness) were added first. The pipe
  case stays: it is the property only a real binary and a real pipe show.
- **N7.** The Linux bats step writes a junit report with timing and uploads it as an artifact.

## Out of scope

- N1, N2, N3, N6 and N8 of #2059: later PRs against this spec.
- `harness-suggest.bats` and `dotf-search.bats` (audit: "keep one smoke each"): their assertions read
  the shipped trigger data and the CLI wrapper's output, and no Go test pins either, so no twin was
  confirmed and nothing was deleted.
- The macOS CI leg and the `nproc` change (`ci/macos-leg`).

## Risks / open questions

- `ci.yml` edits the same bats `run:` line as `ci/macos-leg` (`nproc`); a textual conflict is
  expected and trivial.
- `DOTF_BIN` in the job environment reaches every bats file. Only the five helper users read it.

## Acceptance criteria

- [ ] AC1: a compile error in `cli/` fails the bats files that run the real binary, locally and in CI, instead of skipping them.
- [ ] AC2: with `DOTF_BIN` set, those files run that binary and do not build; set to a non-executable it fails.
- [ ] AC3: each deleted `dotf-agent-run.bats` case has a Go twin that goes red when the behaviour is mutated.
- [ ] AC4: the Linux bats step keeps `--jobs` and `--no-parallelize-within-files`, writes `report.xml` with per-file time, and uploads it even when the run is red.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `<repo>/docs/adr/adr-XXX.md` (if any)
- Audit: vault `10_projects/dotfiles/research/2026-10-07-ci-test-duplication-audit.md`
- Related: #1744 (budget guard fed by the N7 artifact), #2013
