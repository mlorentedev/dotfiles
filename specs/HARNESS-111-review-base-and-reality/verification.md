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
