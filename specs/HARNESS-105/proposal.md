---
id: "HARNESS-105"
type: spec
status: draft
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

- [ ] `shellsafe.Bash` and `shellsafe.PowerShell` are implemented and have 100% unit test coverage.
- [ ] Ad-hoc replacements (e.g. in `review_launch.go`) are replaced by `shellsafe` functions.
- [ ] A `TerminalFailure` error type exists and is correctly handled by the CLI's main execution or top-level command handlers.
- [ ] Throwing a `TerminalFailure` results in a structured JSON output (with `schema` and `retryGuidance`) instead of standard text.

## References

- Bitácora board: mlorentedev/dotfiles#1403
