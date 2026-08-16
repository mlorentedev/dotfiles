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
