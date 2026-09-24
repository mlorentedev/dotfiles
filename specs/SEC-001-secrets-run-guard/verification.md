---
tags: [spec, verification]
created: "2026-09-02"
---

# Verification - SEC-001-secrets-run-guard

This spec landed in #1459 (`a720b9d`, 2026-09-02) with this file still an unfilled template. It was filled on 2026-09-23 after a retroactive review, run at the landing commit, failed it (round 1, below). Every check in `features.json` is a runnable command, and each one exits 0 on this branch.

## Evidence

| AC | Proof | Where |
|---|---|---|
| AC1, AC2 | `TestAssertSafeChildCommand`: the `bare env`, `path to env`, `bare printenv` and `bare export` rows. `TestRunChildPTY_HonoursTheIntrospectionGuard` drives the real run path. | `a720b9d` |
| AC3 | `TestAssertSafeChildCommand` shell rows, now including an absolute path, a redirect with no space, quotes and a backslash inside the word, a line continuation, and upper case | `a720b9d`, then `60d2251` and `3ed6de0` |
| AC4 | The `allowed ...` rows (`goreleaser`, `python3`, `dotf review`, `echo ... environment`, `run-env-check`, `cat .env.example`) | `a720b9d`, `60d2251` |
| AC5 | The table above: 31 rows, safe and unsafe | `cli/internal/cmd/secrets_test.go` |
| AC6 | `ai/claude/settings.json` deny list, and `ai/pi/models.json` `providers.openrouter` (features.json f6) | `a720b9d` |
| AC7 | `TestRedactWriter_*`, including `TestRedactWriter_SecretInTinyChunksNeverLeaks` (writes of 1-3 bytes), and `TestRunChildPTY_RedactsASecretSplitAcrossWrites` | `a720b9d`, SEC-002, `60d2251` |
| AC8 | `docs/lessons/lesson-261-never-test-secret-guards-against-live-credentials-and-redact-at-the-stream-boundary.md` (f7) | `a720b9d` |
| AC9 | `TestSecretsShow_RejectsAgentSession`, `_TTYMasking`, `_RevealFlag`, `_ClipFlag`; `TestDetectAgentSession_*`; `TestAgentSessionMarkersAreDocumented` | `a720b9d`, `60d2251` |
| AC10 | The deny list holds `Bash(sudo:*)`, `Bash(git clean -f*:*)`, `Bash(*dotf secrets show*:*)` and the pipe-to-shell bans (f6) | `a720b9d` |

**Known gap in AC6/AC10, disclosed rather than hidden.** The deny list exists in the template, but it has **never reached an existing installation**. `merge_claude_settings` unions `permissions.allow` and leaves `permissions.deny` out of its merge policy. Measured on msi on 2026-09-23: the deployed `~/.claude/settings.json` has 0 deny rules. The fix is the Go port of the Claude settings deploy (#1339, comment of 2026-09-23), not a setup-script edit. The ACs as written ("the template is hardened") hold; the protection they were meant to give does not yet.

## Test status

- `go build ./...`, `go vet ./...` and `GOOS=windows go vet ./...`: clean.
- `go test ./... -count=1`: 24 packages ok, 0 failed.
- `golangci-lint run` on the pinned 2.12.2: 0 issues.
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
- Every `features.json` verifier exits 0. Two negative controls exit 1: a `-run` pattern that matches no test, and a deny rule that is not in the file.

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

## Decisions made during implementation

- Whole-word matching over a longer boundary class. The class kept missing separators (`/`, `>`, `<`, a newline after a backslash). Splitting on everything that cannot be part of a command word closes the class of misses, not the instances.
- The marker list stays lexical and per-vendor for now. An agnostic marker that every harness exports needs deploy work in six places, so it is #1646 rather than this fix.

## Promotion candidates

- [x] Lesson? No new one. The CLAUDECODE miss is lesson 287's class, "a guard that skips is a guard that passes", and its test now pins the variable name the harness really exports.
- [x] ADR? No. ADR-028's marker list is corrected in place.
- [x] Pattern? No.

## Archive checklist

- [ ] Round 2 review passes
- [ ] `dotf spec archive SEC-001-secrets-run-guard`
- [ ] #1626 records the disposition
