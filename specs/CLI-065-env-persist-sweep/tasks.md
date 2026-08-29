---
tags: [spec, tasks, templates]
created: "2026-08-29"
---

# Tasks - CLI-065-env-persist-sweep

> TDD order. One task = one focused commit. Tick as you go. Frozen at the start of `implementing`.

## Setup

- [x] Branch created from main: `feat/env-persist-sweep`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1][AC2][AC5] `env`: split `UserEnvReader` (Get) out of `UserEnvStore` (Get/Set/Delete); `Delete` of an absent name succeeds — stated on the interface, honoured by the fake and by the registry store (`ErrNotExist` → nil).
- [x] [AC2][AC3] `env`: one pure `Leftovers(marker []string, vars []ResolvedVar) []string` — names the marker lists that no contract variable names, compared case-insensitively (registry value names are; a case-only rename is a rewrite, not a leftover), the marker's own name never among them, empty/missing marker → empty set. Table test.
- [x] [AC1][AC5] `env`: `Persist` reads the marker, deletes the leftovers **before** writing (write-then-delete would delete a case-only rename it just wrote), writes the variables, rewrites the marker only when it differs. Fake records the operation order; a test pins delete-before-set.
- [x] [AC2] `env`: two-run test — run 1 persists A, B; contract retires B; run 2 deletes B and only B while a foreign name F is present throughout and untouched. No-marker run deletes nothing and writes the marker.
- [x] [AC3] `env` + `cmd`: `Retired(reader, vars)` for the read-only callers; `dotf env persist --check` prints `retired: NAME` per leftover and exits non-zero; `persist` prints `removed NAME` and counts removals in its summary; `--help` names the marker.
- [x] [AC4] `doctor`: the adapter satisfies `UserEnvReader` only (its `Set` no-op goes); the row WARNs on leftovers with the remedy; test by status gains the retired case.
- [x] [AC5] `persist_windows.go`: `Delete` via `DeleteValue`, `ErrNotExist` → nil, broadcasts the change like `Set`.
- [x] [AC6] Box: scratch copy of the real contract minus one name, `DOTFILES_REPO_DIR` pointed at it, `dotf env persist` twice, registry read between; then the real contract again to restore.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test (AC6 by the box transcript in `verification.md`)
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go build ./... && go vet ./... && go test ./...`, `GOOS=windows go vet ./...`, `golangci-lint run` (pinned 2.12.2)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
