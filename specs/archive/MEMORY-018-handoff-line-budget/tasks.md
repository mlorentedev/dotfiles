---
tags: [spec, tasks, templates]
created: "2026-11-12"
id: "MEMORY-018-handoff-line-budget"
type: spec-tasks
status: active
owner: manu
---

# Tasks - MEMORY-018-handoff-line-budget

## Implementation

- [x] [AC1] Failing test: a write past 200 lines and under 25,000 bytes is refused, naming the cap, the count and the oldest threads
- [x] [AC1] Add `MemoryLineBudget` and refuse in `CheckMemoryBudget`, sharing the archive suggestions with the byte refusal
- [x] [AC2] Test: a write that holds or lowers the line count of a file past the cap passes; one at or under 200 lines passes
- [x] [AC3] Test: the byte refusal is unchanged for a short file past 25,000 bytes

## Closing

- [x] `go test ./...`, `golangci-lint run`, `GOOS=windows` vet and lint
- [ ] `verification.md` filled in; independent review; archive
