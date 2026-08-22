# Lesson 220 — Four defects, one shape: a thing verified by a proxy that lives somewhere else

**Date:** 2026-08-22
**Context:** A session that shipped the model-tier and capability renders (#1165, #1172) and then spent most of its length fixing the review machinery around them.

## What happened

Four separate defects surfaced in one session. They looked unrelated — a registry, a doctor check,
a launcher, a reviewer pool — and they are the same bug.

| Defect | What was checked | What it actually proved |
|---|---|---|
| **#1182** doctrine marker | "did `definition-of-done` reach this file?" by searching for `"Definition of Done"` | That phrase appears **only in `pr-stewardship`'s provenance blockquote**. It verified one region by finding another's meta-text |
| **#1178** review verdict | "did the review pass?" by reading `review.md`'s `verdict:` | The reviewer **authors its own frontmatter**. A run that wrote nothing left the previous round's PASS in place |
| **#1156** reviewer pool | "can the fallback arm run?" by it being declared in `reviewer-pool.json` | It named `GEMINI_API_KEY`, a secret the registry never held. **The arm had never once run** |
| **#1162** `agy` render kind | "how does this harness get its model?" by the `render` field | The field said `agent-md`; the launcher actually passes `--model`. The map contradicted the launcher |

In every case the check passed for as long as the system was healthy, and kept passing — or started
failing — for reasons unrelated to the thing it claimed to measure.

## Why they are the same bug

Each check reads a **proxy that lives outside the thing it verifies**:

- a phrase in *another record's* prose,
- a field the *reviewed party* wrote,
- a *declaration* that no runtime ever exercised,
- a *description* of behaviour instead of the behaviour.

The proxy and the subject drift independently. When they do, the check reports on the proxy and
says nothing about the subject — in either direction. #1182 is the sharpest: the doctrine was
entirely intact and the check reported three deployment failures, because the *prose it keyed on*
had moved.

## The test that separates them

When adding or reviewing a check, do not ask "does it pass?". Ask:

> **What would this still pass on, if the thing it checks were completely broken?**

Applied to the four above, the answers are immediate: a file with the doctrine deleted but
pr-stewardship intact; a review that never ran; a credential that does not exist; a harness wired
the opposite way.

The corollary is the fix, and all four took the same one: **key the check on the subject's own
observable behaviour.**

- The marker became the rule's **own opening line**.
- The verdict gate compares `review.md`'s **digest before and after** the launch — a fact the
  reviewer cannot author.
- The pool must declare `secret_id` **or** `auth: login`, so "no secret" can never mean "nobody said".
- The render kind was checked against **what `review_launch.go` actually builds**.

## The second-order rule

Three of the four fixes added a guard for the *class*, not the instance:

- every marker must appear in the record it names;
- every pool entry must declare exactly one auth mechanism;
- a harness whose `render` is `adapter` must have no harness-keyed tier, and vice versa.

That is the difference between fixing an instance and closing a class. An instance fix leaves the
next occurrence to be discovered the same expensive way.

## The measurement that started it

None of these were found by reading diffs. #1182 was found by **deploying and running
`dotf doctor`**; #1178 and #1156 by **launching a real review and watching it fail**; #1162 by
**reading the launcher instead of the registry**.

A check that has never been observed failing is a check whose failure mode is unknown.

## Where it bit

`cli/internal/doctor/checks_deploy.go` (`enforcedRegionMarkers`),
`cli/internal/spec/review_request.go` (the sidecar digest),
`harness/reviewer-pool.json` (`auth`), `harness/model-map.json` (`agy.render`).
Guards: `TestEveryDoctrineMarkerIsInItsOwnRecord`, `TestEveryPoolEntryDeclaresItsAuth`,
`TestRenderKindAgreesWithTierKeying`.
