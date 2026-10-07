---
title: "A repo named differently from its vault slug keeps its memory off the vault"
date: "2026-10-06"
---

# A repo named differently from its vault slug keeps its memory off the vault

## Context
A pre-migration audit of a project whose repository directory and vault project slug differ (the
repo is named after the product, the vault project after the product family). Agent sessions in
that repo had been running for a week.

## The Trap
`memlink` resolved the vault memory as `10_projects/<repo basename>/memory`. With no project of
that name, it returned nothing: on a fresh machine `dotf doctor` reports SKIP ("no vault memory
source"), and Claude Code then creates a real per-project memory directory, which doctor only flags
as a WARN if someone runs it. Nothing failed: sessions simply never loaded the
vault handoff, and a feedback memory written there lived only on the local disk, so a workstation
migration would have dropped it. Once that directory is non-empty, `Ensure` never replaces it, so a
later fix to resolution alone would not have recovered the link.

## The Solution
The vault already recorded the mapping in each project's `context.md` frontmatter as `repo_url`.
`resolveVaultMemory` now falls back to the single project whose `repo_url` names the repo (#2022),
and refuses to pick when two projects claim it, because doctor reports an unlinked directory while a
link into the wrong project would be silent. A SKIP from doctor's auto-memory check is a claim that
the repo has no vault project; when it does have one, the SKIP is the bug.
