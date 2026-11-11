---
tags: [spec, tasks]
created: "2026-11-11"
id: "BUG-168-review-error-stub"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-168-review-error-stub

## Implementation

- [x] [AC3] Failing test: `ParseReview` refuses `verdict: ERROR` by name; add `VerdictError`
- [x] [AC1] Failing test: `WriteErrorStub` renders reviewer, SHA, ending and last words from the request and transcript; implement in `spec`
- [x] [AC1] [AC2] Failing test: `spec deadline` writes the stub when the review is absent and leaves an existing one; wire it before the copy-back
- [x] [AC4] `StoppedRunRecord` names an ERROR record; test
- [x] [AC3] The foreground outcome reports an ERROR record without calling it malformed; test

## Closing

- [x] `go test ./...`, `golangci-lint run`, `GOOS=windows` vet and lint
- [ ] `verification.md` filled in; independent review; archive
