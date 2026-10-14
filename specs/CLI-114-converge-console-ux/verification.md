---
tags: [spec, verification, templates]
created: "2026-10-14"
---

# Verification - CLI-114-converge-console-ux

## Evidence

PR 1 (macOS arm64, 2026-10-14):

- [x] AC1 -> `TestColorEnabled` (7 cases: a terminal, no TERM, a pipe, NO_COLOR, NO_COLOR=false, an empty NO_COLOR, TERM=dumb), `TestIsTerminal_NeverForANonFile`, `TestPainter`, `TestDuration`. All failed to compile before `internal/console` existed
- [x] AC2 -> `TestReport_Golden` (plain, plain-verbose-clean, color, color-verbose-clean). Its golden files were committed (`01dcfa7a`) before doctor moved onto `console` (`6df8d647`) and stayed byte-identical after it
- [x] AC3 -> `TestConvergeConsole_Colour`. The plain golden was captured from the unchanged `printConvergeReport` (`de3bbeae`) and is byte-identical after colour. The coloured output, with its SGR codes removed, equals the plain output. `TestConvergeConsole_ARunOnATerminalIsColoured` covers a plan; `TestConvergeConsole_AnAppliedRunColoursTheTerminalNotTheTranscript` covers an apply: the terminal gets the codes and the transcript none
- Mutations, each caught:
  - `runConverge` printing with a zero `Painter` -> `ARunOnATerminalIsColoured` FAIL
  - colour decided on the transcript tee, `convergePainter(out)` -> `AnAppliedRunColoursTheTerminalNotTheTranscript` FAIL
  - the tag printed unpainted -> `TestConvergeConsole_Colour` FAIL
- [ ] AC4, AC5, AC6 -> PR 2 and PR 3
- [ ] AC7 -> every PR (`go test ./...` unchanged elsewhere)

## Test status

- PR 1: `go build`, `go vet`, `GOOS=windows go vet`, `go test ./...` -> exit 0, every package ok; `golangci-lint run` 2.12.2 after `cache clean` -> 0 issues; bats over the 7 files that reference converge's output (`git grep -l -E 'dotf converge|converge on |converge plan' -- 'tests/*.bats'`) -> 1..252, 252 ok
- Manual smoke test (the branch built into `$TMPDIR`, run from the Mac's checkout, `converge --plan`, a dry run): piped -> the tags bare, the same text as before; under `script -q /dev/null` (a pty) -> `\x1b[36m[SKIP]`, `\x1b[35m[CHANGE]`, `\x1b[32m[ OK ]` with the columns aligned; `NO_COLOR=1` under the pty -> bare again

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

-
-

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-114-converge-console-ux/` -> `specs/archive/CLI-114-converge-console-ux/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
