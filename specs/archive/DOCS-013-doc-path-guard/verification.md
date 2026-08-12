---
tags: [spec, verification, templates]
created: "2026-08-11"
---

# Verification - DOCS-013-doc-path-guard

## Evidence

| AC | Claim | Proof |
|---|---|---|
| 1 | Every repo path in `.claude/CLAUDE.md` resolves | `./scripts/check-doc-paths.sh .claude/CLAUDE.md` → `OK`. Before the change the same command reported **8** dead paths |
| 2 | Secrets docs describe ADR-028 | `grep -n env-mapping .claude/CLAUDE.md` → only plain-text mentions naming it as retired; the two "adding a secret" recipes now use `dotf secrets set/verify/run` |
| 3 | Verification Commands covers the Go layer | New *Go layer* block: `go build/vet/test` + `golangci-lint run`, with the `GOLANGCI_LINT_VERSION` pin and the reason an unpinned local run proves nothing (BUG-071) |
| 4 | No hardcoded vault literal | `grep -n 'Projects/knowledge' .claude/CLAUDE.md` → no matches; three occurrences replaced with `$VAULT_PATH` |
| 5 | Guard catches, on real and seeded inputs | `tests/check-doc-paths.bats` cases 3 (dead path), 4 (empty glob), 7 (ALL-CAPS with extension); README's dead `load-secrets.sh` was found by the guard, not by reading |
| 6 | Zero false positives on `AGENTS.md` | `./scripts/check-doc-paths.sh AGENTS.md` → `OK`. An earlier revision reported **13** on the same file; case 6 pins nine of the token shapes that caused them |
| 7 | Suite pins both behaviours | `bats tests/check-doc-paths.bats` → **8/8** |

## Test status

- `bats tests/check-doc-paths.bats` → **8 passed, 0 failed** (re-run against the
  final tree after the last edit, not an earlier state).
- `bats tests/*.bats` → **1205 passed, 1 failed** of 1206.
  The one failure is `not ok 402 converges over a running dotf: a live binary in
  dest is replaced, not refused` — **#807 (BUG-054)**, pre-existing and
  unrelated. Reproduced this session on clean `main` @ `a3f9a10` with no
  working-tree changes, same `coreutils: unknown program 'dotf'` cause.
- `shellcheck scripts/check-doc-paths.sh` → clean.
- `bash -n` and `zsh -n` on the new script → clean; the script was also
  **executed** under `zsh -c`, not merely parsed, because the failure class this
  repo keeps hitting is a construct that runs and answers wrongly rather than
  erroring (`docs/lessons.md`, 2026-08-09).
- `./scripts/check-bats-names.sh tests/` → OK, 82 files.
- `./scripts/check-spec-gate.sh --explain` → `Spec folder touched: yes`,
  active-spec LOC 199 (floor 10).
- Guard clean on all six instruction files: `.claude/CLAUDE.md`, `AGENTS.md`,
  `README.md`, `ai/claude/CLAUDE.md`, `ai/copilot/copilot-instructions.md`,
  `.github/copilot-instructions.md`.

## What was found while verifying

Two defects in the guard itself, both caught by running it rather than
reasoning about it, and both now pinned as tests:

1. **False positives (26, then 13).** The first revision flagged `&>/dev/null`,
   `ai/<agent>/`, the model id `opencode-go/qwen3.6-plus` and more. The control
   run on `AGENTS.md` — a file with no staleness — was what exposed the scale.
2. **A false negative.** The ALL-CAPS placeholder rule intended for
   `sensitive/KEYNAME.secret.age` also swallowed `SKILL.md`, so the guard
   silently skipped `ai/skills/*/SKILL.md`, a genuinely dead glob it existed to
   catch. Fixed by applying the placeholder reading only when the token lacks a
   known extension.

The second is the one worth remembering: a guard that under-reports looks
identical to a clean repo.

## Not verified

- **Deployment.** This change is repo-side only; `.claude/CLAUDE.md` is read
  from the checkout, so no deploy step applies. `README.md` is not deployed.
- **Windows.** The guard is bash and runs in the Linux `test` job. It is not
  wired into any PowerShell path, and nothing on Windows consumes it.

## Sign-off

- [ ] Independent `/adversarial-review` — **owed, not performed.** This session
      implemented the change and therefore cannot review it. `dotf spec archive`
      refuses without a fresh passing `review.md`, so this must be supplied by
      another session before the spec can be archived.

---

## Round 2 — after the adversarial review

The round-1 review (`review.md`, verdict **FAIL**, reviewed_sha `d6681a6`) found
one Major that neither the author nor CodeRabbit caught, plus six Minors. All
were reproduced independently before being accepted.

### The Major

`.claude/CLAUDE.md:51` documented the golangci-lint install as
`@v$(. versions.conf; …)`. A slashless argument to the `.` builtin is searched
on `$PATH` only; bash additionally falls back to the cwd, zsh does not. Under
zsh — the default interactive shell here — it resolved to **empty**, producing
`@v`:

```console
$ bash -c 'echo "v$(. versions.conf; echo "$GOLANGCI_LINT_VERSION")"'
v2.12.2
$ zsh -c 'echo "v$(. versions.conf; echo "$GOLANGCI_LINT_VERSION")"'
zsh:.:1: no such file or directory: versions.conf
v
```

A wrong instruction, shipped in the PR whose purpose was to stop instructions
being wrong, by the same bash/zsh divergence class the file's own
prohibited-patterns table documents twenty lines above it.

Fixed to `. ./versions.conf`; verified `2.12.2` under zsh. The pattern is now a
row in that table, and two tests pin it: one on `versions.conf` specifically,
one class-level over all three instruction files.

### Minors applied

| Finding | Resolution |
|---|---|
| "backticked path is a live claim" oversells the guard (bare names unchecked) | Callout scoped to slash-containing paths, with the blind spot stated |
| Case 6 pins 9 shapes but 3 pass via `is_repo_rooted`, not the filter | Not papered over — the filter is now labelled defense-in-depth in the script, and the test comment says it pins the outcome, not the mechanism |
| `versionMatches` described as callable | "append an entry to the `versionMatches` table" |
| `#!/bin/bash` vs the repo's documented `#!/usr/bin/env bash` | Changed |
| `setup-macos.sh` — live instance of the bare-name blind spot | De-backticked both README mentions |
| red `spec-gate` not disclosed in Evidence | Disclosed here; archive-on-merge resolves when this review lands |

### CodeRabbit findings (PR #922), all reproduced

- README:42 claimed secrets are "auto-loaded at login", contradicting the
  ADR-028 text this PR added at :95. Also stale in the same block: "21 custom
  skills" (37) and "316 BATS tests" (1206).
- The script header still described basename resolution that was removed.
- **Path traversal**: `scripts/../../dotfiles/README.md` passed `is_repo_rooted`
  and was reported `OK` while resolving outside the repo — a false negative.
  CodeRabbit's stated mechanism was slightly off (it predicted a wrong success
  on a nonexistent path); the real shape is silent acceptance. Rejected now,
  with a regression test.

### Round-2 evidence

- `bats tests/check-doc-paths.bats` → **11/11**
- Mutation: a bare `. versions.conf` added to a live README code block turns
  case 11 red; removing it turns it green again
- `zsh -c '. ./versions.conf; echo $GOLANGCI_LINT_VERSION'` → `2.12.2`
- `shellcheck` clean; `bash -n` + `zsh -n` clean
- Guard clean on all instruction files

### Still owed

A **fresh** adversarial review at the new sha. A Major cannot be waived by
re-reading the reviewed commit, so `dotf spec archive` stays blocked until a
passing `review.md` exists for this head.

---

## Round 3 — after the second adversarial review

Round 2 (`reviewed_sha f91a08d`) verified every round-1 finding and every
CodeRabbit finding as genuinely fixed, then returned **FAIL** on three Majors it
found itself, none inherited. All reproduced before being accepted.

| # | Finding | Reproduction | Fix |
|---|---|---|---|
| 1 | The `..` traversal check ran **before** the rooted gate, so it fired on `not-a-real-toplevel/../other/thing.md` — a token the guard promises to ignore by construction | guard exited 1 on a non-rooted token | Moved after `is_repo_rooted`; new case 10 pins it |
| 2 | The zsh-sourcing test's regex required a delimiter before `. file`, so a source line **flush-left in a fenced block** — the exact shape the original bug took — matched nothing | `grep -E` on `. versions.conf` at column 1 → no match | `(^\|[^./a-zA-Z0-9_-])` alternation; mutation-verified in two files |
| 3 | `cli/AGENTS.md` and `ai/hermes/AGENTS.md` are instruction files by this suite's own criterion and were governed by neither list | both exist, both ungoverned | Added; the two lists now derive from one source |
| Minor | The prohibited-patterns blockquote still said "the last two rows" after this PR appended a third | grep | "last three rows" |

Finding 1 is the one worth remembering: **fixing a false negative introduced a
false positive**, in the same guard, in the same PR. Ordering a new check before
the gate that decides "is this ours to judge at all" is how.

Finding 2 is worse in kind. The round-2 `verification.md` claimed that exact
mutation turned the test red. It did not — the claim was written from intent
rather than from a run, and the reviewer caught it by running it. A guard blind
to the canonical form of its own bug is theatre.

### Round-3 evidence

- `bats tests/check-doc-paths.bats` → **12/12**
- Mutation, the form round 2 proved invisible: `. versions.conf` flush-left in a
  fenced block turns case 12 **red** in `README.md` **and** in the newly
  governed `cli/AGENTS.md`; restoring turns it green
- Mutation: a non-rooted `..` token no longer trips the guard (case 10)
- Guard clean on all **eight** instruction files
- `shellcheck` clean; `bash -n` + `zsh -n` clean

### Still owed

A third review at the new sha. Two rounds have each found real defects the
previous one missed, so the base rate here does not support assuming the third
finds nothing.

---

## Round 4 — after the third adversarial review

Round 3 (`reviewed_sha f5ee9a3`) reproduced both round-2 Major fixes by mutation
and confirmed them genuinely correct, verified all twelve bats cases discriminate
by mutating the code each covers, and checked the CLI claims against a
from-source build. It returned **FAIL** on one Major.

**The Major: `ai/agy/AGY.md` was ungoverned.** A real, deployed agent-instruction
pointer, structurally identical to the two files round 2 had just caused to be
added — the same defect class, third occurrence.

### The fix is not the one that was asked for

Round 3's stated fix was one line: add `"ai/agy/AGY.md"` to `instruction_files()`.
That patches the instance and leaves the class. The evidence against it is this
review chain's own record:

| Round | Missing file(s) found |
|---|---|
| 2 | `cli/AGENTS.md`, `ai/hermes/AGENTS.md` |
| 3 | `ai/agy/AGY.md` |

Three rounds, three misses, from a list maintained by hand — which is precisely
the rot this guard exists to catch, reproduced inside the guard's own test.

`instruction_files()` now **discovers** its set from the git index by naming
contract, with an explicit exclusion list carrying a reason per entry
(`harness/` generated, `specs/` frozen, `docs/` historical). A file matching the
contract is governed the day it is staged; excluding one requires writing down
why. Discovery raised the governed set from 8 to 9 immediately, picking up
`ai/agy/AGY.md` without it being named.

New case 8 pins the mechanism rather than the outcome: it stages a probe
instruction file containing a dead path, asserts discovery finds it and the
guard fails on it, then removes it. `git ls-files` reads the index, so a
staged-but-uncommitted file counts — the case that matters, since a new
instruction file arrives staged alongside the code it describes.

Also applied: `README.md`'s "~50 scripts total" corrected to ~40 (actual 39).
Round 3's second Minor — that a non-backticked markdown link to a repo path
would be missed — is accepted as a known limit with no live instance; widening
extraction beyond backticks reopens the false-positive problem that cost two
revisions in round 1.

### Round-4 evidence

- `bats tests/check-doc-paths.bats` → **13/13**
- Discovery governs 9 files, verified by listing them
- Probe test leaves the worktree clean (`git status --short` shows only the
  intended edits)
- `shellcheck` clean; `bash -n` + `zsh -n` clean

### Still owed

A fourth review at the new sha. Three rounds have each found real defects, and
this round changed the test suite's own foundation, so the base rate does not
support assuming the next one finds nothing.

---

## Round 5 — after the fourth adversarial review

Round 4 (`reviewed_sha 20c4807`) reproduced every claim round 4's
`verification.md` made and found them all true — no false claim this round,
unlike round 2. It returned **FAIL** on one Major.

**The Major: the probe case could strand a fabricated instruction file in the
real working tree.** The case stages `ai/probe-agent/AGENTS.md` — it must, since
discovery reads the git index — and cleaned up two lines later. bats aborts a
test body at the first failing command, so an unguarded `git add -N` that failed
(a concurrent session's `index.lock` is enough) would skip the cleanup and leave
a file claiming a dead path in the tree the whole suite shares.

Not contrived: a real commit landed on this branch in this worktree *during* the
review, which is the same concurrency the finding depends on.

Reproduced both ways before and after the fix:

```console
# bats abort skips later lines — confirmed on a minimal case
$ ... false; rm -f "$PROBE"   -> $PROBE still present

# the real scenario, after moving cleanup into teardown()
$ touch "$(git rev-parse --git-dir)/index.lock"
$ bats tests/check-doc-paths.bats      # 2 probe cases fail, as they must
$ ls ai/probe-agent harness/probe-agent
   -> absent; teardown ran despite the failure
$ git status --short                    # no stray files
```

Cleanup moved into `teardown()` rather than guarded with `|| true`: bats
guarantees teardown runs, so the invariant no longer depends on statement order
inside a test body. `PROBE_REL` is set *before* anything is created, so teardown
can clean up even if the very next line fails.

Also applied, from round 4's Minor: a case asserting a file staged under an
excluded prefix (`harness/`) is **not** discovered. The exclusions carried
reasons but no test, so a regression dropping them would have pulled generated
records and frozen specs into the governed set silently.

### Round-5 evidence

- `bats tests/check-doc-paths.bats` → **14/14**
- Fault injection (`index.lock` present): probe cases fail, **nothing leaks**,
  `git status` clean
- Guard clean on all 9 discovered instruction files

### Round 4's remaining Minor, accepted not fixed

`check-doc-paths.sh` has no standalone CI step; enforcement is the one bats
case. Pre-existing since round 1 and outside this PR's diff. The bats suite runs
in the `test` job, so it *is* enforced in CI — a dedicated step would improve
the failure message, not the coverage. Left as-is deliberately.

### Still owed

A fifth review at the new sha, per this chain's convention. Four rounds have
each found a real defect; the last two were in the fix for the previous one.

---

## Round 6 — after the fifth adversarial review

Round 5 (`reviewed_sha 3c6d77d`, run against `main` directly — no PR, no
branch, the feature was already merged) returned **FAIL** on one REAL Major and
one Minor.

**The Major recalibrates the diminishing-returns read from round 4.** It was
not a defect in a previous fix — nothing round 5 found existed because of
anything rounds 2-4 touched. `instruction_files()`'s discovery regex matched
`AGENTS.md` / `CLAUDE.md` / `AGY.md` / `GEMINI.md` / `copilot-instructions.md`
anywhere in the tree, but anchored `README.md` to the repo root only
(`^README\.md$`) — an asymmetry that was never in the exclusion list this file
says every exclusion needs a reason for, and had been there since before round
1. Reproduced by widening the pattern and running the guard against every
non-excluded `README.md` in the repo (`git ls-files '*README.md' | grep -vE
'^harness/|^specs/|^docs/'` → 6 files): `cli/README.md` and `ai/nan/README.md`
had real dead paths, and — dimensioning the blast radius past what the review
itself reported — so did `ai/opencode/README.md`, a third file the reviewer
didn't name. One of the three, `scripts/healthcheck.sh` in `cli/README.md`, is
the exact file whose staleness in `.claude/CLAUDE.md` was this whole spec's
founding incident (#916).

**The Minor**: mutating the discovery alternation (dropping `AGY.md`, the same
silent-drop class rounds 2-4 each independently found once) left all 14 tests
green. Nothing pinned that the discovered set actually contains a member of
each pattern.

### Fix

- `tests/check-doc-paths.bats`: folded `README\.md` into the main alternation
  `(^|/)(...)$` instead of a separate `^README\.md$` anchor.
- `cli/README.md`: de-backticked `scripts/healthcheck.sh` and
  `scripts/doctor.sh` — both retired, named only as history in a sentence
  about what `dotf doctor` consolidates. Matches this guard's own stated
  convention ("to name a path that no longer exists, write it in plain text").
- `ai/nan/README.md`: `specs/SDD-007-ai-tooling-consolidation/proposal.md` →
  `specs/archive/SDD-007-iac-deploy-strategy/proposal.md` (the spec's `id:` is
  `SDD-007-ai-tooling-consolidation`; its folder was named
  `SDD-007-iac-deploy-strategy` before archiving — a real rename this guard
  caught).
- `ai/opencode/README.md`: de-backticked `ai/ollama/`, a forward-looking
  "when wired" reference to a directory that does not exist yet — not a stale
  claim, but backticks make the guard treat it as a live one.
- Two new bats cases: `cli/README.md` is discovered (pins the nested-README
  fix), and a discovery-floor case asserting one real member per pattern
  (`AGENTS.md`, `ai/agy/AGY.md`, `.claude/CLAUDE.md`,
  `ai/copilot/copilot-instructions.md`, `README.md`) — membership assertions,
  not an exact list or count, so it does not resurrect the hand-maintained list
  round 3 replaced with discovery. `GEMINI.md` has no real member in this repo
  today and is deliberately not asserted.

### Round-6 evidence

- `bats tests/check-doc-paths.bats` → **16/16**
- Guard clean on all 6 non-excluded `README.md` files in the repo, not just the
  2 the review named
- Both new cases mutation-verified before being trusted: reverting the
  alternation to drop `AGY.md` turns the discovery-floor case red and leaves
  the nested-README case green; reverting `README.md` back to a root-only
  anchor turns the nested-README case red and leaves the discovery-floor case
  green — each case fails on the mutation it exists for, not on an unrelated
  one
- `shellcheck` clean; `bash -n` / `zsh -n` clean
- `check-md-escapes.sh` clean on every touched file

### Still owed

A sixth review at the new sha. Note for the standing "when to stop" question
this chain keeps asking: round 5's Major was not introduced by fixing round 4
— it was real, independent, and predates this chain's first commit. That
argues against reading round 4's "last two defects were in the previous fix"
as evidence the chain had reached diminishing returns; it argues instead that
five fresh, independent passes have now found five real, distinct defects.

---

## Round 7 — after the sixth adversarial review

Round 6 (`reviewed_sha b5209f6`) returned **FAIL** on one REAL Blocker, found
by running the exact command CI runs rather than by inspection.

**The Blocker, and two honest confessions, not one.**

**(a) The defect itself.** The new bats case added in round 6's fix —
`"check-doc-paths: discovery floor — each governed pattern has a known real
member [#916]"` — used a literal U+2014 em dash instead of a hyphen.
`scripts/check-bats-names.sh` is a required `lint` job step (CURATOR-001/#615)
guarding exactly this: non-ASCII in a `@test` name silently unregisters the
test. Confirmed by the reviewer: the byte is `e2 80 94`, it is new to round 6's
own commit (the same file at `main` is clean under the same script), and the
one-character fix (em dash → hyphen) restores both the name-lint and the full
bats suite.

**(b) What let it through.** Round 6's own `verification.md` entry (above)
listed `bats tests/check-doc-paths.bats` and shellcheck/parser evidence, but
never ran `check-bats-names.sh` or the full `bats tests/*.bats` sweep — the
exact baseline round 1 established and every round since round 2 was supposed
to carry forward. That is a second instance of the class round 2 caught first:
a verification claim that reads as complete without having run everything CI
runs. Recorded here rather than smoothed over, because the honest record is
what gives this chain evidentiary weight for #881; a cleaned-up account of "one
small typo, fixed" would understate what actually happened twice in one spec.

**Aggravating note**: this exact defect class — non-ASCII in an instruction
artifact's structural text — bit this same session once already, earlier in
the DOCS-013 chain (an em dash in a `@test` name, caught the same way). Second
occurrence, and the cost scaled: from a free local lint catch to a full
~140k-token adversarial-review round spent finding a one-character defect.

**Benchmark note for #881, so a future model-recall replay is not
inflated:** this Blocker was found by running a deterministic script CI already
runs — `check-bats-names.sh` — not by anything requiring model judgment. When
replaying a candidate model against this chain's shas, round 6→7 should not be
scored as a test of model capability; it is an argument for running
deterministic checks (name-lint, `bats`, `shellcheck`) *before* an adversarial
review round, not instead of one, regardless of which model reviews.

### Fix

- `tests/check-doc-paths.bats:168`: em dash → hyphen in the `@test` name.
- `scripts/check-doc-paths.sh`: added one doctrine line generalizing round 5's
  point-fix — a path that does not exist *yet* ("will live at X when wired")
  is not a live claim either, so it also belongs in plain text, not backticks.
  Closes round 6's THEORETICAL Minor (the exception existed on one file with
  no general rule written down).
- No code behavior changed; both fixes are text-only.

### Round-7 evidence

- `./scripts/check-bats-names.sh tests/` → `OK (82 file(s) clean)`
- `bats tests/check-doc-paths.bats` → **16/16**
- `bats tests/*.bats` (full suite, the baseline round 1 established) →
  **1213/1214**, the one failure being the pre-existing, disclosed
  `#807`/`BUG-054` case, unrelated to this spec — confirmed by running it, not
  by citing round 6's claim of it
- `shellcheck` clean; `bash -n` / `zsh -n` clean on `check-doc-paths.sh`

### Known local/CI gap (ticketed, not fixed here)

The local pre-commit hook that ran on this branch's commits does not invoke
`check-bats-names.sh` — only CI's `lint` job does. That gap is exactly how
round 6's Blocker shipped past a local commit. Fixing the hook is a repo-wide
change to `.git-hooks`/pre-commit config, out of this spec's scope (the rubric
scores Scope, and this chain has been disciplined about not creeping it) and
low severity on its own (CI does catch it) — but the cost of the gap, measured
in this round, is a full adversarial-review pass burned on a one-character
defect. Ticketed separately rather than folded into this PR.

### Still owed

A seventh review at the new sha. The per-round defects are now mechanical
(a stray character, a missing standard-evidence step) rather than logical —
the convergence signal to watch for is the local loop matching the CI loop,
which this round's fix directly closes.
