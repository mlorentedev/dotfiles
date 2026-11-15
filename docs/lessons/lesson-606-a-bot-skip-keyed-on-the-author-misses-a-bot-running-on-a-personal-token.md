---
id: "lesson-606-a-bot-skip-keyed-on-the-author-misses-a-bot-running-on-a-personal-token"
type: lesson
status: active
title: "A bot skip keyed on the author misses a bot running on a personal token"
created: "2026-11-15"
---

# A bot skip keyed on the author misses a bot running on a personal token

## Context

Renovate was adopted for the `versions.conf` pins (#3284). The fleet runs it centrally from
`mlorentedev/ci-workflows` with a personal access token, so that its PRs trigger CI; a PR opened with
the workflow's own token would not.

## Problem

The spec-gate skips dependency PRs only when the author is `dependabot[bot]` or `renovate[bot]` and the
PR carries `dependencies`. A PR opened with a personal token is authored by the token's owner, so the
skip never applies. Measured with `scripts/check-spec-gate.sh --explain` on a commit moving the 29 pins
that were behind on 2026-11-15: 58 lines, over the 50-line threshold, gate failed. The preset groups
every minor and patch bump into one PR, so the first Renovate PR would have gone red for a reason that
had nothing to do with the bumps.

## Solution

Size the groups instead of widening the gate: one group per `depType`, each under the threshold.
A pin move is two lines, so a group of at most 24 pins stays under 50. `renovate_test.go` reads the
gate's `THRESHOLD=` and fails when a group would outgrow it, so the cap follows the gate. Teaching the
gate to skip a diff that changes only version values is the other option. It widens a gate, so it is
the owner's to decide (#3284). Before relying on a skip keyed on who authored a PR, check who the
author actually is for that integration.
