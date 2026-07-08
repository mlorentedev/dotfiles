---
tags: [spec, verification, secrets, jit]
created: "2026-06-25"
---

# Verification - CLI-024-secrets-run-jit

> Phase 1a evidence. Branch `feat/secrets-run-jit`.

## AC1/AC2/AC5 — env-builder + mapping parser (unit)

`go test ./internal/secrets/...` → `ok`. Covers:
- `ParseMapping`: `VAR=file`, `@VAR=file>dest`, comments, blanks, `~` expansion, non-mapping lines skipped.
- `EnvFor`: env secrets → `VAR=value` with trailing newline stripped; `--only` injects exactly the named vars; decrypt failure fails fast (child never launched with partial secrets).

## AC3 — file secrets materialized 0600

`TestEnvFor_FileSecret_MaterializedDest` → file written to `dest` (parent dirs created), `VAR=<dest>`, content verbatim.

## AC4 — child exit-code propagation + env injection (unit, cross-platform)

`go test ./internal/cmd/ -run TestRunChild` (helper-process pattern, no system shell dependency):
- `TestRunChild_PropagatesExitCodeAndInjectsEnv` → child sees injected `FOO`, returns exit 3 (propagated).
- `TestRunChild_ZeroExit` → (0, nil).
- `TestRunChild_LaunchFailureIsError` → missing binary → error (not exit code).

## AC6 + live end-to-end (real age store on this machine)

Built `dotf`, ran against the deployed `~/.dotfiles/sensitive` + `~/.config/age/key.txt` (value never printed — byte count only):

```
$ dotf secrets run --only NAN_API_KEY -- sh -c 'printf %s "$NAN_API_KEY" | wc -c'
25                         # secret decrypted + injected into the child

$ dotf secrets run -- sh -c 'exit 7'; echo rc=$?
rc=7                       # exit code propagated
```

`dotf secrets run` itself writes nothing to stdout/stderr on success — only the child's output appears. The injected value is present in the child only.

> Note: the parent shell on this machine still has `NAN_API_KEY` set, because the login-time `load-secrets` ambient export is **still active** — removing it is Phase 1b (this PR is purely additive and does not touch shell startup).

## Build / suites

- `go build ./...` → OK.
- `go test ./internal/{secrets,cmd}/...` → `ok`.
- Pre-existing, unrelated `TestEmbeddedTemplatesMatchVault` drift in `initrepo/spec/vault` not touched.

## Deferred — Phase 1b (next PR)

Retire the ambient `load-secrets` sourcing from `.bashrc`/`.zshrc`/`profile.ps1` + the setup eager-load, AND migrate the ambient consumers (`opencode` reads `{env:NAN_API_KEY}`, `agy` reads `ANTHROPIC_API_KEY` — per `profile.ps1`) to launch via `dotf secrets run`. Tracked on #493.
