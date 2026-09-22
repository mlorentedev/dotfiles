---
tags: [spec, verification, templates]
created: "2026-09-21"
---

# Verification - CLI-078-secrets-layout-drift

## Evidence

| AC | Proof |
|---|---|
| AC1 | `TestLayoutDriftReportsAFolderNameNothingCarries` |
| AC2 | `TestLayoutDriftChecksDormantDeclarations` + `TestBWDeclarationsIncludesDormantBlocksOnAgeBackedSecrets` |
| AC3 | `TestLayoutDriftReportsAMisfiledItem` |
| AC4 | `TestLayoutDriftResolvesFieldsTheWayTheReaderDoes` (all four paths, both directions) |
| AC5 | `TestLayoutDriftReportsOneFindingPerProblemNotPerVar` + `TestLayoutDriftDedupesAMisfiledItemAcrossVars` |
| AC6 | `TestDecodeItemsCannotCarryAValue` |
| AC7 | `TestDecodeItemsErrorNeverQuotesTheBody` |
| AC8 | `features.json` f8 — the source, comments stripped, calls no writer |

## Test status

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` — clean
- `go test -count=1 ./...` — green across the module
- `golangci-lint run` at the pinned **v2.12.2** — 0 issues
- All eight `features.json` commands executed: **8/8 pass**
- **Live run against the real store** (185 items, 27 declared targets): four
  findings, each independently confirmed — `dockerhub` unfoldered, and
  `github-cli-pat`, `github-release-pat`, `zoho` absent.

### Mutation

Five mutations, each killed:

| Mutation | Killed by |
|---|---|
| a password reaches `ItemSummary` | `TestDecodeItemsCannotCarryAValue` |
| skip dormant declarations | `TestBWDeclarationsIncludesDormantBlocksOnAgeBackedSecrets` |
| report a misfiled item per var | `TestLayoutDriftDedupesAMisfiledItemAcrossVars` |
| `hasField` forgets the login cases | `TestLayoutDriftResolvesFieldsTheWayTheReaderDoes` |
| drop the per-var field fallback | `TestBWDeclarationsResolvesPerVarFieldOverrides` |

**Three of those five survived the first round**, and the tests that kill them now
were written in response. The dormant-declaration hole is the one worth recording:
the original test handed `LayoutDrift` a struct with `Dormant: true` and passed
happily while `BWDeclarations` — the walk that has to *discover* dormancy — was
never exercised at all. A fixture built past the code under test proves nothing
about it.

## Decisions made during implementation

- **The projection is a type, not a rule.** `itemWire` cannot name a value, so
  `encoding/json` drops passwords during decoding. A redaction step would have
  been a line someone could delete; this is one nothing can reorder past.
- **The `Dotfiles/` prefix lives in the declaration, not in code.** Prepending it
  in `ResolveFolder` would mean the declared value differs from the real one, so
  the report would have to undo the translation to avoid lying.
- **Dormant declarations are checked.** They are the migration target, and three
  of them pointed at items that do not exist. Left unchecked they surface only
  during the migration that needs them.
- **`hasField` mirrors `fieldFromItem` deliberately**, against lesson 279's
  general rule, because the alternative puts a value-resolving path one refactor
  away from a value-free report. Named in the code and pinned by a test.
- **Read-only is structural.** No `--fix`, no writer passed in, and a
  committed check that greps the source (comments stripped) for any write call.

## Promotion candidates

- [ ] Lesson for `docs/lessons/`? **Deferred, deliberately.** The candidate is
      "a fixture built past the code under test proves nothing about it", and it
      is a restatement of lesson 279's family rather than a new class. Recorded
      in this file instead; DoD §2 skip, stated.
- [ ] ADR-worthy? No — this implements ADR-028 §6, it does not amend it. The
      folder NAMES gained a prefix, which §6 left unspecified.
- [ ] Pattern for `00_meta/patterns/`? Not yet. "Project at the producer so the
      type system enforces redaction" is a strong candidate if it recurs in a
      second project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-078-secrets-layout-drift/`
- [ ] Bitácora `#1596` closed with the PR link (ADR-018)
- [ ] Independent adversarial review passed (reviewer != implementer)
