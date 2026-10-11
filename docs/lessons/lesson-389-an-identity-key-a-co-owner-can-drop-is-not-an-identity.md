---
id: "lesson-389-an-identity-key-a-co-owner-can-drop-is-not-an-identity"
type: lesson
status: active
title: "An identity key a co-owner can drop is not an identity"
created: "2026-10-10"
---

# An identity key a co-owner can drop is not an identity

## Context
`dotf harness bind` writes our hooks into each harness's settings file and tags every entry with
a sidecar key, `_managed: dotfiles-harness:<id>`, so a later bind finds its own entries and never
touches a third party's. The marker was the only identity: an entry without it was foreign.

## The Trap
`~/.claude/settings.json` has more than one writer. Measured 2026-10-10 (#2232): the bind at
21:20 left four marked hooks; by 23:54 the file had been rewritten with the same four hooks and
no markers. The other writer kept the keys it knew and dropped the one it did not. From then on
every bind saw four foreign hooks plus four missing ones, and reported drift on a converged
machine. Nothing failed loudly, and the next writer to strip the markers would undo every repair.

The key was ours, but the file was not. A field we add to a document someone else owns survives
only as long as their serializer preserves unknown keys, and nothing promised that.

## The Solution
- Identity is the marker when it is present, else the command signature: a first token whose
  basename is `dotf` or `dotf.exe`, followed by exactly the manifest's arguments. The binary
  path is not part of it, so a moved binary is replaced in place rather than duplicated.
- The currency check compares entries without the marker, so a stripped marker alone is not
  drift. The marker is still written; it is never required.
- Tests strip every marker from a fresh bind and require `changed=false`, and a mutation that
  drops the signature arm fails them. A non-dotf binary with our arguments is never adopted.

## Takeaways
- In a file with several writers, identify your entries by what they do, not by a tag only you
  understand. A tag is a hint the other writers are free to lose.
- "Converged" must be judged on the content you need, never on your own bookkeeping, or every
  foreign rewrite turns into a permanent false drift.
