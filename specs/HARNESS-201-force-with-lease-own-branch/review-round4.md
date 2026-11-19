---
spec: "HARNESS-201-force-with-lease-own-branch"
verdict: "FAIL"
reviewed_sha: "f7b05ee451bfe217958486e59ba66adc89e5f092"
reviewer: "agy/gemini-4-hydrogen-high"
date: "2026-11-19"
---

## Adversarial review

**Scope**: HARNESS-201-force-with-lease-own-branch (`git diff 31360ae33a738ab309af04a302cb8c71084d3e4d...HEAD`, 11 commits on branch, 24 files)
**Sources**: `specs/HARNESS-201-force-with-lease-own-branch/{proposal,tasks,verification,features}.*`, `docs/adr/adr-065-agents-may-force-with-lease-their-own-branch.md`, `docs/lessons/lesson-614-narrowing-a-deny-rule-in-a-merged-list-retires-nothing-without-a-record.md`, `ai/claude/settings.json`, `harness/manifest.json`, `tests/claude-settings-template.bats`, `cli/internal/{harness,cmd,claude,deploy}`

### Spec and task alignment
- `proposal.md`, `tasks.md`, and `features.json` digests match `review-request.json` via `dotf spec digest HARNESS-201-force-with-lease-own-branch`. No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in spec, ADR, or lesson files.
- `tasks.md` ticks all implementation items and leaves only PR opening unchecked; `features.json` marks `f1`–`f7` as `passing`.
- Verified live at `f7b05ee451bf`: `go test ./internal/harness ./internal/claude ./internal/deploy -count=1` passes, `go test ./internal/cmd -v -run 'TestPushGuard|TestNativeDir|TestSafeRemoteName|TestBoundHarnessHooksResolveToRealSubcommands'` passes, `go vet ./...` passes for both `windows` and `linux`, `bats tests/claude-settings-template.bats` passes `33/33` via Git Bash, and mutating `abbreviatesForce` in `cli/internal/harness/pushguard.go:536` (reverted after check) turns `TestGuardCommand_Table/abbreviated_force`, `abbreviated_lease`, and `abbreviated_mirror` red.
- Round 2 fixes (`6b577b69`) closed the previously reported heredoc, wrapper-path, `env -C /dir`, `export GIT_DIR`, `(cd ...)`, `eval --`, `pwsh -Command`, and `$'...'` findings, but three REAL Major bypasses remain against AC1/What when probed against the compiled `dotf.exe` binary.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | guard / nested-shell | (1) `nested` (`cli/internal/harness/pushguard.go:288-294`) copies `inner.dir, inner.dirKnown` back to `g`, so `bash -c 'cd <repoA>' && git push ...` lets a child shell's `cd` override the parent shell's working directory (`repoB` on `main`). (2) `shellString` (`:269`) checks lowercase `'c'` via `strings.ContainsRune(a.text, 'c')`, missing PowerShell's uppercase `-C` (and `-Co`/`-Com`) abbreviation for `-Command`. | Reproduced with compiled `dotf.exe harness push-guard` (`EXIT=0`, empty stderr) for `bash -c 'cd <repoA>' && git push origin --force-with-lease=feat/x:<sha> feat/x` (from `cwd=repoB` on `main`), and for `pwsh -C "git push --force origin main"` under both `Bash` and `PowerShell` payloads; confirmed `pwsh -NoProfile -C "echo EXEC_3"` executes. | UNTESTED (`TestGuardCommand_CdInsideASubshellDoesNotCarryOut` covers `(...)`/`$(...)` only; `TestGuardCommand_SecondRoundBypasses` covers `-Command` only) | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | lexer / redirects | `stripRedirects` (`cli/internal/harness/pushguard_lex.go:256`) skips any word with `w.quoted == true`. A redirection with a quoted target (`>"out.txt" git push --force origin main`, `2>"err.txt" git push --force origin main`) marks the entire `>out.txt` word quoted, so `stripRedirects` leaves it as `cmd[0]`, `byProgram` sees program `">out.txt"` instead of `"git"`, and `ai/claude/settings.json` prefix globs miss it too. | Reproduced with compiled `dotf.exe harness push-guard` (`EXIT=0`, empty stderr) on `>"out.txt" git push --force origin main` and `2>"err.txt" git push --force origin main`; confirmed execution in Git Bash (`>"$tmp/out.txt" echo EXEC_1`, `2>"$tmp/err.txt" echo EXEC_2`). | UNTESTED | code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | guard / wrappers | `skipOptions` (`cli/internal/harness/pushguard.go:157-176`) treats `env -S "git push --force origin main"` (`--split-string`) as an option argument and drops it, returning empty `cmd` (`EXIT=0`). Its `-C` check (`t == "-C" \|\| strings.HasPrefix(t, "--chdir")`, `:164`) also misses attached `env -C<dir>`, leaving `envDir=false` so the lease reads `cwd` instead of `<dir>`. | Reproduced with compiled `dotf.exe harness push-guard` (`EXIT=0`, empty stderr) for `env -S "git push --force origin main"` and against real `repoA`/`repoB` for `env -C<repoB> git push origin --force-with-lease=feat/x:<sha> feat/x`; confirmed `env -S "echo EXEC_5"` executes in Git Bash. | UNTESTED (`TestGuardCommand_SecondRoundBypasses` tests space-separated `env -C /onmain` and `env --chdir=/onmain` only) | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Minor | REAL | lexer / heredocs | `heredocStart` (`cli/internal/harness/pushguard_lex.go:287`) allows only a leading backslash (`\\?`) and excludes `\`, `'`, `"` in unquoted delimiters, so `cat <<E\OF` parses `delim="E"`, never matches `EOF`, and strips trailing `git push --force origin main`. | Reproduced with compiled `dotf.exe harness push-guard` (`EXIT=0`) on `"cat <<E\\OF\nhi\nEOF\ngit push --force origin main"`; confirmed Git Bash executes `EXEC_8`. | UNTESTED | code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`) |
| Minor | REAL | maintainability & docs | Seven functions in `pushguard.go` and `pushguard_lex.go` exceed `AGENTS.md`'s `< 10` cyclomatic complexity target (`feed` 15, `byProgram` 14, `refuseForm` 13, `shellString` 11, `addLong` 11, `unquoted` 11, `stripHeredocs` 11 via `cyclop`; all remain `<= 15`). Additionally, `verification.md:25` and `features.json` evidence lines still cite `952d443c` rather than `6b577b69`. | Observed via `golangci-lint run --no-config --enable=cyclop ./internal/harness` and inspection of `verification.md:25` / `features.json`. | UNTESTED (static lint / doc check) | code + spec (`cli/internal/harness/pushguard*.go`, `specs/HARNESS-201-force-with-lease-own-branch/verification.md`, `features.json`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All 7 AC test suites pass and Round 1/2 fixes hold, but three REAL Major bypasses (`bash -c 'cd ...'` / `pwsh -C`, `>"..."` redirects, `env -S` / `env -C<dir>`) let forced or wrong-repo lease pushes exit `0`. |
| Verification       | B | `go test` (`harness`, `claude`, `deploy`, `cmd`), mutation check on `abbreviatesForce`, `go vet`, and `bats` (`33/33`) pass, though negative tests miss the three bypasses above and `verification.md:25` cites `952d443c`. |
| Scope              | A | Diff is strictly scoped to HARNESS-201; contract digests match and no `[AGENT-DRAFT]` tags remain. |
| Reliability        | C | Never exits 1 and handles malformed payloads safely, but fails open on `>"..."` redirects, `pwsh -C`, `env -S` / attached `env -C<dir>`, and `bash -c 'cd ...'` directory leakage. |
| Maintainability    | B | Clean asymmetric parser with all functions at CC `<= 15` (within Solid bar; 7 sit at 11–15 vs `AGENTS.md`'s `< 10`). |
| Handoff-readiness  | B | Spec, ADR-065, and lesson 614 are complete without draft tags; `verification.md:25` and `features.json` evidence still name `952d443c` instead of `6b577b69`. |

### Verdict
FAIL

### Recommended next steps
- **Code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`)**:
  - Keep `eval` propagating directory changes (`eval "cd ..."`), but do **not** copy `inner.dir`/`inner.dirKnown` back to `g` for child shells (`bash`/`sh`/`pwsh`/`powershell`) in `nested`.
  - In `shellString`, match PowerShell `-Command` case-insensitively including abbreviations (`-c` through `-command`, e.g. `strings.HasPrefix("-command", strings.ToLower(a.text)) && len(a.text) >= 2`).
  - In `skipOptions`, treat `strings.HasPrefix(t, "-C")` (attached `env -C<dir>`) like `-C`, and inspect `-S` / `--split-string` arguments with `g.nested` (or fail closed when `-S` appears).
- **Code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`)**:
  - Track whether the leading operator (`<`/`>`) of a word was quoted rather than skipping the whole word on `w.quoted` in `stripRedirects`, so `>"out.txt" git ...` and `2>"err.txt" git ...` still expose `git`.
  - Optional (Minor): unescape/strip quotes across mid-word `\` / `'` / `"` in `heredocStart` delimiters (`<<E\OF`), or treat mid-word backslash/quote in a delimiter as non-strippable.
- **Spec (`verification.md`, `features.json`) + re-review**: Add regression tests for each reproduced bypass above, refresh test evidence SHA in `verification.md` (and `features.json` before the next round), and run a fresh adversarial review pass.
