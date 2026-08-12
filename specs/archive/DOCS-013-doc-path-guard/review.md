---
spec: "DOCS-013-doc-path-guard"
verdict: "PASS"
reviewed_sha: "21f5d2c579d4fb367bb013f27ff130df30d8d6b5"
reviewer: "claude-sonnet-5"
date: "2026-08-11"
---

## Adversarial review

**Scope**: DOCS-013-doc-path-guard, round 7. Two commits on top of `main` at
`6267bea`: `b5209f6` ("govern nested READMEs in the doc-path guard", round 6's
fix, already reviewed and FAILed on a Blocker) and `21f5d2c` ("em dash in a
bats test name, plus the planned-path doctrine", round 7's fix, responding to
that Blocker). Diffed against `main` in this worktree (`git diff main...HEAD`);
no open PR to cross-reference. Reviewed the full two-commit diff, not only
`21f5d2c` in isolation, since round 6's content was FAILed and this is its
first PASS-eligible re-examination.

Per the task brief, no chat history, handoff notes, or GitHub PR/issue
comments about this spec were read beyond the repo's own
`specs/DOCS-013-doc-path-guard/` folder. `proposal.md`, `tasks.md` and
`verification.md`'s full round-1-through-7 log were read for acceptance
criteria and prior-round history; `review.md` (round 6's, pre-existing in this
worktree) was read only *after* independently reproducing this round's own
claims, to avoid inheriting its frame before forming one.

**Sources**: `specs/DOCS-013-doc-path-guard/{proposal,tasks,verification}.md`,
`scripts/check-doc-paths.sh`, `tests/check-doc-paths.bats`,
`scripts/check-bats-names.sh`, `docs/lessons.md` (2026-06-25 entry, the #607
precedent `check-bats-names.sh` exists to catch), `cli/internal/spec/review.go`
(archive staleness contract), `.github/workflows/ci.yml`,
`.pre-commit-config.yaml` + `git-hooks/`, `ai/nan/README.md`,
`ai/opencode/README.md`, `cli/README.md`, plus direct execution in this
worktree (real checkout, not a disposable clone — working tree confirmed clean
via `git status --short` / `git diff --stat` before writing this file, after
every mutation was applied and reverted):

- `./scripts/check-bats-names.sh tests/` → `check-bats-names: OK (82 file(s) clean)`
- `bats tests/*.bats` (full suite) → **1213 passed, 1 failed, 1214 total**
  (`1..1214`); the one failure is `not ok 410 converges over a running dotf: a
  live binary in dest is replaced, not refused`
  (`coreutils: unknown program 'dotf'`) — matches the disclosed, pre-existing
  `#807`/`BUG-054` signature exactly. Ran twice (once mid-investigation, once
  as a final clean-tree close-out); identical result both times.
- `~/.local/bin/shellcheck scripts/check-doc-paths.sh` → clean (exit 0)
- `bash -n scripts/check-doc-paths.sh` → clean (exit 0)
- `zsh -n scripts/check-doc-paths.sh` → clean (exit 0)
- `./scripts/check-spec-gate.sh --base-ref main --head-ref HEAD --explain` →
  production LOC 345, active-spec LOC 332 (floor 10), spec folder touched:
  yes → `[OK]`
- `./scripts/check-md-escapes.sh` on every file this diff touches → clean
- `grep -rn "AGENT-DRAFT\|AGENT-SUGGESTION" specs/DOCS-013-doc-path-guard/` →
  the only hit is `review.md`'s own prose stating none are present; no live tag
- `gh issue view 925 --json number,title,state,body` → confirmed OPEN, title
  and body match `verification.md`'s citation exactly (local pre-commit hook
  does not run `check-bats-names.sh`; scope note explains why it is not folded
  into this PR)
- Mutation 1 (bats case, reproduced myself, not read from `verification.md`):
  reverted `tests/check-doc-paths.bats:168`'s hyphen back to the literal U+2014
  em dash. `./scripts/check-bats-names.sh tests/` → flags
  `tests/check-doc-paths.bats:168: non-ASCII character in @test name`, exit 1.
  Running the full suite with the mutation in place: `not ok 153 check-bats-names:
  the repo's own tests/ pass (no silent-skip names remain)` fires, exactly as
  round 6's review predicted — a loud, visible CI failure via the wired-in
  meta-test (`tests/check-bats-names.bats:56`), not a silent one. Restored the
  hyphen; `git status --short` and `git diff --stat` both empty afterward,
  `check-bats-names.sh` back to OK.
- Mutation 2 / discrepancy investigated and resolved (see Findings): under the
  em-dash mutation, `bats tests/check-doc-paths.bats` run alone (not the full
  suite) still registered and ran the mutated test — `1..16`, 16 executed,
  `ok 10 ... discovery floor — each governed pattern ...`. Bats 1.13.0 (the
  pinned version, confirmed via `versions.conf` `BATS_VERSION=1.13.0` matching
  the installed `bats --version`) did not silently drop this specific em-dash
  test in this reproduction. This qualifies the historical mechanism described
  in `check-bats-names.sh`'s header and `docs/lessons.md`'s 2026-06-25 entry,
  but does not change round 6's Blocker, which was that the required CI lint
  step (`check-bats-names.sh` directly, and the suite's own meta-test) fails
  deterministically — both reproduced true. See Findings.
- Content-correctness spot checks (independent, not re-reading round 6's
  claims): `specs/archive/SDD-007-iac-deploy-strategy/proposal.md` exists,
  `id:` frontmatter is `SDD-007-ai-tooling-consolidation` — matches
  `ai/nan/README.md:159`. `scripts/healthcheck.sh` / `scripts/doctor.sh` absent
  from disk, referenced only in plain text (no backticks) in `cli/README.md`.
  `ai/ollama/` absent from disk, referenced only in plain text in
  `ai/opencode/README.md:35`.
- Guard run directly against all 14 currently-discovered instruction files
  (`git ls-files '*.md' | grep -E '(^|/)(AGENTS\.md|CLAUDE\.md|AGY\.md|GEMINI\.md|copilot-instructions\.md|README\.md)$' | grep -vE '^harness/|^specs/|^docs/'`)
  → `OK` on every one.
- Swept every governed file for un-migrated "planned path" phrasing
  (`will live at`, `not yet (created|wired|built)`, `when wired`,
  `TBD`, `to be created`, `coming soon`) → the only match is the already
  plain-text `ai/opencode/README.md:35` line; no new live instance.
- Swept every governed file's backticked tokens for the ALL-CAPS +
  unrecognized-extension placeholder-skip gap noted in round 1 → no token in
  the current governed set falls into that shape today (0 matches).

### Spec and task alignment

- Round 6's Blocker (`tests/check-doc-paths.bats:168`'s em dash) is genuinely
  fixed: the byte is a plain hyphen, `check-bats-names.sh` and the full suite
  both pass, and reverting it reproduces the exact failure round 6 described —
  including the specific `not ok 153` meta-test line round 6's review named.
- Round 6's Minor #1 (the "planned path" doctrine gap in
  `scripts/check-doc-paths.sh`'s header) is genuinely fixed: a new comment
  block generalizes the one-file point-fix into a stated rule ("a path that
  does not exist YET ... is also not a live claim ... goes in plain text, not
  backticks"), closing the gap round 5's review first raised and round 6
  applied only the instance-fix for.
- Round 6's Minor #2 (verification.md's evidence scope narrower than round 1's
  baseline) is genuinely fixed: Round 7's `verification.md` entry lists
  `check-bats-names.sh`, the full `bats tests/*.bats` sweep, and
  `shellcheck`/`bash -n`/`zsh -n` explicitly — the same baseline round 1
  established, restored.
- All content fixes from round 6 (README discovery folding, the three
  README.md corrections, the two new bats cases) remain correct — re-verified
  this round both by re-running the suite and by independently checking the
  underlying facts (spec ID, absent files) rather than re-reading round 6's
  own review.
- No mechanism from rounds 1-6 regressed: traversal-check ordering, the
  flush-left zsh-sourcing regex, probe hermeticity under `teardown()`, the
  false-positive/false-negative pair, and README discovery symmetry all
  re-verified true by direct execution of the full suite at HEAD.
- `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags: none present in any spec file.
- `check-spec-gate.sh`: production diff 345 LOC ≥ threshold 50, spec folder
  touched — gate satisfied.
- `proposal.md`/`tasks.md` unchanged since before this round (last touched at
  `6a40da3`, predating both `b5209f6` and `21f5d2c`) — the archive staleness
  gate in `cli/internal/spec/review.go` compares `reviewed_sha` only against
  `proposal.md`/`tasks.md`/`features.json` (deliberately excluding
  `review.md`/`verification.md` per that file's own comment), so this review
  will not be flagged stale against the contract files it gates.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | `scripts/check-doc-paths.sh` header / `docs/lessons.md` 2026-06-25 — historical-mechanism precision | `check-bats-names.sh`'s header and the 2026-06-25 lesson describe non-ASCII `@test` names as causing bats to "silently fail to register" the test. Reproduced directly under the pinned bats 1.13.0: the mutated em-dash test in `tests/check-doc-paths.bats` **did** register and run (`1..16`, 16 executed, `ok 10 ... discovery floor — ...`) both standalone and inside the full 1214-test suite. What actually fired, both times, was the separate, loud meta-test `tests/check-bats-names.bats:56` ("the repo's own tests/ pass") going `not ok` — a visible CI failure, not a silent one, for this specific character in this specific bats version. Does not weaken round 6's Blocker (the required `check-bats-names.sh` / meta-test failure is real and reproduced identically) — it only qualifies the *why* text one layer up, for a repo whose defense-in-depth (the meta-test) has since made the originally-silent failure mode loud again. | `./scripts/check-bats-names.sh tests/` flags it (exit 1); `bats tests/check-doc-paths.bats` alone still shows `ok 10` for the mutated name; `bats tests/*.bats` shows `not ok 153 check-bats-names: the repo's own tests/ pass (no silent-skip names remain)`. `versions.conf` `BATS_VERSION=1.13.0` matches installed `bats --version` exactly (no version-mismatch confound). | `check-bats-names: flags a non-ASCII @test name with file:line` and `check-bats-names: the repo's own tests/ pass (no silent-skip names remain)` (`tests/check-bats-names.bats`) — NOT UNTESTED, both already cover the mechanism that actually fires | docs — optional, not blocking: the header comment and the 2026-06-25 lesson could note that the current repo's own meta-test converts what was historically a silent skip (#607/HARNESS-043) into a loud CI failure, so a future reader does not need to independently discover this the way this review did |
| Question / assumption | informational | `cli/internal/spec/review.go` — archive staleness scope (CLI-034, not this spec) | `checkReviewGate`'s staleness check (`gitStaleness.Stale`) compares `reviewed_sha` only against changes to `proposal.md`, `tasks.md`, `features.json` — never against `scripts/check-doc-paths.sh`, `tests/check-doc-paths.bats`, or any other code this spec's diff touches. This is documented as deliberate in the code's own comment and is consistent across all 7 rounds of this chain (each round's code changes did not by themselves trigger staleness; only a hypothetical future edit to `proposal.md`/`tasks.md` would). Out of scope for DOCS-013 — `review.go` is CLI-034's artifact, not touched by this diff — surfaced only because it bears on whether "PASS + archive" is durable against unreviewed code drift after this sha. Not a DOCS-013 defect. | Read `cli/internal/spec/review.go:15-23` and `:134-157`; `contractFiles = []string{"proposal.md", "tasks.md", "features.json"}` | n/a (design question about a different spec's gate, not a code path in this diff) | vault/spec — a question for CLI-034, not an action item here; noting it so a future session does not need to re-derive it |

No Blockers or Majors. Both rows above are explicitly non-verdict-moving: the
first documents evidence that qualifies a rationale comment without weakening
the actual defect round 6 found and round 7 fixed; the second is a
cross-spec observation, not a finding against this diff.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | Round 6's Blocker is fixed and the fix reproduces exactly as claimed in both directions (mutate/restore); no new defect found after full baseline + mutation + content-correctness verification |
| Verification       | A | All four mandated baselines run and reproduced in this session, not read from `verification.md`; round 7's own evidence list matches round 1's original baseline scope, closing round 6's Minor #2 |
| Scope              | A | Diff is a one-character test-name fix plus a four-line doctrine comment plus spec-doc updates; nothing extraneous |
| Reliability        | A | No runtime behavior changed (both fixes are text-only, confirmed by diff and by identical guard/test behavior pre- and post-fix on every other case) |
| Maintainability    | A | Clear naming, ASCII-only convention now uniformly applied, doctrine comment closes a documented gap with a stated general rule rather than another point-fix |
| Handoff-readiness  | A | `verification.md`'s round-by-round record remains accurate and complete; the local/CI pre-commit gap is ticketed (`#925`, confirmed open, confirmed accurate) rather than silently absorbed or left undocumented |

### Verdict
PASS

Round 6's REAL Blocker (a U+2014 em dash in a new `@test` name, failing the
repo's required `check-bats-names.sh` CI lint step) is fixed with a one-byte
change, reproduced in both directions this session. Round 6's two Minors —
the undocumented "planned path" doctrine gap and the narrowed verification
evidence scope — are both closed. Every mechanism from rounds 1-6 was
re-verified against the full 1214-test suite at HEAD with no regression, and
the one failure in that suite is the disclosed, pre-existing `#807`/`BUG-054`
case, unrelated to this spec. Independent content-correctness spot checks
(the archived spec ID, the absent `scripts/healthcheck.sh`/`doctor.sh`, the
absent `ai/ollama/`) all confirm round 6's README fixes are factually
accurate, not merely guard-satisfying. No Blocker or Major found this round;
the two findings above are explicitly non-verdict-moving (an evidence
qualification and a cross-spec observation). All six rubric dimensions grade
A.

### Recommended next steps (before archive)

- None required to archive. `dotf spec archive DOCS-013-doc-path-guard` is
  advisable: `proposal.md`/`tasks.md` are unchanged since before this round,
  so the staleness gate will not fire against this review's `reviewed_sha`.
- Optional, not blocking: when convenient, note in `check-bats-names.sh`'s
  header or the 2026-06-25 `docs/lessons.md` entry that the repo's own
  meta-test now converts the historically-silent em-dash failure into a loud
  one (see Findings row 1) — purely a rationale-comment clarity improvement,
  no behavior change.
- Not this spec's action item: the `review.go` staleness-scope observation
  (Findings row 2) belongs to CLI-034 if it is ever acted on; DOCS-013's own
  contract files are correctly tracked by the existing gate.
