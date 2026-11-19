---
spec: "HARNESS-201-force-with-lease-own-branch"
verdict: "FAIL"
reviewed_sha: "71fa28e3fe732afe47b0ffb1bbcc40ba75c9b169"
reviewer: "agy/gemini-4-hydrogen-high"
date: "2026-11-19"
---

## Adversarial review

**Scope**: HARNESS-201-force-with-lease-own-branch (`git diff 31360ae33a738ab309af04a302cb8c71084d3e4d...HEAD`, 9 commits on branch, 20 files)
**Sources**: `specs/HARNESS-201-force-with-lease-own-branch/{proposal,tasks,verification,features}.*`, `docs/adr/adr-065-agents-may-force-with-lease-their-own-branch.md`, `docs/lessons/lesson-614-narrowing-a-deny-rule-in-a-merged-list-retires-nothing-without-a-record.md`, `ai/claude/settings.json`, `harness/manifest.json`, `tests/claude-settings-template.bats`, `cli/internal/{harness,cmd,claude,deploy}`

### Spec and task alignment
- `proposal.md`, `tasks.md`, and `features.json` digests match `review-request.json` via `dotf spec digest HARNESS-201-force-with-lease-own-branch`. No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in spec, ADR, or lesson files.
- `tasks.md` ticks all implementation items and leaves only PR creation unchecked; `features.json` marks `f1`–`f7` as `passing`.
- Round 1 fixes verified live: compiled `dotf.exe harness push-guard` now writes `push-guard: ...` to stderr with exit code `2` on refused pushes, `TestGuardCommand_ReviewBypasses` and `TestPushGuard_MsysDriveDirectoryAgainstARealRepository` pass, `go vet ./...` exits 0, `bats tests/claude-settings-template.bats` passes `33/33`, and mutating the `main`/`master` backstop in `cli/internal/harness/pushguard.go` kills `TestGuardCommand_Table/master_with_default_unresolved`.
- Remaining gaps against AC1/What: four REAL bypass families allow `git push --force origin main`, `npm run land:push`, or mismatched-directory lease pushes to exit `0` from the compiled `dotf.exe` binary while also evading the `ai/claude/settings.json` prefix globs.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | lexer / heredocs | `stripHeredocs` (`cli/internal/harness/pushguard_lex.go:253-326`) still swallows real `git push --force` commands in four cases: (1) `inQuotes(lines[i][:m[0]])` checks single-line prefix only, so a multi-line quoted `<<EOF` starts a phantom heredoc; (2) `heredocStart` only allows `[A-Za-z_][A-Za-z0-9_]*` unquoted delimiters, truncating `<<END-OF-MSG` at `-` so `END-OF-MSG` never terminates it; (3) `# <<EOF` comments trigger heredoc stripping; (4) `shellFeeder` requires `(\s|$)` after the shell name, missing `bash<<EOF`, `cat <<EOF \| bash; ...`, and `cat <<EOF \| (bash)`. | Reproduced with compiled `dotf.exe harness push-guard` (`EXIT=0`, empty stderr) on all 6 inputs and confirmed execution in Git Bash (`EXECUTED_1`–`EXECUTED_3`). | UNTESTED (`TestGuardCommand_Heredocs` and `TestGuardCommand_ReviewBypasses` cover single-line quotes and space/EOL-delimited `shellFeeder` only) | code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | guard / wrappers & dirs | `simple` (`cli/internal/harness/pushguard.go:129`) matches `wrappers[t]` on raw `cmd[0].text` instead of `programName(t)`, so `/usr/bin/env git push --force origin main` exits `0`. In addition, repo-directory tracking is bypassed by `env -C <other-repo>`, `GIT_DIR=<other-repo>/.git bash -c '...'` (`byProgram` does not forward `envDir` to `shellString`/`nested`), `export GIT_DIR=<other-repo>/.git && ...`, and subshell `(cd <repoA>) && git push ...` (`splitCommands` flattens `(...)` so `g.chdir` mutates parent `g.dir`). | Reproduced with compiled `dotf.exe harness push-guard` (`EXIT=0`) for `/usr/bin/env git push --force origin main` and against real temp repos `repoA` (`feat/x`) / `repoB` (`main`) for `env -C`, `GIT_DIR=... bash -c`, `export GIT_DIR=...`, and `(cd ...) && ...`. | UNTESTED (`TestGuardCommand_ReviewBypasses` tests bare `env`/`sudo`/`command`/`exec` and direct `env GIT_DIR=... git` only) | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | guard / eval & pwsh & lexer | (1) `byProgram` `case "eval":` (`cli/internal/harness/pushguard.go:202-203`) joins args verbatim without stripping leading `--`, so `eval -- "git push --force origin main"` inspects `"--"` as the program and exits `0`. (2) `shellString` (`:242`) reads only `args[j].text` after `-Command`, so `pwsh -Command git push --force origin main` on `PowerShell` inspects `"git"` alone and exits `0`. (3) Lexer single-quote branch (`pushguard_lex.go:85-92`) does not handle `$'...'` ANSI-C escapes (`\'`), leaving `ok=false`, and `run` (`pushguard.go:98`) only fails closed when `"git"` and `"push"` appear, letting `echo $'hello\'' ; npm run land:push` exit `0`. | Reproduced with compiled `dotf.exe harness push-guard` (`EXIT=0`, empty stderr) and verified execution in Git Bash (`EXECUTED_4`, `EXECUTED_5`) and `pwsh -NoProfile -Command` (`EXECUTED_6`). | UNTESTED | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`) |
| Minor | REAL | maintainability | Five functions in `pushguard.go` and `pushguard_lex.go` exceed `AGENTS.md`'s `CC < 10` target (`golangci-lint --no-config --enable=cyclop`): `escapeOrSubst` (14), `refuseForm` (13), `inQuotes` (12), `addLong` (11), `unquoted` (11). (All are `<= 14`, within the rubric's `<= 15` solid bar.) | Observed via `golangci-lint run --no-config --enable=cyclop ./internal/harness`. | UNTESTED (static lint; `cli/.golangci.yml` excludes `internal/harness`) | code (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_lex.go`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All 7 AC test suites pass and Round 1 findings are fixed, but three REAL Major bypass families allow `--force`, `land:push`, and wrong-repo lease pushes with exit `0`. |
| Verification       | B | Package Go tests (`harness`, `claude`, `deploy`, `cmd`), mutation test on `main`/`master` backstop, `go vet ./...`, and `bats` (`33/33`) pass, but negative tests miss the four bypass families above. |
| Scope              | A | Diff is strictly scoped to HARNESS-201; contract digests match and no `[AGENT-DRAFT]` tags remain. |
| Reliability        | C | Never exits 1 and handles malformed payloads cleanly, but fails open on heredoc edge cases, path-qualified/flag-based wrappers, subshell `cd`, `eval --`, `pwsh -Command`, and parse errors lacking `"git"+"push"`. |
| Maintainability    | B | Well-commented asymmetric parser; all new functions are at CC `<= 14` (within `<= 15`, though 5 sit between 11 and 14 vs `AGENTS.md`'s `< 10`). |
| Handoff-readiness  | A | `proposal.md`, `tasks.md`, `features.json`, `verification.md`, ADR-065, and lesson 614 are complete and consistent. |

### Verdict
FAIL

### Recommended next steps
- **Code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`)**:
  - Update `stripHeredocs` to track quote/comment state across lines (ignoring `<<` inside comments or multi-line quotes), match general unquoted heredoc delimiters (including `-`, `.`, digits, e.g. `[^\s;&|()<>'"]+` / `\\?` unescaping), and widen `shellFeeder` so shell invocations followed by `<`, `;`, `&`, `|`, `)` (not only `\s|$`) keep the heredoc body.
  - Handle `$'...'` ANSI-C quoting in the lexer (or mark `$'` escapes), and update `run()` fail-closed heuristic on `ok == false` to also refuse when `land:push` appears.
- **Code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`)**:
  - Normalize `programName(t)` before checking `wrappers[...]`, handle `env -C` / `--chdir` (marking `envDir = true` or resolving the directory), propagate `envDir` (`dirKnown = false`) into nested `bash -c` / `eval` calls, mark `g.dirKnown = false` on `export GIT_DIR=...` / `GIT_WORK_TREE=...` / `GIT_COMMON_DIR=...`, and isolate `g.chdir` inside `(...)` subshell scopes so `(cd ...)` does not mutate the parent command's directory.
  - Strip leading `--` in `eval` handling, and for `ShellPowerShell` `-Command` join all arguments from `args[j:]` when `args[j]` is not a single script block/string (or check `joinTexts(args[j:])`).
  - Add regression tests for all reproduced cases in `cli/internal/harness/pushguard_test.go`.
- **Contract set (`proposal.md`, `tasks.md`, `features.json`)**: No edits required; update `verification.md` after fixing and run a fresh adversarial review round.
