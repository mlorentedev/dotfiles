---
id: "lesson-404-claude-plugin-install-reads-a-marketplace-copy-it-never-refreshes"
type: lesson
status: active
title: "claude plugin install reads a marketplace copy it never refreshes"
created: "2026-10-13"
---

# claude plugin install reads a marketplace copy it never refreshes

## Context
The from-zero job (#2331) converges a fresh ubuntu and macOS runner twice and runs
`dotf doctor` after each pass. On run 38365444507 the Claude plugin step reported every
declared plugin as failed on both passes, while the same step was green on every machine
that had used Claude Code before (#2378).

## The Trap
A fresh Claude Code already knows the `claude-plugins-official` marketplace, so nothing
looks missing: the marketplace is registered and `claude plugin install gopls-lsp@...`
reaches it. But the install reads the local copy of the marketplace, and that copy predates
the plugins. The CLI said so (`Plugin "gopls-lsp" not found in marketplace ... try claude
plugin marketplace update`), and `install` does not run that update itself. A machine that
had ever opened Claude Code interactively had a refreshed copy, so the defect only existed
on new machines, which is where nobody looks.

## The Solution
`Syncer.Sync` runs `claude plugin marketplace update <name>` once per marketplace that holds
a plugin to install, before the first install from it. It refreshes nothing when every
plugin is present or on a dry run, so a converged box pays no network call. A failed refresh
is not fatal, because the copy may already list the plugin; it is appended to the cause of
any install from that marketplace that then fails.

## Takeaways
- A registered dependency is not a current one. When a tool resolves names against a local
  index (a marketplace, an apt list, a helm repo), refresh the index before resolving on a
  machine you did not build, the same way `apt-get update` comes before `apt-get install`.
- Read the failing CLI's own hint before writing a workaround: here it named the fix.
- A step that only fails on a fresh machine needs a fresh machine in CI. The from-zero job
  found this; no amount of local reruns would have.
