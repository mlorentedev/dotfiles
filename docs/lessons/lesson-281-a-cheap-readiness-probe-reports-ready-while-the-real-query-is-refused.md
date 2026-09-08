---
id: lesson-281
type: lesson
status: active
created: "2026-09-07"
owner: manu
tags: [lesson, github-api, rate-limits, gh-cli, retry, background-jobs, fail-closed]
---

# 281 — A readiness probe cheaper than the work it gates reports ready while the work is still refused

## What happened

Nine bitácora tickets had to be filed and put on the board. Issue creation went
through REST and succeeded. The board writes need GraphQL, and GraphQL was under
a **secondary** rate limit, so the board work was handed to a detached job that
would wait for capacity and then run.

The job's readiness probe was the cheapest GraphQL call there is:

```sh
until gh api graphql -f query='{viewer{login}}' >/dev/null 2>&1; do sleep ...; done
```

It logged `graphql available after 0s` and went straight into the real query,
which failed immediately:

```
2026-09-07T18:47:00 graphql available after 0s
unknown owner type
2026-09-07T18:47:01 FAIL item-list
```

The probe was not lying about itself. `{viewer{login}}` costs one point and was
genuinely being served. `gh project item-list --owner mlorentedev --limit 1200`
was not. Both are "GraphQL"; only one of them was the question that mattered.

Three separate things had to line up to make this hard to see, and each is worth
keeping on its own.

**1. The documented rate-limit endpoint reported full capacity during the
outage.** Measured minutes before, while project queries were being refused:

```
$ gh api rate_limit --jq '{core: .resources.core, graphql: .resources.graphql}'
{"core":{"limit":5000,"remaining":5000,...},"graphql":{"limit":5000,"remaining":5000,...}}

$ gh project list --owner @me
GraphQL: API rate limit already exceeded for user ID 13562150.
```

`remaining: 5000` and `already exceeded`, from the same account, seconds apart.
The secondary limit is not the counter that endpoint reports, so **its `reset`
timestamp is not a valid retry trigger** — scheduling a wake-up from it schedules
a wake-up against the wrong limiter.

**2. `gh` reports the refusal as a configuration error.** `gh project ... --owner
X` must first resolve whether `X` is a user or an organisation, and that
resolution is itself a GraphQL call. When it is refused, the failure surfaces as:

```
unknown owner type
```

which reads as "you typed the owner wrong", not "you are being throttled". The
first instinct is to go check the owner name and the token scopes — both of
which were fine. A capacity failure wearing a configuration error's clothes
sends you to the wrong fix.

**3. The gap is real, not a race.** It is not that the probe ran a moment too
early. A one-point query and a 1200-item project listing sit in different cost
classes, and the throttle lifts for the first well before the second. Retrying
the probe faster would never have closed it.

## Why it happened

The probe was chosen for the property that makes it useless here: it is cheap, so
it can be polled often without itself consuming the budget being waited on. That
reasoning is correct for a *liveness* check — is the service up — and wrong for a
*readiness* check, which is a claim about whether **this specific work** can run
now.

The result is the shape lesson 280 names in a different setting: **a check that
cannot observe the failure it gates on**. No probe leaves the caller uncertain,
so it proceeds carefully. A probe that answers "ready" makes the caller
confident, and it fails halfway through the real work — after `item-add` and
before the field writes, which is precisely the partial state a background job
exists to avoid.

The first job — written before the gap was known — got away with it only because
GraphQL genuinely was available when it ran. Its nine items landed complete. The
second job, written from the same template minutes later, hit the gap and stopped
having done nothing. Same code, different outcome, and the difference was luck.

## The rule

**A readiness probe must exercise the same path as the work it gates: same
authentication, same resolution steps, and the same cost class.** If the real
query resolves an owner, lists a project, or paginates, the probe does too — with
the smallest result the endpoint will return, not the smallest query the API
accepts.

The fix was one line, and the comment matters more than the code:

```sh
# The readiness probe must exercise the SAME path as the real query. A cheap
# `{viewer{login}}` costs one point and answers while the project queries are
# still refused -- gh then reports that refusal as "unknown owner type", which
# reads like a config error rather than a rate limit. Probe with field-list:
# same owner resolution, same project scope, small result.
until gh project field-list "$PNUM" --owner "$OWNER" --format json >/dev/null 2>&1; do
```

It backed off correctly on the next run — `graphql unavailable; sleeping 300s` —
which is the probe finally reporting the state the job is actually in.

Three corollaries:

- **Do not schedule a retry from a limiter you are not being throttled by.** With
  a secondary limit there is no published reset to wait for, so the only honest
  strategy is backoff against the real query, with a declared give-up bound.
  Waiting on `rate_limit.reset` here would have fired into a still-closed door
  and looked like a second, unrelated failure.
- **Read an error's *category* before believing its *text*.** "unknown owner
  type" during work that had just succeeded with the same owner is not about the
  owner. When a message names a cause that was true a minute ago and is
  unchanged, suspect the message.
- **A job that gates on a probe must be idempotent anyway.** The probe reduces
  how often you resume mid-flight; it does not remove the case. Both jobs here
  used `item-add`, which returns the existing item, so a re-run costs nothing and
  repairs a partial one.

## Worked example

The sequence, from the two logs:

| Time | Event |
|---|---|
| 18:27 | `gh project item-list` → `unknown owner type`; `gh api rate_limit` → 5000/5000 on both counters |
| 18:44 | board job: cheap probe passes **and the real work also runs** — nine items added, Status/Priority/Type set |
| 18:46 | a `--limit 500` then `--limit 2000` listing exhausts GraphQL again |
| 18:47:00 | fixup job: cheap probe passes, `item-list --limit 1200` refused, job exits having done nothing |
| 18:47:13 | probe replaced with `project field-list`; job correctly reports unavailable and backs off |

Note the 18:46 row: the listing issued **to verify the writes** is what exhausted
the budget for the follow-up. Verification is not free, and on a point-priced API
a wide read can cost more than the writes it is checking. Prefer verifying from
what the write returned — `item-add` hands back the item id — over re-reading the
whole collection.

## Scope

Not GitHub-specific. It applies to any probe standing in front of work with a
different cost, quota, or permission profile: a `SELECT 1` in front of a report
query, a TCP connect in front of a TLS handshake, a `HEAD` in front of a ranged
`GET`, a token refresh in front of a scoped call. The probe's job is to predict
the work, and it can only do that by resembling it.

Related: [280](lesson-280-a-diagnosis-repeated-often-enough-reads-as-evidence.md)
(a vacuous check leaves you sure instead of uncertain — same failure, different
surface), and the repo's own prohibited-pattern table, whose last five rows are
all cases of a check answering wrongly rather than failing.
