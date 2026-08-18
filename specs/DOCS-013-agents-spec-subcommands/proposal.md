---
id: "DOCS-013-agents-spec-subcommands"
type: spec
status: draft
created: "2026-08-18"
owner: manu
issue: "mlorentedev/dotfiles#1016"
tags: [spec, proposal, docs, cli]
template_version: "1.0"
---

# DOCS-013: AGENTS.md Spec Subcommands Reconciliation

> **Naming**: file lives at `<repo>/specs/DOCS-013-agents-spec-subcommands/proposal.md`.

## Why

`AGENTS.md` § Spec-Driven Development (lines 167-169) previously instructed agents to run `dotf spec fill` (which does not exist in the binary) while omitting `dotf spec review` (which exists and is required by the archive gate). Because `AGENTS.md` is the cross-agent canonical prompt read by every agent, this discrepancy caused agents trying to comply with SDD to hit execution errors or miss running required reviews.

## What

1. Reconciles `AGENTS.md` to clearly distinguish between **binary subcommands** (`dotf spec init`, `dotf spec review`, `dotf spec archive`) and **skill workflow steps** (`/spec fill`, `/spec check`, `/spec bootstrap`).
2. Adds a drift-prevention unit test `TestSpecSubcommandsProseMatchesCode` in `cli/internal/spec/` that parses `AGENTS.md` and verifies every documented `dotf spec <sub>` subcommand matches the actual Cobra command tree.

## Out of scope

- Implementing new binary subcommands under `dotf spec`.
- Large structural refactoring of `AGENTS.md` outside § Spec-Driven Development.

## Risks / open questions

- None. This is a documentation-code reconciliation backed by a deterministic test.

## Acceptance criteria

- [ ] `AGENTS.md` accurately documents `dotf spec` CLI subcommands (`init`, `review`, `archive`) and distinguishes skill-only steps.
- [ ] `TestSpecSubcommandsProseMatchesCode` passes on clean `AGENTS.md`.
- [ ] Mutation test demonstrates `TestSpecSubcommandsProseMatchesCode` goes RED when an invalid subcommand is inserted into `AGENTS.md`.

## References

- Bitácora work-gate: [mlorentedev/dotfiles#1016](https://github.com/mlorentedev/dotfiles/issues/1016)
- Precedent test: `TestIDPatternProseMatchesCode` in `cli/internal/spec/drift_test.go`

- Related patterns: `00_meta/patterns/<pattern>.md` (if any)
