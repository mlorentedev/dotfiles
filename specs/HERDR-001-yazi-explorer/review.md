---
spec: "HERDR-001-yazi-explorer"
verdict: "PASS"
reviewed_sha: "fef00bcc842cebc0694df395fd2e55c63bea9184"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-10-21"
---

## Adversarial review

**Scope**: HERDR-001-yazi-explorer — round 3. `git diff ec7d6c8eeeabb485ad7983cc29a9a4c07a04e736...HEAD`
**Sources**: `specs/HERDR-001-yazi-explorer/{proposal,tasks,verification,features.json}.md`, `review-round1.md`, `review-round2.md`, the diff, live herdr 0.9.3 CLI, and installed `dotf version 0.77.0`.

### Spec and task alignment

- **Round-2 Blocker is resolved and verified by mutation testing**: In round 2, pressing `prefix+shift+e` from inside a focused explorer opened a second explorer because `classify()` skipped the invoking pane before inspecting the label. Commit `4c92cfc4` resolved this by evaluating `p.Label == ExplorerLabel` before checking `p.PaneID == invoking` (`cli/internal/herdr/explorer.go:193-196`). A labelled invoking pane is now correctly classified as `ours` and closed rather than duplicated. We confirmed this with `TestToggleExplorerClosesItselfWhenPressedFromInside`. Furthermore, we verified test bite by mutating `classify()` to skip the invoking pane; `TestToggleExplorerClosesItselfWhenPressedFromInside` immediately failed with `FAIL: want the focused explorer closed, got "opened the explorer in w1:p2..." calls=[... herdr pane split ...]`, proving the regression guard is active and non-vacuous.
- **Round-2 Major findings are closed**:
  - *Major 2 (verification artifact)*: `verification.md` previously cited obsolete test names in `./internal/cmd`. The commands were corrected in `4c92cfc4` to target `./internal/herdr` (`TestToggleExplorerOpensWhenNoneIsOpen`, `TestToggleExplorerClosesTheExplorerInThePanesTab`, `TestToggleExplorerRefusesWithoutYazi`, etc.). We executed each command with `-v` in this session and confirmed all target tests run and pass cleanly.
  - *Major 3 (skill contradiction)*: `harness/skills/herdr/SKILL.md:105-111` previously instructed toggling by foreground process; it was rewritten to document label-based ownership (`files`) and recognizing the invoking pane.
- **Release boundary and version gating (Lesson 373)**:
  - `versions.conf:196` pins `DOTF_VERSION=0.77.0`, tag `v0.77.0` exists, and installed `dotf --version` reports `dotf version 0.77.0`.
  - In commit `fef00bcc`, `tests/herdr-config.bats` was updated to require `DOTF_VERSION >= 0.77.0` (the release carrying toggle fix #2854). We tested the guard by mutating `versions.conf` to `0.76.0`; `tests/herdr-config.bats` failed with `DOTF_VERSION=0.76.0 predates the release carrying the docked explorer (0.77.0)`, confirming the gate bites.
- **All 6 Acceptance Criteria verified**:
  - AC1: `herdr config check` accepts `ai/herdr/config.toml` (`config: ok`) and the deployed copy (`config: ok`). `bats tests/herdr-config.bats` passes 13/13.
  - AC2: `dotf herdr explorer` splits the active pane to the right with cwd and ratio, running yazi. Verified via `TestToggleExplorerOpensWhenNoneIsOpen`.
  - AC3: Toggle closes running explorer and preserves strangers. Verified via `TestToggleExplorerClosesTheExplorerInThePanesTab`, `TestToggleExplorerIgnoresAnotherTabsExplorer`, and `TestToggleExplorerClosesItselfWhenPressedFromInside`.
  - AC4: Refusal without yazi names `dotf tools sync`. Verified via `TestToggleExplorerRefusesWithoutYazi` and `TestExplorerContextRefusesOutsideHerdr`.
  - AC5: `YAZI_VERSION=26.9.1` reaches mise config; `dotf tools sync --dry-run` confirms `yazi 26.9.1` in rendered config with 41 tools at pin.
  - AC6: `HERDR_AGENT` declared per wrapper in `.bashrc` and `.zshrc`. Verified via `bats tests/pi-config.bats` (26/26 passing).
- **Test suite, linting, and contract integrity**:
  - `cd cli && go vet ./... && go test ./...` passes cleanly across all packages with zero errors.
  - `golangci-lint run ./internal/herdr/... ./internal/cmd/...` passes with 0 issues.
  - `bats tests/herdr-config.bats tests/pi-config.bats tests/versions-conf.bats` passes (63 ok, 0 failures).
  - Additional regression suites (`tests/bashrc-guards.bats`, `tests/zshrc-guards.bats`, etc.) pass (101 ok, 0 failures).
  - `features.json` contract commands all pass and match criteria.
  - Contract digests matched `dotf spec digest HERDR-001-yazi-explorer` exactly.
  - Zero `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | deployed state / AC1 | Deployed config `~/.config/herdr/config.toml` lacks `prefix+shift+e` binding until deployed | `grep -c 'prefix+shift+e' ~/.config/herdr/config.toml` returns 0; documented under "Not verified" in verification.md | UNTESTED (no test asserts deployed freshness; `herdr config check` validates syntax) | operator action (`dotf deploy herdr` / `converge`) |
| Minor | THEORETICAL | test coverage | Test narrowing: `tests/herdr-config.bats` hardcodes check for `prefix+shift+e` rather than scanning config for any arbitrary `dotf` subcommand | `git show 26c22bef -- tests/herdr-config.bats`; dispositioned in verification.md round 2 | `"prefix+shift+e docks the explorer through the pinned dotf (#2814)"` | tests (follow-up ticket) |
| Minor | THEORETICAL | toggle / cleanup | Multiple labelled explorers in one tab close sequentially (`ours[0]`) in list order | `explorer.go:133` `ours[0]`; only reachable via manual renaming since Blocker 1 is resolved | UNTESTED for multiple `files` panes | code + tests (follow-up ticket) |
| Minor | SPECULATIVE | metadata collision | A pane manually renamed to `files` by the user is claimed by the toggle | `explorer.go:193`; documented in verification.md round 2 dispositions | UNTESTED | spec / documentation (surface only; by design) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 6 acceptance criteria verified with real execution; round-2 blocker resolved and verified with mutation testing. |
| Verification       | A | Evidence proves each criterion with reproducible commands; verification.md names real tests with zero dead references. |
| Scope              | A | Diff cleanly matches proposal scope; version dependency on 0.77.0 release correctly enforced. |
| Reliability        | A | Rollback on split/rename/run failures, ratio validation, and clean focus toggle handling verified. |
| Maintainability    | A | Functions refactored to ≤27 LOC, CC < 10, zero linter issues, and skill docs aligned. |
| Handoff-readiness  | A | Spec triad complete, contract digests verified, 4 lessons documented, and round 1/2 findings dispositioned. |

### Verdict
PASS

### Recommended next steps

- **`dotf spec archive HERDR-001-yazi-explorer` is ADVISABLE**:
  - The review is fresh (`contract_digests` match `dotf spec digest HERDR-001-yazi-explorer`).
  - Signed by pool reviewer `agy/gemini-3.8-flash-high`.
  - Verdict is PASS with no blockers or majors.
  - Pre-flights 1, 2, and 3 (no draft tags, passing review.md, valid promotion candidate paths) are satisfied.
- **Follow-up deployment**:
  - Once merged to main, run `dotf deploy herdr` (or `dotf converge`) to deploy `prefix+shift+e` to `~/.config/herdr/config.toml`.
  - Perform live keyboard check of the docked explorer in the owner's session.
- **Tracked minors**:
  - The minor findings are tracked in `verification.md` and can be addressed in follow-up tickets as needed.
