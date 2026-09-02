---
tags: [spec, verification, templates]
created: "2026-09-02"
---

# Verification - CLI-070-doctor-next-steps

## Evidence

- [x] AC1 -> test `TestNextSteps` (multi-verb case) + `TestRun_NextStepsBlock/FAIL_with_a_remedy_surfaces_it_after_Results`
- [x] AC2 -> test `TestNextSteps` (WARN line asserted absent from the result)
- [x] AC3 -> test `TestNextSteps` (`` `bw status`: unauthenticated `` diagnostic span asserted not captured)
- [x] AC4 -> test `TestNextSteps` (`dotf tools install` repeated across two FAIL lines, asserted listed once)
- [x] AC5 -> test `TestNextSteps_NoFailNoBlock` + `TestRun_NextStepsBlock/quick_mode_with_no_FAIL_prints_no_block`
- [x] AC6 -> test `TestRun_NextStepsBlock/quick_mode_with_no_FAIL_prints_no_block` (Quick: true, no special-casing in the implementation)
- [x] AC7 -> `go test ./internal/doctor/...` full pass (no test disables color explicitly; live smoke test below shows color-capable terminal output unaffected)

## Test status

- Test suite: `cd cli && go test ./...` -> all 22 packages ok, no regressions (full module, not just the doctor package)
- Manual smoke test: built `dotf` from this branch's tree (`go build -o dotf.exe ./cmd/dotf`) and ran `dotf doctor` live on a real Windows box with an unauthenticated Bitwarden session. Output ended:
  ```
  Results: 121 passed, 3 failed, 10 warned, 32 skipped

  Next steps:
    bw login
    export BW_SESSION=$(bw login --raw) && bw sync
  ```
  Both lines are the `checkBitwardenReach` FAIL's two remedies (`run \`bw login\`` and `recover with \`export BW_SESSION=...\``), deduplicated and printed once each in first-seen order — the exact behavior AC1/AC4 assert, now also observed outside the test suite.
- No regressions in existing test suite: yes

## Decisions made during implementation

- Split what was originally one investigation into two PRs: the pin-floor semantics fix (#1441, independent, obvious-cause, shipped under `skip-sdd`) and this feature. Bundled together they were 108 production LOC pushing an unrelated bug fix through a spec process it didn't need and diluting this feature's own spec.
- Chose to extract remedies by scanning the *rendered transcript* for verb+backtick patterns rather than adding a structured hint field to `Report.Fail()`. Considered and rejected the structured-field approach: it would touch on the order of 100 call sites across the package for a benefit (compile-time-checked hints) that a doctor-package reviewer already gets for free, since the regex and the messages it reads live in the same package and get reviewed together.
- Scoped strictly to FAIL lines, not WARN. `checkGitWindowsFloor`'s WARN carries a real remedy (`upgrade with \`winget upgrade Git.Git\``) that is NOT surfaced in Next steps under this design — deliberate, per `Report`'s own contract that only FAIL drives the non-zero exit Next steps exists to explain.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? no
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — additive, within the existing doctor architecture
- [ ] New pattern candidate for `00_meta/patterns/`? no — single-repo concern

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-070-doctor-next-steps/` -> `specs/archive/CLI-070-doctor-next-steps/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
