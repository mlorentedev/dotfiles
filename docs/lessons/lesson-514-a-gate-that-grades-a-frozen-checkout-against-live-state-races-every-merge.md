---
id: "lesson-514-a-gate-that-grades-a-frozen-checkout-against-live-state-races-every-merge"
type: lesson
status: active
title: "A gate that grades a frozen checkout against live state races every merge"
created: "2026-10-22"
---

# A gate that grades a frozen checkout against live state races every merge

## Context

`dotf spec audit` runs on the test-windows leg of every code PR. It fails any active spec whose tracking issue
is CLOSED (a zombie), so a spec cannot be forgotten after its issue closes some other way (#1087). It reads the
active specs from the run's checkout and asks the forge for each issue's state.

## Problem

On 2026-10-22 test-windows on #2897 failed with `GUARD-003-review-loop-determinism: zombie: #1052 is CLOSED`, a
spec the PR never touched. #2898 had archived GUARD-003 and closed #1052 at 04:30Z, between the run's start
(04:22) and its audit step (04:32). The checkout is a merge ref cut when the run is triggered, so it still held
the active spec, while the issue state was read live and was already closed. The two inputs came from
different moments, and the archive merged in between. A re-run did not help: it reuses the same merge ref.

The first fix checked only `specs/archive/<id>` on the default branch. The live check against a second stale
ref then showed `FIX-WIN-TUI-001` still failing: it had been abandoned, and `spec archive --abandoned` routes to
`specs/archive/_abandoned/<id>`. A unit test written from the reading of the code would not have found that.
Only the live run did.

## Solution

When a spec grades as a zombie, the audit asks the forge whether the default branch already holds the spec
under either archive route (`spec.ExcuseArchivedZombies`, `spec.GHArchiveLookup`). If it does, the finding is
a WARN that says to merge the base. If it does not, the FAIL stands. If that second question cannot be
answered, the FAIL also stands: the zombie question was answered, and an unanswered excuse must not clear it.
On a push to main the default branch is the checkout, so nothing changes there.

## Rule

A gate that compares a snapshot (a checkout, a merge ref, a cached file) with live state (an issue, a
release, a board) races every merge that changes the truth between the two reads. Before failing, ask the
live side whether the base has already caught up, and fail closed when it cannot say. When the base can store
the same thing under more than one path, check every route the producer writes, and prove it against a real
stale ref, not only a fake.
