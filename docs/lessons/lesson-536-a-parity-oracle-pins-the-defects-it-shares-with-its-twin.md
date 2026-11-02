---
id: "lesson-536-a-parity-oracle-pins-the-defects-it-shares-with-its-twin"
type: lesson
status: active
title: "A parity oracle pins the defects it shares with its twin"
created: "2026-11-02"
---

# A parity oracle pins the defects it shares with its twin

## Context

CLI-026 ported `scripts/compile-harness.sh` to Go verb by verb. Each verb was held to the script by a parity
oracle: a bats file that deployed with both engines into two throwaway HOMEs and diffed the result.

## Problem

An oracle that diffs against the twin can only prove the port matches the twin. Three defects the two engines
shared (#2884: an empty `capabilities: []` failing open, two region shapes that dropped the user's lines) never
showed up in a diff; review found them. A fourth, a deployed skill directory keeping every helper its record
ever shipped (#2851), was deliberately kept in Go, because fixing it would have turned the oracle red. The
oracles also deployed into fresh HOMEs, so a behaviour that only shows on a populated HOME, such as a stale file
left behind, was invisible to them in either direction.

## Solution

Treat a parity oracle as scaffolding with an end date. While it is in force, record every defect it pins as a
declared divergence or a deferred ticket that names the oracle's deletion as its unblocking event (#2851 did:
"once the bash twin is retired"). Delete the oracle together with the twin, then work those tickets
immediately: the Go verb is the only engine left, and its own tests, which can assert the correct behaviour,
replace the comparison. An oracle that only seeds empty state also needs a populated-state case before anyone
reads its green as equivalence.
