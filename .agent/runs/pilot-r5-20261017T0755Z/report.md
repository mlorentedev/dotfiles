# Run pilot-r5-20261017T0755Z

**Verdict:** ended `window-closing` after 1 long iteration, about 10h: 3 done and merged under the grant (#2666, #2668, #2674) plus #2687 and #2691, 1 prepared and still open (#2701, CLI-131 archive), 1 parked (#2552), 0 irreversible actions awaiting you. Question this answers: what is waiting on a human? **#2701 only, and it is reversible** (a spec-only archive diff).

## Waiting on you, irreversible first

Nothing irreversible is pending. Landed this run, all under the standing grant with `dotf pr land` on a CLEAN head:

| PR | What | Merge |
|---|---|---|
| #2666 | two-concurrent-runs test for `lane apply` (closes #2617) | grant |
| #2668 | lane prompt: index is rendered on read, `SKIP=lessons-fmt` instruction removed | grant |
| #2674 | CLI-131 round-1 fixes: `lessons --help` text, 4 Closing boxes, re-measured evidence | grant |
| #2687 | lessons 471 and 472 | grant, PR-Agent triaged |
| #2691 | sort-order test for the index, wrapped stray-index error, evidence | grant, PR-Agent triaged |

#2656 (CLI-131 itself) was landed by **you**, not by the grant.

Open for a decision:
- **#2701**: archives CLI-131 after a PASS WITH GAPS (round 3). It carries `Closes #2554`. If the envelope expired before CI settled (17:55Z), land it by hand or with a new run; it needs no code review beyond the triage already on the PR.

## Parked, with the question restated

- **#2552 (CLI-129, `dotf initrepo` scaffolds the lane contract).** Parked behind ADR-050 acceptance, which is the owner's. (A) Accept ADR-050, then the next run takes it; (B) rescope it to the dotfiles half only. Cheapest unblock: accept the ADR.
- **#2553 (CLI-130)** was to be prepared and left open (authority widening, owner lands). Not started: it widens the headless tier's authority and the window went to the CLI-131 review chain.

## Done but unmerged

- #2701 above.

## Evidence index

| Item | Command | Result |
|---|---|---|
| CLI-131 round 1 | `dotf spec review CLI-131-lessons-index-on-read` | FAIL, kept as `review-round1.md` (false Blocker: normalised digests, ticketed #2684) |
| CLI-131 round 2 | same, from the run branch | FAIL, kept as `review-round2.md`: the one REAL Major was the run's own `.agent/` arming commit failing `ci-path-filtering.bats` |
| CLI-131 round 3 | same, from `chore/review-cli-131` cut from main at `2888f9cc` | PASS WITH GAPS (`nan/qwen3.8-flash`) |
| Index tests | `git grep -l lessons -- 'tests/*.bats' \| xargs bats` | 228 ok, 0 not ok |
| Sort test | delete the `sort.Slice` line, run `TestIndex_SortsByNumberNotByFileName` | fails; restored, passes |
| Per-PR queue | `dotf pr triage-queue --pr N` | exit 0 before each land |

## Tickets filed

#2684 (`dotf spec digest`), #2690 (`newestLesson` swallows I/O errors), #2700 (`StrayIndexes` stops at level 1).

## Lessons

464 (a file a pinned tool requires needs a release first), 471 (a reviewer cannot check a digest it cannot recompute), 472 (polling `gh pr checks` from parallel sessions spends the GraphQL budget).

## Budget spent

Iterations: 1 long loop (limit 10). Wall clock about 10h of the 10h window. No dollar figure exists: the provider exposes none for this run. The review chain dominated: three rounds of about 40 minutes each, plus the CI queue (the `review` job took up to about 40 minutes).

## Host-suspend line

Window covered. The first `caffeinate` died at once (backgrounded with `&` inside a background command) and a second exited 144, so I relaunched with `-t 32000`. Other sessions' `caffeinate` processes (`-t 37800`, `-t 41400`, `-t 43200`) also held the Mac awake. No gap was observed in the PR and poll timestamps.

## Not done and why

- A peer (dotfiles-20) asked to add #1975 and #2448 to the envelope. Declined: widening the queue filter is the owner's call.
- GraphQL was exhausted by parallel sessions for about 10 minutes; PRs were opened and polled through REST (lesson 472).

## Next action

Land #2701, then accept ADR-050 so the next run can take #2552.
