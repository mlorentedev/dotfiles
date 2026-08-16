---
tags: [spec, verification]
created: "2026-08-16"
---

# Verification — GUARD-002-review-attestation

Session evidence for the acceptance criteria in `proposal.md`. `features.json`
carries the machine-readable form; this file records what was actually run and
what it printed.

## The evidence this gate was built from

Captured 2026-08-16 and committed as fixtures, because it is the only
reproducible form of a vendor notice that exists solely while a quota is spent:

| PR | `reviews[]` | CodeRabbit comment | `gh pr checks` |
|---|---|---|---|
| #1007 | 0 | rate-limit notice | `CodeRabbit  pass` |
| #1009 | 0 | rate-limit notice | `CodeRabbit  pass` |
| #1013 | 0 | rate-limit notice | `CodeRabbit  pass` |

Three PRs, three parallel sessions, one exhausted account-wide quota, **zero
reviews, all green**. Both #1007 and #1009 were subsequently merged.

Two levers were measured dead while the quota is spent: pushing new commits does
not reclaim a slot, and neither does an explicit `@coderabbitai review`. Only
time does. That is why AC-level auto-retriggering is out of scope — it would not
work, *and* it would make the gate perturb the thing it measures.

## AC1 — offline classification, including the real payload

```
$ bats tests/check-review-attestation.bats
25 tests, 0 failures
```

```
$ scripts/check-review-attestation.sh --payload tests/fixtures/review-attestation/pr-1009.raw.json
[FAIL] declined — coderabbitai posted a notice that no review ran (marker: "rate limited by coderabbit.ai")
       A notice that no review ran leaves this PR UNREVIEWED. Its checks may still be green:
       the reviewer reports its own status, and a skipped review is not a failed one.
```

The gate correctly refuses the real PR that was merged unreviewed.

## AC2 / AC3 — every state, with the right label

| Fixture | Expected | Exit | Reported |
|---|---|---|---|
| `bot-reviewed.json` | attested | 0 | `[OK] attested — reviewed by: coderabbitai` |
| `human-reviewed.json` | attested | 0 | `[OK] attested — reviewed by: someone-else` |
| `self-review.json` | **not** attested | 1 | pending |
| `coderabbit-rate-limited.json` | declined | 1 | `[FAIL] declined — …` |
| `no-output.json` | pending | 1 | `[FAIL] pending — no reviewer output on this PR yet` |

`self-review.json` is the case worth naming: the author's own review does not
attest. A non-empty `reviews[]` is not the question — whether anyone
*independent* looked is, which mirrors `harness/reviewer-pool.json`'s standing
rule that the reviewer must not be the implementer.

`declined` and `pending` both exit 1 and are reported distinctly, per the fourth
constraint in the proposal's Risks.

## AC4 — the escape needs both halves

| Fixture | Label | Rationale | Exit |
|---|---|---|---|
| `disclosed.json` | ✅ | non-empty | **0** |
| `label-only.json` | ✅ | absent | 1 |
| `section-only.json` | ❌ | non-empty | 1 |
| `empty-rationale.json` | ✅ | **heading, nothing under it** | 1 |

## AC5 — a reviewer is config, not code

```
$ bats tests/check-review-attestation.bats -f 'second reviewer'
ok  AC5: a second reviewer declared in config is recognized, with no code change
```

Driven by a config naming `pr-agent`, classified from `second-reviewer.json`
with no change to the script. Asserted in the negative too: the script contains
neither `coderabbitai` nor `pr-agent`, so the registry cannot quietly become
decorative.

## AC6 — fails closed

Malformed JSON, an empty payload, a missing payload, a missing registry and a
malformed registry each exit **2**, and each prints *"Cannot determine whether
this PR was reviewed, so it is NOT treated as reviewed."*

## AC7 — settles without a push

Triggers are `pull_request` and `issue_comment`; the step carries no
`continue-on-error`.

Worth recording, since it cost a test: **YAML 1.1 parses the bare key `on:` as
the boolean `true`**, so `yaml.safe_load(wf)['on']` raises `KeyError` on every
GitHub workflow ever written. The check reads `d[True]` first.

### The gate ran on its own PR, and the first run found a hole in AC7

First live execution, PR #1019, `pull_request` event:

```
[FAIL] pending — no reviewer output on this PR yet
       Proceeding on an unreviewed PR is allowed. Proceeding silently is not.
```

Correct: at 00:58:51 no reviewer had posted. CodeRabbit's rate-limit notice
landed at **00:59:05**, 14 seconds later — and **no second run followed**:

```
$ gh run list --workflow=review-attestation.yml
00:58:45  event=pull_request  completed/failure     <- the only run
```

Two facts behind that, both verified rather than assumed:

1. GitHub reads the `issue_comment` trigger from the **default branch's** copy
   of a workflow. `review-attestation.yml` is not on `main` yet, so no
   `issue_comment` run for this PR was ever possible.
2. More seriously, and true even after merge: a run triggered by
   `issue_comment` is associated with the default branch, **not** with the PR's
   head commit. Confirmed on the head SHA — the `pull_request` run attached as
   a check-run, and an `issue_comment` run would not.

So AC7 as originally written would have shipped **decorative**: the trigger
fires, the classifier runs, and the PR's visible verdict never changes. A
re-run that only appears to re-run — this spec's own defect class, committed
inside the fix for it, and caught only because the gate was pointed at itself.

**Fix:** the job now publishes an explicit commit status (`review-attestation`)
onto the PR's head SHA on both events, with `statuses: write`. The exit code is
captured so the status is always published, and restored in a final step so the
run still fails when the PR is not attested — otherwise capturing the code to
publish it would have turned every verdict green, which would have been the
same bug a third time.

Two assertions added: `statuses: write` plus a `statuses/` API call are present,
and the final `exit "$CODE"` is present.

**Not verified until merge:** the live `issue_comment` re-run. It cannot be,
for reason (1) above. Structural until then, and the honest first proof is the
next PR opened after this one merges.

## Non-vacuity — measured, not assumed

Four mutations, each with valid syntax and a wrong answer. Every one is caught,
and by the tests that should catch it rather than by all of them at once:

| Mutation | Tests that went red |
|---|---|
| declined detection disabled | AC1 ×2, AC5 second-reviewer |
| escape accepts the label alone | AC4 label-only, AC4 empty-rationale |
| fail-closed → fail-open | AC6 ×4 |
| author's own review counts | AC2 self-review |

A suite where every test dies on one mutation is usually asserting one thing
several times. These partition, which is the property worth having.

One test was **observed failing for real, not by mutation**: the first version of
"the workflow does not swallow the gate's failure" grepped the bare string
`continue-on-error` and matched the workflow's own comment explaining why the key
is absent — a guard tripping on documentation *about* the thing rather than the
thing. Same false positive that made #998's archive gate unpassable, caught here
by the assertion being written before the file was considered finished. Now
anchored to a YAML key.

## Lint

```
$ shellcheck --severity=error scripts/check-review-attestation.sh    # CI's severity
CLEAN
$ shellcheck scripts/check-review-attestation.sh                     # all severities
(no findings)
$ bash -n && zsh -n
OK
```

## Not verified here

- **Live CI behaviour.** The workflow has not yet run on a real PR; the first
  exercise is this spec's own PR. Fixtures prove the classifier, not the wiring.
- **Marker durability.** If CodeRabbit changes its HTML marker the gate reports
  `pending` rather than `declined` — still red, so the verdict survives and the
  message degrades. Not something a test can pin.
