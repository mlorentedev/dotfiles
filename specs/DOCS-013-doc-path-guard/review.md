---
spec: "DOCS-013-doc-path-guard"
verdict: "FAIL"
reviewed_sha: "b5209f6d729622a6ceb37a8f9d51af6db6bc289f"
reviewer: "claude-sonnet-5"
date: "2026-08-11"
---

## Adversarial review

**Scope**: DOCS-013-doc-path-guard, round 6. One commit
(`b5209f6`, `fix(docs): govern nested READMEs in the doc-path guard (DOCS-013)`)
on top of `main` at `3c6d77d` (release 0.37.1), fixing round 5's REAL Major
(README.md discovery silently root-anchored while every sibling pattern in the
same regex matched anywhere) and its Minor (no regression floor on the
discovery alternation itself). Diffed against `main` in this worktree
(`git diff main...HEAD`); no open PR to cross-reference.

Per the task brief, `review.md` was intentionally not read as part of forming
this review's frame — only `proposal.md`, `tasks.md` and `verification.md`
(whose round-by-round log carries the same history `review.md` would). Every
verification.md claim relevant to this round was independently reproduced,
including this round's own entries, not taken on the document's word.

**Sources**: `specs/DOCS-013-doc-path-guard/{proposal,tasks,verification}.md`,
`scripts/check-doc-paths.sh`, `tests/check-doc-paths.bats`,
`scripts/check-bats-names.sh`, `.github/workflows/ci.yml`,
`cli/internal/spec/review.go` (frontmatter/staleness contract), plus direct
execution in the actual worktree (not a disposable clone — the working tree
was restored to a clean state, verified by diff, after every mutation):
`bats tests/check-doc-paths.bats`, `bats tests/*.bats` (full suite, 1214
tests), `./scripts/check-bats-names.sh tests/`, `shellcheck`, `bash -n`/`zsh -n`,
`./scripts/check-md-escapes.sh`, `./scripts/check-spec-gate.sh`, and three
guard-condition mutations (README anchor reverted to root-only; `AGY.md`
dropped from the discovery alternation; the new @test name's em-dash replaced
with a hyphen), each applied, tested, and reverted individually.

### Spec and task alignment

- Round 5's Major is genuinely fixed. `tests/check-doc-paths.bats:50` now folds
  `README\.md` into the same `(^|/)(...)$` alternation as every other pattern,
  replacing the round-5-flagged `^README\.md$` anchor. Reproduced by mutation:
  reverting to the root-only anchor turns case 9 ("a nested README.md is
  governed...") red while case 10 ("discovery floor...") stays green; restoring
  turns case 9 green again. Matches verification.md's Round 6 claim exactly.
- Round 5's Minor (no discovery-floor regression test) is genuinely fixed.
  Case 10 asserts five known members by name. Reproduced by mutation: dropping
  `AGY\.md` from the alternation turns case 10 red while case 9 stays green;
  restoring turns it green again. Matches verification.md's claim exactly —
  both mutations behave precisely as documented, in the direction documented,
  with no unrelated case affected.
- The three content fixes are factually correct, not just guard-satisfying:
  `specs/archive/SDD-007-iac-deploy-strategy/proposal.md` exists and its
  frontmatter `id:` really is `SDD-007-ai-tooling-consolidation`, confirming
  `ai/nan/README.md`'s corrected pointer is accurate, not merely resolvable.
  `scripts/healthcheck.sh` and `scripts/doctor.sh` (de-backticked in
  `cli/README.md`) are confirmed absent. `ai/ollama/` (de-backticked in
  `ai/opencode/README.md`) is confirmed absent, consistent with "not yet
  created."
- Ran the guard directly against all 6 non-excluded `README.md` files in the
  repo (`README.md`, `ai/nan/`, `ai/opencode/`, `ai/pi/`, `cli/`,
  `sensitive/`) — all report `OK`, matching verification.md's claim. `ai/pi/`
  and `sensitive/` were not touched by this diff but are newly swept into the
  governed set by the widened regex; both were already clean, so the widening
  introduced no new false failures on files nobody looked at.
- No filename pattern in the discovery alternation carries an anchor asymmetric
  with the others: `AGENTS\.md`, `CLAUDE\.md`, `AGY\.md`, `GEMINI\.md`,
  `copilot-instructions\.md` and `README\.md` all now sit inside the same
  `(^|/)(...)$` group. The exclusion filter (`^harness/|^specs/|^docs/`) is
  separately anchored to string-start for all three prefixes, symmetric with
  itself; no file in the repo today sits under a *nested* `docs/`, `specs/` or
  `harness/` (e.g. `ai/x/docs/README.md`) that this would fail to exclude, so
  it is a live-instance-free asymmetry, structurally identical to the one
  round 5 found live — worth naming, not worth blocking on (see Findings).
- `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags: none present in any spec file.
- `check-spec-gate.sh --base-ref main --head-ref HEAD --explain`: production
  diff 250 LOC ≥ threshold 50, spec folder touched — gate satisfied.
- `proposal.md`/`tasks.md` unchanged since `6a40da3`, predating this round's
  commit — `review.md`'s staleness check (`cli/internal/spec/review.go`) will
  not flag this review against the contract files.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| Blocker | REAL | `tests/check-doc-paths.bats:168` — new `@test` name | The new "discovery floor" test name contains a literal U+2014 EM DASH (`—`) instead of a hyphen: `@test "check-doc-paths: discovery floor — each governed pattern has a known real member [#916]" {`. `scripts/check-bats-names.sh` — the repo's own guard against exactly this class ("bats @test name the runner silently fails to register," CURATOR-001/#615) — flags it, and that script is a **required** step in `.github/workflows/ci.yml`'s `lint` job ("Lint bats @test names (no silent-skipped tests)": `./scripts/check-bats-names.sh tests/`). Running that exact command against this worktree fails deterministically. Confirmed the character is new to this round, not pre-existing: `git show main:tests/check-doc-paths.bats` piped through the same script reports clean; only this round's diff (`git diff main...HEAD` shows the line as a `+` addition) introduces it. This is not a hypothetical CI outcome — it is the literal command CI runs, run here, failing. | `./scripts/check-bats-names.sh tests/` → `tests/check-doc-paths.bats:168: non-ASCII character in @test name (bats silently fails to register it)`, exit 1. `xxd` on the character confirms `e2 80 94` (UTF-8 U+2014). `bats tests/*.bats` (full suite, 1214 tests) surfaces this as `not ok 153 check-bats-names: the repo's own tests/ pass (no silent-skip names remain)`. Mutation-verified the fix: replacing the em-dash with `-` makes both `check-bats-names.sh tests/` and the bats run pass; reverted after confirming. | `check-bats-names: the repo's own tests/ pass (no silent-skip names remain)` (`tests/check-bats-names.bats:54`) — NOT UNTESTED; this existing regression test already catches it, it was simply not run by this round's own verification pass | tests — one-character fix in `tests/check-doc-paths.bats:168`, em-dash to a plain hyphen (or ` - ` / `:`), matching every other `@test` name's ASCII-only convention in the same file |
| Minor | THEORETICAL | `scripts/check-doc-paths.sh` header convention — "planned path" gap | Round 5 flagged that the "backticked = live claim, retired = plain text" convention has no third category for a planned-but-not-yet-built path, and offered two fixes: de-backtick the one live instance, or extend the documented convention. This round took the first (de-backticked `ai/opencode/README.md`'s `ai/ollama/` mention) but not the second. Swept every governed file for the same shape (`will live at`, `not yet (created\|wired\|built)`, `when wired`) and found no second live instance today, so this is not a currently-failing case — but the general gap in the script's own header comment (which enumerates only "live" and "retired," not "planned") is unaddressed, so the next agent who backticks a forward-looking path anywhere in the repo will hit the identical landmine round 5 hit, with no comment warning them off it. | `grep -nE "will live at\|not yet \(created\|wired\|built\)\|when wired" <every governed file>` → one match, already fixed. No second instance. | UNTESTED — no regression test exists for this class, by design (there is nothing to regress against yet) | docs — extend `scripts/check-doc-paths.sh`'s header convention section with an explicit "planned, not yet built" exception (write it in plain text like a retired path), or accept explicitly as a known, documented limit the way round 4 did for markdown-link extraction |
| Minor | informational | `specs/DOCS-013-doc-path-guard/verification.md` — Round 6 evidence scope | Round 6's own evidence list (`bats tests/check-doc-paths.bats` 16/16, guard-clean sweep, mutation tests, `shellcheck`, `bash -n`/`zsh -n`, `check-md-escapes.sh`) omits both a full `bats tests/*.bats` sweep and `./scripts/check-bats-names.sh tests/` — the exact required CI step that would have caught the Blocker above in seconds. Round 1's own verification explicitly ran `./scripts/check-bats-names.sh tests/ → OK, 82 files`; this round narrowed the evidence set relative to that established baseline, and the narrowing is precisely how the Blocker shipped undetected. Not a new mechanism defect — a verification-process gap that explains the Blocker rather than compounding it. | Direct comparison: `verification.md`'s Round 1 evidence lists `check-bats-names.sh`; Round 6's does not. Full suite run in this review (1214 tests, 2 failures: the new Blocker and the pre-existing, disclosed #807/BUG-054) confirms what a full sweep would have shown. | n/a (process finding, not a code path) | spec — when this round is amended, restore a full-suite + `check-bats-names.sh` line to the Round 6 evidence list; a standing practice for future rounds worth stating once rather than re-discovering: re-run the full suite and the naming lint every round, not only the file scoped to the round's own fix |

No other Blockers or Majors. Rounds 1-5's mechanisms — traversal ordering, the
flush-left zsh-sourcing regex, probe hermeticity under `index.lock`, the
false-positive/false-negative pair, and now the README discovery symmetry —
all re-verified true by direct execution and, where a prior round's fix was
itself the subject of a mutation, by mutation. The regression sweep found
exactly one other failure, `not ok 410 converges over a running dotf: a live
binary in dest is replaced, not refused` (`coreutils: unknown program 'dotf'`)
— matches the pre-existing, disclosed #807/BUG-054 signature exactly (same
underlying cause, shifted test number from added tests); not attributable to
this spec.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | D | A real, deterministically reproducible defect ships in the diff: a non-ASCII character in a new `@test` name fails the repo's own required CI lint step outright |
| Verification | C | What was claimed reproduced faithfully (both mutations behave exactly as documented) but the evidence set was narrower than this round's own baseline (round 1 ran `check-bats-names.sh`; round 6 did not), and that narrowing is exactly how the Blocker went unnoticed |
| Scope | A | Diff addresses precisely round 5's Major and Minor, nothing extraneous; the three README content fixes are each independently verified factually correct, not just guard-satisfying |
| Reliability | B | The mechanisms this round touches (discovery regex, two new bats cases) behave correctly and hermetically under the mutations tested; the Blocker is a lint-time defect, not a runtime reliability gap in the guard itself |
| Maintainability | B | Clear naming and extensive, accurate rationale comments elsewhere in the diff; the one em-dash is a typo-class slip, not a structural issue, and is already scored under Correctness |
| Handoff-readiness | A | `verification.md`'s round-by-round record remains exemplary — thorough, accurate about what it claims, and every claim it made for this round reproduced true; only the evidence *scope* narrowed, not its honesty |

### Verdict
FAIL

One REAL Blocker: `tests/check-doc-paths.bats:168`'s new `@test` name contains
a non-ASCII em-dash, which fails `scripts/check-bats-names.sh` — a required
step in the `lint` CI job — deterministically and unconditionally. This is not
a design gap or an edge case; it is the literal command CI runs, run here,
failing on a one-character typo in code this round itself added. Everything
else in the round — the README-discovery symmetry fix, the discovery-floor
regression test, the three content corrections — is genuinely correct,
independently verified, and matches what `verification.md` claims. This FAIL
is narrow and mechanical, not a signal that the round's actual design work is
unsound.

### Recommended next steps (before archive)

- **The fix is one character.** In `tests/check-doc-paths.bats:168`, replace
  the em-dash (`—`, U+2014) between "discovery floor" and "each governed
  pattern" with a plain hyphen (or any ASCII punctuation) so the `@test` name
  is pure ASCII, matching every other test name in the file.
- Re-run `./scripts/check-bats-names.sh tests/` (must report `OK (82 file(s)
  clean)`) and `bats tests/*.bats` (must return to 1 failure, the disclosed
  #807/BUG-054, not 2) before the next review.
- Optional but recommended, not blocking: fold a full-suite run and
  `check-bats-names.sh` back into this round's evidence list in
  `verification.md`, so the record shows what actually gates CI, matching
  round 1's own baseline.
- Optional, not blocking, no live instance: extend
  `scripts/check-doc-paths.sh`'s header with an explicit "planned, not yet
  built" exception, or note the gap as a deliberately accepted limit the way
  round 4 did for markdown-link extraction.
