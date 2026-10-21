---
spec: "HERDR-001-yazi-explorer"
verdict: "FAIL"
reviewed_sha: "26c22befd0ddba247dd66ed13ea9d5761c287d05"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-20"
---
## Adversarial review

**Scope**: HERDR-001-yazi-explorer — round 2. `git diff ec7d6c8eeeabb485ad7983cc29a9a4c07a04e736...HEAD`
(85 files, +3806/−228). Of those, the range also carries 15 unrelated merged PRs (main's history between
#2815 and the 0.76.0 release). This spec's change is `11614b9b` (#2815: `ai/herdr/config.toml`,
`cli/internal/cmd/herdr.go`, `cli/internal/herdr/explorer{,_test}.go`, `.bashrc`, `.zshrc`,
`tests/{herdr-config,pi-config}.bats`, `versions.conf`, `harness/skills/herdr/SKILL.md`, three lessons, the
spec folder) plus the round-2 delta `26c22bef` (the `prefix+shift+e` binding, its bats test, and the spec
artifacts). The review is of the whole range, with the new binding judged against the label rewrite it runs.
**Sources**: `specs/HERDR-001-yazi-explorer/{proposal,tasks,verification,features.json}.md`, `review-round1.md`
+ its dispositions, the diff above, and the live herdr 0.9.3 server this session runs inside (pane `w22:p1`).

### Spec and task alignment

- **The round-1 gaps were actually closed, and I proved the closures bite.** `cli/internal/herdr/explorer.go`
  now identifies its explorer by the `files` **label** and excludes the invoking pane (round-1 finding 1), and
  takes back the split when `pane rename` or `pane run` fails (round-1 finding 2). Mutations run and reverted
  (`git status -- cli` clean afterwards): dropping `|| p.PaneID == invoking` → `TestToggleExplorerNeverTouchesTheInvokingPane`
  **FAIL**; removing the rollback close → `TestToggleExplorerTakesBackAPaneItCannotFill` **FAIL** ("the split must be
  taken back"); `--direction right` → `down` → `TestToggleExplorerOpensWhenNoneIsOpen` **FAIL**.
- **Round-1 finding 3 (deploy-dir `versions.conf` has no `YAZI_VERSION`) is closed on this machine**:
  `~/.dotfiles/versions.conf` now carries the pin (mtime Oct 20 22:13, `grep -c '^YAZI_VERSION='` → 1) and
  `recordsMirror` (`cli/internal/converge/records.go:137-141`) is the committed mechanism that mirrors it.
- **Round-1 finding 4 is closed in `features.json` and NOT in `verification.md`** — see finding 2 below.
- **AC6 is now inside the reviewed scope** (base `ec7d6c8` predates `216fa210`'s wrapper work; `11614b9b`
  squash-carries `.bashrc`, `.zshrc` and `tests/pi-config.bats`), so the round-1 traceability finding is moot.
- **Lesson 373's release boundary holds, and I checked the released artifact, not the tree**: `versions.conf:196`
  `DOTF_VERSION=0.76.0`, tag `v0.76.0` exists, `dotf --version` → `dotf version 0.76.0`, and the **installed**
  binary answers `dotf herdr explorer --help`. The binding therefore names a subcommand a new machine will have.
- **Fresh green, all re-run this session**: `go vet ./...` exit 0; `go test ./...` exit 0 (every package `ok`);
  `bats tests/herdr-config.bats tests/pi-config.bats tests/versions-conf.bats` → **63 ok, 0 not ok** (matches
  verification.md's count); `golangci-lint run ./internal/herdr/... ./internal/cmd/...` → `0 issues.`;
  `HERDR_CONFIG_PATH=ai/herdr/config.toml herdr config check` → `config: ok`, same for the deployed copy.
- **The herdr contract the code assumes is real, not mock-shaped.** `herdr pane split --help` accepts exactly
  `--pane/--direction {right,down}/--cwd/--ratio`; `HERDR_ACTIVE_PANE_ID`, `HERDR_ACTIVE_TAB_ID`,
  `HERDR_ACTIVE_PANE_CWD` exist in the herdr 0.9.3 binary, so `ExplorerContext`'s precedence is genuine; and a
  live `pane rename w22:p1 files` put `"label":"files"` in `herdr pane list`'s output (cleared with
  `--clear`; `pane get` confirms it is gone). The end-to-end path reproduced too: `dotf herdr explorer` →
  `opened the explorer in w22:p3 at <worktree>`, a second call → `closed the explorer in w22:p3`, tab back to
  `w22:p1`. AC2 and AC3 hold **on the path the tests do not cover** (a typed command in a stable pane).
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in `proposal.md`, `tasks.md`, `verification.md` or `features.json`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| **Blocker** | **REAL** | toggle / focus | **Pressing `prefix+shift+e` while the explorer pane is the focused pane opens a SECOND explorer instead of closing it.** The keybinding exports `HERDR_ACTIVE_PANE_ID` = the focused pane; `classify()` skips the invoking pane unconditionally (`if p.TabID != tab \|\| p.PaneID == invoking { continue }`), so the explorer it owns is neither `ours` nor a `stranger` (labelled panes are skipped before the foreground check), `ours` and `strangers` come back empty, and the code splits again. `proposal.md` AC3 states "A second `dotf herdr explorer` in the same tab closes that pane and opens nothing" — it does not, on the interaction this feature exists for (you dock a file browser in order to be *in* it). The round-1 fix for "never close a stranger's yazi" over-corrected: the label, not the pane's identity, is the ownership token, so a labelled invoking pane *should* be a candidate. | Reproduced live in this review with the **installed dotf 0.76.0** (what the binding runs): explorer `w22:p4` existed (label `files`); `HERDR_ACTIVE_PANE_ID=w22:p4 dotf herdr explorer` printed `opened the explorer in w22:p5 at /tmp`; `pane list` then showed `[(w22:p1,None),(w22:p4,files),(w22:p5,files)]`. Both panes closed, tab restored to `w22:p1`. | **UNTESTED** — `TestToggleExplorerNeverTouchesTheInvokingPane` asserts the invariant that *causes* it (its fixture gives the invoking pane `label:"files"` and only checks nothing closes it); no test names the "invoked from the explorer" case | code + tests: treat a labelled invoking pane as `ours` (still never a `stranger`), amend that fixture to an unlabelled pane, add `TestToggleExplorerClosesTheExplorerWhenInvokedFromIt` (asserts `pane close w1:p1`, no `pane split`), then re-check live in a herdr tab |
| **Major** | **REAL** | verification artifact | `verification.md`'s Evidence for AC2, AC3 and AC4 runs `go test ./internal/cmd -run TestHerdrExplorer…`. That package has **no explorer tests any more** (`ls cli/internal/cmd \| grep herdr` → `herdr.go` only; the suite moved to `internal/herdr` and was renamed `TestToggleExplorer*`). I ran the three commands verbatim: each prints `testing: warning: no tests to run` / `[no tests to run]` and **exits 0** — the exact vacuous-pass mode round-1 finding 4 flagged in `features.json`, still live in the evidence file the archive relies on. The claims themselves are true (I re-ran the real tests green with `-v`, and reproduced AC2/AC3 live); the artifact does not prove them. | `cd cli && go test ./internal/cmd -run TestHerdrExplorerOpensOnTheRightWhenNoneIsOpen -v` → `ok … [no tests to run]`; same for the AC3 and AC4 `-run` patterns; the real names: `go test ./internal/herdr -v -run 'TestToggleExplorer\|TestExplorerContext\|TestValidateExplorerRatio'` → 13 `--- PASS` | proof exists (`TestToggleExplorerOpensWhenNoneIsOpen`, `…ClosesTheExplorerInThePanesTab`, `…RefusesWithoutYazi`, `TestExplorerContextRefusesOutsideHerdr`) but the artifact names dead ones | verification.md (outside the contract set — free to fix now; no re-review needed for this one alone) |
| **Major** | THEORETICAL | knowledge artifact / contradiction | `harness/skills/herdr/SKILL.md:105-107`, merged in this same change, tells the reader the opposite of what the code does: *"A right-docked explorer is a shell command that calls the herdr CLI… **Toggle it by the pane's foreground process, not by a pane label: any rename clears the label.**"* That is the exact mechanism round-1 finding 1 refuted (a foreground-process toggle closes a stranger's yazi), and it is the mechanism `ExplorerLabel` was introduced to replace — the shipped comment says so: *"Identity is a label rather than 'a pane running yazi' because a user's own yazi is a pane running yazi too."* The file is a generated skill with `paths: ['**/ai/herdr/**','**/herdr*/**']` and it is loaded into the next agent's context, so a future session that follows it reverts the ownership fix. The contradiction is observed verbatim in one merged tree; the regression it invites is the plausible consequence. | `grep -n 'docked\|label' harness/skills/herdr/SKILL.md` vs `cli/internal/herdr/explorer.go:14-18`; both landed in `11614b9b` | UNTESTED — nothing binds the skill's prose to the code | the skill (`harness/skills/herdr/SKILL.md`, and its vault source `00_meta/skills/herdr/SKILL.md` per the generated header) — not contract set |
| Minor | REAL | deployed state / AC1's second half | AC1 is "…and **the deployed copy**". The live deployed config `~/.config/herdr/config.toml` (mtime **Oct 18 21:07**, i.e. before the binding commit) has **0** occurrences of `prefix+shift+e`, so on this machine the key the change adds is inert until `dotf deploy herdr` (or `converge`, whose record list carries the merge-strategy entry) runs. verification.md's AC1 evidence (`config check` on the deployed copy → `config: ok`) is true of that stale file, so it evidences nothing about this delta. The repo discloses the rest honestly under "Not verified". | `grep -c 'prefix+shift+e' ~/.config/herdr/config.toml` → `0`; `grep -n 'prefix' …` shows only `prefix+d`, `prefix+e`; `ai/deploy.json` entry uses `strategy: merge` (`cli/internal/deploy/merge_toml_test.go:17`) | UNTESTED (no test asserts the deployed copy carries the binding; `herdr config check` validates a file, not freshness) | verification.md + one `dotf deploy herdr` and one live keypress; the AC1 wording itself is contract set — leave it as reviewed |
| Minor | THEORETICAL | test deletion / narrowing | `26c22bef` **deletes** the general guard `@test "a binding never names a dotf subcommand the pinned release does not have (#2814)"` (it grepped the whole config for any `command = "dotf herdr explorer"`) and replaces it with a per-binding test that hardcodes `key = "prefix+shift+e"` and its command string. The lesson-373 property survives *for this binding* — I proved it: `DOTF_VERSION=0.75.0` → `not ok 1 prefix+shift+e docks the explorer through the pinned dotf`, reverted → `ok`. What is lost is coverage of the *next* binding naming a tree-only subcommand, which the old grep would have caught. Retention fields mostly recorded (the deleted test, its failure mode, `tasks.md`'s line + lesson 498 for the narrowing, and the focused command `bats tests/herdr-config.bats -f "docks the explorer"`); field 2 (stronger proof that remains) is now shown above; field 3 (non-test callers) n/a. | `git show 26c22bef -- tests/herdr-config.bats` (−`_pin_ge`, −the grep guard, +the hardcoded block check); mutation run and revert | named and bitten: `"prefix+shift+e docks the explorer through the pinned dotf (#2814)"` | tests (option: keep a config-wide scan that every `command = "dotf …"` subcommand is in the pinned release's `--help` tree), or disposition as accepted debt |
| Minor | THEORETICAL | multiple explorers / ordering | Once two labelled explorers exist (finding 1 makes that reachable by ordinary keypresses), the toggle closes `ours[0]` — whichever `pane list` yields first, not the one beside the invoking pane — and it takes one press per stacked pane to unwind. Bounded, but it is the state finding 1 creates. | `explorer.go` `if len(ours) > 0 { … ours[0] }`; my repro left two `files` panes in `w22` | `TestToggleExplorerClosesTheExplorerInThePanesTab` covers one; the two-ours case UNTESTED | code + tests, together with finding 1 |
| Minor | REAL | quality rule (function length) | `ToggleExplorer` is 43 lines from `func` to `}` (repo standard: < 40). Nothing enforces it: `.golangci.yml` enables `gocyclo` at `min-complexity: 15` with `path-except` exclusions, no `funlen`, and the package lints clean. CC itself is fine. | `awk '/^func ToggleExplorer/,/^}/' cli/internal/herdr/explorer.go \| wc -l` → 43; `golangci-lint run ./internal/herdr/...` → `0 issues.` | n/a | code (extract the open branch once finding 1 touches the function anyway) |
| Minor | SPECULATIVE | ownership token collision | A user who renames any pane `files` by hand (`herdr pane rename <id> files`) hands that pane to the toggle; `classify` trusts the label with no second check. Surface only — the label is the documented token, and the alternative (a hidden marker) is not expressible in herdr's config-free pane metadata. | `explorer.go` `if p.Label == ExplorerLabel { ours = append(...) }`; no test for a hand-labelled pane | UNTESTED | — (surface only; do not gate) |
| Question | UNVERIFIED | keybinding focus | Does a `type = "shell"` command's `pane split` **focus** the new pane? If it does, finding 1 fires on the *immediate* second press, not just after the user navigates in. My CLI-initiated splits left `focused:true` on `w22:p1`, and `pane split --help` exposes `--focus`/`--no-focus` without stating the default for a keybinding-driven command. Only a real `prefix+shift+e` press settles it — the owner's keyboard, or one added test that drives the command with the explorer as the active pane. | `herdr pane list` → `[(w22:p1,True),(w22:p4,False),(w22:p5,False)]` after two CLI splits | UNTESTED | tests / owner answer |
| Question | UNVERIFIED | popup vs docked toggle (carried from round 1) | Is a `prefix+e` **popup** running yazi listed in `pane list` for the tab? With the label rewrite the failure mode changed but is still unproven: an unlabelled pane whose foreground is yazi is now a `stranger`, so the docked toggle would *refuse* ("a pane running yazi is not this explorer") while a popup is open, instead of docking. No popup was open in the read-only `pane list` I took, and I could not open one without the keyboard. | round-1 same question; `classify()`'s `strangers` branch; live `pane list` had no popup entries | UNTESTED | tests if popups do list |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | C | AC1/AC2/AC4/AC5/AC6 verified (live + fresh suite), but AC3 — the toggle — is violated on a reproduced, ordinary path. |
| Verification | C | Every criterion is reproducible, yet three of six evidence commands in `verification.md` match no test and pass vacuously; `features.json` is corrected and I re-ran all six. |
| Scope | B | `26c22bef` is exactly the promised follow-up (binding + artifacts in line with the code); the rest of the range is main's merged history, and the release-ordering hold is honoured. |
| Reliability | C | Rollback on `rename`/`run` failure, ratio validation, refusal without yazi and outside herdr all exist and bite; stacking explorers and list-order close do not. |
| Maintainability | B | Lint clean, small single-purpose functions, comments state WHY; one 43-line function over the repo's 40-line rule and no `funlen` to catch it. |
| Handoff-readiness | C | Three lessons + ADR-none declared + honest "Not verified"; but the shipped herdr skill contradicts the shipped identity mechanism, and the evidence file names dead tests. |

Aggregation: three C, no D → PASS WITH GAPS by rubric; **one Blocker (REAL)** → FAIL. Use the more severe path.

### Verdict

**FAIL**

Round 1's two reliability Majors were fixed and I verified the fixes bite. The fix for the first one is the
cause of this round's Blocker: excluding the invoking pane from *being closed* also excluded it from *being
recognised*, so the docked key opens a second explorer whenever the explorer has focus — reproduced against the
released `dotf` 0.76.0 the binding runs, in a live herdr tab.

### Recommended next steps

None of these are contract-set edits; `proposal.md`, `tasks.md` and `features.json` stay exactly as reviewed
(the AC3 sentence is the contract the code must meet, so the code changes to it, not the other way round).

1. **Finding 1 (Blocker) — minimum set to flip this to PASS.** In `classify`, let the invoking pane be an `ours`
   candidate when its own label is `ExplorerLabel` and keep it out of the `stranger` set (an unlabelled invoking
   pane stays untouchable, which is what round-1 asked for). Amend `TestToggleExplorerNeverTouchesTheInvokingPane`
   so its invoking pane is unlabelled (it currently asserts the buggy invariant), and add
   `TestToggleExplorerClosesTheExplorerWhenInvokedFromIt` asserting `pane close <invoking>` and **no** `pane split`.
   Re-check live in a herdr tab, then request round 3 (`dotf spec review HERDR-001-yazi-explorer`), because a REAL
   Blocker needs a fix *and* a re-review, not a fix alone.
2. **Finding 2 (REAL Major) — `verification.md`.** Point the AC2/AC3/AC4 commands at `./internal/herdr` and the
   `TestToggleExplorer*` names, and re-run them with `-v` so a future reader sees 13 `--- PASS` instead of
   `[no tests to run]`. Add the round-1 finding-1/2 closures' live re-check once step 1 lands.
3. **Finding 3 — the skill.** Rewrite the §4 bullet in `harness/skills/herdr/SKILL.md` (and its vault source, per
   the `generated_from` header) to the label mechanism, or delete the foreground-process advice; leaving it is an
   invitation to undo the ownership fix.
4. **Finding 4 — the owner's machine.** `dotf deploy herdr`, then `HERDR_CONFIG_PATH=~/.config/herdr/config.toml
   herdr config check`, then one real `prefix+shift+e` press; that press also answers the focus Question (step 1
   depends on knowing whether the immediate second press stacks). Record the result in `verification.md`'s
   "Not verified" section either way.
5. **Findings 5–8** — disposition in `verification.md` (apply / ticket on #2814 / decline with a reason). The
   deleted general guard is the one worth a ticket: one commit from now, a second dotf-naming binding lands
   unguarded.
6. `dotf spec archive` is **not advisable** in the current state: the gate would accept this file's frontmatter
   (fresh contract digests, a pool reviewer), but the verdict it parses is FAIL and it Blocks(). Archive after
   step 1 lands and round 3 signs it.
