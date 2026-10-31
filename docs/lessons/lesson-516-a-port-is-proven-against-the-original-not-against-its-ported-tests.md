---
id: "lesson-516-a-port-is-proven-against-the-original-not-against-its-ported-tests"
type: lesson
status: active
title: "A port is proven against the original, not against its ported tests"
created: "2026-10-22"
---

# A port is proven against the original, not against its ported tests

## Context

TOOL-031 moves `scripts/pr-agent-route.sh` into Go as `dotf pr route` (#2893, CLI-101 #2217). The script has a bats
suite, and the first plan was to port each bats case to a Go test and call the port done when they passed.

## Problem

Ported tests check what their author thought to assert. Two behaviours of the script were in no bats case, and the
port got both wrong while every ported case passed:

- Under a forced provider (`PR_AGENT_PROVIDER=anthropic`) with no Anthropic member answering, the script prints
  its `::warning::` even when the risk route has already chosen the reviewer. The port skipped the warning on that
  path, because it only built candidates when a draw was needed.
- The probe's request body was a `map[string]string` per message, which `encoding/json` serialises in sorted key
  order (`content` before `role`). The script sends `role` first. Nothing reads the order, but "byte-identical to
  the script" was the claim, and it was false.

## Solution

A differential test runs the real script (bash, jq and curl) and `Route` on the same inputs against one httptest
server, and compares the key=value outputs, every `::warning::` line, the sequence of probes and the exit class:
523 subtests over the fixture pools and every draw point. It found the warning ordering on its first run. The body
order surfaced when the probe was compared byte-for-byte with the script's request. A drift test also holds Go's
allowlist and price rows equal to the bats file's until the script is deleted.

## Rule

When a port replaces a working implementation, keep the original runnable until the port lands and test the two
against each other on generated inputs. Ported unit tests prove the port matches the old tests; only the
differential test proves it matches the old behaviour. Delete the original, and the differential test with it, in
the change that switches callers over.
