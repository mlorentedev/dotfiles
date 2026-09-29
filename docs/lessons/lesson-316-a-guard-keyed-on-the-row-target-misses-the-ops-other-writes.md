---
id: lesson-316
type: lesson
status: active
created: "2026-09-28"
owner: manu
tags: [lesson, secrets, authorization, review, mutation-testing]
---

# 316 — A guard keyed on the row's target misses the op's other writes

## What happened

`dotf secrets curate` refuses to touch an item the registry declares, because `reconcile` owns those items. The check ran once per plan row, on the row's **target**. That covered every op but one. `merge-delete` deletes its target and **writes to a second item**: it carries the URIs the keeper lacks. The keeper was never checked, so a merge whose keeper the registry declares would have edited that item silently.

The mutation check did not catch it. It disabled the ownership guard, a test failed, and "registry ownership" was recorded as covered. The guard itself was covered. The seam it did not reach had neither a test nor a mutant, and a mutation run can only kill a mutant of code that exists. The independent review (SEC-006 round 1, `review-round-1.md`) found it by reproducing the write.

## The rule

Key an authorization check on **each write the operation makes**, not on the row that asked for it. List every item an op writes to, then place the guard on each one. The check belongs where the write is decided (`keeperRefusal` runs when `Carry` is non-empty), so a read-only use of the same item stays allowed.

Record mutation coverage **per seam, not as a count**. "12 guards killed" hid a missing thirteenth guard. A line per seam (target ownership, keeper ownership) makes the missing one visible.

Refs: SEC-006 (#1784), lesson 312.
