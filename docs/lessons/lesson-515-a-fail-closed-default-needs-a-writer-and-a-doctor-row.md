---
id: "lesson-515-a-fail-closed-default-needs-a-writer-and-a-doctor-row"
type: lesson
status: active
title: "A fail-closed default needs a writer and a doctor row"
created: "2026-10-21"
---

# A fail-closed default needs a writer and a doctor row

## Context

ADR-032 section 8 makes a machine with no declared identity deny every
non-local pool. The reader (`LoadMachinePolicy`) shipped with #1213. For two
months nothing wrote the id and nothing reported its absence (#1236, #1547).

## Problem

The default was right and the loop around it was open. The only way to satisfy
the gate was a hand edit of a JSON file, which the repo forbids, and the only
report of the gap was the dispatch itself refusing, at the moment someone needed
it. It read as a broken dispatch, not as an undeclared machine. The Mac ran
unattended sessions for weeks without an advisor because of it.

## Solution

Ship a fail-closed gate with its three halves in one change: the reader that
denies, the idempotent verb that satisfies it (`dotf env identity set`, which
never derives the value and never overwrites a declared one), and a `doctor`
row that names that verb before the first use. A default that denies is only
safe when the way out is one command away and the denial announces itself.

## Prevention

When a PR adds a deny-by-default reader, the review asks two questions: which
command writes the value, and which `doctor` row reports it missing. A ticket
that answers "nothing" to either is the same gap, filed late.
