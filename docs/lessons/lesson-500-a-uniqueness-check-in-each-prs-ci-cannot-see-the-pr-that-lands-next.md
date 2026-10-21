---
id: "lesson-500-a-uniqueness-check-in-each-prs-ci-cannot-see-the-pr-that-lands-next"
type: lesson
status: active
title: "A uniqueness check in each PR's CI cannot see the PR that lands next"
created: "2026-10-20"
---

# A uniqueness check in each PR's CI cannot see the PR that lands next

## Context

`dotf lessons fmt --check` fails when two lessons share a number. It runs in every PR's CI, on that PR's merge
ref, and main does not require a branch to be up to date before it merges.

## Problem

Twice in two days, two PRs each took "the highest number on my base, plus one" and both landed green: lesson 478
(#2742, #2748) and lesson 496 (#2823, #2819, two minutes apart). Each PR's CI ran against a main without the other
lesson. Different slugs meant no git conflict and a `CLEAN` merge state. The duplicate check then failed on main
and on every open PR until someone renumbered by hand (#2751, #2826).

## Solution

`dotf pr land` lists the lessons on the current base right before it merges, through one read of the git trees
endpoint. It refuses a PR whose new lesson number another file already holds (#2752). It does not count a file
the PR removes or renames away, so a slug rename still lands. The read and the merge are seconds apart rather
than hours, and the land lock serialises lands from one machine; a merge from elsewhere inside that window is
still caught by the check on main.

## Rule

A check for a property of the whole tree (uniqueness, a total, an index) that runs per PR proves the property
only for the base that PR was checked on. When merges do not require an up-to-date branch, repeat the check
where the merge happens, against the live base, or accept that main finds out first.
