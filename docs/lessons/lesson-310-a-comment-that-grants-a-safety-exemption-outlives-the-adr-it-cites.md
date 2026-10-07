---
id: "lesson-310-a-comment-that-grants-a-safety-exemption-outlives-the-adr-it-cites"
type: lesson
status: active
title: "A comment that grants a safety exemption outlives the ADR it cites"
created: "2026-09-26"
owner: manu
tags: [lesson, secrets, doctor, adr, exemption, dr]
---

# A comment that grants a safety exemption outlives the ADR it cites

## What happened

`dotf doctor` used to FAIL on a `sensitive/*.secret.age` that no registry entry claimed. BUG-078 (#971) downgraded that to a WARN: `migrate` drops a secret's `age:` pointer, and the comment next to the check said the leftover blobs were "the ADR-028 floor" and must survive. That sentence was never checked against ADR-028. ADR-028 §5 names a different floor, the whole-vault escrow `sensitive/dr/bitwarden-export.age`, and its ratification gated deleting the blobs on that escrow, not keeping them.

By 2026-09-25 the escrow existed and had been verified with the offline key alone (#1000). The exemption still held. 31 committed blobs were decryptable copies that no Bitwarden rotation would ever reach, in a public repository. Every machine's `~/.dotfiles/sensitive/` kept its own copy, because the deploy never pruned (#802). Three places still depended on them: a rollout script's decryption fallback, a Windows CI step that planted a dummy blob, and two READMEs telling the reader to create one.

## The rule

An exemption from a safety check has to carry the condition under which it expires, and a test that fails when that condition is met. "These are the floor" was true only until the escrow shipped, and nothing connected the two. When the condition was met, the WARN kept firing on every doctor run, and a WARN that fires on every run is read as noise.

When a comment cites an ADR to justify weakening a check, quote the ADR's own sentence, not a paraphrase. A paraphrase drifts and a quote can be checked.

Refs: CLI-036 (#938), #971, #802, #1000.
