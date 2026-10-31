---
tags: [spec, verification, templates]
created: "2026-09-23"
---

# Verification - GUARD-017-protection-as-code

## Evidence: PR-A (declare and detect)

Everything was run on 2026-09-23 in `dotfiles-wt-protection-as-code` (branch `feat/protection-as-code`, base `origin/main` `2c6af81`).

- [x] **AC1**, live: `dotf forge protection check` (dev build) returned `14 repositories: 9 ok, 0 drift, 5 declared state, 0 unanswerable`, exit 0, in 0.9 s.
  - The declaration was generated from the live GET responses by a scratch script that is not committed. A second, independent implementation, the Go normaliser in `check`, agrees with it on every field. That agreement is the evidence the generation is faithful.
  - Unit tests with a fake forge: `TestProtectionCheckReportsDrift`, `TestProtectionCheckProtectionRemovedIsDrift`, `TestProtectionCheckUnprotectedStates`, `TestProtectionCheckUnanswerable`.
  - CLI exit contract: `TestForgeProtectionCheckExitClean`, `…ExitOnDrift`, `…ExitWhenUnanswerable`, `…RepoFilter`.
- [x] **AC2**, live: `dotf doctor --verbose` prints `[branch-protection]` with `[ OK ] 9 repositories match their declaration`, plus 5 Skips that carry their declared reasons. `TestCheckBranchProtection` pins five mappings:
  - a confirmed state is a Skip;
  - drift is a FAIL naming repo and field;
  - unanswerable is a WARN and never a PASS;
  - unavailable is a Skip with its reason;
  - `gh` absent is a Skip and never a PASS.
- [x] **AC3**: `TestProtectionApprovalsRationalePinned` pins `required_approving_review_count = 0` and the rationale's sha256 (`ba5430c8…`), and it checks every repo that has a review requirement against the policy.
- [x] **AC4**: `TestProtectionSchemaRejectsAnIncompleteRepo` rejects six shapes:
  - neither an object nor a state;
  - a state without a reason;
  - both an object and a state;
  - an object missing a field;
  - a check without its source;
  - a key that is not a slug.

  `TestProtectionSchemaAcceptsEachDeclaredShape` covers both valid shapes.
- **Normalisation**, against real GET fixtures captured 2026-09-23 (`testdata/get-{dotfiles,iris,pollex}.json`, URLs stripped): the wrappers are unwrapped, an omitted block is null (iris has no status checks, pollex has no pull-request requirement), check order is ignored, a changed `app_id` is drift, and a live `restrictions` is drift.

## Test status: PR-A

- `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./... -count=1` is clean. `go test -race ./internal/forge/ ./internal/cmd/` is clean. `golangci-lint run ./...` at the pinned 2.12.2 reports `0 issues.`
- **Mutation battery.** 13 mutants, one per invocation, each under `systemd-run --user --scope -p MemoryMax=1500M -p MemorySwapMax=0` and restored with `git checkout` in a trap.
  - One survived at first: a normaliser that **hard-coded `enforce_admins: true`**, because every captured fixture has it true. `TestProtectionNormaliseReadsEveryFlag` now flips each of the 9 flags in the raw response and requires the normalised field to follow. The mutant, and a `lock_branch` variant, are now killed.
  - The final result is 13 of 13 killed by a test failure, none by a build error.
- **LOC.** PR-A adds 278 executable production lines, 142 declaration lines (import/type/const/var, measured with a `go/ast` classifier), 10 lines of help text and 74 braces.
- **No shell or PowerShell script was touched**, and **nothing was written to any forge**: every live call was a GET.

## Decisions made during implementation

- **The first declaration records the live state.** Drift detection therefore starts at zero, and PR-B's apply has nothing to converge except the deliberate change, `spec-gate` on dotfiles.
- **An `unavailable` repo is not queried.** Its 403 is already explained by the declaration, so a call would add latency and nothing else.
- **`resume` is not declared.** It is archived (`archived: true`), and the `unavailable` state is justified by the three non-archived private repos with specs (fae-brain, knowledge, openkm-brain). This corrects the D-6 note on #1625, which named `resume`.
- **Declaration path `forge/`**, the collaborate layer: GitHub-side state declared in git. This was the open question in `proposal.md`, and the owner has not yet answered; it is a default, and a rename if they choose otherwise.

## Evidence: PR-B (apply)

Run on 2026-09-25 in `dotfiles-wt-protection-apply` (branch `feat/protection-apply`, base `origin/main` `81940a8`).

- [x] **AC5**, against a stateful fake forge (`cli/internal/forge/apply_test.go`, 12 tests): a PUT replaces what the next GET returns, as GitHub's does.
  - `TestProtectionApplySendsCompleteBodyAndConverges`: the PUT body carries exactly the 11 keys the endpoint takes, `restrictions` as an explicit null; the result is re-read; the second run reports `unchanged` and writes nothing.
  - `TestProtectionApplyFailsWhenTheReReadDisagrees`: a write the forge accepts and does not apply is a FAIL naming the field.
  - `TestProtectionApplyRefusesUnreportedContext`, four ways: never reported, reported by another app, reported only outside the last 5 merged pull requests, reported only on a pull request that never merged. Nothing is written in any of them.
  - A context reported as a commit status counts (`review-attestation` is one). Contexts already required are not re-checked. `--dry-run` writes nothing and still runs the preflight. `required_signatures`, which PUT does not accept, goes to its own endpoint. Declared states are skipped without a call; apply never removes protection.
  - CLI exit contract: `TestForgeProtectionApply{DryRunPlansAndWritesNothing,UnchangedExitsClean,ExitsNonZeroWhenAReadFails}`.
- [x] **Mutation battery**: 9 mutants, one per run, each killed by a test failure: `restrictions` made omitempty, the re-read check disabled, `--dry-run` ignored, the preflight disabled, the source app ignored, unmerged pull requests counted, the 5-PR window unbounded, the signatures call dropped, an unprotected branch not counted as a change.
- [x] **Live, read-only**, before the declaration changed: `dotf forge protection apply --dry-run` over all 14 declared repositories reported `changed=0 (0 applied, 0 planned), 9 unchanged, 5 skipped, 0 refused, 0 failed`, exit 0. The declaration and the forge agree on every field of every protected repository.
- [x] **Live, read-only**, after declaring `spec-gate` on `mlorentedev/dotfiles`: the same command plans exactly one field, `required_status_checks.checks`, adding `spec-gate@15368`, and the preflight passes, because `spec-gate` reported on the recent merged pull requests. Exit 0; nothing was written.
- [x] **AC6**, owner-run after merge (done; measured below): `dotf forge protection apply --repo mlorentedev/dotfiles`, then `gh api repos/mlorentedev/dotfiles/branches/main/protection -q '.required_status_checks.contexts'` includes `spec-gate`, and a second `apply --dry-run` reports `changed=0`. Until then, `dotf forge protection check` and `dotf doctor` report that one field as drift, which is the declaration being ahead of the forge on purpose.

## Test status: PR-B

- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean. `go test -p 1 ./... -count=1`: 25 packages ok, exit 0. `golangci-lint run ./...` at the pinned 2.12.2: `0 issues.`
- **LOC.** About 300 executable production lines by a heuristic count (declarations, help text and brace-only lines excluded): `apply.go` 109, `preflight.go` 106, `cmd/forge.go` 87, of which 20 move existing code into `loadForgeDecl`. Excluded: 39 declaration lines, 21 lines of `--help` text, 17 braces.
- **No shell or PowerShell script was touched, and nothing was written to any forge.** Every live call was a GET.

## Evidence: stale-declaration refusal (2026-10-15)

On 2026-10-15 the owner ran `apply` and `check` from a main checkout that predated #2545: both reported `UNCHANGED`/`OK` while the live forge had `strict: true` and main declared false. After `git pull` the same commands reported `[APPLIED] ... strict: true -> false` and `check` 0 drift. Fixed here at the owner's request (fix, not ticket). Lesson 442.

- [x] `forge.CheckFresh` fetches origin and byte-compares the checkout's declaration with origin's default branch's. `TestCheckFresh` against real git histories (bare origin plus a second clone that lands a change): fresh; stale behind a landed change (names `origin/main`); behind on other files only (fresh); an unmerged local edit; the declaration absent from main; an origin with no default branch while a local `main` exists; an unreachable origin; `origin/HEAD` naming `trunk` wins over `origin/main`.
- [x] `check`, `apply` and `apply --dry-run` refuse a stale declaration without asking the forge: `TestForgeProtectionRefusesAStaleDeclaration`. `--unmerged` skips the gate: `TestForgeProtectionUnmergedSkipsTheFreshnessGate`.
- [x] Mutations, each killed by a test failure (`-count=1`): `CheckFresh` never reporting a difference; a local `main` fallback in the default-branch resolution (survived at first: the no-remote row failed at the fetch before reaching resolution, so the row now uses a reachable origin whose HEAD names a missing branch); the gate dropped from `loadForgeDecl`.
- `go vet` and `GOOS=windows go vet` on `internal/forge` and `internal/cmd`: clean. `go test ./...`: ok. `golangci-lint run` (pinned) on both packages: 0 issues.
- [x] After the #2557 review (PR-Agent, three findings): `forge.LandedDeclaration` compares without fetching, and `dotf doctor` WARNs when the declaration differs from origin's default branch as last fetched (`TestCheckBranchProtectionWarnsOnADeclarationThatHasNotLanded`: a stale checkout whose old declaration still matches the forge, which PASSed before). The sweep does not fetch, and an unanswerable comparison draws nothing there: the gate is `check`/`apply`. `apply --unmerged` refuses without `--dry-run` (`TestForgeProtectionApplyUnmergedNeedsDryRun`), so an unlanded declaration is never written. The fetch is bounded at 60 s. Mutations on the doctor WARN, the `--unmerged` guard and the comparison (always "same") are each killed.

## Review dispositions (2026-10-22, agy/gemini-3.8-flash-high, PASS WITH GAPS)

- Promotion candidates held placeholders: applied, answered below.
- `proposal.md` showed AC5 and AC6 unticked: applied, both ticked (checkbox state is outside the contract digest).
- PR-Agent on #2959: f6 piped `dotf` into `grep`, so a non-zero `dotf` exit with `changed=0` in its output still passed. Applied after the review: the output is captured first, so `dotf`'s exit status gates the check. The criterion f6 verifies is unchanged; only its command was tightened.
- f6 needs network and `gh` auth: declined. AC6 is defined as a live check against the forge, so an offline command could not prove it; f6 is run by hand, as the other live evidence in this file was.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-442-a-declaration-read-from-a-checkout-is-only-as-current-as-the-checkout.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the one policy decision, D-1 (no required approvals), is recorded in proposal.md and pinned by TestProtectionApprovalsRationalePinned
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: forge protection is declared once, in dotfiles, for every repository; no other project carries its own

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/GUARD-017-protection-as-code/` -> `specs/archive/GUARD-017-protection-as-code/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

## Evidence: AC6 and closing (2026-10-22)

Run from `chore/guard-017-ac6-close`, base `origin/main` `d9700c74`.

- [x] **AC6**, live: `gh api repos/mlorentedev/dotfiles/branches/main/protection -q '.required_status_checks.contexts'` returns `lint`, `lint-powershell`, `test`, `test-windows`, `review-attestation`, `cli-gate`, `spec-gate`, `changes`, `knowledge-gate` and `from-zero-gate`; `spec-gate` is at index 6.
- [x] `dotf forge protection apply --dry-run` over the whole declaration: `12 repositories: changed=0 (0 applied, 0 planned), 9 unchanged, 3 skipped, 0 refused, 0 failed`, exit 0.
- [x] `dotf forge protection check`: `12 repositories: 9 ok, 0 drift, 3 declared state, 0 unanswerable`, exit 0, `[OK] mlorentedev/dotfiles`. The drift that PR-B left on purpose is gone.
- [x] `features.json` gains f5 (AC4, `TestProtectionSchema*`) and f6 (AC6, the live check above). Each of f1-f6's verification commands exits 0.
- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean. `go test ./internal/forge/... ./internal/cmd/... ./internal/doctor/... -count=1`: ok. `golangci-lint run` at the pinned 2.12.2 on those packages: `0 issues.`
- The declaration names 12 repositories, not the 14 of PR-B: #2200 (`7b067099`) removed `fae-brain` and `openkm-brain` when they moved to Gitea. Both were `unavailable` state entries, so no protected repository left the declaration: PR-B skipped 5 and this run skips 3, while the protected repositories are 9 in both.
