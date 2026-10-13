---
id: "lesson-399-a-deny-rule-that-mixes-wildcards-denies-nothing"
type: lesson
status: active
title: "A Claude Code deny rule that mixes * with :* denies nothing"
created: "2026-10-12"
---

# A Claude Code deny rule that mixes * with :* denies nothing

## Context
`ai/claude/settings.json` has carried a deny list since #1089 (2026-08-19), deployed by the
`claude-settings` entry to every machine. Six of its rules were written as
`Bash(git push *--force*:*)`, `Bash(rm -rf ~/*:*)`, `Bash(git reset --hard*:*)` and the like.

## The Trap
Claude Code has two rule forms. A trailing `:*` is a prefix match on everything before it,
read literally. A `*` anywhere else is a wildcard, and only in a rule without the `:*`
suffix. A rule that has both is read as the prefix form, so its inner `*` is a literal
asterisk: `git push *--force*:*` matches only a command that literally contains
`git push *--force*`. The six rules looked like protection for nearly two months and denied
nothing. Nothing failed: the settings parsed, every other rule worked, and a denied command
that is never denied makes no noise. Claude Code 2.x began warning about them at startup,
which is how they were found.

## The Solution
Each rule uses one form: `Bash(prefix:*)` with no other `*`, or a wildcard rule with no `:*`
(`Bash(git push *--force*)`). `tests/claude-settings-template.bats` fails on any rule in
`deny`, `allow` or `ask` that ends in `:*` and holds another `*`, and asserts that the
destructive git and rm rules are present in the wildcard form.

## Takeaways
- A permission rule fails open. A rule that matches nothing reads exactly like a rule whose
  command never ran, so its syntax needs a test, not a review.
- A deny list matches spellings, not intent: `git push -f` slips past `*--force*`. List the
  common forms of each destructive command and treat the list as best-effort, never as a
  sandbox.
- A merge deploy that unions lists adds the corrected rules but never removes the malformed
  ones: an existing box keeps them, warning included. `dotf doctor` reports them and `--fix`
  removes them, applying the same validation Claude Code runs at startup (#2350).
- When a tool offers two pattern syntaxes, a guard that rejects their mixture is cheaper than
  remembering which one wins.
