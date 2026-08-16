---
id: "CLI-038-secrets-probe"
type: spec
status: draft
created: "2026-08-16"
issue: "mlorentedev/dotfiles#1012"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — CLI-038-secrets-probe

TDD order: the safety tests come first, because they are the feature. A probe
that reports well but leaks once has failed completely, so the tests that pin
"no value ever reaches output" are written before any output exists to check.

## 0. Seam (other lane, blocking for 3+)

- [ ] `BWServeClient.Probe(method, path) (ProbeResult, error)` landed in
      `cli/internal/secrets/bwserve.go` by the lane that owns that file. Signature
      agreed 2026-08-15; carries `Status`, `ContentType`, `Size`, `Body`, uses the
      same `httpClient()`/`baseURL()` as `call()`, returns `ErrBWServeUnreachable`
      wrapped on transport failure with `Status 0`, and treats a non-2xx as a
      successful observation rather than an error.
- [ ] Confirm no error path in that method can carry body bytes.

## 1. Safety tests (red first — these ARE the feature)

- [ ] Test: a 2xx item body containing a sentinel value never appears in probe
      output. Default mode.
- [ ] Test: the same, with `--raw` set. This is the flag that could regress the
      whole ticket, so it is pinned separately rather than folded into the above.
- [ ] Test: a non-2xx body IS shown under `--raw`, capped to the bound.
- [ ] Test: a non-2xx body is NOT shown without `--raw`.
- [ ] Test: fingerprints are 12 hex chars and differ for differing values, match
      for identical ones — the property that makes rotation verifiable.

## 2. Report shaping (pure, no I/O)

- [ ] Extract the response → report transformation as a pure function over
      `ProbeResult`, so every case above is table-testable without a server.
- [ ] Emit: status, content-type, size, envelope validity (`json.Valid`),
      `success`/`message`, field names, value lengths, `sha256[:12]`.
- [ ] Malformed/non-JSON body: report validity false and the size, never the bytes
      (outside `--raw` + non-2xx).
- [ ] Item with no fields, and a field with an empty value: no panic, no blank
      fingerprint that could read as "unset" when it is "empty".

## 3. Wiring

- [ ] `probe.go` in `cli/internal/secrets/` — new file, calls the seam.
- [ ] `dotf secrets probe <secret-id>` resolves the id through the registry, not
      an arbitrary URL. Rejects an unknown id with the same error shape the rest of
      the facade uses.
- [ ] `--raw`, `--count N` flags.
- [ ] `--count N`: report the outcome distribution (status → occurrences); no
      value output regardless of N.
- [ ] **`--count` MUST reuse one client across all N iterations.** Not a
      performance choice — it decides whether the flag works at all. Measured in
      the `cli/internal/secrets/` lane (2026-08-15): 24 requests at each of
      1/2/4/8-way parallelism gave 96/96 clean, while the *same* 6-request
      keep-alive chain gave 2, then 1, then 0 × HTTP 500 across three consecutive
      runs. So the failure tracks connection reuse, and a fresh client per
      iteration would sample only the always-clean case — `--count` would then
      report the bug as ABSENT, which is worse than not having the flag: it would
      be evidence for the wrong conclusion. A test pins that N iterations issue
      through one client.
- [ ] Read-only: assert no unlock/sync/set/rotate call exists on this path.

## 4. Verification

- [ ] `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` at the
      `versions.conf` pin.
- [ ] Live run against the real daemon, output captured into `verification.md`.
      Safe by construction if the ACs hold — and that demonstration IS the evidence
      the ticket asks for.
- [ ] Live run against a deliberately wrong id, to show the failure path.
- [ ] `--count 10` against the real daemon: the distribution output is the artifact
      #988 needed and nobody could produce safely.

## 5. Knowledge

- [ ] `docs/lessons.md`: name this command as the sanctioned probe on the existing
      redaction entry, replacing the prose-only rule. (PR #1013 amends that entry
      with the third instance; sequence after it to avoid a conflict.)
- [ ] Close #1012 with the command that closed it.

## Out of scope reminders

Fixing #988; changing `call()`'s no-body contract (#1007); any write path; a
general-purpose HTTP client; the daemon cache staleness (#1015).
