# run orch2-2731-20261018T0057Z

**Run orch2-2731-20261018T0057Z ended done-unmerged after 1 iteration and about 1h: 1 done, 0 parked, 0 merged, 1 unmerged.** This run refilled orchestrator run orch2-20261017T2045Z, relayed by session dotfiles-20 with the owner's authorisation.

What is waiting on a human: #2738, which the owner lands. Nothing irreversible was done: nothing merged, nothing released, nothing run as root.

## Waiting on you

1. **#2738 (`fix/sshtarget-launchctl-print`, `d853f8a8`, draft): owner-land.** Two reasons:
   - It is stacked on #2733, which adds `schedule.LaunchdLoaded`. Once #2733 merges, merge main into the branch (never a force-push), mark it ready, request `/review`, then triage.
   - It wrote two test fakes outside the envelope (see Scope below). The scope was not widened, so the standing grant does not cover the land (lesson 469).

## Parked

None.

## Done but unmerged

- #2738 closes #2731. `Launchd.Enabled` and the darwin `MeshAtBootFor` read only exit 113 as "not loaded", through `schedule.LaunchdLoaded` with one adapter.
  - `Enabled` returns any other failure as an error.
  - The mesh note says the read failed, and asserts no state.
  - `schedule` and env-persist were not edited.

## Scope

The envelope armed `write_paths: cli/internal/sshtarget/**`. The fix also needed:

| File | Envelope status | Change |
|---|---|---|
| `cli/internal/converge/workstation_test.go` | **frozen_paths** | the faked 113 carries an `ExitCode()` |
| `cli/internal/doctor/checks_workstation_test.go` | outside write_paths | the same, plus the noMesh launchctl case |

Both are test fakes for sshtarget's consumers, not env-persist files. Without them, every unloaded-sshd test fails with the new, correct error.

**Arming lesson:** to protect one file (env-persist), I froze its whole package. That made the fix's consumer tests unwritable. Freeze the file, not the package. Before arming a reader fix, grep the consumer packages for fakes of the value being reinterpreted (here `exit status 113`). The owner decides whether this becomes a numbered lesson; it was not written to `docs/lessons/`, which is also outside the envelope.

## Evidence

| Check | Command | Exit |
|---|---|---|
| build / vet / windows vet | `go build ./... && go vet ./... && GOOS=windows go vet ./...` | 0 |
| tests | `go test ./...` | 0 |
| lint | `golangci-lint run ./...` (2.12.2, the pin), and `GOOS=windows` on sshtarget, converge and doctor | 0 issues |
| red first | the new tests failed on the old readers (exit 5 and no-exit-status cases) | 1, as intended |
| mutations | the adapter drops `launchctl`; `Enabled` swallows the error; the mesh note ignores a failed read; the adapter drops the error | 4 of 4 caught |
| live, read-only | `launchctl print system/com.openssh.sshd` / `system/com.tailscale.tailscaled` | 0 / 113 |
| live plan | `go run ./cmd/dotf converge --plan --only workstation-target --repo ~/Projects/dotfiles` | plans only `state record` (#2716) and keeps the app note: the 113 path survives |

The production runners keep the exit code: `sshtarget.Exec` and doctor's `CommandOutput` both return `CombinedOutput`'s raw `*exec.ExitError`.

## Notes

- **#2713** had already landed at `bfd95b5`, before this refill was relayed.
- **#2683** was merged by the owner (`aecc1ae7`). Its worktree and branch are removed.
- **#2681** is behind main again since #2683 landed. Not touched under this envelope.
- **Spec-gate's local pre-push** counts the stacked base's lines (87 against this change's ~20). I pushed with the wrapper's documented first-push values (`SDD_LABELS=skip-sdd`, `SDD_PR_BODY` = the PR body). The PR carries the same label and rationale, and CI re-checks them.
