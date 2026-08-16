---
id: "BUG-081b-pi-runtime-secret"
type: spec
status: verifying
created: "2026-08-16"
issue: "mlorentedev/dotfiles#987"
tags: [spec, verification, secrets, pi]
template_version: "1.0"
---

# Verification — BUG-081b-pi-runtime-secret

## AC1 — the source uses pi's syntax

```
$ grep -o '"apiKey":[^,]*' ai/pi/models.json
"apiKey": "${NAN_API_KEY}"
$ grep -c '{env:' ai/pi/models.json
0
```

## AC2 — render leaves it untouched (a passthrough)

`TestRender_LeavesShellStyleVariablesUntouched`: a config containing `${VAR}` and
`$VAR` comes back byte-identical with zero substitutions reported. This is the
assertion the entire "no setup edits" scope decision rests on, so it is pinned
rather than assumed.

## AC3 / AC4 — doctor FAILs on both defects

`TestAgentConfig_SeverityByShape`, one case per shape:

| deployed `apiKey` | verdict |
|---|---|
| `{env:NAN_API_KEY}` | FAIL — "that is dotf's syntax, not pi's" |
| a bare literal | FAIL — "holds a literal credential for provider(s) nan" |
| `${NAN_API_KEY}` | OK |
| `$NAN_API_KEY` | OK |
| `!dotf secrets show NAN_API_KEY` | OK |

Both `$` forms and the `!command` form pass because all three are things pi
resolves; accepting only the braced form would fail a correct config, which is
how a guard gets disabled.

Absent config → SKIP (a machine without pi has nothing broken). Unparseable JSON
→ WARN, never OK: "could not be checked" is not "fine".

Every case also asserts the sentinel credential never appears in the report. A
check that printed a credential to prove a credential was on disk would have
reproduced the defect it reports.

## AC5 — observed red against real state

Against the actual deployed config on this machine, which still carries a
materialised literal:

```
[Agent config secrets]
  [FAIL] /home/manu/.pi/agent/models.json holds a literal credential for
         provider(s) nan — a deployed config must reference the environment
         ("${VAR}"), never carry the secret; re-run setup to redeploy from
         source (BUG-081b)
```

The provider is named; the value is not.

## AC6 — no setup script changes

```
$ git diff --stat origin/main -- setup-linux.sh setup-windows.ps1
(empty)
```

## AC7 — ADR

`docs/adr/adr-034-agent-config-secrets-resolve-at-runtime.md`. Records the
posture change, the two rejected alternatives (honest-fallback-and-keep-
materialising; pi's `!command` form, blocked on #952), and the consequence that
the guard goes red on any machine that has not yet redeployed.

## The load-bearing claim, tested directly

`#987`'s own evidence describes a machine state that has since been reverted, so
the resolver was re-tested from primary source — importing pi's shipped
`dist/core/resolve-config-value.js` and comparing fingerprints, never values:

```
sentinel fp                  : 3387dac8dd02
"${NAN_API_KEY}"  resolves   : 3387dac8dd02   <- correct
"$NAN_API_KEY"    resolves   : 3387dac8dd02   <- correct
"{env:NAN_API_KEY}" resolves : f3407edcacbd   <- the literal string
sha256("{env:NAN_API_KEY}")  : f3407edcacbd   <- exact match
unset "${NAN_API_KEY}" throws:
  Failed to resolve API key from environment variable: NAN_API_KEY
```

`f3407edcacbd` matches the fingerprint #987 captured off the wire with a capture
server — two independent methods, same answer.

Two findings fell out of that reading:

1. **The failure mode gets strictly better, not merely safer.** Today the
   unresolved placeholder is sent as the token and the server answers 401,
   indistinguishable from a bad credential. After this change an unset variable
   names itself.
2. **pi's own preflight approves the broken config.**
   `isConfigValueConfigured("{env:NAN_API_KEY}")` returns `true`, because a
   string with no `$` has no variables that could be missing. So pi did not warn
   either.

## Toolchain

`go build ./...`, `go vet ./...`, `go test ./internal/...`,
`golangci-lint run ./internal/...` (v2.12.2, the `versions.conf` pin) — all clean.
`scripts/check-doc-paths.sh` passes on the new ADR.

## Known consequence, not a regression

`dotf doctor` will report FAIL on this machine until the pi config is
redeployed. That is the point of the guard: the deployed copy still holds the
credential that leaked, and the reminder is mechanical rather than a note
someone has to remember. Redeploying is what clears it, and it is also what
removes the second copy from the pending `NAN_API_KEY` rotation.
