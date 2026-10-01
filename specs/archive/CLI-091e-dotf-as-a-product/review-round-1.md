---
spec: "CLI-091e-dotf-as-a-product"
verdict: "FAIL"
reviewed_sha: "10116e859ea450951cc94a8e494a686934fc25af"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: CLI-091e-dotf-as-a-product (track E row E1), whole diff `0013ea5b8b45cec7ab950e91b93eb8f1defc0025...HEAD` (`10116e85`), not the delta since any earlier round.
**Sources**: `specs/CLI-091e-dotf-as-a-product/{proposal,tasks,verification,features}.json|md`; `git diff 0013ea5b...HEAD` (33 files); built binary `/tmp/dotf-review` (`cli/cmd/dotf`) and a normalized dump of `--help` for all 95 command paths; `git log`; base worktree `/tmp/cli091-base` (added and removed for the red-green run).

### Spec and task alignment

- The change does what the spec describes on the whole: the root/`spec`/`init`/`hooks`/`tools`/`secrets`/etc. help strings now describe behaviour, and `cli/README.md` is rewritten with `Install`, `Commands` and `Develop` sections. AC2 holds; every README claim I could check is true (see rubric rationale).
- **AC1 does not hold as written.** The `What` section says "none of it cites an internal id or the word 'twin'"; three lines of live help text still cite `DX-006`, an internal spec id in this repository (`specs/DX-007-orca-cli-bootstrap/`, `docs/adr/audit-007-cli-convergence-state.md`). The guard passes only because `DX` is missing from its hand-written prefix list, so the guard reports a tree as clean that is not clean.
- All `[x]` tasks map to diff evidence (test file, help rewrites, README rewrite). The lint/test claims in `tasks.md` and `verification.md` reproduce (see Verification rubric).
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` markers remain in the spec folder.
- Freshness: the normalised contract digests recorded in `review-request.json` still match the current files (`spec.ContractDigests`, checked from an in-package test that was removed after the run), so this review describes the current contract.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | spec-vs-code / guard coverage | `--help` still prints an internal id. `dotf orca` Example and `dotf orca tune-hooks` Short and Long all cite `DX-006` (and "lesson 111"). The `What` requirement — "Every piece of text `--help` prints … none of it cites an internal id" — is unmet; the guard passes only because `DX` is absent from `internalRef`'s hardcoded prefix list. The declared risk covers "a new prefix", and `DX` is not new: `specs/DX-007-*` exists and `DX-006` is referenced across the repo. | `/tmp/dotf-review orca --help` and `... orca tune-hooks --help` (lines: `dotf orca tune-hooks # Repair Orca's generated Copilot hooks (DX-006)`; `Repair Orca's generated Copilot hooks: raise timeoutSec and swap the slow POST (DX-006)`; `… "hook errored" (DX-006, lesson 111)`). Mutation: adding `\|DX` to the regex makes the guard red on exactly those 3 lines (edit reverted; tree clean). | UNTESTED — `TestHelpTextHasNoInternalReferences` is the only candidate test and it passes on the offending tree | code (`cli/internal/cmd/orca.go`) + tests (extend the prefix list; better, derive prefixes from `specs/*` / `docs/adr/*` so a missing entry cannot silently grant a pass) |
| Minor | REAL | verification evidence | `tasks.md` and `verification.md` claim "49 offending lines at the start" / "49 failures before the rewrite". The shipped regex finds **51** at the launcher-resolved base `0013ea5b`, reproducibly. | Base worktree at `0013ea5b` + `help_text_test.go` copied in: `go test ./internal/cmd/ -run TestHelpTextHasNoInternalReferences` printed 51 `mentions` lines on two runs (first hit: `"dotf" Long mentions "ADR-020"`). | UNTESTED (it is an evidence number, not a behaviour) | spec (`verification.md`, safe to edit; `tasks.md` is contract) |
| Minor | REAL | scope | `proposal.md` declares error messages out of scope ("They are a later row"), yet the diff edits a runtime error string: `spec init`'s WIP refusal loses " (#770)". The removal is not needed by AC1, which covers `--help` only. No test breaks because `spec_test.go:245` asserts `"10 active"`, `"limit is 10"`, `"--over-wip-limit"` only. | `cli/internal/cmd/spec.go:519-522` in the diff; `cli/internal/cmd/spec_test.go:245-260`; `#770` still present in `cli/internal/spec/wip.go:12` comments. | `TestSpecInitRefusesAtWipLimit` (message substrings) does not cover the removed text | code (revert) or spec (widen Out of scope) |
| Minor | THEORETICAL | guard completeness | The guard checks `Short`, `Long`, `Example` and `LocalFlags()` usages only. `--help` also prints `Use` (Usage line), `Aliases`, flag default values and `Deprecated`; none are checked. The issue pattern `#[0-9]{2,5}` misses 1-digit and 6+-digit issues. A bare `ADR`/`GUARD` with no number is a known accepted gap (`verification.md`). No current instance of a `Use`/alias id exists, so this is a latent hole, not an observed one. | code read of `cli/internal/cmd/help_text_test.go:23-31`; `grep 'Use:\s+"[^"]*(ADR|GUARD|CLI|DX|HARNESS)-[0-9]'` on `cli/internal/cmd/*.go` returned nothing. | UNTESTED | tests |
| Minor | SPECULATIVE | jargon | Internal jargon survives outside the id rule: `dotf agent auto --task "open a ticket for the bitacora"` (Example) and `spec archive` Long's "(a 00_meta/ path in the vault)". AC1's letter is "an internal id or 'twin'", so neither is in scope of the guard; surface only, do not gate. | normalized `--help` dump, `/tmp/allhelp.txt` lines 96 and 1596. | UNTESTED | code (optional) / spec (widen AC if intended) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | AC2 met and verified; AC1's stated behaviour unmet (an internal id is still printed), the guard's green is a false clean. |
| Verification       | C | `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...` (27 packages, 0 FAIL) and both `features.json` commands reproduce, but the baseline count is wrong (51 not 49) and "0 after" is only true for the guard's own pattern. |
| Scope              | C | A declared non-goal was crossed (a runtime error string edited) and two other specs' archives (`CLI-091b`, `TOOL-023`) ride in the same diff; each is small and documented in commit messages, but together they exceed "minor side-changes documented". |
| Reliability        | B | Help-text-only change plus one error-string edit; no runtime logic touched; the tree builds and the Windows leg vets. |
| Maintainability    | C | The guard is a hand-maintained prefix allow-list where one missing entry silently grants a pass — exactly the failure that occurred; the rest of the test is short, clear and table-less but adequate. |
| Handoff-readiness  | B | Spec updates shipped in-session (proposal/tasks/verification + `features.json`), promotion lines answered with reasons; one evidence number is wrong. |

### Verdict
FAIL

### Recommended next steps

Set routing: `proposal.md`, `tasks.md` and `features.json` are the **contract set** (a change invalidates this review); everything else is free to edit.

1. **Fix (code + tests)** — remove `DX-006` (and `lesson 111`) from the `dotf orca` / `dotf orca tune-hooks` help strings in `cli/internal/cmd/orca.go`; extend `internalRef` so the miss cannot recur. Prefer deriving the id prefixes from the tree (`specs/*` directory names, `docs/adr/adr-*`) over another hand list, or add a second named test that fails when a repo spec prefix is absent from the allow-list. This is the minimum action that flips the Blocker.
2. **Re-run the review after that fix.** The staleness check keys on the contract digests, so a code+tests-only fix leaves this `review.md` looking fresh to `dotf spec archive`. Do not archive on it — the verdict is about the state before the fix. Since a re-review is already required, correcting `tasks.md`'s "49" in the same round is acceptable and will be re-reviewed.
3. **Correct `verification.md`** (outside the contract set): 49 → 51, with the base commit named, or drop the number and name the command.
4. **Disposition the out-of-scope error-string edit** in `verification.md`: revert it, or record why `spec init`'s WIP message was treated as in-scope despite the proposal's Out-of-scope line.
5. **Optional follow-up ticket** for the guard-completeness gaps (`Use`/`Aliases`/defaults unchecked) and for widening the issue-number pattern.

### Unverified

- `golangci-lint run ./...` reported "0 issues" but emitted a config/cache warning naming an unrelated worktree (`../dotfiles-wt-lookup/...`), so that run is weaker evidence than the claim in `tasks.md`; not re-run in a clean cache. UNVERIFIED.
- `go install github.com/mlorentedev/dotfiles/cli/cmd/dotf@latest` was not executed (needs network). Indirect check only: the repo has 0 `cli/v*` tags and 85 plain `v*` tags, and a plain build prints `dotf version dev`, which is consistent with the README's pseudo-version note.
- The README's install one-liner was not fetched end-to-end; `scripts/install-dotf.sh` exists on both `main` and `origin/main`.
