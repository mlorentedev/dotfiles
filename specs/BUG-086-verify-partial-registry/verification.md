---
tags: [spec, verification, templates]
created: "2026-08-16"
---

# Verification - BUG-086-verify-partial-registry

## Evidence

| AC | Proof |
|---|---|
| AC1 | `TestSecretsVerify_MalformedEntryDoesNotHideTheRest` |
| AC2 | `TestSecretsVerify_ScopedAwayFromDefect` |
| AC3 | `TestSecretsVerify_ScopedToTheDefect` |
| AC4 | `TestSecretsVerify_DuplicateIdBlamesTheSecondOne` |
| AC5 | `TestParseRegistryPartial_DefectiveSecretsNeverReachEntries` |
| AC6 | `TestParseRegistry_StrictStillFailsOnFirstDefect` + the whole pre-existing suite |
| AC7 | `TestSecretsVerify_StructuralFailureStillAborts` |
| AC8 | `TestSecretsVerify_ScopedToDuplicateIdReportsBothHalves`, `TestSecretsVerify_SeveralDefectsOnOneId` — both observed failing with the short-circuit reintroduced |

### Mutation check — the guards were observed failing

A guard never seen failing is not evidence (#898). `verify` was pointed back at the
strict door (the exact defect) and the four behavioural guards failed:

```
--- FAIL: TestSecretsVerify_MalformedEntryDoesNotHideTheRest
    verify output missing "FAILED"
--- FAIL: TestSecretsVerify_ScopedAwayFromDefect
--- FAIL: TestSecretsVerify_ScopedToTheDefect
    the named defect should be a FAILED row
--- FAIL: TestSecretsVerify_DuplicateIdBlamesTheSecondOne
```

Restored, suite green.

### No behaviour change on a healthy registry

```
$ dotf secrets verify        # real registry, built from this branch
33 ok, 0 missing, 0 failed
```

That is the point: this changes what happens when the registry is broken, and nothing
about what happens when it is not.

## Adversarial review — PASS, and what was done with its findings

`nan/deepseek-v4-flash` against `e033302`, **4m 48s**, verdict **PASS**, 0 blockers,
0 majors, 3 minors. Verdict in `review.md`.

| # | Class | Finding | Disposition |
|---|---|---|---|
| 1 | Minor / REAL | `validateSecret` was 72 lines against AGENTS.md's `< 40` guideline; `scopeVerify` at 41 | **APPLIED** — extracted `checkExpose`, `checkBWFolder` and `checkVarNames`. 72 → **31 lines**, no behaviour change |
| 2 | Minor / THEORETICAL | a token that is both a defect id and a valid var name produces two rows; correct but undocumented and untested | **APPLIED** — pinned by `TestSecretsVerify_TokenThatIsBothADefectIdAndAValidVar` and documented at the branch |
| 3 | Minor / SPECULATIVE | `kept` copies each validated secret; O(n) beyond the YAML decode for very large registries | **DECLINED** — the registry holds 33 secrets and the reviewer scoped it to "~500+". Optimising an unmeasured path would trade a clear implementation for a guess |

Finding 1 was a real violation of this repo's own Code Quality Rules, found by reading
the code against the stated threshold rather than by running anything — the kind a test
suite structurally cannot produce.

Finding 2 is the sharper one. Two *different* secrets can be named by one token — a
malformed entry with `id: FOO` and a well-formed entry exposing a var called `FOO` — and
answering for only one of them would hide a real result behind a name collision. The
behaviour was already correct; what was missing was anything asserting it stays correct.

**Both applied without touching `proposal.md`, `tasks.md` or `features.json`.** The
review's suggested remedy for finding 2 was to document it in AC8, but those are the
contract files the staleness check watches: editing one after `reviewed_sha` invalidates
the review that asked for the edit, and buys another round (#998, HARNESS-072). The same
information lives in a test and a code comment, which carry it better anyway — a test
cannot go stale silently.

## Test status

- `go build ./... && go vet ./... && go test ./...` — every package ok
- `golangci-lint run` (pinned 2.12.2) — **0 issues**
- No regressions: the pre-existing registry suite passes unchanged, which is also AC6's
  evidence — those tests assert the strict door's messages, and the strict door is now a
  thin policy over the partial one.

## Decisions made during implementation

- **Strict implemented on top of partial, not beside it.** One set of per-secret checks,
  two policies. Two independent validators is the shape that produced BUG-084: the moment
  one moves, they disagree silently. `ParseRegistry` is now literally
  `ParseRegistryPartial` + "any defect → first error".
- **Defective secrets are excluded, not flagged in place.** The package's other code
  documents that it relies on validation having run.
- **Defect vars are not expanded.** The entry is exactly what could not be read, so
  naming the vars it "would have" exposed is guesswork; the row is keyed by secret id.
- **Structural failures stay fatal.** "Cannot read the document" and "read it, and this
  entry is wrong" are different conditions, and only the second has anything to report.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? No — the lesson this belongs to is already written
      (*"a check whose precondition the architecture forbids reports SKIP forever"* and
      the #997 family). This is another instance, not a new rule.
- [ ] ADR-worthy? No.
- [ ] Pattern for `00_meta/`? No.
