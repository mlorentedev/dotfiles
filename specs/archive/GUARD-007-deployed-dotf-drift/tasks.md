---
tags: [spec, tasks, templates]
created: "2026-09-01"
---

# Tasks - GUARD-007-deployed-dotf-drift

> TDD order. One task = one focused commit. Tick as you go.
> `[P]` — no dependency on another unchecked task. `[AC<n>]` — satisfies acceptance criterion #n.

## Setup

- [x] Branch created from main: `feat/detect-deployed-dotf-drift` (base `6e0d180`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — all three resolved in place

## Implementation

- [x] [AC1] Stamp `-X main.commit={{ .FullCommit }}` in `cli/.goreleaser.yaml`; add `var commit` to `cmd/dotf/main.go`
- [x] [AC1] Thread the commit through `cmd.New(version, commit)` → `newVersionCmd`; add the `--commit` flag
- [x] [AC2] Pin the installer contract: default output stays one line (`TestVersionDefaultOutputStaysSingleLineForTheInstallers`), `--commit` prints the bare value
- [x] [P] [AC6] Expose `RepoDir` on `doctor.Config` — `loadConfig` already computed it and threw it away
- [x] [AC3] [AC5] [AC6] Write the table test covering all eight states, then `checkDotfProvenance` to satisfy it
- [x] [AC4] Harden the pathspec to `:(top)cli` after measuring the 4-from-root / 0-from-subdir split; pin it with a fail-first test
- [x] Register the check in `doctor.go` after `checkAgentSkillsMigrated`

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] Every acceptance criterion has a `features.json` entry with a non-vacuous verification command — **all six executed, all six exit 0**
- [x] Type checks pass (`go build`, `go vet`, `GOOS=windows go vet`)
- [x] Lint passes (`golangci-lint run` → 0 issues, v2.12.2 = the pin)
- [x] No unrelated changes in the diff — no `.sh`/`.ps1` touched
- [x] `verification.md` filled in, including the fail-first transcript
- [ ] PR opened referencing this spec folder

## Machine-readable features

`features.json` holds six entries, one per acceptance criterion. `state` is `pending` on every one: the agent cannot write `passing` — only the harness may, after running `verification` and capturing exit 0.

## Notes for the reviewer

Two things are worth an adversarial look rather than a skim:

1. **`:(top)cli` is the whole check's correctness.** Without it the count reads 0 from any CWD that is not the repo root, and 0 is the clean answer — a guard that fails silently in exactly the way #1158 describes. The fail-first transcript in `verification.md` shows it caught.
2. **The pre-stamp branch treats a non-zero exit as the answer, not as an error.** If that reads as swallowing a failure, the counter-argument is that the state is real and live on this machine right now, and any other handling reports "cannot probe" where the truth is "this binary predates the stamp".
