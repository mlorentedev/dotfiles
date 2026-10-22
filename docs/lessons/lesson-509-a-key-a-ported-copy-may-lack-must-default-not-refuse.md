---
id: "lesson-509-a-key-a-ported-copy-may-lack-must-default-not-refuse"
type: lesson
status: active
title: "A key a ported copy may lack must default, not refuse"
created: "2026-10-21"
---

# A key a ported copy may lack must default, not refuse

## Context

`dotf pr triage-queue` reads `harness/review-attestation.json` from the
repository it is asked about. The file is hand-ported: dotfiles and twelve
other repositories each carry a copy. GUARD-003's review (F1, #2870) found
that a triage record counted no matter who wrote it, so the fix adds
`triage.standing`, the author associations whose record counts.

## Problem

The obvious shape copies `triage.marker`: refuse a registry that does not
declare the key, fail-closed. For the marker that is right, because no marker
is safe to assume. For a new key it would break the queue in every repository
whose copy predates it, on the release that ships the check. kubelab measured
that failure for the marker itself: its copy omitted the key, the queue
answered "declares no triage.marker" on every call, and because exit 1 also
means "pending" nobody read it for the whole life of the registry.

## Solution

An absent `standing` takes a default in code, `OWNER, MEMBER, COLLABORATOR`,
the set `pr-agent.yml` already trusts to ask for a review. An explicit empty
list is still refused. dotfiles declares the key anyway, and tests hold the
three spellings together: the code default, the registry and the workflow's
`fromJSON` list.

## Takeaway

Before making a key required, count the copies of the file that will be read
by the new code. Refuse an absent key only when no value is safe; when one is,
default to it, and make the default the secure value rather than the
permissive one. "Fail-closed" means not answering wrongly, and an
unanswerable check that runs everywhere is read nowhere.
