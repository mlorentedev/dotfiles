---
id: lesson-132-never-read-a-locked-secret-store-as-absent-discrim
type: lesson
status: active
created: "2026-06-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 132: Never read a locked secret store as "absent" — discriminate before create, or you spawn duplicates

**Context**: `dotf secrets set`/`migrate` (#612) write a value into a Bitwarden item, creating the item when it does not exist. `BWPut.SetField` deliberately refuses to create; the create path lives in the command (C3, #621).

**Problem**: `bw get item <x>` fails the SAME way (exit 1) whether the item is genuinely missing OR the vault is locked / not-logged-in. A create-if-absent path that treats *any* read failure as "absent" will, against a locked vault, CREATE a duplicate of an item that already exists but was merely unreadable — silently, and worse under a non-interactive `--yes`.

**Solution**: Discriminate by the store's specific signal, not a generic error. `BWGet` wraps `ErrBWItemNotFound` ONLY when bw's message is "Not found."; a locked/unauthenticated vault yields a different message and falls through to fail-loud. `applySet` switches on the sentinel, so the create branch is reached only on a genuine not-found. Belt-and-braces: create still needs an interactive confirm or `--yes`. (`ErrBWFieldNotFound` similarly separates "item present, field missing" = append, from "item missing" = create.)

**Rule**: A create-if-absent against an auth-gated store must distinguish "genuinely absent" from "unreachable/locked" by a *specific* signal (a typed not-found, a status probe) before creating — never on an ambiguous error. When the only signal is a CLI message, match it narrowly, gate the create behind a confirm/`--yes`, and document the fragility (a structured API like `bw serve`, #622, replaces the string-match later).
