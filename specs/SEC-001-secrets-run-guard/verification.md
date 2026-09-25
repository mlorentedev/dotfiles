---
tags: [spec, verification]
created: "2026-09-02"
---

# Verification - SEC-001-secrets-run-guard

This spec landed in #1459 (`a720b9d`, 2026-09-02) with this file still an unfilled template. It was filled on 2026-09-23 after a retroactive review, run at the landing commit, failed it (round 1, below). Round 1's fixes merged in #1655. Round 2 failed them, and round 3 failed round 2's fixes (`d8f8b11`); both sets of fixes are on `fix/sec001-round3`. Every check in `features.json` is a runnable command, and each one exits 0 on this branch.

## Evidence

| AC | Proof | Where |
|---|---|---|
| AC1, AC2 | `TestAssertSafeChildCommand`: the `bare env`, `path to env`, `bare printenv` and `bare export` rows, and since round 3 a Windows path, an `.exe` suffix in either case, and `busybox <applet>`. `TestRunChildPTY_HonoursTheIntrospectionGuard` drives the real run path. `TestSecretsRun_RefusesBeforeResolvingSecrets` counts the secret reads of a refused command: zero, so "without decrypting" holds. | `a720b9d`, round 3 |
| AC3 | `TestAssertSafeChildCommand` shell rows: an absolute path, a redirect with no space, quotes and a backslash inside the word, a line continuation, upper case, and since round 4 the shell's own option grammar (`--`, `+c`, a cluster, `-o`/`-O`/`--rcfile` arguments, a script operand). `TestShellCommandString_AgreesWithRealShells` runs each argv shape through the real bash, zsh, sh and dash and requires them to execute the operand the parser names. | `a720b9d`, then `60d2251`, `3ed6de0`, round 4 |
| AC4 | The `allowed ...` rows (`goreleaser`, `python3`, `dotf review`, `echo ... environment`, `run-env-check`, `cat .env.example`) | `a720b9d`, `60d2251` |
| AC5 | `TestAssertSafeChildCommand`: 55 rows, 44 refused and 11 allowed | `cli/internal/cmd/secrets_test.go` |
| AC6 | `ai/claude/settings.json` denies `Bash(env:*)`, `Bash(printenv:*)` and `Bash(export -p:*)`, and `ai/pi/models.json` registers `providers.openrouter`: features.json f6, which checks all three entries since round 3. Template-scoped (see the next paragraph). | `a720b9d`, round 3 |
| AC7 | `TestRedactWriter_*`, including `TestRedactWriter_SecretInTinyChunksNeverLeaks` (writes of 1-3 bytes), and `TestRunChildPTY_RedactsASecretSplitAcrossWrites` | `a720b9d`, SEC-002, `60d2251` |
| AC8 | `docs/lessons/lesson-261-never-test-secret-guards-against-live-credentials-and-redact-at-the-stream-boundary.md` (f7) | `a720b9d` |
| AC9 | `TestSecretsShow_RejectsAgentSession`, `_TTYMasking`, `_RevealFlag`, `_ClipFlag`; `TestDetectAgentSession_*`; `TestAgentSessionMarkersAreDocumented`; `TestAgentSessionMarkers_CoverEveryHarness`, which requires a known marker for every harness `harness/model-map.json` declares | `a720b9d`, `60d2251`, round 3 |
| AC10 | The deny list holds `Bash(sudo:*)`, `Bash(git clean -f*:*)`, `Bash(*dotf secrets show*:*)` and the pipe-to-shell bans (f6) | `a720b9d` |

**AC6/AC10 are template-scoped (owner decision, 2026-09-24).** The deny list exists in the template, but it has **never reached an existing installation**. `merge_claude_settings` unions `permissions.allow` and leaves `permissions.deny` out of its merge policy. Measured on msi on 2026-09-23: the deployed `~/.claude/settings.json` has 0 deny rules. Round 2 failed the ACs for reading as met while that protection was inert (F2). They now say what they measure, the template, and the proposal's Out of scope names the deploy as #1339's, the Go port of the Claude settings deploy. The protection they were written for is not in place until #1339 lands.

## Test status

- `go build ./...`, `go vet ./...` and `GOOS=windows go vet ./...`: clean.
- `go test ./... -count=1`: 25 packages ok, 0 failed (round 4).
- `golangci-lint run` on the pinned 2.12.2: 0 issues.
- Round 3 mutation run, one compiling mutant at a time, restored after each: **10 of 10 killed**. Each of `AI_AGENT`, `PI_CODING_AGENT`, `OPENCODE` and `COPILOT_CLI` dropped from the list; the early guard in `run` removed; `.exe` kept; `\` not treated as a separator; case kept; `busybox` not unwrapped; `ash` not inspected.
- Mutation run against the final code, one compiling mutant at a time, restored after each run. **10 of 10 killed.** The mutants:
  - `/` or `<>` made word characters;
  - no line-continuation fold;
  - quotes kept (single and double, separately);
  - backslashes kept;
  - case-sensitive matching;
  - `CLAUDECODE` dropped;
  - the marker loop ignoring the environment;
  - the child environment keeping the unlock credentials.

  A build-error mutant was redone as a compiling one; it does not count (lesson 284). On the intermediate code (`60d2251`), one mutant **survived**: reading the cleaned form only. It showed that the as-written pass caught nothing the shell runs, so that pass was deleted (`3ed6de0`) rather than kept untested.
- Round 4 mutation run on the shell-argv parser, one compiling mutant at a time. The script first asserts that the unmutated suite passes, and counts a mutant only when its tests compile (`go vet`). **10 of 10 killed**: `--` not an end of options; the operand after `--` off by one; `+c` not a flag; `-o` or `-O` consuming nothing (two mutants); a long option's argument not consumed; a later flag resetting c; `typeset` dropped; the first operand off by one; an operand read without the c flag. The first run of that set reported kills while the test file did not compile. That is how the baseline and compile checks came to be added. It was rerun before any count was recorded.
- Every `features.json` verifier exits 0 (round 4). Two negative controls exit 1: a `-run` pattern that matches no test, and a deny rule that is not in the file.

## Review round 1 — dispositions

`nan/glm5.3-flash`, verdict FAIL, reviewed at `a720b9d`.

| # | Finding | Disposition |
|---|---|---|
| F1 | Blocker: split writes shorter than the secret bypass the hold-back | **Already fixed on main** by SEC-002's prefix-aware hold-back. It was true at `a720b9d`, but `holdBack` on main withholds any trailing proper prefix whatever the write size. Pinned by `TestRedactWriter_SecretInTinyChunksNeverLeaks`. |
| F2 | Major: `sh -c '/usr/bin/env'` and `sh -c 'env>x'` bypass the snippet check | **Applied** (`60d2251`, `3ed6de0`). The snippet is split into whole shell words, read as the shell reads them. Still true on main before this fix. |
| F3 | Major: unfilled verification.md, false closing ticks, placeholder features.json | **Applied**: this file, `features.json` (7 runnable checks), and `tasks.md`, whose two false ticks are now called out. |
| F4 | Minor: `bash -c 'echo env'` is refused | **Declined.** Failing closed is the intended bias for a tripwire. AC4 names the legitimate tools, and they run. `run-env-check` and `.env.example` are pinned as allowed. |
| F5 | Minor: shells outside the list (`fish`, `pwsh`) are not inspected | **Deferred to #1650** (SEC-004). It is not theoretical on Windows, where `pwsh` and `cmd` are the everyday shells and need their own vocabulary. |
| F6 | Minor: indirection (`e=en; ${e}v`) is not refused | **Declined.** It is outside the declared scope ("arbitrary payload content"). A snippet can compute any command name at run time, and no lexical check sees that. |
| F7 | Minor: `buildChildEnv` is production-dead | **Applied**: `childEnviron`, called by the run path and by the tests. |
| F8 | Minor: ADR-028 omits `ANTIGRAVITY_CLI` | **Applied**, and widened: see the next row. |
| Q | Do mixed-glob deny patterns match? | **Deferred with #1339.** Deny rules do not deploy at all today (see the known gap above), so their syntax is verified when the port makes them live. |
| — | Found while applying F8: AC9's refusal looked for `CLAUDE_CODE`, but Claude Code exports `CLAUDECODE` | **Applied** (`60d2251`). The markers are one declared list, and a test keeps both documents naming every entry. The agnostic remainder is #1646 (SEC-003): pi, opencode, Codex and Copilot export no marker the list knows. |

## Review round 2 — dispositions

`nan/deepseek-v4-flash`, verdict FAIL, reviewed at `1a03ae6` (the #1655 branch).

| # | Finding | Disposition |
|---|---|---|
| F1 | Major: pi exports `AI_AGENT` and `PI_CODING_AGENT`, which the list did not know, so `secrets show` printed plaintext in a pi session | **Applied**, and widened to every harness. Each marker was read off the harness rather than assumed: `CLAUDECODE` and `AI_AGENT` in a live Claude Code session; pi's two in its CLI and RPC entry code and its documentation ("child processes inherit both"); `OPENCODE`, `COPILOT_CLI` and `ANTIGRAVITY_AGENT` live, under `opencode run`, `copilot -p` and `agy --print`. Codex is not installed here, so its `CODEX_THREAD_ID` and `CODEX_SANDBOX` come from std-env's agent table. A clean login shell carries none of them, so none fires for a human. `TestAgentSessionMarkers_CoverEveryHarness` holds the table independently of the list and requires a row for every harness `harness/model-map.json` declares. |
| F2 | Major: AC6/AC10's deny list reaches no existing install | **Applied, by rewording (owner decision, 2026-09-24).** Both ACs are template-scoped, and the proposal's Out of scope names the deploy as #1339's. See the paragraph under Evidence. |
| F3 | Minor: AC6's three deny entries are checked by nothing | **Applied**: f6 checks them, and removing any one makes f6 exit 1. |
| F4 | Minor: `secrets run -- env` decrypts before refusing | **Applied**: `run` calls the guard before loading the registry. `runChild` and `runChildPTY` still check, for their other callers. `TestSecretsRun_RefusesBeforeResolvingSecrets` counts zero reads, and a positive control proves the counter counts. |
| F5 | Minor: `env.exe`, a Windows path to it, and `busybox env` bypass the base-name check | **Applied**: `commandName` takes the last element under either separator, lower-cased, without `.exe`; `busybox <applet>` is checked as `<applet>`, and `ash` joins the inspected shells for `busybox ash -c`. |
| Q (F6) | Wrapper and run-time indirection (`nice env`, `xargs env`, `sh -s`, `en$'v'`) | **Declined as out of scope**, and now stated under Out of scope. A lexical guard cannot see a command another command runs, or a name assembled at run time; the redactor still scrubs every injected value of 6 bytes or more. |
| F7 | Minor: #1646's premise is stale | **Applied**: #1646 is updated, and this round covers what it asked for. |

## Review round 3 — dispositions

`agy/gemini-3.1-pro-high`, verdict FAIL, reviewed at `d8f8b11`.

| # | Finding | Disposition |
|---|---|---|
| F1 | Blocker: `bash -c -- env` and `bash -c -i env` run the snippet uninspected, because the guard read the argument after `-c` | **Applied.** `shellCommandString` reads the shell's options the way the shell does, and the command string is the first operand once they end. Each of the argv shapes was run through the real bash, zsh, sh and dash before the test was written, and `TestShellCommandString_AgreesWithRealShells` keeps them honest in CI. |
| F2 | Blocker: `sh +c env` runs the snippet uninspected | **Applied** by the same parser: `+c` sets the c flag. |
| F3 | Major: `typeset` prints the environment like `declare` | **Applied**: it is an introspection word. |

**Why three rounds found three sets of bypasses (owner decision, 2026-09-24).** A lexical guard over shell syntax cannot be complete, and AC4 requires interpreters that can print the environment to run. The owner chose to fix these with a real option parser, not another heuristic, and to state the threat model the ACs had left implicit. The guard is a tripwire for the accidental shapes; the redactor (AC7) is what protects the values. It is now under Out of scope, so a later review measures the guard against the boundary it claims. The severities above assume the guard is a boundary: what a bypass lets through is the injected keys' names and any value under 6 bytes, not the values themselves.

## Decisions made during implementation

- Whole-word matching over a longer boundary class. The class kept missing separators (`/`, `>`, `<`, a newline after a backslash). Splitting on everything that cannot be part of a command word closes the class of misses, not the instances.
- The marker list stays lexical, per vendor plus the generic `AI_AGENT`. Round 1 deferred the other harnesses on the belief that none exported a marker; round 2 showed pi did, and measuring found one for each of them. No deploy work is needed, and `AGENT_SESSION` is kept for a harness that exports nothing of its own.

## Promotion candidates

- [x] Lesson? Round 1: no new one, since the CLAUDECODE miss is lesson 287's class. Round 3: **lesson 289**, a list tested by looping over itself cannot see a missing member, with how to measure what a harness exports.
- [x] ADR? No. ADR-028's marker list is corrected in place.
- [x] Pattern? No.

## Archive checklist

- [ ] Round 4 review passes
- [ ] `dotf spec archive SEC-001-secrets-run-guard`
- [ ] #1626 records the disposition
