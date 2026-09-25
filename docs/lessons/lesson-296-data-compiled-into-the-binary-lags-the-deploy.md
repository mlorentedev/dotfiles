---
id: lesson-296
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, harness, deploy, release, routing]
---

# 296 — Data compiled into the binary lags the deploy

## What happened

SKILL-001 retired `verification-before-completion` and deployed the new skill records the same day. The prompt hook kept suggesting the retired skill on every git-workflow prompt. The installed `dotf` (0.58.0) carried the skill dependency map compiled in, and the map still expanded a routed skill to the retired one. A binary built from main did not. So the retirement reached the hook only when a release was installed, hours after the records deployed.

## The rule

Data that describes deployed content belongs with that content. The router now reads each record's `requires:` at run time from the deploy dir (HARNESS-147, #1714). The compiled map is only the fallback for a machine with no records, and a test keeps the two equal. When a binary must carry a copy of deployed data, test that the copy matches the source, and deploy the data before the binary that reads it.

Refs: #1693, #1714.
