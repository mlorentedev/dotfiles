---
tags: [spec, verification, templates]
created: "2026-09-21"
---

# Verification - HARNESS-136-model-limit-drift

## Evidence

| AC | Proof |
|---|---|
| AC1 | commit `51e9ffe`; `dotf doctor --verbose` reports **7 models match the provider catalog** on this branch and nine findings on `main`. Values cross-checked against `~/.cache/opencode/models.json` and <https://nan.builders/docs/models>, which agree on every limit the latter states. |
| AC2 | `TestModelLimitsFailsOnAnOverstatedContextWindow` |
| AC3 | `TestModelLimitsWarnsOnAnUnderstatedOutputCap` |
| AC4 | `TestModelLimitsResolvesTheSameIdPerProvider` |
| AC5 | `TestModelLimitsSkipsWhenTheCatalogIsNotCached` |
| AC6 | `TestModelLimitsIgnoresAModelTheCatalogDoesNotPublish`, `TestModelLimitsIgnoresAnUnpublishedLimit` |
| AC7 | `TestModelLimitsFailsOnAnUnparseableDeclaration` |
| AC8 | `features.json` f8 — the source contains no write call and takes no fix flag |

## Test status

- `go test -count=1 ./...` — green across the module; the seven new tests run and
  pass (`-run TestModelLimits -v`, 7/7).
- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` — clean.
- `golangci-lint run` at the pinned **v2.12.2** (`versions.conf`) — 0 issues.
- `bats tests/pi-config.bats` — 18/18, the suite that reads `ai/pi`.
- **Mutation, proving the assertions are not vacuous.** Four mutations, each
  killed by the test written for it and by no other: drop the provider scoping
  (1 test), swap the two severities (2 tests), drop the unpublished-limit guard
  (1 test), let an absent catalog PASS (1 test).
- **Proof by consequence, both directions.** The same binary reports
  `(1 checks, all ok)` against this branch and nine findings — 2 FAIL, 7 WARN —
  against `main`.

## Decisions made during implementation

- **Doctor, not CI.** The truth side is `~/.cache/opencode/models.json`, a
  machine-local file. Committing a snapshot would let CI run the check at the
  cost of a second hand-maintained copy of numbers someone else publishes —
  which is the defect being reported, moved one indirection away. Same split
  `checkModelPins` documents.
- **Severity by direction, not by tidiness.** Over-declaration FAILS because the
  provider rejects a request the config invited; under-declaration WARNS because
  nothing breaks. One severity for both would either cry wolf or bury the
  request-breaking half.
- **Provider-scoped lookup, and it was not theoretical.** The first pass matched
  on the bare model id and reported `nan/qwen3.8-flash` as needing 1000000 — it
  had read openrouter's row for a same-named model. AC4 exists because that
  happened, not because it might.
- **Two commits, not one.** The data correction is verifiable against published
  sources; the check is the mechanism that keeps it true. Separating them keeps
  `fix:` and `feat:` honest for release-please.
- **AC1's verification asserts the PASS line, not the absence of findings.** The
  first draft negated a `grep` for FAIL/WARN and passed against `main`, where the
  check does not exist: no section, no findings, no failure. An absent check read
  as agreement — the exact confusion the check's own SKIP-never-PASS rule exists
  to prevent, reproduced in its verification. It now asserts the section RAN and
  reported agreement.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? **Yes** — `grep -q -v` is not
      portable: ugrep returns 1 where GNU grep returns 0, so a guard written with
      it passes in CI while appearing to fail locally. Found while verifying this
      spec. Belongs beside the existing shell-compatibility lessons.
- [ ] ADR-worthy decision? No — this applies an existing split (HARNESS-067's
      doctor-not-CI reasoning), it does not establish one.
- [ ] New pattern candidate for `00_meta/patterns/`? No — one project so far.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-136-model-limit-drift/` -> `specs/archive/HARNESS-136-model-limit-drift/`
- [ ] Bitácora board ticket `#1594` moved to Done / closed with PR link (ADR-018)
- [ ] Independent adversarial review passed (reviewer != implementer)

## Adversarial review, round 1 — disposition

`review.md` round 1: **FAIL**, `nan/qwen3.8-flash`, reviewed `c7a8adc`. Committed
verbatim; round 2 reviews the follow-up PR's head.

| Severity | Finding | Disposition |
|---|---|---|
| Major (REAL) | F1 — an unreadable declaration WARNs, so doctor exits 0 on a declaration nobody read; AC7's unreadable half unimplemented | **Applied** — `readDeclaration` FAILs on any read error but absence; `TestModelLimitsFailsOnAnUnreadableDeclaration` (mode 000), red with the FAIL mutated back to WARN |
| Major (THEORETICAL) | F2 — `compared++` counts a catalog row that publishes no limit, printing "1 models match" | **Applied** — `compareModel` returns how many limits were comparable; a model counts only when one was. `TestModelLimitsDoesNotCountAModelWithNoPublishedLimits`, red with the old counting restored |
| Minor | F3 — the provider-scoping example named openrouter, which keys the row `qwen/qwen3.8-flash` | **Applied** — proposal and doc comment now name the real collisions, measured against the cache |
| Minor | F4 — "no checkout → SKIP" unpinned | **Applied** — `TestModelLimitsSkipsOutsideACheckout`, red with the SKIP mutated to PASS |
| Minor | F5 — `checkModelLimits` 84 lines, CC≈17 | **Applied** — split into `readDeclaration`, `readCatalog`, `compareModel`; the orchestrator is now under the AGENTS.md bar. The sibling `checkModelPins` has the same shape and is **not** touched here (out of this spec's scope) |
| Minor (SPECULATIVE) | F6 — an empty `HOME` makes the catalog path cwd-relative | **Declined, with reason** — package-wide convention shared by every doctor check, not introduced here; the reviewer marked it surface-only |
| Minor | F7 — `features.json` all `pending`; `tasks.md` PR box unticked | **Applied** — every feature command executed and its result recorded; box ticked |

Mutation for this round: 3 mutations on the three behaviours the review found
unpinned, 3 killed.
