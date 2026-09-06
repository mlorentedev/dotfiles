---
tags: [spec, verification, templates]
created: "2026-09-05"
---

# Verification - HARNESS-111-review-base-and-reality

## Evidence

Every command below was executed in this session.

- [x] **AC1** `TestResolveReviewBaseIsTheParentOfTheCommitThatAddedTheSpec`,
      `TestResolveReviewBaseIsEmptyWhenTheSpecIsNotCommitted` — real git repositories, not fixtures.
- [x] **AC2** `TestResolveReviewBaseGivesANonEmptyDiffOnMainAfterTheWorkLanded`. It asserts the
      **anchor first**: `git diff --name-only main...HEAD` must be empty in the fixture, or the
      test `t.Fatal`s rather than passing. Without that, it could pass on a repository where the
      old behaviour also happened to work.
- [x] **AC3** `TestSpecReviewRefusesWhenNoBaseCanBeResolved`,
      `TestSpecReviewRefusesWhenTheBaseIsTheHead`, and
      `TestSpecReviewLaunchesAndStatesTheBaseWhenOneResolves` — the third exists so the first two
      cannot pass vacuously by the launcher refusing everything.
- [x] **AC4** `TestReviewPromptStatesTheResolvedBase`.
- [x] **AC5** `TestWriteReviewRequestRecordsTheBase` and
      `TestReviewRequestWithoutBaseSHAStillParses`.
- [x] **AC6** `tests/guard-review-verdict-honours-reality.bats`, 6 assertions, mutation-proven.
- [x] **AC7** full loop below.

## Test status

```
go build ./...                        rc=0
go test ./...                         rc=0   (whole module)
GOOS=windows go vet ./...             rc=0
GOOS=darwin  go vet ./...             rc=0
golangci-lint run ./...               0 issues   (2.12.2, the versions.conf pin)
bats tests/guard-review-verdict-honours-reality.bats   6/6 ok
```

### Guard mutation

The guard has to fail when the contradiction returns, or it is decoration. Reintroducing the exact
old line (`FAIL — at least one blocker or major OR rubric has at least one D`) with the anchor
asserted before the patch:

```
ok 1 guard: the adversarial-review skill exists where the gate expects it
not ok 2 guard: the verdict rule qualifies Major by reality, not severity alone
not ok 3 guard: the unqualified 'any blocker or major' rule is gone
ok 4 guard: PASS WITH GAPS is reachable for a THEORETICAL-only Major
not ok 5 guard: a Blocker still blocks regardless of its reality tag
ok 6 guard: the Reality classification the verdict defers to is still present
MUTATED rc=1
restored: guard green again
```

## One thing this change does NOT take effect without

**The reviewer reads the DEPLOYED skill, not the repository's.** `compile-harness.sh` writes
`~/.pi/agent/skills/adversarial-review/SKILL.md`, and that copy is what a launched review opens.
Measured after the edit:

```
diff -q harness/skills/adversarial-review/SKILL.md ~/.pi/agent/skills/adversarial-review/SKILL.md
  -> DIFFERS
```

So root 2's fix is inert until a deploy runs. Stated here rather than discovered when the next
review returns FAIL on a THEORETICAL finding and somebody concludes the fix did not work.

Root 1 has no such gap: it is compiled into `dotf`.

## Decisions made during implementation

- **The base is the spec folder's add-commit parent, not the PR's base branch.** The PR route needs
  network, `gh` auth and human-editable metadata, and it answers differently per PR. The chosen
  route is offline and deterministic, and it is right in both the branch and the post-squash cases
  — the second being the one that was broken.
- **Refuse rather than degrade.** A review with an empty diff does not error; it quietly becomes a
  reading of the spec folder and reports findings with nothing executed behind them. Round 4 shows
  what that produces: one THEORETICAL finding and one factual error. A launch that cannot be scoped
  is refused.
- **Root 2 is a documentation contradiction, so it is fixed in documentation.** Parsing the
  findings table in Go was considered and rejected: the verdict is self-reported either way, so a
  markdown parser adds a fragile surface and no trust. The rule the reviewer applies is what was
  wrong.
- **The two-layer test split is deliberate**, and it is BUG-093 round 3's lesson applied: real-git
  tests prove the resolver's answer is right, seam tests prove the caller honours it. A seam test
  passes over a wrong implementation; a resolver test cannot see whether anyone acts on it.

## Root 3 — what was executed

- **The two defective forms are gone from both copies, and the routing rule is in both.** Run
  verbatim, the f7 command returns `0` then `4` for the repo record and `0` then `4` for the vault
  source. A bare `grep -c "before archive"` returns `3` per copy and is the **wrong** command: two
  are the skill describing when it runs, and the third is the new instruction not to use the phrase.
- **`TestStaleRefusalOffersTheExitThatKeepsTheReview` is mutation-proven.** Reverting `review.go` to
  the previous three-exit message fails it with *"refusal must name the exit that keeps the review"*;
  restoring returns it to PASS. It asserts three things, not one: the exit is present, it says where
  the dispositions go, and it is offered **before** each of the three overrides.
- **`bash scripts/compile-harness.sh --check` exits 0** — `OK: no harness drift` — so the edited
  record is still consistent with what the harness expects to generate.
- **The bodies of the two copies are byte-identical modulo frontmatter**, confirmed by diffing them
  with the frontmatter stripped.
- **Go layer clean**: `go build`, `go vet`, `go test ./...`, `GOOS=windows go vet`,
  `GOOS=darwin go vet` all rc=0; `golangci-lint run` 0 issues on the pinned 2.12.2.

### Not yet true at the time of writing, and it matters

There is a **third** copy of this skill — the deployed one the reviewer actually opens at
`~/.pi/agent/skills/adversarial-review/SKILL.md` and `~/.claude/skills/adversarial-review/SKILL.md`.
Measured after the fix landed in the record and the vault, both deployed copies still read **2
defective forms, 0 routing-rule lines**. `compile-harness.sh --deploy` must run after #1543 merges
and before this spec's own review is launched, or round 1 of HARNESS-111 will be reviewed under the
very template HARNESS-111 fixes — and will produce the list root 3 forbids.

## Decisions made during implementation, root 3

- **The gate is correct and the template was wrong**, and that direction was established by
  measurement before anything was designed: of the ten lines that changed in BUG-093's contract
  files, **zero** were acceptance criteria and **zero** were `behavior` fields — six `evidence` and
  four `verification` strings, plus prose.
- **Field-granular staleness was considered and rejected.** Exempting `verification` would exempt
  the command the reviewer *ran*, opening the "get a pass, rewrite in the working tree, archive"
  bypass that `review.go:187` already names; and `proposal.md` prose has no field structure to key
  on. Recorded in `tasks.md` under *Not done, deliberately*, because it is the attractive wrong turn.
- **`--force-without-review` and `review: waived` were both available and both refused.** The first
  discards a review that exists and passed; the second asserts none happened. Each is a lie in the
  opposite direction from the stale evidence line it would paper over.
