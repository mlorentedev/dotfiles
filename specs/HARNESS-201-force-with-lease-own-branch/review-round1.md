---
spec: "HARNESS-201-force-with-lease-own-branch"
verdict: "FAIL"
reviewed_sha: "9b3504d8b58acfb69709417c0316bcfb2868c8e7"
reviewer: "agy/gemini-4-hydrogen-high"
date: "2026-11-19"
---

## Adversarial review

**Scope**: HARNESS-201-force-with-lease-own-branch (`git diff 31360ae33a738ab309af04a302cb8c71084d3e4d...HEAD`, 6 commits, 18 files)
**Sources**: `specs/HARNESS-201-force-with-lease-own-branch/{proposal,tasks,verification,features}.*`, `docs/adr/adr-065-agents-may-force-with-lease-their-own-branch.md`, `docs/lessons/lesson-614-narrowing-a-deny-rule-in-a-merged-list-retires-nothing-without-a-record.md`, `ai/claude/settings.json`, `harness/manifest.json`, `tests/claude-settings-template.bats`, `cli/internal/{harness,cmd,claude,deploy}`

### Spec and task alignment
- `proposal.md` contract digests match `review-request.json` via `dotf spec digest HARNESS-201-force-with-lease-own-branch`. No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags exist in spec/ADR/lesson files.
- `tasks.md` ticks AC1–AC7 implementation items and leaves only the PR-opening step unchecked; `features.json` entries remain `state: pending` with empty `evidence`.
- **AC2 fails in the real binary**: `dotf harness push-guard` sets `SilenceErrors: true` and returns `withExitCode(2, ...)` without writing to `cmd.ErrOrStderr()`, so the compiled binary exits `2` with empty stderr (`0` bytes), masked by `runPushGuardCmd` appending `err.Error()` in `harness_pushguard_test.go`.
- **AC1/parser coverage gaps**: Four REAL bypass paths allow forced pushes or `GIT_DIR` / `-C<dir>` overrides through with exit `0` (`stripHeredocs`, `skipOptions`/`prefixWords`, nested `(...)` inside `"$(...)"`, and attached `-C<dir>`/`-c<kv>`).
- **AC3–AC7**: `TestPushGuardAndGateCoexistOnPreToolUse`, `TestBoundHarnessHooksResolveToRealSubcommands`, `TestClaudeTemplate_*` (in `internal/claude` and `internal/deploy`), `bats tests/claude-settings-template.bats` (33/33 via Git Bash), and ADR-065 / lesson 614 are present and pass, with a minor AC3 gap where `/c/...` is not exercised against a real git repo in `internal/cmd` tests.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | cmd / AC2 | `dotf harness push-guard` exits 2 with **empty stderr** on refused pushes because `newHarnessPushGuardCmd` sets `SilenceErrors: true` (`cli/internal/cmd/harness_pushguard.go:55`), `runPushGuard` returns `withExitCode(gateExitBlock, ...)` (`:96`) without writing to `cmd.ErrOrStderr()`, and `cli/cmd/dotf/main.go:93-100` suppresses error output when `SilenceErrors` is true (unlike `harness_gate.go:249`). `harness_pushguard_test.go:72-75` masks this by appending `err.Error()` onto `errb.String()`. | Reproduced with compiled `dotf.exe harness push-guard` on `git push --force origin main`: `EXIT=2`, `OUTPUT=[]`. | UNTESTED | code + tests (`cli/internal/cmd/harness_pushguard.go`, `cli/internal/cmd/harness_pushguard_test.go`) |
| Major | REAL | lexer / heredocs | `stripHeredocs` (`cli/internal/harness/pushguard_lex.go:238-272`) runs regex matching before quote lexing and only checks `shellFeeder` against `lines[i][:m[0]]` without leading paths, swallowing real forced pushes after quoted `<<EOF`, `cat <<EOF \| bash`, or `/bin/bash <<EOF`. | Reproduced with compiled `dotf.exe`: `"echo \"<<EOF\"\ngit push --force origin main"` -> `exit=0`; `"cat <<EOF \| bash\ngit push --force origin main\nEOF"` -> `exit=0`; `"/bin/bash <<EOF\ngit push --force origin main\nEOF"` -> `exit=0`. | UNTESTED (`TestGuardCommand_Heredocs` covers only unpathed `bash <<` and `sh <<`) | code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | guard / wrappers | `skipOptions` (`cli/internal/harness/pushguard.go:140-145`) stops at option arguments (`-u FOO`, `-a foo`, `-p`) and discards `env` variable assignments (`GIT_DIR=...`), while `prefixWords` (`:112-116`) does not skip `command`/`exec` flags, letting wrapped forced pushes or `env GIT_DIR=...` lease pushes pass. | Reproduced with compiled `dotf.exe`: `env -u FOO git push --force origin main`, `sudo -u root git push --force origin main`, `command -p git push --force origin main`, `exec -a foo git push -f origin main`, and `env GIT_DIR=/x/.git git push origin --force-with-lease=feat/x:<sha> feat/x` all returned `exit=0`. | UNTESTED (`TestGuardCommand_Table` tests bare `command`/`FOO=1`/`GIT_DIR=` only) | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | lexer / substitution | A subshell `(...)` inside double-quoted `"$(...)"` (`echo "$( (true); git push --force origin main)"`) prematurely pops the `$(` stack frame on the inner `)` (`cli/internal/harness/pushguard_lex.go:180-189`), restoring `lx.q = '"'` and turning the following `; git push --force origin main` into literal quoted text. | Reproduced with compiled `dotf.exe` on `echo "$( (true); git push --force origin main)"` -> `exit=0`. | UNTESTED | code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`) |
| Major | REAL | guard / git-globals | `gitGlobals` (`cli/internal/harness/pushguard.go:290-296`) only matches separate `-C <dir>` and `-c <kv>` tokens; attached `-C<dir>` and `-c<kv>` (valid git CLI syntax) fall through `strings.HasPrefix(t, "-")` and are ignored. | Reproduced with compiled `dotf.exe`: `git -C<nonexistent> push origin --force-with-lease=feat/x:<sha> feat/x` -> `exit=0` (read `cwd` repo instead); `git -cremote.origin.push=+refs/heads/main:refs/heads/main push origin` -> `exit=0`. | UNTESTED | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Minor | REAL | guard / nested-shell | `shellString` (`cli/internal/harness/pushguard.go:195-204`) passes `args[i+1]` blindly to `g.nested`, so `bash -c -- 'git push --force origin main'` inspects `"--"` instead of the command string. | Reproduced with compiled `dotf.exe` on `bash -c -- 'git push --force origin main'` -> `exit=0`. | UNTESTED | code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`) |
| Minor | REAL | maintainability | Five functions in `pushguard.go` and `pushguard_lex.go` exceed `CC < 10`, and two exceed `CC <= 15`: `gitGlobals` (17), `unquoted` (16), `escapeOrSubst` (14), `refuseForm` (13), `addLong` (11). | Observed via `golangci-lint run --enable-only=cyclop ./internal/harness/pushguard.go ./internal/harness/pushguard_lex.go`. (`cli/.golangci.yml` scopes `gocyclo` to `internal/cmd/`.) | UNTESTED (static lint) | code (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_lex.go`) |
| Minor | REAL | tests / AC3 | Real-repo tests in `cli/internal/cmd/harness_pushguard_test.go` only pass `C:\...` (`t.TempDir()`) to `gitLine`; `/c/...` is only tested in unit tests (`TestNativeDir`, `TestGuardCommand_DirectoryResolution`) and not end-to-end against real `git`. | Code inspection of `cli/internal/cmd/harness_pushguard_test.go:79-148`. | `TestNativeDir`, `TestGuardCommand_DirectoryResolution` (split) | tests (`cli/internal/cmd/harness_pushguard_test.go`) |
| Minor | THEORETICAL | globs / remote-url | `ai/claude/settings.json` has `git * push` fallback globs for `--force`, `-f`, and bare `--force-with-lease`, but omits `git * push` variants for `--mirror` and `main:`/`master:` leases when the hook is down; `newGitPushEnv` skips `DefaultBranch` resolution when `remote` is a URL (`safeRemoteName` false), leaving non-`main`/`master` default branches unblocked via URL push. | Code inspection of `ai/claude/settings.json:28-43` and `cli/internal/cmd/harness_pushguard.go:106-119`. | `TestClaudeTemplate_PushDenyRulesArePinned`, `TestSafeRemoteName` | code (`ai/claude/settings.json`, `cli/internal/cmd/harness_pushguard.go`) or disposition in `verification.md` |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | D | AC2 fails in the compiled binary (`exit 2` with empty stderr), and four REAL parser/wrapper bypasses allow forced pushes with `exit 0`. |
| Verification       | C | Package Go tests and `bats` (Git Bash) pass, but `runPushGuardCmd` masks empty stderr by appending `err.Error()` and `/c/...` lacks real-repo e2e coverage. |
| Scope              | A | Diff is strictly scoped to HARNESS-201 across `ai/claude/settings.json`, `harness/manifest.json`, `cli/internal/*`, bats test, ADR-065, lesson 614, and spec files. |
| Reliability        | C | Never exits 1 and fails closed on unknown `cd`/detached HEAD, but fails open on quoted `<<EOF`, `\| bash`, `/bin/bash <<`, `env -u`/`command -p`, nested `( )` in `"$(...)"`, and attached `-C`/`-c`. |
| Maintainability    | C | Clear structure and comments, but `gitGlobals` (CC 17) and `unquoted` (CC 16) exceed `CC <= 15` (5 functions > 10 via `cyclop`). |
| Handoff-readiness  | B | Spec, ADR-065, and lesson 614 are complete without draft tags; `features.json` entries remain `pending` with empty `evidence`. |

### Verdict
FAIL

### Recommended next steps
- **Code + tests (`cli/internal/cmd/harness_pushguard.go`, `cli/internal/cmd/harness_pushguard_test.go`)**: Write `"push-guard: " + v.Reason` to `cmd.ErrOrStderr()` before returning `withExitCode(gateExitBlock, ...)`, and update `runPushGuardCmd` (or add a `main.run` / binary test) to assert `errb.String()` directly without appending `err.Error()`.
- **Code + tests (`cli/internal/harness/pushguard_lex.go`, `cli/internal/harness/pushguard_test.go`)**: Fix `stripHeredocs` so `<<` inside quotes is ignored, heredocs piped to shells (`| bash`, `| sh`, etc.) or fed to path-qualified shells (`/bin/bash <<`) are not stripped, and track `(` nesting depth inside `$(...)` so inner subshells do not pop the `$(` quote frame early; add regression tests for each.
- **Code + tests (`cli/internal/harness/pushguard.go`, `cli/internal/harness/pushguard_test.go`)**: Handle wrapper flags/env assignments (`env -u X`, `env GIT_DIR=...`, `sudo -u X`, `command -p`, `exec -a X`), attached `-C<dir>` / `-c<kv>` in `gitGlobals`, and `--` after `bash -c`; add regression tests for each.
- **Code (maintainability)**: Split `gitGlobals` (CC 17) and `unquoted` (CC 16) (and optionally `escapeOrSubst`, `refuseForm`, `addLong`) to bring cyclomatic complexity within `<= 15` (ideally `< 10` per `AGENTS.md`).
- **Tests (`cli/internal/cmd/harness_pushguard_test.go`)**: Add a real-repo `/c/...` path test case in `TestPushGuard_AgainstARealRepository` (on Windows) to exercise `nativeDir` end-to-end with `git`.
- **Re-review**: After applying the code and test fixes above and updating `verification.md`, run a fresh adversarial review pass (`proposal.md`, `tasks.md`, and `features.json` require no edits).
