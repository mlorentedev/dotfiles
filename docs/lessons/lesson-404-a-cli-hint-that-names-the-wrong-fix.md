---
id: "lesson-404-a-cli-hint-that-names-the-wrong-fix"
type: lesson
status: active
title: "A CLI hint that names the wrong fix: claude-plugins-official was not stale, it was not registered"
created: "2026-10-13"
---

# A CLI hint that names the wrong fix: claude-plugins-official was not stale, it was not registered

## Context
The from-zero job (#2331) converges a fresh ubuntu and macOS runner twice. On run 38365444507
every declared Claude plugin failed to install on both ubuntu converges (the macOS runner installs
no Claude Code, #2317), while the same step was green on every machine that had used Claude Code
before (#2378).

## The Trap
The CLI explained its own failure: `Plugin "gopls-lsp" not found in marketplace
"claude-plugins-official". Your local copy may be out of date — try claude plugin marketplace
update claude-plugins-official`. The first fix did exactly that: refresh each marketplace once
before installing from it. Its unit tests passed, and it changed nothing. A from-zero run
dispatched on the branch before the PR existed (38366816961) showed why:

```
Failed to update marketplace(s): Marketplace 'claude-plugins-official' not found.
Available marketplaces: anthropic-plugin-directory
```

The marketplace was not stale; the fresh box did not have it at all. The `latest` Claude Code the
runner installs no longer registers `claude-plugins-official` on its own, and the install error
reads the same for a missing marketplace as for a stale one. Every machine already in use had it
registered from an older Claude Code, so the defect could only exist on a new machine.

## The Solution
`ai/claude/marketplaces.json` declares where each plugin's marketplace comes from
(`claude-plugins-official` → `anthropics/claude-plugins-official`, whose `marketplace.json` still
carries that name and the three plugins). Before the first install from a marketplace,
`Syncer.Sync` lists the registry once, then adds the marketplace from its source when the box
lacks it, or refreshes it when it has it. A converged box or a dry run makes none of these calls.
A failed preparation does not stop the install; it is named in the cause of an install that then
fails. A Go test fails when a declared plugin's marketplace has no source.

The sources are their own file because every released `dotf` reads `plugins.json` with
`DisallowUnknownFields`: a new field there would have failed the plugin step on any box whose
binary predates it.

## Takeaways
- A hint is the tool's guess at the cause. Test it against the state before building on it:
  `claude plugin marketplace list` on the failing machine would have answered in one line.
- Prove a fix on the environment that failed, before the PR. The unit tests encoded the hint,
  so they passed for the wrong fix; only the dispatched from-zero run could refute it.
- Something a tool used to do for you implicitly (register a default marketplace) is a
  dependency. Declare it, or a version bump removes it silently.
- An unpinned tool in CI (`latest` Claude Code) is why the runner and every workstation
  disagreed. That gap is what made this reproducible, and also what will make the next one.
