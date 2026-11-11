---
tags: [spec, tasks]
created: "2026-11-11"
id: "BUG-167-archive-quotes-dead-reviewer"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-167-archive-quotes-dead-reviewer

## Implementation

- [x] [AC1] Failing test: a quota error in the transcript reaches the archive refusal with the reviewer named; implement in `withTurnCapHint`
- [x] [AC2] Move `reviewerLastWords` to `spec.ReviewerLastWords`; keep both ends of a long output on rune boundaries; test
- [x] [AC3] Existing turn-cap tests pass unchanged

## Closing

- [x] `go test ./internal/spec ./internal/cmd`, `golangci-lint run`, `GOOS=windows` vet and lint
- [x] `verification.md` filled in
