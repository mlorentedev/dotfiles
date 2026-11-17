---
tags: [spec, tasks, templates]
created: "2026-11-12"
id: "BUG-172-update-installer-converge-failure"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-172-update-installer-converge-failure

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `fix/3395-update-install-vs-converge`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] Failing test `TestRunPin_ConvergeFailsInsideInstaller` in `internal/update`: failed steps from the installer's converge give `converge-failed` naming them; none gives `install-failed`
- [x] [AC1] [AC2] `Deps.InstallerConverge`, asked only after `Install` fails, and the `converge-failed` outcome in `update.Run`
- [x] [AC1] [AC2] [AC3] Failing test `TestFailedConvergeSince` in `internal/cmd`: fresh, same-second, stale, passing, missing and unreadable reports, and a failed run with no failed step
- [x] [AC1] [AC2] [AC3] `failedConvergeSince` reads `last.json`, wired into `dotf update` with the installer's start time

## Closing

- [x] `go test`, `go vet` and `golangci-lint run`, natively and with `GOOS=windows`
- [ ] `verification.md` filled in; independent review; archive
