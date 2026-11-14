---
tags: [spec, verification, templates]
created: "2026-11-13"
id: "TOOL-039-forge-pr-draft-ready"
type: spec-verification
status: active
owner: manu
---

# Verification - TOOL-039-forge-pr-draft-ready

## Evidence

- [x] AC1 -> commit `d90ed3c4` / `TestGiteaCreatePRDraftSendsAPrefixedTitle`, `TestGiteaCreatePRDraftFailsWhenTheInstanceIgnoresThePrefix`, `TestGiteaCreatePRWithoutDraftKeepsTheTitle` (cli/internal/forgewrite/draft_test.go)
- [x] AC2 -> commit `d90ed3c4` / `TestGiteaReadyPRStripsThePrefix` (four spellings), `TestGiteaReadyPRFailsWhenTheTitleStillReadsAsDraft`, `TestGiteaReadyPRWritesNothingForAReadyPullRequest`
- [x] AC3 -> commits `d90ed3c4`, and the `isDraft` read-back after it / `TestGitHubReadyPRSendsTheMutationWithTheNodeID`, `TestGitHubReadyPRSkipsTheMutationForAReadyPullRequest`, `TestGitHubReadyPRFailsWhenTheAnswerIsNotReady`
- [x] AC4 -> commit `d90ed3c4` / `TestForgePRReadyMarksTheNumberedPullRequest`, `TestForgePRReadyTakesExactlyOnePositiveNumber`, `TestForgePRHelpNamesDraftAndReadyOnBothForges` (cli/internal/cmd/forge_write_test.go)

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean. `go test ./... -count=1`: every package ok.
- `golangci-lint run` and `GOOS=windows golangci-lint run` on `internal/forgewrite` and `internal/cmd`: 0 issues.
- Mutations: dropping the `draft: true` check after a Gitea create turns `TestGiteaCreatePRDraftFailsWhenTheInstanceIgnoresThePrefix` red; dropping the ready short-circuit on GitHub turns `TestGitHubReadyPRSkipsTheMutationForAReadyPullRequest` red.
- The instance in use answers `draft` on every pull request: Gitea 1.27.3, `personal/resume#808` read on 2026-11-14 had `"draft": false`.
- Manual smoke test on gitea.kubelab.live, run 2026-11-14 with a binary built from `f0aea7a6`, as the agents' account on `personal/resume` (owner's choice): `dotf forge pr create --draft` opened `personal/resume#811`, which the API answered as `"title": "WIP: chore: check dotf forge drafts", "draft": true`. `dotf forge pr ready 811` then answered `"title": "chore: check dotf forge drafts", "draft": false`. A second `pr ready 811` exited 0 and wrote nothing. `dotf forge` has no close verb, so the owner closes #811 and deletes its branch `chore/forge-draft-check` by hand.
- No regressions in the existing suite: yes. The one removed test, `TestGiteaCreatePRRefusesDraftBeforeAnyRequest`, asserted the refusal this spec replaces.

## Decisions made during implementation

- One PR rather than the two the decision comment on #3406 planned: the production change is about 150 lines, so splitting it would double the review for no smaller risk.
- The Gitea prefix dotf writes is `WIP: `, the first of Gitea's defaults. `pr ready` removes either default in any case, because a person may have retitled the pull request by hand.
- Out-of-scope work ticketed: #3477 (harness text and the `dotf forge pr ready` allow rule, owner-only paths).

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? no: nothing went wrong to learn from; the Gitea prefix behaviour is recorded in ADR-052 and the code
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-052-one-forge-write-surface-in-dotf.md
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: specific to dotf forge

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-039-forge-pr-draft-ready/` -> `specs/archive/TOOL-039-forge-pr-draft-ready/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
