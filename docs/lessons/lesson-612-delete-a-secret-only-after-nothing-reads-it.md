---
id: "lesson-612-delete-a-secret-only-after-nothing-reads-it"
type: lesson
status: active
title: "Delete a secret only after nothing reads it"
created: "2026-11-16"
---

# Delete a secret only after nothing reads it

## Context

#3379 retired the agents' Gitea token from the workstation. #3563 removed the only reader: the
`agent_secrets` routing in `forge/hosts.json` and the registry entry. The owner deleted the
Bitwarden item while that PR was still open.

## Problem

`dotf` reads `forge/hosts.json` and the registry from the dotfiles checkout. Until #3563 merged and
that checkout was pulled, a write to a `personal/` repository still resolved the deleted item. A
live session's `dotf forge pr create` failed with `bw item not found`. Separately, `dotf secrets
verify` kept reporting OK for both deleted items, because `bw serve` answers from its local cache
until it syncs.

## Solution

Retire a secret in this order: merge the change that stops reading it, update the checkout `dotf`
reads (and release the binary when the reading logic lives there), then delete the store item.
Until #3578 makes `dotf secrets verify` sync first, an OK from it says only that the `bw serve`
cache holds the item, not the vault. Sync `bw serve` before trusting it on a deletion.
