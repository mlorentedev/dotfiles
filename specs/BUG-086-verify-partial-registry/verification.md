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
