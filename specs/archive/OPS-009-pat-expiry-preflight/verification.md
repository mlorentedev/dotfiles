---
tags: [spec, verification, ops, secrets, pat, doctor, ci]
created: "2026-06-17"
---

# Verification - OPS-009-pat-expiry-preflight

> Status: **implementing** — both surfaces landed and statically verified. The Go
> `dotf doctor` section is fully unit-covered offline; the scheduled Action's
> end-to-end behaviour (AC5: issue open/update) is observable only after merge via
> a `workflow_dispatch` run, recorded below once available.

## Evidence

- [x] **AC1** (one probe per unique `github.*` filename) → `TestCheckPATExpiry_ProbesEachFilenameOnce`: `github.token` mapped by both `GITHUB_PERSONAL_ACCESS_TOKEN` and `RELEASE_TOKEN`, plus a non-github `dockerhub.token`; asserts the recording `HTTPGet` is called **exactly once**. `TestCheckPATExpiry_FallsBackToSecondAlias` further asserts that when only the *second* alias is exported the token is still probed (no false SKIP) — `patSecret` keeps every alias and resolves the first non-empty one.
- [x] **AC2** (classification on every branch) → `TestCheckPATExpiry_Classification` table, one row per branch: HTTP 401 → FAIL (`Failures()==1`); expiry ≤ threshold → WARN; valid+runway → PASS; token env-unset → SKIP; network error → WARN; plus 200/no-header → PASS, unexpected 500 → WARN, at/past-expiry-but-200 → WARN.
- [x] **AC3** (threshold default 14, override via `DOTF_PAT_EXPIRY_WARN_DAYS`) → table rows: a fixed 20-day runway is **PASS** at the default 14 and flips to **WARN** under `DOTF_PAT_EXPIRY_WARN_DAYS=30`; a non-numeric override (`abc`) falls back to 14 and emits a WARN.
- [x] **AC4** (not invoked under `--quick`) → `TestCheckPATExpiry_QuickSkipsProbe`: `Run(Options{Quick:true})` with a token in env and a recording `HTTPGet` → **zero** HTTP calls.
- [~] **AC5** (`pat-expiry.yml` exists, scheduled + dispatch, probes both secrets, opens/updates a labelled issue) → file present; `actionlint` clean; YAML parses; embedded `run` block `bash -n` clean. **End-to-end issue create/update pending** a post-merge `workflow_dispatch` run (needs the secrets, which live in repo settings).
- [x] **AC6** (`System` gains `HTTPGet`+`Now`; `realSystem()` wires `net/http`+`time.Now`; no real network in tests) → `system.go` seam members present and wired with a 5s-timeout client; `go test ./...` passes with the test `HTTPGet` returning a canned/offline response (no socket opened).
- [x] **AC7** (existing sections + full suites stay green) → `go test ./...` all packages `ok`; the new section only appends after `checkSecrets`.

## Test status

- `go test ./...` (from `cli/`) → all packages `ok` (`internal/doctor` included).
- `go test ./internal/doctor/ -run TestCheckPATExpiry -v` → 5 test functions (11 classification subtests + 4 scenario tests), all PASS.
- `go vet ./internal/doctor/` → clean (the two `unused parameter: sys` infos are pre-existing on `loadContractSection`/`checkSecrets`, not this change).
- `actionlint .github/workflows/pat-expiry.yml` → clean.
- `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/pat-expiry.yml'))"` → valid; `bash -n` on the extracted `run` block → clean.
- **Pending (user / post-merge):** `bats tests/` full suite; a real `dotf doctor` run against a live token (manual smoke); `gh workflow run pat-expiry.yml -f warn_days=3650` to force the alert path and confirm one `pat-expiry` issue per flagged secret (then re-run to confirm it *comments* rather than duplicates — R6 dedupe).

## Decisions made during implementation

- **Probe endpoint:** `GET /user` — the cheapest authenticated REST call; its response carries `github-authentication-token-expiration` for any token that has an expiry. An absent header (non-expiring token) → PASS, not an error (R3).
- **Two seam members, not one:** `HTTPGet` (network) **and** `Now` (clock). "Days until expiry" must be deterministic under test, so the clock is injected; the fixed `2026-06-17` test clock makes the header offsets in the table exact.
- **WARN never escalates after a 200:** a token that just authenticated is not dead, so the worst outcome past a 200 is "rotate soon" (WARN) — only HTTP 401 (or env-unset → SKIP) is reachable as FAIL. This is the contract that keeps offline/transient states from breaking a `dotf doctor` exit code (R1, R7).
- **Action does its own probe (no shell-out to the binary):** `Report.ExitCode()` is non-zero only on FAIL, so WARN ("expiring soon") is invisible to exit status — `dotf doctor` could not signal "expiring" to CI. The workflow re-implements the ~15-line probe instead; the duplication is deliberate and documented (R5/R7).
- **Issue dedupe:** a fixed `pat-expiry` label + stable per-secret title (`PAT expiry: <NAME>`); an existing open issue is **commented**, not duplicated, so a weekly cron never spams (R6). The label is created idempotently (`gh label create --force`) before the first `gh issue create`.
- **Injection-safe workflow:** every `${{ }}` (the `warn_days` dispatch input, the three secrets, `GITHUB_TOKEN`) is bound through the job `env:` block and consumed as a quoted `${VAR}` in `run:`; no untrusted event field is interpolated into the script. Token names are hardcoded literals.
- **Multi-alias resolution (post-review):** CodeRabbit flagged that the original dedupe kept only the first env alias per `.age` filename, so a shell with only `RELEASE_TOKEN` (not `GITHUB_PERSONAL_ACCESS_TOKEN`) set would falsely SKIP `github.token`. Fixed: `patSecret.envVars` now holds all aliases and `resolvePATToken` probes the first non-empty one. `checkPATExpiry` was also split into `probePATSecret` + `reportPATExpiry` to stay within the AGENTS.md < 40-line / < 10-complexity budget, and the `realSystem` `HTTPGet` errors are now wrapped with the URL.
- **Deviation — `context.Context` not threaded through `System` (AGENTS.md "propagate context in all I/O"):** the 5s client timeout already bounds the only network call, and `doctor.Run` has no cancellable parent context; adding it to `HTTPGet` alone (while `LookPath`/`CommandOutput` stay context-free) would be an inconsistent partial change. Deferred to a seam-wide refactor — see promotion candidate below.

## Promotion candidates

- [ ] Vault pattern `secrets-rotation` (cross-repo): "every repo with PAT-backed Actions secrets needs an expiry preflight — detect before the red CI run, dedupe the reminder by label+title". Generalises beyond dotfiles.
- [ ] Lesson for `docs/lessons.md`: "a WARN that doesn't move the exit code can't be observed by CI via the tool's status — give the CI surface its own probe rather than shelling out" (the R7 generalisation).
- [ ] Follow-up ticket (REFACTOR): thread `context.Context` through the whole `doctor.System` seam (`HTTPGet` + `CommandOutput`) and propagate from `doctor.Run`, satisfying the AGENTS.md I/O-context rule consistently rather than per-method.

## Archive checklist

- [ ] `proposal.md` frontmatter → `status: archived`
- [ ] Folder moved to `specs/archive/OPS-009-pat-expiry-preflight/`
- [ ] Issue #422 closed by the PR
- [ ] AC5 end-to-end (`workflow_dispatch` issue create/update) observed + recorded above before archiving
