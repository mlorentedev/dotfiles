---
spec: "HARNESS-081-dependabot-labels-provisioned"
verdict: "PASS"
reviewed_sha: "c4a003c1b01e1f210144fdc86c7a9438bee62f9f"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-02"
---

## Adversarial review

**Scope**: HARNESS-081-dependabot-labels-provisioned (dotfiles#1222 — provision the labels a repo's dependabot.yml declares)
**Sources**: `specs/HARNESS-081-dependabot-labels-provisioned/{proposal,tasks,verification,features.json}`; `git diff 1a6a8b076952ab2c4796a17b28c7d1ff561e5baf...HEAD` (commits `31ccb79d` and `c4a003c1`); live verification of Go unit suites, bats suites, binary-level CLI execution against stubs, and mutation testing in this session.

### Spec and task alignment

- Diff maps 1:1 onto the proposal's "What" and AC1–AC5 without scope creep or test deletions.
- Contract digests in `review-request.json` match `dotf spec digest HARNESS-081-dependabot-labels-provisioned` exactly across all three contract files (`proposal.md`, `tasks.md`, `features.json`).
- No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` markers exist anywhere in the spec files.
- `features.json` entries remain in `pending` state with empty evidence (honoring the contract that only the harness marks `passing`). All 5 feature verification commands were executed directly this session and passed with exit 0.
- `tasks.md` accurately tracks progress: all implementation and closing check boxes are ticked except "PR opened", which correctly remains open prior to PR creation.
- Round 1 review findings have been completely dispositioned in `verification.md`:
  - The declared-side case sensitivity gap was resolved by adding `TestApplyDependabotLabelsMatchesDeclaredCaseInsensitively`.
  - The unreadable-repository degradation gap was resolved by adding `TestApplyDependabotLabelsSkipsWhenTheRepoCannotBeRead`.
  - The literal "404" substring check was tightened to "HTTP 404".
  - The `--limit 1000` observation was declined with clear operational rationale.

### Verification performed (this session, fresh)

- Clean builds, linting, and formatting:
  - `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./...` passed with exit 0.
  - `golangci-lint run ./internal/initrepo/... ./internal/cmd/...` reported 0 issues.
  - `shellcheck scripts/bitacora-rollout.sh` reported 0 issues.
- Test suites:
  - `go test -v ./internal/initrepo/...` executed fresh and passed (0 failures).
  - `bats tests/bitacora-rollout.bats tests/stub-real-pairing.bats` passed 22/22 tests under `/bin/bash` 3.2.
- Binary smoke tests:
  - Built `cmd/dotf` and executed `dotf init github` against stubbed `gh` environments:
    - Verified `--dry-run` correctly lists missing labels and makes no creation calls (`[DRY-RUN] would create dependabot labels on owner/name: ci`).
    - Verified failed `gh label create` exits 1 and names both the failed label and target repository.
    - Verified alternate extension `.github/dependabot.yaml` parses and detects declared labels properly.
- Mutation testing:
  - Mutated `have[strings.ToLower(name)]` to `have[name]` in `cli/internal/initrepo/labels.go:83`: `TestApplyDependabotLabelsMatchesDeclaredCaseInsensitively` failed immediately (caught). Reverted.
  - Mutated HTTP 404 check to accept any read error: `TestApplyDependabotLabelsSkipsWhenTheRepoCannotBeRead` failed immediately (caught). Reverted.
  - Mutated filename list to omit `"dependabot.yaml"`: observed that unit test suite passes because the mock stub in `labels_test.go` always returns 404 for `.yaml` (captured as Minor finding below). Reverted.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | test-coverage | Alternate `.github/dependabot.yaml` extension is supported in code (`for _, name := range []string{"dependabot.yml", "dependabot.yaml"}`), but `stubLabelsGh` in `labels_test.go` unconditionally returns HTTP 404 for `.yaml`. Mutating the list to drop `"dependabot.yaml"` survives the unit test suite. | Mutation run this session: removing `"dependabot.yaml"` from `labels.go:110` leaves `go test ./internal/initrepo/ -run TestApplyDependabotLabels` passing. Production binary was verified manually against a stubbed `.yaml` fixture and functioned as expected. | UNTESTED | tests (`cli/internal/initrepo/labels_test.go`) |
| Minor | SPECULATIVE | scale | `gh label list --limit 1000` silently omits labels beyond 1000; repos exceeding 1000 labels would attempt duplicate creations (which fail loudly). | Carried over from Round 1; author dispositioned as declined in `verification.md` (no repository in scope approaches 1000 labels). | UNTESTED | — (surface only; declined) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All five ACs verified across unit, binary, and bats suites; negative and degradation paths covered. |
| Verification       | A | Commands and outputs reproducible with zero external dependencies; mutations confirmed red-green. |
| Scope              | A | Diff matches proposal 1:1; zero creep; no test deletions (5 lines modified, 666 added). |
| Reliability        | A | Idempotent read-before-write proven; unreadable repository and label creation errors fail loudly. |
| Maintainability    | A | Functions under 50 lines with CC ≤ 10; golangci-lint clean; rationale documented in code. |
| Handoff-readiness  | A | Spec triad complete; previous review round dispositions tracked in verification.md; PR checklist kept accurate. |

### Verdict
PASS

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed and must NOT be edited prior to archiving (to preserve digest stability).

For the implementer outside the contract set:
1. **tests (optional follow-up)**: In a future test cleanup, extend `stubLabelsGh` to allow serving `.github/dependabot.yaml` fixtures so the fallback extension branch has direct unit test regression coverage.
2. **archive**: `dotf spec archive HARNESS-081-dependabot-labels-provisioned` is **advisable** and ready to proceed once the PR is opened and merged.
