---
id: "lesson-494-tools-sync-reads-the-deploy-dirs-versions-conf-outside-the-checkout"
type: lesson
status: active
title: "tools sync reads the deploy dir's versions.conf outside the checkout"
created: "2026-10-19"
---

# tools sync reads the deploy dir's versions.conf outside the checkout

## Context

Adding `YAZI_VERSION` to `versions.conf` looked like it had not taken effect,
and the two runs of `dotf tools sync` disagreed with each other:

```
$ cd cli && /tmp/dotf-tree tools sync --dry-run     # runs inside the checkout
would write /Users/manu/.config/mise/conf.d/dotfiles.toml:
  yazi 26.9.1
$ cd /tmp && /tmp/dotf-tree tools sync              # runs OUTSIDE the checkout
40 tool(s) at their pin; nothing to do
```

The rendered mise config on disk never gained the pin, so `yazi` stayed installed
but inactive (`mise ERROR yazi is a bin however it is not currently active`), and
`dotf tools version yazi` said "no version found on PATH".

## Cause

`tools sync` resolves its `versions.conf` with `env.ResolveVersionsPath()`: the
checkout's when the working directory is inside one, the **deployed** copy
otherwise. The deployed copy is whatever the last release left there, so a pin
that exists only in the tree is invisible to `sync` run from anywhere else — and
the run reports a clean "nothing to do" for the 40 pins it did read, which looks
like success, not like a stale source.

## Rule

When validating a new or bumped pin, run the sync **from the checkout** (or pass
`--versions` explicitly) and confirm the pin reaches the rendered config:

```
grep -n '<tool>' ~/.config/mise/conf.d/dotfiles.toml
```

A "nothing to do" from outside the checkout says nothing about the tree's pins.
The second run from the checkout is still the idempotence check: "nothing to do"
there, with the pin on disk, is the pass.
