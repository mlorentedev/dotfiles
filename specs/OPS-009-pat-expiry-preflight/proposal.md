---
id: "OPS-009-pat-expiry-preflight"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-17"
issue: "dotfiles#422"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, ops, secrets, ci, pat, doctor]
template_version: "1.0"
---

# OPS-009-pat-expiry-preflight

> **Naming**: file lives at `<repo>/specs/OPS-009-pat-expiry-preflight/proposal.md`. Implements the chosen approach for issue #422 (detect expiring/expired PAT secrets before they break CI).

## Why

<!-- from issue #422 -->

A classic PAT expired silently and broke release-please's first run with `Bad credentials` (the `github.token` PAT backing `RELEASE_TOKEN` / `GITHUB_PERSONAL_ACCESS_TOKEN`). **Nothing warned before expiry** — it surfaced only as a red CI run during the release-please adoption (CLI-011, #369). The existing `checkSecrets` doctor section validates *structural* integrity (every `env-mapping.conf` entry points to an existing `.age` blob, no orphans) but says nothing about whether a token is still **alive** or **about to expire**. That liveness/expiry blind spot is exactly what bit us; this ticket closes it so rotations become proactive, not post-outage.

## What

Add a PAT-expiry preflight across **two surfaces**, each idiomatic to its runtime:

1. **`dotf doctor` — local developer surface.** A new `checks_pat.go` section ("PAT expiry") that enumerates the GitHub PAT-backed secrets from `env-mapping.conf` (filename prefix `github.`), reads each token from the environment (already exported by `load-secrets`), probes `GET https://api.github.com/user`, and reads the `github-authentication-token-expiration` response header. Classification: invalid/expired → **FAIL** (drives exit 1, the dead-token case that broke CI); expiring within the threshold → **WARN** (advisory, rotate soon); valid with comfortable runway → **PASS**; token not in env → **SKIP** (fresh shell, no alarm); network unreachable → **WARN** (offline, not a setup failure). Threshold defaults to **14 days**, overridable via `DOTF_PAT_EXPIRY_WARN_DAYS`. The check runs in the full sweep only — **skipped in `--quick`** (the SessionStart hot path must stay fork-free and offline).

2. **Scheduled GitHub Action — the "before it breaks CI" surface.** `.github/workflows/pat-expiry.yml`, weekly cron + `workflow_dispatch`, probes the `RELEASE_TOKEN` and `BITACORA_PAT` Actions secrets the same way and, when a token is invalid or within the threshold, **opens or updates a deduplicated tracking issue** (stable label `pat-expiry`) so a rotation reminder lands in the backlog *before* the next release run goes red. This is the surface that actually prevents the outage class, since `dotf doctor` only helps when a human runs it locally.

Both surfaces lean on two new `System` seam members — `HTTPGet` (the network is precisely the "non-deterministic external surface" `system.go` says belongs behind the seam) and `Now` (a clock seam, so "days until expiry" is deterministic under test).

## Out of scope

- **Non-GitHub PATs** (`dockerhub.token`, `cloudflare.api-token`, …) — the `github-authentication-token-expiration` header trick is GitHub-specific. Generalising expiry detection to other providers is a follow-up.
- **Automatic rotation / Bitwarden round-trip** — HARNESS-022 (#378). This ticket *detects*; it does not rotate.
- **Per-purpose token convention** — OPS-007 (#321) owns the "one PAT for everything is an antipattern" policy.
- **Fine-grained vs classic PAT migration** — orthogonal; the check handles whichever the token is, by reading the same header.
- **Failing CI on expiry** — the scheduled job *alerts* (opens an issue); it deliberately does not red-fail unrelated pipelines.

## Risks / open questions

- **R1 — network inside `dotf doctor` adds latency + offline flakiness.** A blocking HTTP GET on every full `dotf doctor` could hang offline. **Mitigation:** 5s client timeout; network-error → **WARN** (never FAIL); the check is **skipped in `--quick`**, which is the only latency-sensitive path (SessionStart). The full sweep already runs many `<tool> --version` probes + the ~2.8s harness-drift gate, so one bounded GET is in budget.
- **R2 — token env var absent in a fresh shell.** If `secrets_refresh` has not run, `GITHUB_PERSONAL_ACCESS_TOKEN` is empty and a naive check would scream. **Mitigation:** env-unset → **SKIP** with a hint ("run `secrets_refresh`"), never FAIL.
- **R3 — expiry header format.** The live classic PAT returns `github-authentication-token-expiration: 2026-09-15 07:11:31 UTC`. **Mitigation:** parse that exact layout (`2006-01-02 15:04:05 MST`) with an ISO-8601 fallback; an **absent** header (non-expiring token) → PASS, not an error.
- **R4 — the Action needs a token to open issues, distinct from the ones it checks.** **Mitigation:** the workflow uses the default `GITHUB_TOKEN` (has `issues: write`) for the `gh issue` calls; the secrets under test are used *only* as the `Authorization: Bearer` for the liveness probe.
- **R5 — duplicated date math (Go check vs shell workflow).** Both compute "days until expiry". **Mitigation / accepted:** they run in different contexts (local shell env vs Actions secret), the shell side is ~15 lines, and forcing the Go binary into CI just to read one header would couple more than the duplication costs. Documented as deliberate.
- **R6 — issue spam.** A weekly cron could open a new issue every run. **Mitigation:** dedupe on a fixed `pat-expiry` label + stable per-token title; update the existing open issue instead of opening duplicates; close/comment when the token is healthy again is a nice-to-have, not required.
- **R7 — WARN is invisible to exit codes.** `Report.ExitCode()` is non-zero only on FAIL, so "expiring soon" (WARN) cannot be detected by the Action via `dotf doctor`'s exit status. **Consequence:** this is *why* the Action does its own focused probe rather than shelling out to the binary — confirmed by reading `report.go`.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1** — A full `dotf doctor` run prints a "PAT expiry" section that enumerates each *unique* `github.*` PAT-backed secret once (so `github.token`, mapped by both `GITHUB_PERSONAL_ACCESS_TOKEN` and `RELEASE_TOKEN`, is probed a single time). *Verify:* table test asserting one probe per filename + manual run.
- [ ] **AC2** — Classification is correct on every branch: HTTP 401 → FAIL (exit 1); expiry ≤ threshold → WARN; valid + runway → PASS; token env-unset → SKIP; network error → WARN. *Verify:* `checks_pat_test.go` table tests, one row per branch, using a fake `System`.
- [ ] **AC3** — The threshold defaults to 14 days and is overridable via `DOTF_PAT_EXPIRY_WARN_DAYS`. *Verify:* tests with the env set to a value that flips a fixed-clock case from PASS to WARN.
- [ ] **AC4** — The check is **not** invoked under `--quick`. *Verify:* a test runs `Options{Quick:true}` with a fake `HTTPGet` that records calls and asserts zero calls.
- [ ] **AC5** — `.github/workflows/pat-expiry.yml` exists, triggers on `schedule` + `workflow_dispatch`, probes `RELEASE_TOKEN` and `BITACORA_PAT`, and opens/updates a `pat-expiry`-labelled issue when a token is invalid or within the threshold. *Verify:* `actionlint`/yaml parse + a recorded `workflow_dispatch` dry-run in verification.md.
- [ ] **AC6** — `System` gains `HTTPGet` + `Now`; `realSystem()` wires `net/http` (with timeout) + `time.Now`; unit tests make **no real network call**. *Verify:* `grep` for the seam members + `go test ./...` passes with networking unavailable.
- [ ] **AC7** — The existing doctor sections and the full `bats` + `go test` suites stay green; no behavioural change outside the new section. *Verify:* `go test ./...` + `bats tests/`.

## References

- Issue: dotfiles#422 (OPS-009)
- Incident: release-please first run `Bad credentials` (CLI-011 / #369 adoption); rotation persisted in PR #423
- Code: `cli/internal/doctor/system.go` (seam), `cli/internal/doctor/report.go` (Status/ExitCode contract), `cli/internal/doctor/checks_deploy.go` (`checkSecrets`, the structural sibling), `cli/internal/doctor/doctor.go` (sweep wiring)
- Related: OPS-007/#321 (per-purpose token convention), HARNESS-022/#378 (Bitwarden bidirectional secrets), ADR-021 (the healthcheck/doctor Go consolidation)
- Candidate vault pattern: `secrets-rotation` (cross-repo — every repo with PAT-backed Actions secrets)
