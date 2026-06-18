---
tags: [spec, tasks, ops, secrets, pat]
created: "2026-06-17"
---

# Tasks - OPS-009-pat-expiry-preflight

> TDD order: extend the seam, write the table tests against a fake `System`, then make them pass. One task = one focused commit where practical.

## Setup

- [x] Branch created from main: `feat/ops-009-pat-expiry-preflight`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Scope confirmed with user: **both** surfaces (`dotf doctor` check + scheduled Action); threshold **14 days** (env `DOTF_PAT_EXPIRY_WARN_DAYS`)
- [x] Live header format captured from the rotated token: `github-authentication-token-expiration: 2026-09-15 07:11:31 UTC`

## Implementation — `dotf doctor` check (Go)

- [x] Extend `System` (`system.go`) with `HTTPGet(url string, headers map[string]string) (int, http.Header, error)` and `Now() time.Time`; wire `realSystem()` to `net/http` (5s timeout client) + `time.Now`
- [x] `checks_pat.go`: `checkPATExpiry(sys, cfg, rep)` — parse `env-mapping.conf`, select `github.*`-backed mappings, dedupe by filename, resolve a representative env var per token, probe `GET /user`, parse the expiry header, classify (SKIP/WARN/FAIL/PASS) against the threshold
- [x] Threshold helper: read `DOTF_PAT_EXPIRY_WARN_DAYS` via `sys.env`, default 14; reject non-numeric → default + WARN
- [x] Wire `checkPATExpiry` into `doctor.go`'s full sweep (after `checkSecrets`), **not** under `--quick`
- [x] `checks_pat_test.go`: table tests with a fake `System` (canned `HTTPGet` + fixed `Now` + `Getenv`) covering every branch in AC2/AC3/AC4; assert one probe per filename (AC1) and zero probes under `--quick`

## Implementation — scheduled Action

- [x] `.github/workflows/pat-expiry.yml`: `schedule` (weekly) + `workflow_dispatch`; probe `RELEASE_TOKEN` + `BITACORA_PAT`; on invalid or ≤ threshold, open/update a `pat-expiry`-labelled issue (dedupe by stable title) using the default `GITHUB_TOKEN`
- [x] `actionlint` clean; `workflow_dispatch` dry-run recorded in verification.md _(actionlint clean + YAML/bash -n verified; dispatch dry-run recorded as pending post-merge — needs the live secrets)_

## Closing

- [x] `go test ./...` green; existing doctor sections unaffected (AC7) — `bats tests/` left for the user to run
- [x] Each acceptance criterion mapped to an executable check in `verification.md`
- [x] `dotf doctor` help/Long text mentions the new section if the others are enumerated there
- [ ] PR opened referencing this spec folder + issue #422
