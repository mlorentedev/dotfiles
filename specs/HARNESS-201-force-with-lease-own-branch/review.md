---
spec: "HARNESS-201-force-with-lease-own-branch"
verdict: "FAIL"
reviewed_sha: "053b9f774eea951f6c5ca263e50c01b8374e9cab"
reviewer: "agy/gemini-4-hydrogen-high"
date: "2026-11-19"
---

## Adversarial review

**Scope**: HARNESS-201-force-with-lease-own-branch (`git diff 31360ae33a738ab309af04a302cb8c71084d3e4d...HEAD`, 16 commits on branch, 30 files)
**Sources**: `specs/HARNESS-201-force-with-lease-own-branch/{proposal,tasks,verification,features}.*`, `docs/adr/adr-065-agents-may-force-with-lease-their-own-branch.md`, `docs/lessons/lesson-614-narrowing-a-deny-rule-in-a-merged-list-retires-nothing-without-a-record.md`, `ai/claude/settings.json`, `harness/manifest.json`, `tests/claude-settings-template.bats`, `cli/internal/{harness,cmd,claude,deploy}`

### Spec and task alignment
- `proposal.md`, `tasks.md`, and `features.json` digests match `review-request.json` via `dotf spec digest HARNESS-201-force-with-lease-own-branch`. No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in spec, ADR, or lesson files.
- `tasks.md` ticks all setup, implementation, and closing items except opening/readying the PR; `features.json` marks `f1`–`f7` as `passing`.
- Verified live at `053b9f77`:
  - `go test ./internal/harness ./internal/claude ./internal/deploy -count=1 -timeout 5m` passed (`harness` 45.438s, `claude` 2.132s, `deploy` 4.821s).
  - Targeted `go test ./internal/cmd -run 'TestPushGuard|TestNativeDir|TestBoundHarnessHooksResolveToRealSubcommands' -count=1 -timeout 5m` passed (54.666s; full `internal/cmd` package was not run in this pass).
  - `go vet ./...` passed for both Windows and `GOOS=linux`.
  - `golangci-lint run --no-config --enable-only govet,errcheck,staticcheck,unused,ineffassign,gocyclo ./internal/harness ./internal/cmd ./internal/claude ./internal/deploy` reported `0 issues.` (default `golangci-lint run` fails on `gofmt` because git checks out CRLF on Windows while index EOLs are LF).
  - `bats tests/claude-settings-template.bats` passed `33/33` under Git Bash (`exit 0`).
  - Mutation test in `cli/internal/harness/pushguard.go:778` (disabling the `main`/`master` backstop in `refuseDefault`, reverted cleanly) turned `TestGuardCommand_Table/master_with_default_unresolved` red (`exit 1`).
- Probing the compiled `dotf.exe harness push-guard` binary and verifying execution against Git Bash (`C:\Program Files\Git\bin\bash.exe`), `git`, and `pwsh` reproduced five REAL Major bypasses.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | guard / `env -S` argument splitting | `simple` (`cli/internal/harness/pushguard.go:132-139`), `envScript` (`:195-218`), and `skipOptions` (`:223-246`) check the `-S` string in isolation while still letting `skipOptions` consume trailing options (`--force`) or return non-`git` remainders (`push`, `+main`). `env -S "git push" --force origin main`, `env -S "git" push --force origin main`, and `/usr/bin/env -S "git push origin" +main` all exit `0` in `push-guard`, and the `/usr/bin/env` / `FOO=1 env` / `command env` forms also evade `ai/claude/settings.json`'s `Bash(env:*)` deny rule. | Reproduced with compiled `dotf.exe harness push-guard` (`exit=0`, empty stderr) on all three commands; confirmed Git Bash executes all three against a bare remote (`Everything up-to-date`). | UNTESTED (`TestGuardCommand_ThirdRoundBypasses` / `FourthRoundBypasses` test `env -S` only when the force flag is inside the `-S` string) | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | lexer / `#3611` heredoc backstop | `heredocFeedsAShell` (`cli/internal/harness/pushguard_lex.go:377, 390-401`): (1) checks `strings.Contains(b, "push") && forceLike.MatchString(b)` per single line of `bodies`, so splitting `push` and `--force` with a line continuation (`cat <<EOF \|\ngit push \\\n  --force origin main\nEOF\nbash`) exits `0`; (2) `forceLike` (`--force\S*\|--mirror\|...`) omits `--mir` (unambiguous abbreviation of `--mirror` accepted by `git`), so `cat <<EOF \|\ngit push --mir origin\nEOF\nbash`, `b''ash <<EOF\ngit push --mir origin\nEOF`, and `bash \\\n<<EOF\ngit push --mir origin\nEOF` exit `0`. | Reproduced with compiled `dotf.exe harness push-guard` (`exit=0`, empty stderr) on all four inputs; confirmed `git push --dry-run --mir origin` succeeds in real `git` (`* [new reference] origin/main -> origin/main`) and all four execute in Git Bash. | UNTESTED (`TestGuardCommand_HeredocFeederBackstop` tests single-line `--force`/`-f`/`+main`/`--mirror` only) | code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | lexer + guard / pipeline `cd` scope | `operator` (`cli/internal/harness/pushguard_lex.go:250-255`) calls `lx.markSub()` only on the command *before* `\|` or `&`, not on the right-hand stage of a pipeline before `&&`/`\|\|`/`:`/`;`. In `bash`/`sh`/`dash`, `echo hi \| cd <feat_dir> && git push origin <lease> feat/x` (and `; git push ...`) executes `cd <feat_dir>` in a subshell while `git push` runs in the parent `cwd` (`<main_dir>` on `main`), yet `g.cd` (`cli/internal/harness/pushguard.go:316-322`) updates `g.dir` to `<feat_dir>` and exits `0`. | Reproduced with compiled `dotf.exe harness push-guard` (`exit=0`, empty stderr) from `cwd=<main_dir>` for both `&&` and `;`; confirmed Git Bash `pwd` stays in `<main_dir>` after `echo hi \| cd <feat_dir>`. | UNTESTED (`TestGuardCommand_CdBeforeAPipeOrAmpersand` tests `cd /repo \| git push ...` only) | code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | lexer + guard / `PowerShell` tool | Under `tool_name="PowerShell"`, `push-guard` is the sole guard (`ai/claude/settings.json` has `Bash(...)` rules only). (1) `escapeOrSubst` (`cli/internal/harness/pushguard_lex.go:137-140`) puts `\n` into the word on PowerShell `` ` ``+`\n` (`git \`\npush --force origin main` -> `"git\npush"`), whereas `pwsh` treats it as a line continuation separating `git` and `push`. (2) `byProgram` (`cli/internal/harness/pushguard.go:293`) omits `Push-Location` (`pushd` is an alias for `Push-Location`). (3) `byProgram` (`:301`) omits `iex`/`Invoke-Expression` (`iex "git push --force origin main"`). | Reproduced with compiled `dotf.exe harness push-guard` (`exit=0`, empty stderr) under `tool_name="PowerShell"` for all three inputs; confirmed `pwsh -NoProfile` executes `git push --force origin main` for (1) and (3) and switches `PWD` to `<main_dir>` for (2). | UNTESTED (`TestGuardCommand_PowerShell` covers single-line `;` commands and `pwsh -Command` only) | code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | guard / unrecognized wrappers (`timeout`, `nice`) | `wrappers` (`cli/internal/harness/pushguard.go:151`) omits `timeout` and `nice`. `timeout 30 git push --force origin main` and `nice git push --force origin main` exit `0` in `push-guard`, match no `ai/claude/settings.json` deny rule, and execute `git push --force origin main` in Git Bash. | Reproduced with compiled `dotf.exe harness push-guard` (`exit=0`, empty stderr) on both commands; confirmed Git Bash executes `timeout 5 git ... push --dry-run --force origin main` and `nice git ... push --dry-run --force origin main` (`Everything up-to-date`). | UNTESTED | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Minor | REAL | spec evidence & complexity | `verification.md:30` and `features.json` (`f1`–`f7`) still cite `ead41879` instead of `053b9f77` (`HEAD`), which modified `pushguard_lex.go` and `pushguard_test.go`. Also `golangci-lint` (`cyclop`) reports 7 functions in `pushguard.go`/`pushguard_lex.go` between 11 and 15 (`byProgram` 14, `addLong` 11, `refuseForm` 13, `unquoted` 15, `operator` 12, `stripHeredocs` 13, `feed` 15) vs `AGENTS.md`'s `< 10` target (all `<= 15`, passing `cli/.golangci.yml`). | Observed in `verification.md`, `features.json`, `git log`, and `golangci-lint run --no-config --enable-only cyclop ./internal/harness ./internal/cmd`. | UNTESTED (static check) | code + spec (`cli/internal/harness/pushguard*.go`, `specs/HARNESS-201-force-with-lease-own-branch/verification.md`, `features.json`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All 7 AC test suites pass and the `#3611` backstop covers single-line split-shell heredocs, but five REAL Major bypasses (`env -S` trailing args, multi-line/`--mir` heredocs, RHS pipe `cd`, PowerShell `` `\n ``/`Push-Location`/`iex`, `timeout`/`nice`) allow forced or wrong-dir pushes with exit `0`. |
| Verification       | B | `go test` (`harness`, `claude`, `deploy`, targeted `cmd`), Windows & Linux `go vet`, `golangci-lint`, `bats` (`33/33`), and `refuseDefault` mutation check pass, though negative tests miss the five reproduced bypasses, full `cmd` was not run, and `verification.md`/`features.json` cite `ead41879`. |
| Scope              | A | Diff is strictly scoped to HARNESS-201; `contract_digests` match `review-request.json` and no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain. |
| Reliability        | C | Never exits 1 and handles malformed payloads safely, but fails open on `env -S "..." --force`, multi-line/`--mir` heredoc feeders, `... \| cd <dir> && ...`, PowerShell `` `\n ``/`Push-Location`/`iex`, and `timeout`/`nice`. |
| Maintainability    | B | Clean asymmetric parser with all new functions `<= 15` CC (0 above 10 in `harness_pushguard.go`; 7 at 11–15 in `pushguard*.go`), passing `cli/.golangci.yml`. |
| Handoff-readiness  | B | Spec, ADR-065, and lesson 614 are complete without draft tags, though `verification.md` and `features.json` evidence cite `ead41879` instead of `053b9f77`. |

### Verdict
FAIL

### Recommended next steps
- **Code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`)**:
  - When `env -S` / `--split-string` is present, combine the `-S` words with the remaining positional words after `env`'s own options (rather than scanning `env`'s remaining arguments as `env` options or dropping `splitScript` before `push ... --force`).
  - Add `timeout` (handling its duration argument) and `nice` (handling `-n <adj>` / `--adjustment=<n>`) to `wrappers`.
  - In `byProgram`, handle PowerShell `Push-Location` alongside `Set-Location`/`pushd`, and `iex`/`Invoke-Expression` alongside `eval`.
- **Code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`)**:
  - In `heredocFeedsAShell`, evaluate the joined heredoc body across lines (normalizing `\<newline>` continuations) instead of per single line, and match git's unambiguous force abbreviations (`--mir`..`--mirror`, `--force\S*`).
  - Mark `cd`/`pushd` on the right-hand side of a pipeline (`| ...`) as subshell-scoped (`sub = true`) just like the left-hand side.
  - In `escapeOrSubst` for `ShellPowerShell`, treat `` ` `` followed by `\n` (or `\r\n`) as a line continuation boundary (`endWord` / whitespace) rather than appending `\n` inside the word.
  - Add named regression tests in `cli/internal/harness/pushguard_test.go` for all five reproduced bypasses.
- **Spec (`verification.md`, `features.json`) + re-review**: Refresh evidence SHAs in `verification.md` and `features.json` after fixing and run a fresh adversarial review pass before `dotf spec archive`.
