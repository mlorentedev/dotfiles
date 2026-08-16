---
id: "CLI-038-secrets-probe"
type: spec
status: verifying
created: "2026-08-16"
issue: "mlorentedev/dotfiles#1012"
tags: [spec, verification, secrets, security]
template_version: "1.0"
---

# Verification — CLI-038-secrets-probe

## AC1 — reports status, shape, field names, lengths and fingerprints

Live, against the real daemon and the real registry entry:

```
$ dotf secrets probe NAN_API_KEY
HTTP 200  application/json; charset=utf-8  670 bytes
envelope: success=true
values (never printed — length and fingerprint only):
  data.creationDate                len=24    60d0cd27c65c
  data.fields[0][api-key]          len=25    322dfb364a4e
  data.folderId                    len=36    871840ccbdfb
  data.id                          len=36    e085ac77e2dd
  data.key                         len=180   ffe5392e317a
  data.name                        len=11    6ec740532af0
  data.object                      len=4     4a33eacd5fa6
  data.revisionDate                len=24    60d0cd27c65c
```

`data.fields[0][api-key]` shows the field-label exception working: the label is
visible (the fact that diagnoses a mis-declared mapping, #990) while its value is
a fingerprint.

## AC2 — a sentinel value never appears in output, in any mode

`TestShapeProbe_NeverEmitsAValue` loops both `raw` states over a payload carrying
the sentinel in two places (`fields[].value` and `login.password`).

## AC3 — a 2xx body never prints, including with `--raw`

`TestShapeProbe_RawNeverShowsA2xxBody`. The bound lives in `ShapeProbe`, not at
the call site, so no caller can opt out of it.

## AC4 — `--raw` prints non-2xx bodies only, capped

`TestShapeProbe_RawShowsNon2xxBody` and `TestShapeProbe_RawBodyIsCapped` (10 000
bytes in, capped at 512, and the report says `truncated`).

**Not reproduced live, stated rather than implied.** The non-2xx path was not
exercised against the real daemon: after #1018 the daemon would not re-enter the
poisoned state, and the probe's own item-id resolution takes longer than the
~0.5 s window, so every live attempt returned 200. This AC rests on unit
coverage.

## AC5 — the probe goes through `BWServeClient`

`TestProbe_UsesTheSameTransportAsCall` asserts `Probe` and `call` present as the
same client. Without it the "same transport as the read path" claim is a comment
rather than a property — and it is the property that makes the tool worth having,
since a hand-rolled client cannot reproduce the condition being chased.

## AC6 — `--count N` reports a distribution, prints no value

```
$ dotf secrets probe NAN_API_KEY --raw --count 6
6 probes, one client, in order:
  HTTP 200  6/6
```

## AC7 — read-only

No unlock, sync, set or rotate on this path. `--count 0` and `--count -5` are
refused rather than silently doing nothing:

```
$ dotf secrets probe NAN_API_KEY --count 0
Error: --count must be at least 1, got 0
```

## AC8 — `docs/lessons.md` names this as the sanctioned probe

Recorded on the existing redaction entry.

## Toolchain

`go build ./...`, `go vet ./...`, `go test ./internal/...` and
`golangci-lint run ./internal/...` (v2.12.2, the `versions.conf` pin) all clean,
re-run after merging `origin/main` at `e66120f` so the result reflects the
backend-pinning changes that landed underneath this branch.

## What the tool found on its first live run

This is the evidence that the ticket was worth building, and it was not planned.

A credential had been reported rotated. Probing it compared the stored
fingerprint against the value that had leaked earlier the same day:

```
fingerprint of the leaked value : 322dfb364a4e
fingerprint stored in the vault : 322dfb364a4e
```

Unchanged after an explicit `POST /sync` returning HTTP 200, so not the
stale-cache path — and `~/.pi/agent/models.json` holds the same fingerprint. Both
stores still carried the leaked value; the rotation had not reached either.

**A liveness probe would have reported success.** An unrevoked old credential
authenticates exactly as well as a new one, so "does it work?" answers yes for a
rotation that never happened. Only "did it change?" catches it, and answering
that used to require printing the value — which is how the leak occurred in the
first place. The tool answered it without printing anything.

## Defects found in this tool, by this tool

Both are the failure class the ticket exists to close, reproduced inside its own
implementation, and both were found by probing the command with the inputs that
mean "do nothing":

1. `--count 0` ran the loop zero times and exited 0 — success reported for work
   never done.
2. `--raw --count N` silently ignored `--raw`, in the one combination where the
   flag earns its keep.

Fixed and pinned by `TestProbeCmd_NonPositiveCountIsRefused` and the multi-probe
reporting path. Worth recording as a pattern: the happy path gets written and
tested first, and the branches where nothing happens are the ones nobody looks
at.

## Adversarial review disposition

**PASS** — `nan/deepseek-v4-flash`, `reviewed_sha 39689e1`, 5 findings, all Minor,
none blocking. Every one applied rather than deferred; the code and tests changed,
the spec contract files did not, so the review is not invalidated (staleness is
scoped to `proposal.md`, `tasks.md`, `features.json`).

| # | Reality | Finding | Disposition |
|---|---|---|---|
| 1 | REAL | `is2xx`/`isSuccess` duplicated across packages | **Applied** — exported `secrets.Is2xx`, deleted the cmd copy. Two callers must not drift about which bodies are safe. |
| 2 | REAL | `ProbeItemID` holds a credential-bearing search body; safe by scoping, not structurally | **Applied** — `TestProbeItemID_ReturnsOnlyTheIDNeverBodyBytes` pins that only the id leaves, on the success, not-found and ambiguity paths. |
| 3 | THEORETICAL | non-string `value` in the field-label path matched no case and vanished | **Applied, and it was a real gap.** A `{"name":"enabled","value":true}` field disappeared from a report whose job is to describe shape. `nil` and default branches added, reporting the TYPE and never the value. |
| 4 | THEORETICAL | field labels are printed verbatim on domain knowledge, not validation | **Applied as documentation, in the code rather than the spec.** The assumption now sits at the exception it governs. Deliberately not heuristically validated: guessing what "looks secret" would trade certain diagnostic power for an unreliable filter. |
| 5 | SPECULATIVE | cmd tests covered only refusals, so a wiring regression would pass | **Applied** — added a `probeClient` seam (matching this package's existing seam idiom) and a positive test driving `--raw --count 4` against an httptest server that alternates 200/500, asserting the distribution, the non-2xx body under `--raw`, and that the sentinel never appears. |

Finding 3 is the one worth keeping: it is the same shape as the two defects found
earlier in this command — the branches where nothing happens are the ones nobody
looks at. Here the walker had a case for every type it expected and no case at
all for the rest, so the unexpected input produced silence instead of a report.
