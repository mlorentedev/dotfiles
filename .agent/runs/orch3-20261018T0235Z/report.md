# Run orch3-20261018T0235Z

**Verdict:** run orch3-20261018T0235Z — ended `queue-empty` after 2 items / about 30 min of a 10 h window — 1 done (#2591), 1 investigated and parked (#2724), 0 merged (`merge: never`), 2 PRs open for you. Waiting on you: land #2746 first (main is red without it), then #2747 and #2748. Nothing irreversible happened.

## Waiting on you, in order
1. **#2746** (not mine): deletes the run artifacts #2729 put on main. Main's `test` has been red since `e6eb0f76` (`HARNESS-041`: `.agent/` is in neither list). Every code PR inherits it, #2747 included. Its own `test` is green.
2. **#2747** (closes #2591, option B): the job `if` of `pr-agent.yml` no longer reads the payload's `draft`; the existing `reviewable` step, which reads the PR from the API, is the only draft gate. Guard test replaces `draft PRs are skipped`. Cost: a draft push starts a short run that takes a place in the `queue: max` lane; the follow-up if that is noisy is an ungated `gate` job. Triaged: pr-agent's one finding (a run id in the body) applied, three "verify" notes declined with reasons.
3. **#2748** (refs #2724): the lesson. Docs only. Triaged, no findings.

## Parked
- **#2724**, Blocked on the board. The brief's hypotheses were tested and none holds. The `workflow_run` trigger works: on #2718 it fired 2 s after pr-agent finished, resolved PR 2718 and its head correctly, and published `success`; it started its job 4 min 41 s after creation (a runner queue), so the status stayed at an earlier, correct `failure` for 4 min 55 s after the review landed and `dotf pr land` read it there. Latency of `review-attestation` runs: `workflow_run` median 203 s busy and 130 s quiet. Evidence tables are on the issue.
  - (A, recommended) `dotf pr land` treats a status older than the newest reviewer comment as in flight: polls up to about 8 min, reruns the workflow once on timeout, then refuses. Go in `cli/`, outside this envelope, and needs a spec slot (WIP 13 of 10).
  - (B) publish the verdict from the last step of `pr-agent.yml`: removes one runner start, but that job runs PR-controlled code, so the classifier must come from the base branch and the reviewer workflow gains `statuses: write`. Security-relevant, owner-lands.

## Evidence index
Per item, the commands and output are in the PR bodies and the `## Review triage` comments of #2747 and #2748 and in the comments on #2724.
- #2591: new test failed before the change (`AssertionError: the job gate decides from the event payload's draft flag`); 114 ok, 0 not ok on the five pr-agent and dependabot bats files after; `actionlint` exit 0; `dotf lessons fmt --check` OK.
- #2747 CI on `395f8397`: all green except `test`, which fails only `HARNESS-041` for `.agent/`, same failure as main (`fac9ff60`).
- #2748 CI on `168a1567`: green; its `test` skipped the bats suite (docs-only path filter), so it says nothing about main.
- `dotf pr triage-queue --pr 2747` and `--pr 2748`: `[OK] no reviewer output is awaiting a disposition`.

## Reviewers
pr-agent reviewed both PRs on their final heads. CodeRabbit posted only rate-limit notices.

## Debt and corrections
- Linked #2730's unreviewed final head on #1756 (push gate, one new commit under the threshold of 3), no new ticket, as asked.
- A mistake caught late: the first triage on #2748 explained why `test` was green with a guess; I measured it, and a correction is posted on the PR.
- The envelope's `armed_at` (02:35:00Z) is about four minutes later than the real arming (about 02:31Z). I left the committed file untouched rather than edit the envelope mid-run; it only makes the window read slightly later than it was.
- I claimed #2724 after posting the analysis, not before. Assigning it flipped its board status to In Progress; I set Blocked again and checked.
- Lesson numbers: 476 is used by #2743 and #2736, 477 by #2747, 478 by #2748. Renumber on a clash.
- Not touched on purpose: the `issue_comment` cancel case of #2738, noted on #2591 by you.

## Budget
2 of 2 queue items, plus the #1756 comment. About 15 min of waiting for CI and the reviewer on each PR; no GraphQL refusals (REST reads throughout, `gh pr ready` worked).

## Host
`caffeinate -i -t 36000` started with the arming; released at close.

## Next action
Land #2746, then merge the base into #2747 if it needs it and land #2747 and #2748. Decide A or B on #2724.
