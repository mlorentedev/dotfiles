---
tags: [spec, verification, templates]
created: "2026-09-29"
---

# Verification - GUARD-005a-malformed-review-artifact

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> `TestVerifyReviewProducedCatchesAMalformedVerdict` and
  `TestSpecReviewForegroundRejectsMalformedReview`
- [x] Criterion 2 -> `TestVerifyReviewProducedAcceptsAFreshVerdict`
- [x] Criterion 3 -> `TestVerifyReviewProducedCatchesARunThatWroteNoFile`
  and `TestVerifyReviewProducedCatchesAnUnchangedVerdict`

## Test status

- Fail-first: `go test ./internal/spec -run '^TestVerifyReviewProducedCatchesAMalformedVerdict$' -count=1`
  failed because `VerifyReviewProduced` returned nil for the malformed WIN-014
  artifact.
- Targeted behavior: `go test ./internal/spec ./internal/cmd -run '^(TestVerifyReviewProducedCatchesAMalformedVerdict|TestSpecReviewForegroundRejectsMalformedReview)$' -count=1`
  -> both packages pass.
- Package suites: `go test ./internal/spec ./internal/cmd` -> pass.
- Static checks: `go build ./...`, `go vet ./internal/spec ./internal/cmd`,
  `golangci-lint run --new-from-rev=main`, `git diff --check`, and
  `jq empty specs/GUARD-005a-malformed-review-artifact/features.json` -> pass.
- Manual smoke test: not required; the foreground completion seam is covered
  directly without launching a real external reviewer.
- No regressions: the pre-existing missing-file, unchanged-file, and fresh
  verdict tests pass in the package suite.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Reuse `ParseReview` after the existing file/digest checks so the launcher and
  archive gate share one machine-readable verdict contract.
- Preserve missing and unchanged review precedence because those errors diagnose
  the runner writing nothing more precisely than a generic parse failure.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: the prior GUARD-005 spec and
  lesson 215 already preserve the general provenance and parser-boundary lessons.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: this closes
  a validation gap in an existing contract.
- [x] New pattern candidate for `00_meta/patterns/`? no: no cross-project pattern
  beyond the already recorded output-validation rule.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/GUARD-005a-malformed-review-artifact/` -> `specs/archive/GUARD-005a-malformed-review-artifact/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
