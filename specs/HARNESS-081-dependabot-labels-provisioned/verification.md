---
tags: [spec, verification, templates]
created: "2026-11-02"
id: "HARNESS-081-dependabot-labels-provisioned"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-081-dependabot-labels-provisioned

## Evidence

- [x] AC1 -> `TestApplyDependabotLabelsCreatesOnlyTheMissingOnes`, `TestApplyDependabotLabelsSecondRunCreatesNothing`, `TestApplyDependabotLabelsMatchesDeclaredCaseInsensitively` (`cli/internal/initrepo/labels_test.go`)
- [x] AC2 -> `TestApplyDependabotLabelsDryRunCreatesNone`
- [x] AC3 -> `TestApplyDependabotLabelsWithoutDeclarationsDoesNothing` (both the missing file and the no-labels case); an unreadable repository is a skip, not "none declared": `TestApplyDependabotLabelsSkipsWhenTheRepoCannotBeRead` (non-404 contents read, failed `label list`), and the rollout fails a skip (`tests/bitacora-rollout.bats`, "a failing or skipped dotf init github fails the rollout")
- [x] AC4 -> `TestApplyDependabotLabelsFailsLoudlyWhenACreateFails`; `newInitGithubCmd` returns the error, so the command exits non-zero
- [x] AC5 -> `tests/bitacora-rollout.bats`, the five tests under "GitHub repo defaults via dotf"

## Test status

- Go: `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` all ok; `golangci-lint run` 0 issues (pinned version).
- Shell: `shellcheck scripts/bitacora-rollout.sh` clean; `bats tests/bitacora-rollout.bats tests/stub-real-pairing.bats` 22/22 under macOS `/bin/bash` 3.2 (the script's shebang).
- Manual smoke test: none against a live repository. The rollout is the owner's to run (`--check` first) with `BITACORA_PAT`; the stubs pin the gh and dotf calls it makes.
- No regressions in existing test suite: yes.

## Decisions made during implementation

- The rollout calls `dotf init github` instead of duplicating the gh calls, so the defaults have one owner and the script grows by one function, not by its logic.
- A `[WARN]` skip from dotf fails the repo in the rollout. At a terminal a skip is right; in a 27-repo rollout nobody reads it, and an unprovisioned label is the silent failure the issue is about.
- Labels read from the target repository's own `dependabot.yml` at run time, not from a list in dotf, so there is no second list to drift from the declaration.

## Review round 1 dispositions (FAIL, nan/glm5.3-flash, 31ccb79d)

- Major, declared-side case match untested: applied. `TestApplyDependabotLabelsMatchesDeclaredCaseInsensitively`; the reviewer's mutation (`have[name]`) now fails the suite.
- Major, unreadable-repo degradation untested: applied. `TestApplyDependabotLabelsSkipsWhenTheRepoCannotBeRead`; the mutation that treats every read error as a 404 now fails the suite.
- Minor, bare `404` substring match: applied. The match is now `HTTP 404`.
- Minor, `--limit 1000` on `label list`: declined. No repository is near 1000 labels, and a label past it would be a duplicate create, which fails loudly.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the one finding (Dependabot never creates a custom label) is the issue's premise and is recorded in the code comment and the proposal; nothing new was learned building it.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a new default inside an existing command, decided on #1222.
- [x] New pattern candidate for `00_meta/patterns/`? no: GitHub- and Dependabot-specific, one project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-081-dependabot-labels-provisioned/` -> `specs/archive/HARNESS-081-dependabot-labels-provisioned/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
