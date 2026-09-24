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

Pending.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/GUARD-017-protection-as-code/` -> `specs/archive/GUARD-017-protection-as-code/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
