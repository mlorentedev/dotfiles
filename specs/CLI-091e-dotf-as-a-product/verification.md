---
tags: [spec, verification, templates]
created: "2026-09-29"
---

# Verification - CLI-091e-dotf-as-a-product

## Evidence

- [x] AC1 -> test `TestHelpTextHasNoInternalReferences` (49 failures before the rewrite, 0 after)
- [x] AC2 -> `cli/README.md` rewritten; `features.json` f2 checks the three sections and runs `TestReadmeHasNoInternalReferences`, which holds the README to the same pattern as `--help` (added from #1850's review; mutation-checked with an `ADR-041` line)

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...`: green. `GOOS=windows go vet ./...`: green. golangci-lint 2.12.2: 0 issues.
- No test or script matched the old help strings (searched `tests/`, `scripts/`, the setup scripts).

## Decisions made during implementation

- The examples used real spec ids (`AI-001-ollama-public`, `HARNESS-071-reviewer-pool`), which the guard flags. They now use a neutral `FEAT-001-dark-mode`, which shows the id format without pointing at this repository's history.
- Error messages keep their ADR references for now (see Out of scope).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: the round-1 Blocker (a hand-kept prefix list missed `DX`, so the guard reported a dirty tree as clean) is the class lesson 322 already records for the CI path filter; a bare `GUARD` with no number escaping it is recorded in #1850's review triage
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a documentation and help-text change, with no design decision
- [x] New pattern candidate for `00_meta/patterns/`? no: one repository's CLI help

## Round 1 review and what changed after it

Round 1 (`review-round-1.md`, `nan/deepseek-v4-flash`) reviewed `10116e8`: **FAIL**. That branch was then rebased onto `8d7aca5` (#1883), which changed no file under this spec folder; the round-1 files were moved verbatim to `review-round-1.md` and `review-request-round-1.json` in `24ec7ca`.

| Finding | Disposition |
|---|---|
| Blocker: `dotf orca` and `dotf orca tune-hooks` help still printed `DX-006` and "lesson 111"; the guard passed because `DX` was missing from its hand-kept prefix list | Fixed. The three help strings in `cli/internal/cmd/orca.go` name behaviour only. The guard now reads the id prefixes from `specs/` and `specs/archive/` on top of the fixed list, fails if it finds none, and also matches `lesson N`. Red first: before the orca edit it failed on exactly the three lines the review named. |
| Minor: "49" is 51 at the review base `0013ea5b` | Accepted: the reviewer reproduced 51 twice with the shipped pattern at that base, and the "49" in `tasks.md` and in the Evidence line above has no recorded command behind it. `tasks.md` is part of the contract set and is left as reviewed; this line is the correction. |
| Minor: `spec init`'s WIP refusal lost " (#770)", an error message the proposal puts out of scope | Kept, and recorded here as the one exception. It shipped in #1850, already on `main`: the refusal was new in #1861, a day old, and no test or hook matched the removed text (`TestSpecInitRefusesAtWipLimit` asserts the count, the limit and the flag). Reverting it would put an issue number back into a message a user reads. |
| Minor: the guard reads `Short`, `Long`, `Example` and flag usage only | Deferred to #1891, together with the runtime output of `dotf orca tune-hooks` and `dotf doctor`, which still prints `DX-006`. Those strings are error and output messages, so they stay out of this spec's scope. |
| Minor: jargon ("bitacora", "00_meta/") | Declined: AC1 covers internal ids and "twin". Neither is an id, and the review marks it do-not-gate. |

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-091e-dotf-as-a-product/` -> `specs/archive/CLI-091e-dotf-as-a-product/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
