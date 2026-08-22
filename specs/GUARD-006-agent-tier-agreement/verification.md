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
future reader can tell "everything resolved" from "nothing was looked at" — the distinction this
repo has now measured failing four separate ways.

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./... -count=1` -> clean
- `golangci-lint run` -> `0 issues.` at the `versions.conf` pin
- `gofmt -l internal/` -> clean apart from `internal/doctor/report.go`, which is **#1154**,
  pre-existing on main and untouched here
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
