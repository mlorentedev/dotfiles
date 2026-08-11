---
spec: "DOCS-013-doc-path-guard"
verdict: "FAIL"
reviewed_sha: "f5ee9a3e68d7629788f12202ffb936f05e14461e"
reviewer: "claude-sonnet-5"
date: "2026-08-11"
---

## Adversarial review

**Scope**: DOCS-013-doc-path-guard, round 3. No PR open;
`git -C /home/manu/Projects/dotfiles-wt-r2 diff main...HEAD` (base `6267bea`, head `f5ee9a3e`)
touching `scripts/check-doc-paths.sh`, `tests/check-doc-paths.bats`, `.claude/CLAUDE.md`,
`README.md`. `main` already carries round 1 (#922); this branch is rounds 2+3.
**Sources**: `specs/DOCS-013-doc-path-guard/{proposal,tasks,verification}.md`, read only
*after* independently reviewing the diff and running the code, per instruction. Every
mutation and factual claim below was executed in this session, not inferred from
`verification.md`'s prose.

### Method

Read the diff and the full `scripts/check-doc-paths.sh` / `tests/check-doc-paths.bats`
cold. Ran the guard directly under `zsh scripts/check-doc-paths.sh …` (interpreting the
script's actual content under zsh, not just its bash shebang) against all governed files —
clean. Then mutation-tested each of the twelve `@test` cases individually: edited the
one code path each test claims to cover (disabled a branch, reordered two checks, dropped
a filter clause, flipped a condition), ran only that test with `bats -f`, confirmed red,
reverted via a saved baseline copy, confirmed the full suite green again before moving to
the next. Independently reproduced the zsh-vs-bash `.` builtin behavior in a scratch
directory (not by reading the test). Built `dotf` from `cmd/dotf` (unchanged between
`main` and this branch — confirmed via `git diff main...HEAD --stat -- cli/`, empty) and
ran its subcommands to check every CLI claim in the prose. Cross-checked the governed-file
list against `harness/manifest.json`'s deploy targets, `setup-linux.sh`/`setup-windows.ps1`
deploy logic, and a repo-wide `find` for agent-instruction-shaped files.

### Spec and task alignment

Round 2 (`reviewed_sha f91a08d`) returned FAIL on three Majors: traversal-check ordering,
a regex blind spot on flush-left source lines, and two ungoverned instruction files
(`cli/AGENTS.md`, `ai/hermes/AGENTS.md`). All three are addressed in this diff, and two of
the three are cleanly fixed — see Findings for the third. The one Minor (stale "last two
rows" blockquote) is also fixed. Acceptance criteria in `proposal.md` are unchanged since
round 1 and not stale relative to this review (no commit after `6a40da3` touches
`proposal.md`, `tasks.md`, or `features.json`).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | governed-file list | `ai/agy/AGY.md` is an instruction/pointer file structurally identical to the two files round 3 added (`cli/AGENTS.md`, `ai/hermes/AGENTS.md`) — same "read AGENTS.md first, agent-specific extensions" shape as `ai/claude/CLAUDE.md` — but is absent from `instruction_files()` in `tests/check-doc-paths.bats`. `check-doc-paths.sh` is invoked nowhere else in the repo (`grep -rl check-doc-paths` outside `tests/` and the script itself returns nothing), so this bats list is the entire enforcement surface. `ai/agy/AGY.md` is deployed by both `setup-linux.sh:509-514` and `setup-windows.ps1:1397-1404` (each verifies the deployed copy points at `AGENTS.md`), and this very PR's own `README.md:63` documents it as a peer of `ai/claude/CLAUDE.md` in the Structure tree. A #916-class incident (the precipitating bug for this entire feature) landing in `AGY.md` would ship silently, forever — exactly the failure mode this spec exists to close, on a file this round's own fix pattern (widen the governed list) should have caught. `check-doc-paths.sh ai/agy/AGY.md` currently exits 0, so there is no live incident today — this is a coverage gap, not an active false-negative. | `find ai -maxdepth 1 -type d` → `agy claude copilot hermes nan opencode pi`; `grep -n AGY.md setup-linux.sh setup-windows.ps1`; `bash scripts/check-doc-paths.sh ai/agy/AGY.md` → `OK` (file itself is currently clean, but ungoverned) | UNTESTED — no bats case in `tests/check-doc-paths.bats` names or exercises `ai/agy/AGY.md` | tests — add `"ai/agy/AGY.md"` to `instruction_files()` in `tests/check-doc-paths.bats` |
| Minor | REAL | README accuracy | `README.md`'s Structure tree claims `# ~50 scripts total` for `scripts/`; actual count is 39 files (no subdirectories). Pre-existing since round 1 (`git show main:README.md` has the same line before this branch), not a regression introduced by this round, and not part of this diff — but it is a factual statement in a file this feature's own AC governs ("every repo path... resolves"), and it fails the same "prose is a live claim" spirit the guard enforces for backticked paths (the count itself isn't backticked, so `check-doc-paths.sh` cannot and does not catch it — out of the guard's declared scope by design). | `find scripts -maxdepth 1 -type f \| wc -l` → 39 | UNTESTED — no test checks prose counts | docs — correct the README count (e.g. "~40") or drop the approximation |
| — | THEORETICAL | guard design | The guard only extracts backticked tokens (`grep -o '`[^`[:space:]]*`'`); a markdown link whose visible text is *not* backticked (`[see here](docs/foo.md)` rather than `` [`docs/foo.md`](docs/foo.md) ``) bypasses token extraction entirely, not just the already-disclosed bare-filename blind spot. No live instance exists today — every repo-relative markdown link in both governed files was checked by hand and either is also backticked (and thus already covered) or resolves. Surfacing only; not scored against the verdict (THEORETICAL, no reproduction). | Manual `grep -noE '\]\([^)]+\)'` sweep of `README.md` and `.claude/CLAUDE.md`, all 9 targets resolved | UNTESTED | spec/docs — worth a line in the script's own "what counts as a repo path" header if the convention should also apply to plain markdown links |

### Independently reproduced (not merely trusted from `verification.md`)

- **Traversal-ordering fix (round 2 Major #1):** reordering the `..`-escape check back to
  *before* `is_repo_rooted` (the exact bug shape) turns case 10
  (`a .. token that is not repo-rooted stays ignored`) red; restoring the order turns it
  green. The fix in the diff (`is_repo_rooted` gate before the `case "/$token/" in */../*`
  block) is correct.
- **Regex blind spot fix (round 2 Major #2):** round 2's `verification.md` claimed a
  mutation "turns case 11 red... this session did not run it" is the exact failure class
  round 2 itself caught in round 1's write-up. I did not repeat that mistake: I directly
  grepped a flush-left `. versions.conf` line inside a fenced block with both the old
  (delimiter-required) and new (`(^|[^./a-zA-Z0-9_-])`-alternation) regex — the old one
  misses it, the new one catches it. Also reintroduced the literal historical bug
  (`. versions.conf` in place of `. ./versions.conf`) into a scratch copy of
  `.claude/CLAUDE.md` and confirmed case 12 goes red, then reverted.
- **All twelve `@test` cases discriminate** what their names claim — each was mutation-
  tested individually (chmod -x for case 1; a real broken path injected into a scratch
  copy of a governed file for case 2; disabling/inverting the relevant `if`/condition for
  cases 3, 4, 5, 7, 8; reordering for case 10; removing the traversal case entirely for
  case 9; targeted removal of filter clauses for case 6). One nuance on case 6: of its
  nine sub-shapes, only the `<`/`>` branch of the metacharacter filter is load-bearing
  (removing it makes `ai/<agent>/` a false failure, because `ai` is a real top-level repo
  directory); the `&` branch is redundant with `is_repo_rooted` for the tested `&>/dev/null`
  shape, same as the three shapes the script's own comment already discloses for the
  leading-character filter. This matches the test's own comment ("pins the OUTCOME... not
  the mechanism") and is not a new gap — case 6 still fails on a real regression
  (`<`/`>` removed), just not on every conceivable sub-mutation.
- **zsh vs. bash `.` builtin claim:** reproduced from scratch in a throwaway directory,
  independent of the test file — `. testconf.conf` (no slash) resolves the variable under
  bash and fails silently (empty variable) under zsh; `. ./testconf.conf` works under both.
  Matches the claim in `.claude/CLAUDE.md`'s prohibited-pattern table and the
  `versions.conf` install snippet.
- **CLI/doc factual claims**, checked against a `dotf` binary built from `cmd/dotf` at this
  head (`cli/` is byte-identical to `main` on this branch): `dotf secrets run`,
  `dotf secrets ls/verify/show/set`, `dotf env path <KEY>`, `dotf doctor --fix` (repairs
  `core.hooksPath` and junctions — both confirmed in `cli/internal/doctor/`) all exist and
  match their described behavior. `cli/internal/doctor/checks_golangci.go` exists and calls
  `matchPin`; `versionMatches` is a real table in `checks_tools.go`. 37 `SKILL.md` records
  under `harness/skills/` (matches "37 custom skills"). 1210 `@test` cases across 82 `.bats`
  files (matches "1200+ BATS tests"). `env-mapping.conf`, `doctor.sh`, `healthcheck.sh`
  confirmed absent from the repo (matches the "retired" claims). All markdown links in both
  files resolve. `shellcheck scripts/check-doc-paths.sh` clean; `check-md-escapes.sh` clean
  on all four touched files; `docs-drift.bats` (a separate content-sync guard, unaffected by
  this diff) still 6/6.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | The reordering and regex fixes are genuinely correct and mutation-verified; the one real defect is the incomplete governed-file list (Major above). |
| Verification       | B | Every specific round-3 evidence claim I re-ran (case 10, case 12, zsh execution, shellcheck) reproduced cleanly; historical caution from round 2's unverified claim is why I re-ran rather than trusted, and round 3 held up. |
| Scope              | A | Diff is precisely the three round-2 Majors + one Minor, nothing extraneous. |
| Reliability        | B | `set -euo pipefail`, correct exit codes (0/1/2) on all paths tested, including usage error and not-a-file. |
| Maintainability    | B | Dense but load-bearing comments explain WHY at each branch; no dead code. |
| Handoff-readiness  | B | `verification.md` is thorough and its Round 3 section's claims reproduce; `proposal.md`'s acceptance text still says "six instruction files" against an actual eight (soon nine) — small, pre-existing drift, not touched this round. |

### Verdict
FAIL

One REAL Major (incomplete governed-file list — the exact defect class round 3's own fix
was supposed to close, on a file meeting the identical criterion) forces FAIL under the
skill's severity-axis rule, independent of the all-B rubric. This is a narrower gap than
rounds 1 and 2 found: no active false negative or false positive ships today, the fix is a
one-line addition to an existing list, and everything else examined — the traversal-
ordering fix, the regex-alternation fix, all twelve tests' discriminating power, and every
sampled factual claim in both governed files — held up under direct execution.

### Recommended next steps (before archive)

- Add `"ai/agy/AGY.md"` to `instruction_files()` in `tests/check-doc-paths.bats` (one line;
  `governed_files()` derives from it automatically, so no second edit needed). Re-run
  `bats tests/check-doc-paths.bats` to confirm it stays green at 12/12 with the ninth file
  included, and update `verification.md`'s "Guard clean on all **eight**" claim to nine.
- While there: double check no other `ai/<agent>/*.md` pointer file was missed. This
  session's `find ai -maxdepth 1 -type d` found exactly seven agent directories
  (`agy claude copilot hermes nan opencode pi`); `nan`, `opencode`, and `pi` carry only
  human-facing `README.md` config docs (not "SYSTEM META-INSTRUCTION" pointer files reading
  `AGENTS.md`), so they are correctly out of scope — but a fresh grep is cheap insurance
  given this is the second round in a row this exact category has needed a fix.
- Optional, non-blocking: correct README's "~50 scripts total" (actual 39), and consider
  whether the guard's backtick-only extraction should also flag non-backticked markdown
  links to repo paths — both are Minor/THEORETICAL and do not need to gate this round.
- Re-run `/adversarial-review` at the new head after the fix — given the base rate so far
  (three rounds, three distinct real defects, two of them in code the previous round had
  already examined), a fourth pass over a one-line, low-risk fix can reasonably be light,
  but should still exist per this feature's own "incident-to-guard" logic.
