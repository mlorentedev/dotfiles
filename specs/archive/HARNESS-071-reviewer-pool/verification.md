---
tags: [spec, verification, templates]
created: "2026-08-13"
---

# Verification - HARNESS-071-reviewer-pool

## Evidence

- [x] **A review signed outside the pool cannot be archived** -> `TestArchiveBlocksReviewerOutsideThePool`,
      plus a live smoke against the real pool file rather than a fixture (below).
- [x] **Absent pool skips, malformed pool refuses** -> `TestArchiveSkipsThePoolCheckWhenNoPoolExists`,
      `TestArchiveRefusesAMalformedPool` (4 shapes: bad JSON, empty array, missing
      key, blank id).
- [x] **Membership is exact, not prefix** -> `TestArchiveRejectsAReviewerThatMerelyResemblesAPoolEntry`.
      A prefix match would admit `…-flash-lite`.
- [x] **The pin does not depend on per-machine state** -> `TestReviewerCommandPinsProviderAndModelExplicitly`.
- [x] **Both gate and launcher refuse an out-of-pool model** -> `TestResolveReviewerRefusesAModelOutsideThePool`.

## Test status

- `go build ./...`, `go vet ./...` -> clean.
- `go test ./...` -> every package `ok`.
- `golangci-lint run` -> `0 issues`, on the **pinned** 2.12.2 from `versions.conf`
  (BUG-071: a local binary on another major reports 0 issues on code CI rejects).
- All 10 `features.json` commands pass and each runs at least one real test
  (1/2/6/2/1/5/3/3/3/1).

**Mutation tests.** Each behaviour broken deliberately, the guarding test observed
going red, then reverted. Run through a harness that ABORTS when a mutation fails
to apply — added after an earlier battery in this repo reported two false
`SURVIVED`s from a `sed` pattern that anchored one tab against a line indented
with two and silently matched nothing. A mutation that never applied is
indistinguishable from a mutant that survived.

| Gate mutation | Result |
|---|---|
| gate never consulted (`checkReviewerPool` removed) | detected |
| membership becomes a prefix match | detected |
| malformed pool passes instead of refusing | detected |
| empty pool treated as no-pool | detected |
| absent pool refuses instead of skipping | detected |

| Launcher mutation | Result |
|---|---|
| pi loses its explicit `--provider` (falls back to google) | detected |
| model no longer pinned, runner default wins | detected |
| agy keeps its 5m default timeout | detected |
| transcript no longer requested | detected |
| shell quoting removed — a prompt would execute | detected |
| empty `--reviewer` no longer defaults to the primary | detected |

**Live smoke**, built binary against the real `harness/reviewer-pool.json`:

```text
$ dotf spec archive ZZZ-001-smoke        # review.md signed claude-opus-5
Error: review.md records reviewer "claude-opus-5", which is not in harness/reviewer-pool.json
the models allowed to review are: nan/deepseek-v4-flash, agy/gemini-3.1-pro-high
re-run /adversarial-review on one of them, declare `review: waived` with a reason
in proposal.md, or pass --force-without-review

$ dotf spec archive ZZZ-001-smoke        # signed nan/deepseek-v4-flash
[OK] Archived

$ dotf spec review HARNESS-071-reviewer-pool --dry-run
Reviewer:   nan/deepseek-v4-flash (pi, primary)
[DRY RUN] would run:
  tmux new-session -d -s review-HARNESS-071-reviewer-pool -c <repo> 'dotf' 'secrets'
  'run' '--' 'pi' '--print' '--provider' 'nan' '--model' 'deepseek-v4-flash' …

$ dotf spec review HARNESS-071-reviewer-pool --reviewer claude-opus-5 --dry-run
Error: reviewer "claude-opus-5" is not in harness/reviewer-pool.json
available: nan/deepseek-v4-flash, agy/gemini-3.1-pro-high
```

## Both paths — status, stated precisely

The acceptance criterion is that each configured path produces a **real review**,
because a fallback never observed doing the job is decoration (#898).

| Path | Evidence | Clears the bar? |
|---|---|---|
| `nan/deepseek-v4-flash` via pi | BUG-074 round 3: a full adversarial review that re-ran the spec's own mutation battery rather than trusting its table, closed six prior Majors, and raised four new Minors (#956) | **yes** |
| `agy/gemini-3.1-pro-high` via agy | 2026-08-14: a full adversarial review of this spec — `review.md` (verdict FAIL, `reviewed_sha b24e105`) and a 58 KB transcript over 147 steps, running `go test` eight times, carrying the worktree's real HEAD | **yes** |

Both rows now clear the bar, and the second one took four attempts to get there.
That history is the useful part, because every failure looked like a success:

1. `--print` swallowed `--model` — a session greeting at exit 0.
2. Detached, agy auto-**denied** every tool call — `{"status":"SUCCESS","response":""}` after 14s.
3. With permissions granted it ran commands in its own install dir — `git rev-parse`
   failed, the suite was unreachable, and it wrote a **well-formed all-A PASS**
   having executed nothing. That artifact would have satisfied a naive reading of
   AC7, which is why the bar for ticking it is reach rather than existence.
4. With `--add-dir`, real reach — and a FAIL with findings that reproduce.

A PASS from attempt 3 and a FAIL from attempt 4 are the same shape on disk. Only
the transcript tells them apart, which is why AC6's transcript is not a nicety.

## Findings from the 2026-08-14 review, and their dispositions

| # | Severity | Finding | Disposition |
|---|---|---|---|
| 1 | Blocker | `poolWasTracked` used `git log -1 -- <file>`, which returns the commit that *deleted* the file, so a formally retired gate blocked every future archive — while the error message instructed exactly that retirement | **Fixed.** Now asks `git ls-tree HEAD`. Reproduced first in a scratch repo (4 commands); guarded by `TestArchiveAllowsAPoolRetiredInACommit`, observed red against the old implementation in isolation |
| 2 | Major | `gitStaleness` read committed history only, so uncommitted edits to a contract file passed as fresh — the cheapest possible bypass of the gate | **Fixed.** Adds a `git status --porcelain` check scoped to the three contract files. Test written red first; `TestGitStalenessIgnoresUncommittedNonContractFiles` pins the scope so a review's own in-flight artifacts never self-stale |
| 3 | Minor | `agy`'s `--print` ordering is positionally fragile | **Declined**, with the reviewer's own reasoning: `TestReviewerCommandGivesAgyThePromptAsThePrintValue` already pins the shape, and the reviewer graded it TESTED |

Finding 1 also resolved a contradiction this document carried: the decision below
says deleting the file disables the check deliberately, while the fail-closed
branch added later (from CodeRabbit on #959) made deletion permanent breakage.
Both are now true of different things — absent from HEAD is a recorded decision,
present in HEAD but gone from the tree is a loss.

## Decisions made during implementation

- **A dedicated file, not a key in `manifest.json` and not `model-map.json`.**
  doctor's harness-drift section reads the manifest, so adding a key risks
  machinery this slice should not touch; `model-map.json` is H-044's future
  schema (neutral tier -> provider *per agent*), a different shape that would
  have to migrate this file if it squatted on the name.
- **Ids, not tool names.** `agy models` lists `claude-opus-4-6-thinking` and
  `claude-sonnet-4-6` beside the Gemini family, so pinning the tool guarantees
  nothing about provider diversity.
- **Absent pool skips rather than refuses.** `dotf` runs in repos with no pool.
  Deleting the file disables the check — deliberately, because that deletion is a
  visible diff, the same auditable-escape philosophy as `review: waived` needing
  a stated reason.
- **The launcher refuses too.** Not duplication: the gate catches a review that
  already ran on the wrong model, the launcher stops it from running at all.
- **`shellQuote` deduplicated** rather than renamed — a byte-identical copy
  already existed in `spec_test.go`, so the test now uses the production one.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — a default is not a pin:
      BUG-074's third round ran on the intended model only because one machine's
      unversioned config happened to say so, while the runner's documented
      default was a different provider entirely.
- [ ] ADR-worthy decision? **not on its own** — it implements the independence
      the adversarial-review skill already requires. If H-044 lands, the
      relationship between that model-map and this pool deserves recording there.
- [ ] New pattern candidate for `00_meta/patterns/`? **maybe later** — "enforce
      the policy at the artifact, not at the runner" generalises, but one
      instance is not a pattern yet.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/HARNESS-071-reviewer-pool/`
- [ ] Bitácora ticket #955 closed with the PR links (ADR-018)
- [ ] Promotions above executed (if any)
