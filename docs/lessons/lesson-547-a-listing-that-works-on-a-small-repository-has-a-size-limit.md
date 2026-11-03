---
id: "lesson-547-a-listing-that-works-on-a-small-repository-has-a-size-limit"
type: lesson
status: active
title: "A listing that works on a small repository has a size limit"
created: "2026-11-02"
---

# A listing that works on a small repository has a size limit

## Context

`LandedHeads` on Gitea read every closed pull request, because the list endpoint has no head filter (lesson 497). It was measured on a small repository and worked. On `personal/resume`, with 453 closed pull requests, each page took about 5.5s, so the 30-second limit stopped it at page 6, and the 10-page cap would have refused it anyway. The swagger's by-head route, `GET /pulls/{base}/{head}`, answered in 1.2s and resolves a deleted branch.

## Rule

- Before reading a whole collection to find one row, look for the route that names the row. "The list has no filter" does not mean the API has no lookup.
- When a read has both a time limit and a page cap, check both against the largest real repository. Removing one may only reveal the other.
- Keep a read error apart from a parse error. A body cut off by the deadline reported "not readable", which pointed at the format rather than the clock (#3089).

## Check

`TestGiteaLandedHeadsReadsThePullOfThisBranchAgainstTheDefaultBranch` asserts two requests per branch, and `TestGiteaReadCutOffSaysSo` asserts the cut-off read.
