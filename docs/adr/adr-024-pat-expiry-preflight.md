---
id: "ADR-024-pat-expiry-preflight"
type: adr
status: accepted
owner: manu
date: "2026-06-18"
issue: "dotfiles#422"   # OPS-009 — the spec that implements this decision
tags: [architecture, decision, secrets, ci, pat, doctor]
created: "2026-06-18"
---

# ADR-024: PAT-expiry preflight — two surfaces, each with its own probe

## Status

Accepted

## Date

2026-06-18

## Context

A classic GitHub PAT (`github.token`, backing `RELEASE_TOKEN` / `GITHUB_PERSONAL_ACCESS_TOKEN`) expired silently and broke release-please's first run with `Bad credentials` (CLI-011, #369). Nothing warned before expiry — it surfaced only as a red CI run. The existing `dotf doctor` `checkSecrets` section validates *structural* integrity (every `env-mapping.conf` entry points to an existing `.age` blob, no orphans) but is blind to whether a token is still **alive** or **about to expire**. OPS-009 (#422) closes that blind spot.

Two facts shaped the design:

1. **There are two distinct consumers.** A *local developer* benefits from an at-a-glance liveness check during a full diagnostic run; *CI* needs a proactive alert that lands a rotation reminder in the backlog **before** the next release run goes red. The local check only helps when a human runs it.
2. **`Report.ExitCode()` reflects only `StatusFail`.** `WARN`/`SKIP`/`INFO` are advisory and never move the exit code (the healthcheck/doctor exit contract, confirmed by reading `report.go`). "Expiring soon" is a **WARN** — so it is invisible to anything that reads `dotf doctor`'s exit status.

## Options Considered

Constraints: **C1** detect expiry proactively (not post-outage) · **C2** the local full-sweep stays usable offline and is skipped on the latency-sensitive `--quick` SessionStart path · **C3** the CI surface must alert on *expiring-soon*, not just *dead* · **C4** no real network in unit tests · **C5** GitHub-specific header trick is acceptable (non-GitHub PATs are out of scope).

| Option | C1 | C3 | Notes |
|---|---|---|---|
| **A** — `dotf doctor` check only | ok | **gap** | local-only; never fires in CI; the outage class survives |
| **B** — scheduled Action shells out to `dotf doctor` and reads `$?` | ok | **gap** | exit code is 0 for WARN — sees only already-dead tokens (the outage), never "expiring soon" |
| **C** — two surfaces, each with its own probe | ok | ok | local check (Go, behind the `System` seam) + scheduled Action (independent ~15-line shell probe) |

## Decision

**Option C — two surfaces, deliberately not sharing the probe.**

- **`dotf doctor` — local surface.** A new `checkPATExpiry` section enumerates the unique `github.*` PAT-backed secrets from `env-mapping.conf`, probes `GET /user`, reads the `github-authentication-token-expiration` header, and classifies: invalid/expired → **FAIL**; expiring within `DOTF_PAT_EXPIRY_WARN_DAYS` (default 14) → **WARN**; healthy → **PASS**; env-unset → **SKIP**; offline → **WARN**. Network + clock are isolated behind two new `System` seam members (`HTTPGet`, `Now`) so the table tests run fully offline (C4). It runs in the full sweep only, never under `--quick` (C2).
- **Scheduled Action — CI-prevention surface.** `pat-expiry.yml` (weekly cron + `workflow_dispatch`) runs its **own** probe of `RELEASE_TOKEN` and `BITACORA_PAT` and opens/updates a deduplicated `pat-expiry`-labelled issue when a token is invalid or within the threshold.

The two surfaces **do not share code**. The Action does NOT shell out to `dotf doctor`, because the binary's exit code cannot express "expiring soon" (it is a WARN). The ~15-line shell duplication is the cost of keeping the CI alert able to fire on the state that actually matters (C3).

## Consequences

### Positive

- The outage class (expired PAT → red release run) is caught proactively, before it bites.
- The local check is fully unit-testable offline via the `System` seam — the first network-touching check in the `doctor` package, and the seam absorbs it without a real socket.
- The Action alerts without red-failing unrelated pipelines (it opens an issue, deliberately not a CI failure).

### Negative

- **Duplicated date math** (Go check vs shell workflow). Accepted and documented: forcing the Go binary into CI just to read one header would couple more than the ~15 lines cost, and the binary couldn't signal WARN anyway.
- GitHub-specific: the expiry-header trick does not generalise to non-GitHub PATs (`dockerhub.token`, …) — explicitly out of scope.

### Neutral

- The `System` seam grew `HTTPGet` + `Now`. Threading `context.Context` through the seam (the idiomatic Go I/O-cancellation rule in AGENTS.md) was deferred to a follow-up refactor — the 5s client timeout bounds the only call, and adding context to one seam method while the others stay context-free would be an inconsistent partial change.
- The check is a sibling of `checkSecrets`: structural integrity vs liveness/expiry, side by side.

## References

- OPS-009 (#422) — the spec implementing this decision (`specs/archive/OPS-009-pat-expiry-preflight/`)
- ADR-021 — the healthcheck/doctor Go consolidation this check extends
- CLI-011 (#369) — release-please adoption; the incident that motivated this
- PRs: #427 (feature), #429 (alias-resolution + review follow-ups)
- Lesson: "A WARN that doesn't move the exit code is invisible to CI" (`docs/lessons.md`, 2026-06-18)
- Candidate vault pattern: `secrets-rotation` (cross-repo)
