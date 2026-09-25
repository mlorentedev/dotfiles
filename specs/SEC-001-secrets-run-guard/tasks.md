---
tags: [spec, tasks, templates]
created: "2026-09-02"
---

# Tasks - SEC-001-secrets-run-guard

## Setup

- [x] Branch created: feat/secrets-run-guard
- [x] proposal.md is complete and acceptance criteria are testable
- [x] No open questions left in proposal.md

## Implementation

- [x] [AC1, AC2] Implement assertSafeChildCommand in cli/internal/cmd/secrets.go
- [x] [AC3] Implement shell wrapper introspection inspection in cli/internal/cmd/secrets.go
- [x] [AC4, AC5] Add unit test suite TestAssertSafeChildCommand in cli/internal/cmd/secrets_test.go
- [x] [AC6] Update ai/claude/settings.json permissions.deny list
- [x] [AC6] Register openrouter provider in ai/pi/models.json
- [x] [AC7] Implement stream-level output redactor redactWriter and TestRedactWriter
- [x] [AC8] Document lesson 261 in docs/lessons/
- [x] [AC9] Implement secrets show solutions 1-2-3 (reveal, clip, TTY masking, agent refusal) and 4 unit tests
- [x] [AC10] Harden ai/claude/settings.json permissions deny list

## Review round 1 (2026-09-23, retroactive, FAIL)

Reviewed at the landing commit `a720b9d` (#1459). Every finding's disposition is in `verification.md`.

- [x] F1: pin the split-write redaction case at 1-3 byte writes (already closed on main by SEC-002's prefix-aware hold-back)
- [x] F2: match whole shell words, so `/usr/bin/env` and `env>x` are refused, and tests pin each part of the shell's reading
- [x] F3: fill verification.md, replace the placeholder features.json, and correct the two closing ticks below
- [x] F7: replace the production-dead `buildChildEnv` with `childEnviron`, shared by the run path and its tests
- [x] F8: name every agent-session marker in AGENTS.md and ADR-028, held by a test
- [x] AC9 (found while applying F8): recognise `CLAUDECODE`, the variable Claude Code actually exports
- [x] Round 2 review ran (FAIL, below); it did not pass

## Review round 2 (2026-09-23, FAIL)

`nan/deepseek-v4-flash` at `1a03ae6` (#1655). Every finding's disposition is in `verification.md`.

- [x] F1: recognise the markers of every harness `harness/model-map.json` declares (pi's `AI_AGENT` and `PI_CODING_AGENT`, plus opencode, Copilot and Codex), measured rather than assumed, and a {harness -> marker} test that fails on an absent marker
- [x] F2: restate AC6 and AC10 as template-scoped; the deny-list deploy is #1339 (owner decision, 2026-09-24)
- [x] F3: `features.json` f6 checks AC6's three deny entries
- [x] F4: refuse before resolving any secret, and a test that counts the reads
- [x] F5: a Windows path or `.exe` suffix, and `busybox <applet>`, name the command they run
- [x] F6: state the indirection boundary under Out of scope
- [x] F7: update #1646, whose premise this round makes obsolete
- [ ] Round 3 review passes before archive

## Closing

Two of these boxes were ticked at landing without being true: `verification.md` was an unfilled template, and `features.json` held a placeholder, so no criterion had a runnable check. Both hold after round 1's fixes. AC6, AC8 and AC10 are artifacts rather than behaviour, so they are checked by `features.json` commands, not by tests.

- [x] Every acceptance criterion maps to a test or an artifact check in features.json
- [x] Type checks and go test ./... pass
- [x] verification.md filled in
- [x] PR opened referencing this spec folder
