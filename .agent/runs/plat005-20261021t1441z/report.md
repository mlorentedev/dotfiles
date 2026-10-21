# Run plat005-20261021t1441z

**Verdict:** ended `queue-empty` after 1 iteration, about 1h40 (14:41Z to about 16:20Z). 1 done, 0 parked, 1 merged, 0 unmerged. Nothing waiting on the owner is irreversible.

## Waiting on you

- Nothing irreversible is pending. Landed: #2862 (`6c095e8d`, Refs #2763), the OpenSSH 8.9p1 `sshd -T` fixture pair.
- ADR-055 (#2850) is merged as `proposed` by you at 14:35Z. Accepting it is still yours. Two text defects are ticketed in #2858 (decision 9 vs 10 on `run`'s stdout; the JSON record can echo a credential the caller prints).
- #2859: allow rules per harness for the new verbs, yours to land when the verbs exist.
- PLAT-005 spec: still 12 active specs against the limit of 10. None of the five cleanup sessions had archived one when the run ended. It needs at least three slots, and `--over-wip-limit` is yours to lift.

## Parked

None. The rest of the area was blocked, not owner-decided, and was left alone:
- Windows 9.5p2 `sshd -T` fixture and guest acceptance (#2763): wait for the session verbs (#2842). Status comment posted on #2763.
- Implementing ADR-055: `cli/internal/vm/**` and `cli/internal/cmd/vm*.go` were frozen in the envelope. Implementing a `proposed` ADR under the grant would accept it by the back door, and the spec is WIP-blocked.
- #2841, #2790, #2794, #2388: owner A/B or owner at the machine.

## Done but unmerged

Nothing.

## Evidence index

- Fixture: raw `sshd -T` of `1:8.9p1-3ubuntu0.18` in `ubuntu:22.04`, unprivileged user, scratch host key. `go test ./internal/sshtarget/` exit 0; `GOOS=windows go vet ./internal/sshtarget/` exit 0. Output pasted in the PR's triage comment.
- #2862 triage: pr-agent finding "fixture not from 8.9p1" declined with evidence (`PerSourceMaxStartups`/`NetBlockSize` date from 8.7; the dump has no `persourcepenalties` and no `requiredrsasize`, the 10.2p1 one has both); evidence finding applied. `dotf pr triage-queue --pr 2862` exit 0. `dotf pr land 2862` printed `[MERGED]`.
- Tickets: #2858 (PLAT-006), #2859 (PLAT-007), both on the board as Backlog / P2 / chore.
- Triage comment on #2850 (the merged ADR PR) naming the two findings and #2858.

## Budget spent

1 of 4 iterations, about 1h40 of 6h, 1 of 3 items. No dollar figure exists for this run.

## Host-suspend line

`caffeinate -i -t 21600` was started in the background at arming; the window was covered while the lid stayed open and the run was active. Not independently verified after the run ended.

## Disclosures

- `dotf pr land` waited on the `review-attestation` status, which showed `fail` ("coderabbitai could not review", advisory) until the workflow re-ran, then merged. CodeRabbit never reviewed; pr-agent was the only reviewer on #2862 and #2850.
- The PR author did this triage on #2862 and #2850.
- Spec-gate: #2862 pushed with `skip-sdd` and a stated rationale (test data and one table entry), the #2823 precedent.

## Next action

Owner: read ADR-055 for acceptance, then decide whether to amend it via #2858 first. Once three specs are archived, `dotf spec init PLAT-005-...` and the three PRs of #2842.
