---
id: "lesson-499-a-commit-pin-of-a-docker-action-does-not-pin-its-image"
type: lesson
status: active
title: "A commit pin of a docker action does not pin its image"
created: "2026-10-20"
---

# A commit pin of a docker action does not pin its image

## Context

`pr-agent.yml` pinned PR-Agent by commit (`The-PR-Agent/pr-agent@8e5a9295…  # v0.47.0`), and
`tests/pr-agent-config.bats` bound that commit to an audited contract of upstream blob SHAs. Dependabot
then opened #2829 to bump the commit to v0.48.0. The contract test failed it, as designed, because
ten audited files had changed.

## Problem

While re-auditing, the action's shape showed that the pin had never held. `action.yaml` runs
`Dockerfile.github_action_dockerhub`, which is only `FROM pragent/pr-agent:github_action`, a rolling
Docker Hub tag. That tag had moved to the v0.48.0 build on 2026-10-16. So the v0.47.0 "pinned" runs had
been executing v0.48.0 code for days. Run 39527011920 (2026-10-21) logs
`pragent/pr-agent:github_action@sha256:80edef71…`, the v0.48.0 image. The audited contract described
sources that were not running.
Every check passed. The commit was immutable, the blobs at that commit matched, and the test checked the
wrapper, which is the one part that does not run the review.

## Solution

Pin the image instead: `uses: docker://pragent/pr-agent@sha256:<digest>  # v0.48.0` (ADR-054, #2830).
The contract records `image_digest`. Its `audited_ref` is the commit that the image's build attestation
names (`gh attestation verify … --signer-workflow …/publish.yml`), and the network leg re-derives it, so
nobody types it in.

## Rule

Before a pin is called immutable, follow it to the bytes that execute. For `runs: using: docker`, read the
Dockerfile. A `FROM <image>:<tag>` there means the commit pins a pointer, not the code, and the pin must be
the image digest. The same holds one level down: a contract checked against the pinned ref verifies the
ref, not what runs, unless the ref is what runs (lesson 171).
