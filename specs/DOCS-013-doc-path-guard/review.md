---
spec: "DOCS-013-doc-path-guard"
verdict: "FAIL"
reviewed_sha: "3c6d77d768c2afdbbe3e009e5cd55fd52fe590bb"
reviewer: "claude-sonnet-5"
date: "2026-08-11"
---

## Adversarial review

**Scope**: DOCS-013-doc-path-guard, round 5. Merged directly to `main` (squash
`caa7af5`, PR #924); no open PR to diff against. Reviewed the CURRENT STATE of
the feature on `main` at `3c6d77d` (`caa7af5` plus one unrelated release commit
`3c6d77d` and one unrelated spec-grammar commit `6267bea`, neither of which
touches the governed files). Read the round 1-4 history in `verification.md`
only to identify what must already hold as a floor, then re-derived everything
independently rather than trusting the round-4 `review.md`'s FAIL-with-fix
narrative or round 5's self-reported fix in `verification.md`.

**Sources**: `specs/DOCS-013-doc-path-guard/{proposal,tasks,verification}.md`,
`scripts/check-doc-paths.sh`, `tests/check-doc-paths.bats`,
`cli/internal/spec/review.go` (frontmatter/staleness contract), plus direct
execution: `bats`, `shellcheck`, `bash -n`/`zsh -n`, manual fault injection
(`index.lock` during the probe tests) and two guard-condition mutations, all
performed in a disposable local clone (`git clone --local`) so the real working
tree was never touched. Also ran the guard directly against every
`README.md` in the repo, not only the governed set, to check the discovery
contract's own boundary rather than only the files it already declares itself
responsible for.

### Spec and task alignment

- Round 4's Major (`ai/probe-agent/AGENTS.md` could be stranded in the real
  tree if `git add -N` failed mid-test) is genuinely fixed, verified by
  reproducing the exact failure condition: `touch .git/index.lock`, then
  `bats tests/check-doc-paths.bats`. Both probe cases (8 and 9) fail as they
  must (`git add -N` returns 128), and `find ai harness -iname '*probe*'` plus
  `git status --short` afterward show nothing — `teardown()`'s unconditional
  cleanup holds under the concurrency scenario it was written for. This is a
  real regression test of a real prior defect, not a re-read of the diff.
- All 14 bats cases pass, three separate ways: on `main` directly, in the
  disposable clone at the same sha, and again in the clone with the
  `index.lock` fault present (cases 8/9 correctly `not ok`, the other 12 stay
  green). `shellcheck`, `bash -n`, `zsh -n` clean.
- `instruction_files()` discovers exactly the 9 files every prior round
  settled on (`.claude/CLAUDE.md`, `AGENTS.md`, `README.md`,
  `.github/copilot-instructions.md`, `ai/agy/AGY.md`, `ai/claude/CLAUDE.md`,
  `ai/copilot/copilot-instructions.md`, `ai/hermes/AGENTS.md`,
  `cli/AGENTS.md`), and the guard reports `OK` on every one of them.
- Two guard-condition mutations, run to confirm the suite is not vacuous:
  dropping the ALL-CAPS/`KNOWN_EXT` guard's negation (line 128) turns cases
  4, 6, 7 and 11 red — the suite catches this class, including the exact
  `SKILL.md` false-negative round 1 already fixed once. Removing `AGY\.md`
  from the discovery alternation (simulating a silently re-dropped file, the
  same defect class rounds 2-4 each found once) leaves the suite **fully
  green** — see Findings.
- `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags: none present in any spec file.
- Full regression sweep, `bats tests/*.bats` on `main` at `3c6d77d`: **1212
  total, 1 failure** (`not ok 408 converges over a running dotf: a live binary
  in dest is replaced, not refused`) — the same pre-existing, disclosed,
  unrelated failure `verification.md` names (BUG-054/#807). No regression
  attributable to this spec.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| Major | REAL | `tests/check-doc-paths.bats` — `instruction_files()` discovery contract | The discovery regex is asymmetric with no stated reason: `AGENTS\.md`, `CLAUDE\.md`, `AGY\.md`, `GEMINI\.md`, `copilot-instructions\.md` are matched anywhere (`(^\|/)…$`), but `README\.md` is matched **only at repo root** (`^README\.md$`). This is not in the exclusion list the same file requires ("excluding one requires saying so below, with a reason") — it is a silent asymmetry in the positive pattern. Consequence, reproduced directly: `./scripts/check-doc-paths.sh cli/README.md` reports 2 missing paths — `scripts/doctor.sh` and, specifically, **`scripts/healthcheck.sh`, backticked**, the exact file whose staleness in `.claude/CLAUDE.md` is this spec's own founding incident (proposal.md "Why": "two sessions in two days acted on one of them — `./scripts/healthcheck.sh`"). `./scripts/check-doc-paths.sh ai/nan/README.md` reports a second dead path, `specs/SDD-007-ai-tooling-consolidation/proposal.md` (that spec ID was reused for a different feature — `specs/archive/SDD-007-iac-deploy-strategy` — so the reference is simply wrong, not just moved). Neither file is in the governed set, so neither failure is caught anywhere: not by this guard, not by any other bats case, not by CI. `docs/` and `specs/`-prefixed README.md files would already be excluded by the existing `grep -vE` line if the anchor were widened, so widening it does not sweep in the two files the exclusion list already reasons about — only genuinely ungoverned ones (`cli/`, `ai/nan/`, `ai/opencode/`, `ai/pi/`, `sensitive/`). | Reproduced directly against the real repo tree (not the clone): `./scripts/check-doc-paths.sh cli/README.md` → `2 missing path(s)`; `./scripts/check-doc-paths.sh ai/nan/README.md` → `1 missing path(s)`; `ls scripts/doctor.sh scripts/healthcheck.sh` → both `No such file or directory`. Confirmed no other test or CI step covers these files (`grep -rln "cli/README\|ai/nan/README" tests/*.bats scripts/*.sh` → no matches). | UNTESTED — no case in `tests/check-doc-paths.bats` exercises README.md discovery outside repo root, and no case pins the discovered set against a regression that narrows it (see the next finding, which is the general form of this one) | code (`tests/check-doc-paths.bats:43`, widen the README alternative to `(^\|/)README\.md$`, matching every sibling pattern) + tests (a case pinning `cli/README.md` or an equivalent nested README as discovered) + docs (fix the two stale references this reveals: de-backtick or update `cli/README.md`'s `scripts/healthcheck.sh`/`scripts/doctor.sh` mentions per this guard's own "retired path → plain text" convention, and correct or drop `ai/nan/README.md`'s dead spec pointer) |
| Minor | REAL | `tests/check-doc-paths.bats` — discovery has no floor | Mutation-tested: removing `AGY\.md\|` from the discovery alternation in a disposable clone (simulating exactly the silent-drop defect class rounds 2, 3 and 4 each found once — a governed file quietly falling out of the set) leaves all 14 cases green. Nothing in the suite asserts the discovered set contains at least the 9 files every round has settled on, so a future edit narrowing the pattern (a typo in the alternation, a misplaced `\|`) would pass CI silently — the exact failure mode `verification.md` itself names ("a guard that under-reports looks identical to a clean repo"), now demonstrated against the discovery mechanism instead of the path-checking mechanism it was written to fix. | Reproduced: `sed` removed `AGY\.md\|` from `tests/check-doc-paths.bats:43` in the disposable clone, ran `bats tests/check-doc-paths.bats` → `1..14`, all `ok`. Restored, `git status --short` clean afterward. | UNTESTED | tests — add a case asserting `instruction_files() \| wc -l` is at least the known floor (or asserts specific known members are present), so a regression in the alternation itself is caught, not only a regression in what the alternation is applied to |
| Minor | THEORETICAL | `scripts/check-doc-paths.sh` header convention vs. `ai/opencode/README.md` | The "backticked = live claim, retired = plain text" convention has no third category for a **planned, not-yet-built** path. `ai/opencode/README.md:35` reads "Future Ollama provider docs will live at `ai/ollama/` when wired" — backticked, and `ai/ollama/` does not exist, so if README.md discovery is widened (per the Major above) this line would newly fail even though it is neither a live claim nor a stale one. Not a bug in the shipped guard today (the file isn't governed), but it is a real gap the fix for the Major above will walk directly into. | Manual read; `ls ai/ollama` confirms absent | UNTESTED | docs — de-backtick this one line when README discovery widens, or spec — extend the convention with an explicit "planned path" exception before more of these accumulate |
| Minor | informational | `scripts/check-doc-paths.sh` — CI wiring | Unchanged since round 4: no standalone CI step; enforcement is only the "every instruction file's repo paths resolve" bats case, which does run in the required `test` job. Pre-existing since round 1, not scored against this round. | n/a | n/a |

No Blockers found in the mechanism rounds 1-4 already fixed: traversal
ordering, the flush-left zsh-sourcing regex, the probe-hermeticity fix, and
the false-positive/false-negative pair from round 1 all re-verified true by
direct execution and, where a prior round's fix was itself the subject of a
mutation, by mutation.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | C | The governed set is internally correct and now hermetic under fault, but the discovery contract has an undocumented boundary that provably lets real stale claims (including the founding-incident path) through today |
| Verification | B | Every testable claim in `verification.md` reproduced by direct execution across three independent runs (main, clone, fault-injected clone); the round-5 "no leaks under index.lock" claim specifically confirmed, not just re-read |
| Scope | B | The round-4→5 discovery rewrite is a deliberate, justified widening of the original 6-file list, consistent with this chain's own precedent; this round's finding is about the widening being incomplete, not about scope creep |
| Reliability | C | Mutation-tested: the discovery mechanism has no regression floor, so a future narrowing of the governed set (accidental or not) fails silently, which is the specific failure mode this spec exists to prevent |
| Maintainability | B | Clear naming, extensive rationale comments in both files, `shellcheck` and both shell parsers clean, no dead code |
| Handoff-readiness | B | `verification.md`'s round-by-round record is thorough, accurate, and the round-5 fix claims all reproduced true; this round's findings were not yet captured anywhere |

### Verdict
FAIL

One REAL Major (UNTESTED): `README.md` discovery is silently root-anchored
while every sibling pattern in the same regex is not, and that asymmetry is
not a documented exclusion but an oversight — proven by two currently-tracked,
ungoverned files (`cli/README.md`, `ai/nan/README.md`) that carry real dead
backticked paths right now, one of them a second occurrence of the exact path
(`scripts/healthcheck.sh`) that motivated this spec in the first place. A
guard whose own governed-file boundary reproduces the bug class it exists to
catch, in a file sitting one directory below the ones it does check, cannot
be called done.

### Recommended next steps (before archive)

Two independent ways to flip this to PASS — either is small:

- **(a) Widen the boundary.** Change `tests/check-doc-paths.bats:43`'s
  `^README\.md$` to `(^|/)README\.md$`, matching every other alternative in
  the same regex. `docs/` and `specs/` README.md files are already excluded
  by the existing `grep -vE '^harness/|^specs/|^docs/'` line, so this sweeps
  in exactly `cli/README.md`, `ai/nan/README.md`, `ai/opencode/README.md`,
  `ai/pi/README.md`, `sensitive/README.md` — nothing already reasoned about.
  Then fix the two now-caught stale references (de-backtick or correct, per
  this guard's own convention) and add a case pinning at least one nested
  README as discovered.
- **(b) Document it as a deliberate exclusion.** If root-only README.md is
  actually intended (e.g. because subdirectory READMEs are judged closer to
  `docs/` than to `AGENTS.md`), say so in the exclusion-reasons comment block
  the same way `harness/`, `specs/` and `docs/` already are, and add a case
  pinning that a nested README.md is correctly NOT discovered — matching how
  case 9 already pins the `harness/` exclusion. This path does **not** fix
  `cli/README.md`'s and `ai/nan/README.md`'s stale content, which would then
  need a separate, non-DOCS-013 ticket, disclosed rather than silently
  dropped (`feedback_fix_or_ticket_tech_debt`).
- Either path: add a regression test for the discovery floor itself (the
  Minor above) — e.g. assert `instruction_files()` returns at least the known
  9 files, or contains a fixed list of must-have members — so a future
  narrowing of the alternation is caught the same way a future narrowing of
  the path-checking logic already is.
- The `ai/opencode/README.md` "planned path" Minor is not blocking; note it
  in the same PR if convenient, otherwise it is fine to leave for whoever
  picks up (a).
