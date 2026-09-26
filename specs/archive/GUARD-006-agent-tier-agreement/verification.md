---
tags: [spec, verification, templates]
created: "2026-08-22"
---

# Verification - GUARD-006-agent-tier-agreement

## Evidence

All 7 `features.json` verifiers executed in this session and passing; each propagates the runner's
exit status and pins its test by unique name rather than by position (lesson 217).

Beyond the fixtures, the check was run against the **real tree** with a binary built from this
branch, which is the part a fixture cannot establish:

```console
$ DOTFILES_DIR=<this worktree> dotf doctor --verbose
[Routing registry]
  [ OK ] harness/model-map.json is present, parses, and satisfies its schema (4 pools, 5 harnesses)
  ...
  [ OK ] every declared agent tier resolves for its deploy targets (1 checked)
```

One pair checked, which is correct: `agents.deploy` holds one target (`claude`) and
`harness/agents/` holds one record (`curator`, `model: top`). The count is in the message so a
reader of `--verbose` output can tell how many pairs "everything resolved" covers. It does not
separate that from "nothing was looked at": at zero pairs the check prints nothing, by design,
since silence is not a pass line (corrected after the retroactive review, round 2).

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./... -count=1` -> clean
- `golangci-lint run` -> `0 issues.` at the `versions.conf` pin
- `gofmt -l internal/` -> clean. The landing commit also removed the trailing blank line gofmt
  flagged in `internal/doctor/report.go` (#1154). An earlier version of this line called that file
  untouched, which was wrong (corrected after the retroactive review, round 2).
- The doctor package's own suite passes unchanged, which is the evidence that wiring a new check
  into `checkModelMap` did not disturb the existing ones

## Decisions made during implementation

- **Doctor, not `compile-harness.sh --check`.** The `--check` mode runs in the CI `lint` job, which
  installs no Go and has no `dotf`; resolving tiers there would report drift on a perfectly good
  record purely because the machine lacks the resolver. That conflates a property of the deploy
  ENVIRONMENT with a property of the committed RECORD. Doctor already loads this registry and runs
  where `dotf` exists by definition.
- **Two narrowings, both to avoid the failure mode that kills a diagnostic.** Scoped to
  `agents.deploy` targets, and honouring each record's `targets:` list. A tier gap for a harness
  nothing deploys to is a real question (#1170) but it is not drift; a persona scoped to one harness
  judged against every other is a false positive on correct data. Either would train the reader to
  skip the line, and then the true positive goes with it.
- **`recordTargets` defaults an ABSENT list to EVERY harness**, matching the render. That direction
  is the sharp edge — inverted, every scoped persona fails against every other harness — so it has
  its own table test rather than only being exercised through the check.
- **Silence on inputs it does not own.** An absent manifest or record dir is
  `checkCompileHarnessDrift`'s diagnosis. Two failures for one cause makes both worth less.
- **No `--fix`.** The repair is either "declare a tier" or "change the record", and which is right
  is a judgement about intent, not something to automate.

## Retroactive review (2026-09-25, W1.4 of #1625)

Reviewed from a detached worktree at the landing commit `e54263c` (#1174), with that commit's own
`dotf` first on `PATH`, so the reviewer's scope was `e54263c^...HEAD` and not everything merged
since. All 7 `features.json` verifiers passed at the landing commit before either round.

**Round 1, `nan/deepseek-v4-flash`, FAIL.** Transcript kept outside the repo
(`~/.local/state/dotf/review-transcripts/GUARD-006-r1.jsonl`). Dispositions:

| Finding | Disposition |
|---|---|
| Major: `recordTargets` matched entries exactly; the render matches the harness name anywhere on the first `targets:` line, so a quoted entry hid drift and a block-style list raised a false FAIL | **Applied** in `d85d951`: the check uses the render's rule, and `TestRecordTargetsAgreesWithTheRender` runs the real `skill_targets_agent` against it. The render's rule itself is #1733 (HARNESS-159). |
| Major: AC1's count had no test | **Applied** in `d85d951`: `(1 checked)` and `(2 checked)` asserted |
| Minor: the unparseable-manifest branch had no test | **Applied** in `d85d951` |
| Minor: `checkAgentTiersResolve` was ~70 lines, CC ≈ 14 | **Applied** in `d85d951`: split into three functions |
| Minor: `copilot` contains `pi` in the render's substring match | **Ticketed**: #1733, together with the block-style defect |
| Minor: the unparseable manifest is a WARN where the proposal says the check stays silent | **Declined**: AC7 requires no FAIL, and a WARN is not one. Saying the check did not run is what keeps silence from reading as a pass; `TestAgentTiersMissingInputsAreNotFailures` now pins both halves. |
| Minor: the landing commit removed a trailing blank line in `report.go` | **Declined**: it was gofmt debt (#1154) paid in passing; recorded under Test status above |

**Round 2, `nan/deepseek-v4-flash`, PASS WITH GAPS**, at `519cf34` (`e54263c` plus the fix).
`review.md` and `review-request.json` are this round's. Two earlier launches produced no review
and are not rounds: the random draw picked `agy/gemini-3.1-pro-high`, whose key the landing
commit's registry does not know, and `nan/mimo-v2.5` was cut off by the provider's concurrency
limit (HTTP 429). Dispositions:

| Finding | Disposition |
|---|---|
| Major THEORETICAL: the `model:` reader disagrees with the render's `skill_field` on CRLF, `model : top` and an indented `---` | **Ticketed**: #1740 (GUARD-020). Fixing it changes the reader, so it gets its own reviewed change rather than landing unreviewed after this verdict. |
| Major THEORETICAL: first-wins on a repeated key had no test | **Applied** in `62806cb`; the last-wins mutant now fails |
| Minor THEORETICAL: the indent guard and the `record_dir` default had no test | **Applied** in `62806cb`; both mutants now fail |
| Minor THEORETICAL: nothing asserted that a FAIL suppresses the pass line | **Applied** in `62806cb`; the mutant now fails |
| Minor REAL: the gofmt sentence above was stale | **Applied**: corrected above |
| Minor REAL: the count does not cover the zero case | **Applied** to the wording above; **declined** as a code change, because at zero pairs silence is deliberate and is not a pass line |

## Promotion candidates

- [ ] Lesson for `docs/lessons/`? no - the transferable point (a diagnostic that fires on correct
      data gets skipped, taking the true positive with it) is already recorded in the check's own
      doc comment, which is where someone changing it will read.
- [ ] ADR-worthy? no - ADR-035 already established the registry and its doctor-check precedent.
- [ ] Vault pattern? no - single-project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/GUARD-006-agent-tier-agreement/`
- [ ] Bitacora #1164 closed with the PR link
- [ ] `/adversarial-review GUARD-006-agent-tier-agreement` run and PASSing
