---
id: "lesson-614-narrowing-a-deny-rule-in-a-merged-list-retires-nothing-without-a-record"
type: lesson
status: active
title: "Narrowing a deny rule in a merged list retires nothing without a record"
created: "2026-11-19"
---

# Narrowing a deny rule in a merged list retires nothing without a record

## Context

HARNESS-201 let the agent force-with-lease its own branch. The old deny glob
`Bash(git push *--force*)` also matched `--force-with-lease=<branch>:<sha>`, and a deny rule beats any
hook decision, so the template had to narrow before anything could be allowed.

## Problem

Editing `ai/claude/settings.json` looked like the whole change. `permissions.deny` is a list and the
deploy merge unions lists, so the old glob stays in every settings file already deployed; the template
read right and the box kept denying the lease form. Two more traps sat on the same path. A glob such as
`...--force-with-lease=main:*` ends in the `:*` prefix form with another `*` before it, which Claude
Code rejects and which denies nothing (lesson 399); only a test over the template caught it. And a
glob cannot say "lease allowed, plain force refused", so the rule that mattered (the destination is not
the default branch) could not live in the template at all.

## Solution

Test the deployed effect, not the template text: deploy the old template, then the new one, through the
real merge with a record, and assert the old rule is gone and a rule the owner added stays. The
record is what removes it, so state the case with no record too (the old glob stays, the box is no
weaker, only not yet relaxed). Keep the rules a glob cannot judge in a hook that parses the command, and
pin the glob list and the allow list so a later edit cannot widen them silently. The pair of tests is in
`cli/internal/deploy/claude_push_rules_test.go` and `cli/internal/claude/push_rules_test.go`.
