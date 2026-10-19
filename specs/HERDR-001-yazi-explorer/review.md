---
spec: "HERDR-001-yazi-explorer"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "fbfedfc371a53148dd2766ced13f042762dd5ccb"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-18"
---
## Adversarial review

**Scope**: HERDR-001-yazi-explorer — `git diff 216fa210d6ddbd46a5eb89b22235ef8f6f2f7cf4...HEAD` (12 files, +869/−1: `ai/herdr/config.toml`, `cli/internal/cmd/{herdr.go,herdr_test.go,root.go}`, `tests/herdr-config.bats`, `versions.conf`, three lessons, the spec folder).
**Sources**: `specs/HERDR-001-yazi-explorer/{proposal,tasks,verification,features.json}.md` + the diff above + the live herdr 0.9.3 CLI on this machine.

### Spec and task alignment

- AC1–AC5 are implemented inside the stated scope; **AC6 is not**. `git log --all --grep=HERDR_AGENT` returns exactly one commit, `216fa210 fix(shell): name the agent in the wrapper…` — the review base. `.bashrc`, `.zshrc` and `tests/pi-config.bats` are untouched by this diff, yet `tasks.md` ticks three `[AC6]` implementation boxes and `proposal.md` says "The same change also lands the agent-detection fix (#2813)". It is on the branch, not in the reviewed change.
- The claimed order is honoured: no `dotf herdr explorer` binding exists in `ai/herdr/config.toml`, and the guard test that enforces it (`"no binding calls a dotf subcommand the pinned release does not have"`) is present and active while `DOTF_VERSION=0.75.0`.
- Everything else I could check reproduced. Fresh, this session: `go vet ./...` exit 0; `go test ./...` exit 0 (all packages `ok`); `bats tests/herdr-config.bats tests/pi-config.bats tests/versions-conf.bats` → 63 `ok`, 0 `not ok`; `HERDR_CONFIG_PATH=ai/herdr/config.toml herdr config check` → `config: ok`, and the **deployed** copy `~/.config/herdr/config.toml` → `config: ok` with `command = 'yazi'`, `key = 'prefix+e'` and `agent_panel_sort = 'priority'` actually present.
- The herdr CLI grammar the code assumes is real: `herdr pane close <pane_id>`, `herdr pane run <PANE_ID> <COMMAND>...`, `herdr pane process-info --pane <ID>`, and live `herdr pane list` returns `result.panes[].{pane_id,tab_id}` — the exact shape `herdrPaneList` decodes. A mock-only suite would have been the obvious place for a wrong-argv bug to hide; it is not wrong.
- Bite verified by mutation, then reverted (`git status` clean apart from this file): `--direction right` → `down` makes `TestHerdrExplorerOpensOnTheRightWhenNoneIsOpen` FAIL; `command = "yazi"` → `"yarai"` makes bats #10 FAIL. Both tests fail for the right reason, so the happy path is not vacuous.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | reliability / toggle | The toggle closes the **first pane in the tab whose foreground is `yazi`**, with no exclusion for the pane the command was typed into and no preference for the explorer it opened itself. A user who runs `yazi` by hand in their own pane, then presses the docked toggle, gets that pane closed (shell, or a running agent, killed) while the real explorer stays open. `explorerPane()` returns on the first match; ordering is whatever `pane list` yields. | `cli/internal/cmd/herdr.go` `explorerPane`/`paneRunsYazi`; `TestHerdrExplorerClosesTheOneAlreadyRunning` asserts `pane close w1:p1` where `w1:p1` **is** the invoking pane (`herdrPaneEnv(t, "w1:p1", …)`), so the suite encodes the behaviour rather than bounding it | UNTESTED for the two-yazi-panes case (`TestHerdrExplorerIgnoresOtherTabsExplorer` only covers another tab) | code + tests |
| Major | THEORETICAL | reliability / rollback | If `herdr pane run <id> yazi` fails after `pane split` succeeded, the error returns and the freshly split **empty pane stays open** — the exact outcome AC4 exists to prevent ("not an empty pane"). Same shape when yazi starts and dies immediately (bad TERM, broken `yazi.toml`): the command prints `opened the explorer in …`, the next call sees no yazi and opens a *second* pane, so explorers stack. No cleanup path, no "close what I just made" on error. | `herdr.go` RunE: `splitRight` then `herdrRun("pane","run",id,"yazi")` with the id discarded on failure; preflight (`herdrLookPath`) only covers *absent* yazi | UNTESTED — `TestHerdrExplorerRejectsASplitWithoutPaneID` covers a missing id, not a failed `pane run` | code + tests |
| Major | THEORETICAL | provisioning | Both the refusal message ("run `dotf tools sync`") and the proposal ("provisions yazi through mise on every OS; no manual install step exists") assume `sync` sees `YAZI_VERSION`. `env.ResolveVersionsPath()` → `resolveInCheckout` prefers a checkout, **else the deploy dir's copy** (lesson 494, written by this same change). On this machine `~/.dotfiles/versions.conf` (mtime Oct 18 12:32) is byte-identical to the repo's except for the missing `YAZI_VERSION` line, and nothing in this diff updates it — only `setup-linux.sh`'s copy step ever does. So on a deploy-dir machine, the remedy the error names does not provision the tool the keybinding needs. Disclosed confound: this branch is unmerged, so today's run may resolve `~/Projects/dotfiles` rather than the deploy dir; the trap itself is the one lesson 494 documents. | `diff` of repo vs `~/.dotfiles/versions.conf` → one line, `> YAZI_VERSION=26.9.1`; `cd $HOME && dotf tools sync --dry-run` → "40 tool(s) at their pin; nothing to install", no `yazi` line in the would-write dump, while `~/.config/mise/conf.d/dotfiles.toml:42` holds `yazi = "26.9.1"` | UNTESTED — `f5`'s command greps the checkout only; bats `"the explorer's yazi comes from a pin"` likewise | code (message) + tests; spec claim is contract set — disposition, not edit |
| Minor | REAL | spec traceability | AC6's three `[x]` tasks and its `features.json` entry `f6` have **no diff evidence in this scope** (base commit `216fa210` carries the wrappers and `tests/pi-config.bats`). The criterion does hold here — `bats tests/pi-config.bats` passes inside the 63/63 run — but the spec claims work it did not do in this change, which misreads the changelog for anyone bisecting. | `git log --all --grep=HERDR_AGENT` → `216fa210` only; diffstat lists no `.bashrc`/`.zshrc`/`tests/pi-config.bats` | `bats tests/pi-config.bats` (passes; not part of this diff) | spec (contract set → track/disposition) |
| Minor | REAL | correctness of artifacts | `features.json` `f5`'s verification is `grep -E '^YAZI_VERSION=…' versions.conf`, which proves a key was typed, not AC5's two halves (the pin *reaches the rendered mise config*; a second sync reports nothing to do). The harness can only ever mark `f5` passing from a file that renders nowhere. `f1`/`f6` similarly stand in for `herdr config check` and live detection with static greps. | `specs/…/features.json` vs `proposal.md` AC5; rendered proof exists on disk (`conf.d/dotfiles.toml:42`) but no command in the contract checks it | `dotf tools sync` from the checkout + `grep yazi ~/.config/mise/conf.d/dotfiles.toml` (run in verification.md, not codified) | spec (contract set → track/disposition) |
| Minor | REAL | docs / stale comment | `ai/herdr/config.toml` says "unlike **the docked binding below**, it needs no dotf subcommand" — the file contains no docked binding (`grep -n 'dotf herdr explorer'` → only the guard test reads it). The comment describes a follow-up that is deliberately out of scope, so it reads as drift to the next editor. | `cat ai/herdr/config.toml`; `grep -n "herdr explorer" ai/herdr/config.toml` → no hit | bats `"no binding calls a dotf subcommand the pinned release does not have"` asserts absence, so the comment contradicts a passing test | code (comment) |
| Minor | REAL | git history | Commit subject `fbfedfc3 feat(herdr): a yazi explorer on demand, and the toggle docked right` claims the docked toggle; the proposal lists "The docked keybinding itself" under **Out of scope** and no binding is added. Durable record contradicts the diff it names. | `git log --oneline` + diffstat | n/a | none in code — fix in the squash/PR message |
| Minor | THEORETICAL | CLI input validation | `--ratio` is passed through with no bounds: `strconv.FormatFloat(ratio,'f',2,64)` emits `0.00`, `-1.00`, `1200.00`, `+Inf`, `NaN` straight into `herdr pane split --ratio`, whose help declares no range. A degenerate value yields an invisible or refused split; in the invisible case the toggle cannot find the pane it cannot see. | `herdr.go splitRight`; `herdr pane split --help` (`--ratio <FLOAT>`, no bounds) | `TestHerdrExplorerHonoursTheRatioFlag` covers `0.45` only; out-of-range UNTESTED | code + tests |
| Question | UNVERIFIED | cross-boundary | Does a `prefix+e` **popup** running yazi appear in `herdr pane list` for that tab with `foreground_processes: [yazi]`? If it does, a `dotf herdr explorer` typed while a popup is open would `pane close` the pane hosting the popup. Not reachable from this session: no popup was open in the live `pane list` I read, and opening one needs the owner's keyboard. Author or owner should confirm against herdr's popup semantics. | live `herdr pane list` (read-only) showed no popup entries; no code/test asserts the popup-vs-split interaction | UNTESTED | tests (if popups do list) |
| Minor | REAL | hygiene / dates | `lesson-494` and `lesson-495` frontmatter carry `created: "2026-10-19"`; this session's date is 2026-10-18 (`date` → `Sun Oct 18 21:14 MDT 2026`), as do `proposal.md`/`tasks.md`. Off-by-one durable records sort wrong. | `head -8` of both lesson files | n/a | docs |
| Minor | REAL | handoff | `proposal.md` leaves every acceptance criterion `[ ]` while `tasks.md` ticks the "covered by at least one test" closing box, and `verification.md`'s count ("62 ok") is one below what the same three files report now (63). Stale-by-one is harmless; the mismatch between the two files' notion of "done" is what a reader trips on. | proposal.md AC list vs tasks.md closing block; fresh bats run → 63 ok | n/a | spec + verification.md (verification.md is outside the contract set — free to edit) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | AC1–AC5 verified live and by mutation-bitten tests; negative paths exist but the toggle's "first yazi pane wins" and no-rollback-on-`pane run` are unargued gaps. |
| Verification | B | Every criterion has a reproducible command and I re-ran almost all of them green; two counts/claims drifted (62 vs 63, AC6 sourced outside the diff). |
| Scope | B | Diff matches the proposal and the deliberate release-ordering hold; lesson-495 is session collateral, and the commit subject overstates what landed. |
| Reliability | C | Failure paths after a split are unhandled (orphan empty pane), `--ratio` is unvalidated, and the pane the toggle closes is chosen by list order, not by ownership. |
| Maintainability | B | All functions short and single-purpose, comments explain WHY, no dead code; one comment contradicts the file it sits in. |
| Handoff-readiness | B | Spec triad, three lessons, follow-up tracked on #2814, out-of-scope stated; contract-side artifacts (features.json f5, proposal AC ticks) left stale. |

Aggregation: no D, one C → PASS WITH GAPS minimum. No Blocker; no finding labelled REAL at Major severity, so the FAIL path is not taken.

### Verdict

**PASS WITH GAPS**

Gaps tracked, not fixed, each with a disposition line for the implementer to record in `verification.md` (outside the contract set):

1. *Toggle picks the first yazi pane in the tab* — code fix (exclude the invoking pane, or remember the pane id it opened) + a named test for two-yazi-panes; owner: follow-up on #2814.
2. *`pane run` failure leaves an empty pane; instant-exit yazi stacks explorers* — code fix (close the split on run failure) + test.
3. *Provisioning assumes a checkout; the deploy dir's `versions.conf` has no `YAZI_VERSION`* — either the refusal names `dotf tools sync` run from the checkout (or `--versions`), or a doctor check flags a pin missing from the source sync will read.
4. *`features.json` f5/f1/f6 verify less than their criteria* — contract set: disposition in `verification.md`, fix carried to the next spec touch, not edited now.
5. *AC6 sourced outside the review scope; commit subject overstates the diff; stale comment; lesson dates; proposal AC ticks* — apply in code/docs at will (none is contract-set except the proposal's AC wording, which stays as reviewed).

### Recommended next steps

- **Before `dotf spec archive`**: it is **advisable** — the review is fresh (contract digests unchanged; I did not touch `proposal.md`, `tasks.md`, `features.json`), passing, and signed by a pool member (`nan/qwen3.8-flash`). Nothing here blocks the archive.
- Disposition findings 1–3 in `verification.md` as applied / ticketed / declined-with-reason; findings 1 and 2 are the ones that can bite the owner's live session, so a follow-up issue on #2814 with the reproduction path stated is the cheapest honest exit.
- Ask the owner to answer the popup-vs-`pane list` question (finding "Question"), since only a live herdr session can settle it; if popups do appear, finding 1 escalates from THEORETICAL to REAL and needs the fix before the docked binding lands.
- Fold findings 6–7 (stale comment, commit subject) into the follow-up PR that finally lands `prefix+e`-for-the-dock, so the squash message describes what merged.
