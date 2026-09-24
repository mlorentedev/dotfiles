---
id: lesson-288
type: lesson
status: active
created: "2026-09-23"
owner: manu
tags: [lesson, secrets, bitwarden, verification, silent-failure]
---

# 288 — An inventory is a hypothesis until each row is probed

## What happened

Converging the Bitwarden store on 2026-09-23, every plan came from a written
inventory: which repositories consume `BITACORA_PAT`, which legacy items duplicate
a canonical one, which credentials are live. Probed row by row, value-free, most
of it was wrong.

- **Consumers.** GitHub code search found 7 repositories reading `BITACORA_PAT`.
  Reading every repository's workflow files found 23. The 16 it missed held tokens
  from June onward; several no longer existed, and knowledge's board sync was
  failing with `Bad credentials`. A sync scoped to the search results would have
  "unified" 9 of 23 and left the rest broken.
- **Duplicates.** The June inventory paired legacy items with canonical ones. Of
  the four pairs the plan blocked, three were not duplicates at all. The Gmail
  note held 1 token against 62 backup codes, none in common. The Hetzner SSH
  "copy" was a different key. The canonical `hetzner-ssh` held a *public* key.
- **Liveness.** `dotf secrets verify` reported `HETZNER_API_TOKEN` OK. Both
  copies answer 401; the live token was in another repository's store.
  `TS_AUTHKEY`'s two values were both past Tailscale's 90-day maximum.
- **The script that measured.** Checking which repositories read the token, a
  `for f in $files` loop found none in all 14 repositories, because zsh does not
  word-split an unquoted variable. The empty result looked like a finding. Only
  the observation that those repositories had `add-to-project` runs exposed it.

## Why it happens

An inventory records what someone believed when they wrote it, and nothing
updates it when the store moves. A search index answers what it has indexed, not
what exists. `verify` answers whether a value resolves, not whether it works.
Each is a proxy, and each fails the same way: a clean, plausible answer about
the part it can see.

## The rule

1. **Before any write driven by an inventory, probe each row by consequence.**
   Use an authenticated read for a token (`HTTP 200/401`), the public fingerprint
   for a key, a token-set comparison for a list of codes. Never print the value.
2. **Enumerate from the source, not from an index.** List the repositories and
   read their files; do not trust code search for "who uses this".
3. **A result of zero needs a positive control.** When a sweep finds nothing,
   check it against a case known to be positive before believing it. Here that
   case was repositories that demonstrably ran the workflow.
4. **Record the disproved rows where they were believed.** The inventory doc
   now names registry ids instead of repeating item names, and the registry
   carries a note saying why each retired entry went.

Refs: #586, #1596, #1624, CLI-082. Relates to lesson 283 (a portability bug that
answers "nothing found").
