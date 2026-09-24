---
tags: [spec, verification, templates]
created: "2026-09-23"
---

# Verification - SDD-042-content-bound-review

## Evidence

All commands were run on 2026-09-23 in `dotfiles-wt-squash-aware-staleness` (branch `feat/squash-aware-staleness`, based on `origin/main` `8a68a49`). Every `-run` criterion went through the §5 `gorun` guard, which fails unless at least one test ran.

- [x] **AC1**: `gorun ./internal/spec/... 'Stale|Squash|Rebase'` → `tests_run=17`. The new real-git fixtures are `TestStaleSquashLandingIsAccepted`, `TestStaleRebaseLandingIsAccepted`, `TestStaleMergeLandingIsAccepted` and `TestStaleContractEditRefused`.
  - Each fixture launches a review at a branch head, applies the review's own "tick the boxes" finding, lands on `main`, and then (squash and rebase) runs `branch -D` + `reflog expire` + `gc --prune=now`. It asserts the reviewed commit object is really gone, because without that it would prove nothing about #1566.
  - **Red before the fix**, reproducing the original defects exactly:
    - squash and rebase: `reviewed_sha … is not a commit in this history (rewritten by a rebase?)` (#1566)
    - merge: `features.json, tasks.md changed after reviewed_sha`, which was only the ticked boxes (#998 part 2)
- [x] **AC2**: `gorun ./internal/spec/... 'TestContractDigest'` → `tests_run=7`. It covers the checkbox fold at any depth and for `*` bullets, `[x]` in prose staying significant, reworded criterion text changing the digest, the CRLF fold, `state`/`evidence` in `features.json` being folded while `behavior` is not, malformed JSON still digesting by its bytes, and an absent file → `""`.
- [x] **AC3**: `gorun ./internal/spec/... 'TestStaleLegacyAbsentObjectNamesTheCause'` → `tests_run=1`. A legacy request (no `contract_digests`) plus a pruned object is refused with "not in this clone" and "contract digests", and never with "changed after". `TestGitStalenessUnresolvableShaIsStale` was updated deliberately: it now forbids the old `rebase?` guess.
- [x] **AC4**:
  - `gorun ./internal/spec/... 'TestArchiveBypassRecorded'` → `tests_run=6`. It covers: a reason is required (the spec does not move), the record of what the gate would have refused, the "nothing" case, both flags, no `proposal.md`, and a proposal with no frontmatter.
  - `gorun ./internal/cmd/... 'TestArchiveBypassRecordedViaCLI'` → `tests_run=1`, end to end through `dotf spec archive --force-with-drafts [--reason]`.
  - `TestArchiveForceWithoutReviewOverridesFail` now also asserts that the record says it overrode a FAIL.
- [x] **AC5**: `gorun ./internal/spec/... 'TestArchiveRefusalsNameNoBypassFlag'` → `tests_run=1`.
  - The test parses every string literal in the package (the one allowed site is `checkBypassRequest`) and also checks 5 real refusals by behaviour.
  - Red before the rewrite: 16 violations, 12 in the source and 4 by behaviour.
  - Three older tests encoded the previous contract ("the refusal names every escape"). They were updated deliberately to assert that the flag is absent.
- [x] **AC6**: the review was launched with this branch's dev build (`go build ./cmd/dotf`). Its `review-request.json` records `contract_digests`, and the reviewer verified them key for key against a recomputation from disk. This spec's archive, run with the same build, is decided by those digests.

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean. `go test ./... -count=1`: every package ok. `golangci-lint run ./...` at the pinned 2.12.2: `0 issues.`
- `bats tests/guard-review-verdict-honours-reality.bats tests/skills-pipeline.bats tests/reviewer-pool.bats tests/agents-md.bats`: 52/52. These read the re-rendered skills.
- `bash scripts/compile-harness.sh --check`: `OK: no harness drift` after the `--refresh`. The refresh touched only the two skills.
- **Mutation battery**: 12 mutants, one per invocation, each under `systemd-run --user --scope -p MemoryMax=1500M -p MemorySwapMax=0` and restored with `git checkout` in a trap. All 12 were **killed by a test failure**.
  - Two first attempts were build errors: an unused `bytes` import, and unused `if` variables. Per lesson 284 those prove nothing, so they were re-run as compiling mutations and killed.
  - The mutants: no CRLF fold, no checkbox fold, no harness-field fold, a fold too wide (`behavior`), a checkbox fold that hits prose, a digest mismatch ignored, the digest path never taken, the launcher recording no digests, `--reason` not required, the bypass not recorded, the gate refusal not captured, no double-quote unescape.
- **No shell or PowerShell script was touched.** The change is Go, the spec, and the skill render.

## Decisions made during implementation

- **Digests are taken from disk at launch, not from `HEAD`.** They record what the reviewer actually read. Comparing against disk at archive time also keeps the uncommitted-edit bypass closed without a separate `git status` check.
- **Normalisation is deliberately narrow.** It folds line endings, list checkbox markers only (a bracketed `x` in prose stays significant), and `state`/`evidence` in `features.json`. A mutant that also folded `behavior` was killed, so the width is pinned in both directions.
- **A bypassed check still runs.** That way the record says *what* was overridden (for example "verdict FAIL" or "no review.md"), not only which flag was typed. This answers #998's point that `--force-without-review` is a false name when what it skips is freshness.
- **`frontmatterFields` now honours YAML quote escapes** (`\"` and `\\` in double quotes, `''` in single quotes). The bypass reason is free text, and without the fix a reason containing a quote read back truncated at it. Found by `TestArchiveBypassRecordedWhatTheGateWouldHaveRefused` going red.
- **A FAIL refusal no longer offers a waiver.** It now says a FAIL is resolved by its findings, which matches #1625 W1.4 rule 3 (a FAIL is never waived). `review: waived` is still honoured by the gate; it is simply not advertised to a FAIL.
- **Skill SSOT.** `harness/skills/{spec,adversarial-review}/SKILL.md` are generated (`generated_from: 00_meta/skills/…`). The edit went into the vault (knowledge `2db025b6`, direct to `master`), and the repo copies are the `compile-harness.sh --refresh` render.
- **Merge order.** This branch is based on `main`. It will conflict with #1630 in `frontmatterFields`: #1630 adds `yamlCommentStart` to the unquoted branch, and this change edits the quoted branch. Resolve by keeping both.
- **Out of scope, already ticketed:** #1154. `gofmt -l` lists 4 files on `main` (`internal/orca/orca.go`, …) and no linter enforces formatting. Re-measured and commented there.

## Review dispositions

The reviewer was `nan/glm5.3-flash`, drawn from the pool by this branch's launcher. It reviewed `12d3755` in about 15 minutes and returned **PASS**: no Blockers, 5 Minors, 1 Question.

This review is substantive. It re-ran the build, vet, the full suite, lint and the 52 bats, and re-counted `tests_run=17`. It applied **4 compiling mutants of its own**, all killed, and it probed the normalisation with inputs of its own.

| # | Finding | Disposition | Reason |
|---|---|---|---|
| 1 | Ordered-list checkboxes (`1. [ ]`) are not folded, so ticking one reads as drift | **apply** | Measured: 9 ordered-list checkboxes across `specs/`, so this is reachable, not theoretical. `listCheckbox` now also matches `\d+[.)]`. `TestContractDigestIgnoresOrderedListTicks` went red first, and it also pins that rewording an ordered task still changes the digest. |
| 2 | Widening `contractFiles` (W3.6) would stale every digest-carrying review, and the message would misdescribe why | **defer to W3.6** (#1153) | This is a design decision for that spec: the policy for a missing key, and a message along the lines of "no recorded digest for this file". Carried into #1153 verbatim. |
| 3 | The FAIL refusal says "not overridden", yet a recorded `--force-without-review` does override it | **apply** | The message is now "a FAIL is resolved by its findings: apply them in a follow-up, then re-review". It still names no bypass flag (AC5 test). |
| 4 | The `unquoteScalar` comment shows a typographic quote | **apply** | Root cause: **gofmt** (Go ≥ 1.19) rewrites two apostrophes in a doc comment into `”`. The comment now describes the escape in words. |
| 5 | `review-request.json` is untracked | **apply** | Committed together with `review.md`. |
| Q | `--reason` without a `--force-*` flag is accepted silently | **skip** | A stray `--reason` grants nothing and records nothing, so rejecting it would add a refusal with no safety value. |

The fixes for 1, 3 and 4 are code and comment changes made **after** the review (`fix(spec): fold ordered-list checkboxes too…`). Freshness is decided by the contract files, which none of the three touch.

## Promotion candidates

- [x] Lesson for `docs/lessons/`? **no** new one. Wave 1's lesson is lesson-287 (#1630). The build-error mutants are lesson 284's rule, applied.
- [x] ADR-worthy decision? **no**. The contract change is recorded in this spec and in `--help`.
- [x] New pattern candidate? **no**.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/SDD-042-content-bound-review/` -> `specs/archive/SDD-042-content-bound-review/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (none needed)
