---
id: "ADR-043-a-wip-limit-on-active-specs"
type: adr
status: accepted
owner: manu
date: "2026-09-30"
supersedes: []
extends: [adr-039-knowledge-asked-for-by-the-pr]
issue: mlorentedev/dotfiles#770
tags: [process, decision, sdd, specs, wip]
created: "2026-09-30"
---

# ADR-043: `dotf spec init` refuses a new spec at the WIP limit

## Context

This repository starts work faster than it finishes it. Active specs on `main`, measured 2026-09-30:

- There are 32 active specs.
- 5 have every task ticked and were never reviewed or archived.
- 8 are at 80% or more.
- 11 are under 30%, several untouched since June.

The knowledge gate (HARNESS-160) is typical: it works, and its last four tasks were never done. #770, the issue about this residue, had been open since August.

Nothing stopped a new spec. `spec init` checks only that an open issue gates it. Every new start is locally justified, so the pile grows one reasonable decision at a time. ADR-039 and lesson 301 already record the pattern: a practice that no command asks for is not followed.

## Decision

`dotf spec init` refuses to scaffold while the repository has its limit of active specs or more:

- **Active** means a folder under `specs/` with a `proposal.md`, not counting `specs/archive/`.
- **The limit** is 10, unless the repository declares another in `specs/.wip-limit` (one positive integer). A malformed file is an error, not a silent default.
- **The refusal** names the count and the limit, and gives the two ways out: archive a finished spec, or abandon a stalled one with `dotf spec archive --abandoned`.
- **`--over-wip-limit "<reason>"`** scaffolds anyway. It writes `wip_override: "<reason> (N active, limit L, date)"` into the new proposal's frontmatter, so the reviewer and the archive see it. A blank reason is not an override.

## Consequences

- **At merge time, every new spec in dotfiles needs an override** (32 active against a limit of 10) until the closing sweep in #770 runs. That is the intended pressure.
- **kubelab has 26 active specs and hive has 5.** kubelab blocks too, unless it declares its own limit in `specs/.wip-limit`.
- **An override is visible, not free.** `grep wip_override specs/` lists every one.
- **Abandoning counts as finishing.** `archive --abandoned` keeps the intent on the issue and takes the spec out of the count.
