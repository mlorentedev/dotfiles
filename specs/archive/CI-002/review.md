---
spec: "CI-002"
verdict: "PASS WITH GAPS"
reviewed_sha: "bade1483c8228dc182d13d058735aae08dfe17fe"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-10-10"
---
## Adversarial review

**Scope**: CI-002
**Sources**: `specs/CI-002/{proposal,tasks,verification}.md`, `.github/workflows/ci.yml`, `cli/internal/cmd/pi.go`, `tests/pi-packages.bats`

### Spec and task alignment
- All acceptance criteria are met and implemented correctly in Go (since HARNESS-139).
- Evidence in `verification.md` is reproducible. Features in `features.json` pass.
- Mutation test for the ordering of the skip vs probe passes.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | ci | `DOTFILES_SKIP_PI_PACKAGES` is passed to `test-windows` but not to the `integration` container. If `pi` is ever installed in the test container, Linux CI will run the pi package reconcile on every PR, bypassing the filter entirely and undoing this spec's win. | `.github/workflows/ci.yml` sets it for Windows, but `integration`'s `docker run` passes no env variable. | UNTESTED | code |
| Minor | THEORETICAL | correctness | Any non-empty string sets the skip, so `DOTFILES_SKIP_PI_PACKAGES="0"` or `"false"` will skip the reconcile. | `if os.Getenv(skipEnv) != ""` in `cli/internal/cmd/pi.go`. | `tests/pi-packages.bats` (only tests `"1"`) | code |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Criteria met, minor negative-path gap on the env var value ("0" skips). |
| Verification       | A | Excellent mutation testing and fully reproducible json evidence. |
| Scope              | A | Diff matches proposal exactly; no creep. |
| Reliability        | B | Graceful degradation on missing tools, but theoretical gap on Linux CI integration env var. |
| Maintainability    | A | Reconcile logic in Go is clean and tests are robust against renames. |
| Handoff-readiness  | A | Spec is fully updated, features.json verified, lessons applied. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- Pass `DOTFILES_SKIP_PI_PACKAGES` to the `integration` job's `docker build` and `docker run` commands in `.github/workflows/ci.yml` so that it honors the filter when `pi` is eventually installed in the container.
- Consider strictly checking `os.Getenv(skipEnv) == "1"` or `"true"` in `cli/internal/cmd/pi.go` to avoid false positives with `"0"`.
