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

## Closing

- [x] Every acceptance criterion from proposal.md is covered by tests
- [x] Type checks and go test ./... pass
- [x] verification.md filled in
- [x] PR opened referencing this spec folder
