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
- [ ] **`--count` must be able to issue a `/status` and then item reads**, because
      that is the actual reproducer. `GET /status` poisons the daemon's item-read
      path for roughly half a second: 10 item reads before it returned 200×10, and
      10 immediately after returned 500×10 (measured twice, independently, on
      2026-08-15). Upstream bitwarden/clients#20951, a switchMap/ReplaySubject
      disposal race.
- [ ] **Superseded rationale, kept deliberately.** An earlier draft of this task
      required reusing one client across iterations, on the theory that the fault
      tracked connection reuse. That was **falsified**: `DisableKeepAlives` over
      360 requests moved failures 35.0% → 32.8%, i.e. not at all. A rival theory
      of mine — concurrency — was falsified too (24 requests at each of 1/2/4/8-way
      parallelism: 96/96 clean). Both survived a first test and died on a second.
      The note stays because the shape of the error is the reusable lesson: **the
      measurement was the cause.** Every attempt to observe the daemon's health
      called `/status` first and damaged the next read, so three sessions'
      contradictory numbers were all correct and all sampling one half-second
      window from different distances.
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
