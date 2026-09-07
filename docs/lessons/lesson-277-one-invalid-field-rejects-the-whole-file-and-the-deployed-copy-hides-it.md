---
id: lesson-277
type: lesson
status: active
created: "2026-09-05"
owner: manu
tags: [lesson, pi, schema, deploy, guard, mutation]
---

# 277 — One invalid field rejects the whole file, and the deployed copy hides it until a deploy

## What happened

`ai/pi/models.json` declared `"audio"` in one model's `input` array. pi 0.84.4 does not
degrade the one model that fails validation; it rejects the entire file and reports
`No models available`. One field took out `pi`, the `qq`/`qf` wrappers and every NaN arm of
the adversarial-review pool at once.

It surfaced far from its cause: `dotf spec review` died with `No models match pattern
"nan/glm5.3-flash"` while `dotf secrets verify` said the API key resolved fine. The reviewer
pool looked down; the key was never the problem.

The bad field shipped in #1471 and sat unnoticed for four commits, because the deployed copy
on the machine was still an older, valid one. The repo was wrong and the machine worked, and
the two are indistinguishable until a deploy overwrites the copy. This is the mirror image of
[lesson 136](lesson-136-a-cli-that-reads-its-config-from-the-deployed-copy.md) and
[lesson 173](lesson-173-a-merged-pr-is-not-a-deployed-change-and-the-deplo.md): there the
deployed copy was stale and hid a fix; here it was stale and hid a break.

The first version of the guard was vacuous. It set `PI_CONFIG_DIR`, which pi ignores, so pi
read the real config and passed while the file under test was broken. `PI_CODING_AGENT_DIR`
is the variable pi honours, verified by pointing it at a deliberately broken copy and getting
the schema error back.

## The lesson

**A validator that rejects the whole file on one bad field turns every field into a single
point of failure for every consumer of that file, and the symptom appears in whichever
consumer is asked first.** When a subsystem reports "down", ask what it reads before asking
what it authenticates with.

**A checkout that is wrong and a machine that works look identical.** Any file that a deploy
copies needs a guard that validates the checkout copy, not the installed one, because the
installed one is the previous release.

**A guard that points a tool at a fixture must prove the tool looked at the fixture.** Break
the fixture on purpose and require the failure; a green run against an unread fixture is the
same green as no guard.

## Applied

- `tests/guard-pi-models-schema.bats`: an explicit allow-list of modalities, so a new one must
  be verified against the installed pi rather than assumed; and, when pi is present, pi's own
  verdict on the exact file the repo ships, exercised via `PI_CODING_AGENT_DIR`.
- Fixed in #1539.
