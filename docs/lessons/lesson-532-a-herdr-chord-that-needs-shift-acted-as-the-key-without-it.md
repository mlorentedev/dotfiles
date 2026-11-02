---
id: "lesson-532-a-herdr-chord-that-needs-shift-acted-as-the-key-without-it"
type: lesson
status: active
title: "A herdr chord that needs shift acted as the key without it"
created: "2026-11-02"
---

# A herdr chord that needs shift acted as the key without it

## Context

HERDR-001 (#2814) added two custom keys: `prefix+e` opens yazi in a popup, and `prefix+shift+e` ran
`dotf herdr explorer` to dock yazi as a split. `herdr config check` accepted both, and `dotf herdr explorer` run by
hand opened and closed the real split.

## Problem

On the owner's Mac, `ctrl+s` then `shift+e` opened the `prefix+e` popup, not the docked split. The two bindings
acted as one key. Nothing local could see it: the config validated, the binding test matched the key string, and
the command was correct. The failure lived between the keyboard and herdr, where the shift state did not arrive as
a distinct chord. The mechanism was not isolated (terminal, keyboard protocol or herdr's key parsing); only the
effect was observed.

## Solution

The docked explorer moved to a plain letter, `prefix+f` (#2992). Herdr 0.9.5 binds no default there, which the
config reference lists. The owner confirmed on the real terminal that `ctrl+s` then `f` docks and undocks it. A
bats test (`no custom herdr key relies on shift`, #3012) rejects any `[[keys.command]]` that names `shift`, so a
shifted chord needs a real-keyboard check before it can be added.

Two rules follow. A key binding is verified by pressing it: a green `config check` says the file parses, not
that the key reaches the program. And a custom key picks a plain letter that the built-in keymap leaves free,
checked against the config reference rather than assumed.
