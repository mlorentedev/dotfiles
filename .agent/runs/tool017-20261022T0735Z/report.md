# Run tool017-20261022T0735Z

**Verdict:** in progress (draft written at 09:55Z; updated at disarm).

## Waiting on you, irreversible first

- **#2924** (AC4/AC6/AC8, the reproduction gate): triaged, CI green at `62708bcb`. It touches `harness/`, so the owner merges it. **Before merging:** run `scripts/compile-harness.sh --refresh` on its branch. The vault source of `pr-review-triage` now carries the same body (vault `5d6a49c5`, the applied REAL finding), so the record's `generated_sha` names the old source and `dotf doctor` reports it stale. That is a `harness/` write, frozen for this run. If #2935 lands first, #2924 needs main merged into it (both edit the TOOL-017 `tasks.md`/`verification.md`). This run does that if it is still armed.
- **Vault commit `5d6a49c5`** (pushed to master): the `pr-review-triage` skill source gains the REAL/refuted rule. It is irreversible in the sense that every machine pulling the vault sees it; main's rendered record matches only once #2924 lands.

## Parked

None.

## Done but unmerged

- **#2935** (closes #2926): `--defect-when-passes`, row quoting, the #1059 replay ticked, lesson 518. It was triaged (one REAL finding, refuted with a GFM-render reproduction). The pending pr-agent review queue blocked the land.

## Evidence index

| Item | Command | Exit |
|---|---|---|
| #2935 | `cd cli && go test ./...` | 0 |
| #2935 | `golangci-lint run` | 0 (0 issues) |
| #2935 | `GOOS=windows go vet ./...` | 0 |
| #2935 | #1059 replay, `dotf pr reproduce --probe {flagless,multiline}.diff --defect-when-passes -- bats tests/ci-age-pin.bats` at `1719024f` | did-not-reproduce / reproduced |
| #2924 | `go run ./cmd/dotf pr triage-queue --pr 2924` (the gate's own branch) | 0 |
| #2935 | `go run ./cmd/dotf pr triage-queue --pr 2935` | 0 |

## Budget

Items: 2 of 4. No dollar figure exists for this run.

## Host

`caffeinate -i -t 28800` from 07:35Z: the window is covered unless the lid closed.

## Next action

Land #2935 once its pr-agent review completes; then merge main into #2924.
