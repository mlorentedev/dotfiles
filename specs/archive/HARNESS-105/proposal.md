---
id: "HARNESS-105"
type: spec
status: archived
created: "2026-08-30"
issue: "mlorentedev/dotfiles#1403"
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-105: Refactor shellQuote and implement JSON Error Latches

## Why

Our CLI currently relies on ad-hoc string replacements for shell quoting and plaintext error outputs for terminal failures. This leads to brittle command executions and unbounded agent retries when faced with unparseable errors. Implementing deterministic `shellsafe` functions and structured JSON error latches provides structural guarantees that prevent shell injection and agent hallucination.

## What

1. A new `cli/internal/shellsafe` package exposing `Bash(string)` and `PowerShell(string)` quoting utilities.
2. Refactoring of existing shell string replacements in the codebase to use the new package.
3. A new `TerminalFailure` error type in the CLI orchestrator that, upon return, forces the emission of a strict JSON payload (a "latch") instructing the agent to cease retrying (`retryGuidance: "Do not retry"`).

## Out of scope

- Refactoring every single `fmt.Errorf` in the codebase; the JSON latch is only for critical CLI terminal failures (like orchestrator failures or SDD halts).
- Implementing full Organic Routing based on file count thresholds (that is covered by HARNESS-104).

## Risks / open questions

- **Risk**: Agents might not understand the JSON format if not configured to read it. **Mitigation**: We use a very clear `retryGuidance` field in English that LLMs can naturally parse.

## Acceptance criteria

- [x] `shellsafe.Bash` and `shellsafe.PowerShell` are implemented and have 100% unit test coverage.
- [x] Ad-hoc replacements (e.g. in `review_launch.go`) are replaced by `shellsafe` functions.
- [x] A `TerminalFailure` error type exists and is correctly handled by the CLI's main execution or top-level command handlers.
- [x] Throwing a `TerminalFailure` results in a structured JSON output (with `schemaName` and `retryGuidance`) instead of standard text.

## References

- Bitácora board: mlorentedev/dotfiles#1403

## Archive provenance — disclosed 2026-08-31

This spec archived **without a passing review on disk**, and the only review
artifact it produced records **FAIL**. Reconstructed from git, not remembered:

| Commit | What it did |
|---|---|
| `b63eef1` | the implementation; `review-request.json` names it as `reviewed_sha` |
| `dd47c5d` | **archived the spec** — no `review.md` accompanied it into `specs/archive/` |
| `b3fcd17`, `150a048`, `76aa61a`, `70e15af` | fixes for the Blocker below, all **after** the archive |

`review.md` was written by a **second** review round pointed at `dd47c5d` — the
archive commit itself — and returned FAIL on one REAL Blocker: a wrapped
`TerminalFailureError` printed the wrapper prefix before the JSON latch, breaking
the strict-prefix parsing orchestrators rely on. Because `dotf spec review`
writes to `specs/<id>/review.md` without checking whether the spec has already
been archived, that write **recreated `specs/HARNESS-105/`** as a ghost directory
holding one file. Everything that enumerates `specs/*/` counted this spec as
active from then on. `spec.Archive` cannot cause this split — it moves the whole
directory with `os.Rename` — so the writer that arrived afterwards is the cause.
Filed against GUARD-005 (#1157), whose `review-request.json` sidecar is the
adjacent half of the same machinery.

**The Blocker is fixed in `main`, verified rather than assumed:**
`cli/cmd/dotf/main.go` unwraps with `goerrors.As(err, &tfe)` before printing, so
the latch survives wrapping. No re-review was ever run against the fixed tree,
which is why this note exists instead of a PASS verdict.

`review.md` was moved here **byte-for-byte unmodified**
(`sha256:dd7fe1c7e08d7118eb61162d2037d3ff30d8bd1218b7abb94b139fe59930d59d`). It
sits beside its own `review-request.json`, which until now recorded a review
having been *requested* with no verdict attached — a less complete record than
the FAIL it was missing. Moving it neither ratifies nor retracts the verdict; it
puts the evidence where the spec's own history is.
