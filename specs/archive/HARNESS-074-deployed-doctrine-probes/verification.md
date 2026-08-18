---
id: "HARNESS-074-verification"
type: spec
status: draft
owner: manu
tags: [spec, verification, templates]
created: "2026-08-18"
---

# Verification - HARNESS-074-deployed-doctrine-probes

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof:

- [x] AC1: `dotf doctor` fails when an enforced region declared in manifest is missing from a deployed surface.
  - Proof: `TestCheckDeployedDoctrine_MissingRegion` in `checks_deployed_doctrine_test.go` asserts doctor reports `FAIL` and names both the missing region and the surface.
- [x] AC2: `dotf doctor` passes when all deployed surfaces carry their declared enforced regions.
  - Proof: `TestCheckDeployedDoctrine` ("all deployed surfaces present and contain enforced regions -> pass") passes cleanly.
- [x] AC3: Mutation test demonstrates doctor goes RED when an enforced region is removed from a deployed target.
  - Proof: `TestCheckDeployedDoctrine_MissingRegion` verified exit code 0 when asserting failure on mutated target.

## Test status

- Unit test suite: `go test -count=1 ./...` in `cli/` -> 100% passing across all 15 packages.
- No regressions: clean test execution.

## Decisions made during implementation

- Kept deployed doctrine target inspection decoupled from git repository location so `dotf doctor` works offline and in standalone mode.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? No.
- [x] ADR-worthy decision? No.
- [x] New pattern candidate? No.

