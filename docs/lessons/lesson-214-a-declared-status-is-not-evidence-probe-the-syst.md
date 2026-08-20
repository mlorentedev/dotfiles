---
id: lesson-214
type: lesson
status: active
created: "2026-08-20"
owner: manu
tags: [lesson, sdd, specs, verification, guards]
---

# 214 — A declared status is not evidence, and a guard that exists is not a guard that covers

**Context**: `specs/` had drifted to 45 active folders. An audit on 2026-08-19/20
took it to 20. Two distinct traps showed up, and both are about trusting a
declaration instead of the system.

## A spec's `status:` field records an intention, not an outcome

Fifteen specs had `status: draft` or `implementing` while their issues were closed
and their work merged. Four more, with no declared issue at all, turned out to be
**shipped** — `AI-022` (both setup scripts install the hive daemon; the service was
running), `SDD-004` (its 937-LOC target pair no longer exists, absorbed by
`dotf mem session-start`), `MEMORY-001-cross-agent-session-bridge` (live as
`dotf mem session-end`), `POLISH-004` (merged as `679126e`).

The field is written when work starts and nobody returns to it. **The only reliable
source is the running system**: probe the acceptance criteria, do not read the
header.

## A one-line probe is not verification either

Probing quickly is how the second trap opens. `POLISH-003`'s probe was *"does the
`lint-powershell` job exist?"* — it does, so the spec looked shipped. Reading the
acceptance criteria showed the job **hardcodes four files while the repo has 21**.
Seventeen production scripts, including `utils.ps1`, `install-dotf.ps1` and
`hive-serve-supervisor.ps1`, were never linted at all.

"A guard exists" and "a guard covers" are different claims, and the first is the one
that is easy to check. Three of six specs classified as shipped by probe did **not**
survive reading their ACs.

## The archive gate has two independent blind spots

`SDD-038` archives a spec when a PR closes its issue. It misses in two ways, and
only the first is obvious:

| | Closing keyword | Spec declares `issue:` | Fires? |
|---|---|---|---|
| The fifteen | ✗ closed by hand (`closed \| actor=... \| commit=-`) | ✓ | no |
| `POLISH-004` | ✓ | ✗ predates the gate | no |

A guard keyed on one half of a mapping fails silently when the other half is absent.
Tracked on #1087.

**Generalises to**: any audit over declared metadata. Ask what the system does, then
ask whether the thing you just checked is the thing the criterion required.

**Related**: [[lesson-212]], [[lesson-213]].
