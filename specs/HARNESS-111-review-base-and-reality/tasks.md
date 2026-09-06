---
tags: [spec, tasks]
created: "2026-09-05"
---

# Tasks - HARNESS-111-review-base-and-reality

## Root 1 — the review base

- [x] `ResolveReviewBase(repoRoot, specDir)` — parent of the commit that first added the folder;
      local, offline, deterministic, and correct on a branch AND on main after a squash.
- [x] `ReviewRequest.BaseSHA` with `omitempty`, so sidecars written before the field still parse.
- [x] `WriteReviewRequest` takes and records the base.
- [x] `ReviewPrompt` states the base and forbids substituting one; it also states that the scope is
      the WHOLE change and not a delta since the last round.
- [x] `dotf spec review` refuses when no base resolves, and when base == HEAD.
- [x] Seams (`resolveReviewBase`, `headSHAOf`) so the command-level fixtures, which build a fake
      `.git` directory rather than a repository, can drive both answers.

## Root 2 — the verdict contradiction

- [x] Make the skill's verdict list agree with its own Reality rule: FAIL on any Blocker or a REAL
      Major; a THEORETICAL/SPECULATIVE-only Major routes to PASS-WITH-GAPS with a disposition.
- [x] State in the skill WHY it is spelled out twice, with the measurement, so the next editor does
      not "simplify" it back into a contradiction.
- [x] `tests/guard-review-verdict-honours-reality.bats` — 6 assertions covering both directions.

## Verification

- [x] Real-git tests for the resolver, including the empty-`main...HEAD` case with an anchor.
- [x] Command-level refusal tests plus the positive direction, so the refusals cannot pass
      vacuously.
- [x] Guard mutation-proven: reintroducing the old verdict line turns 3 of 6 assertions red.
- [x] `go build`, `go test ./...`, `golangci-lint run`, `GOOS=windows/darwin go vet` clean.

## Not done, deliberately

- [ ] Backfilling `base_sha` into archived specs — they are historical records.
- [ ] Softening Majors in general, or Blockers at all. Out of scope, argued in the proposal.
- [ ] Parsing the findings table in Go to compute the verdict. The verdict is and always was the
      reviewer's judgment; the defect was the rule it applied, not who applied it. A markdown-table
      parser in a gate would be a new fragile surface for no added trust.
