---
id: "lesson-504-a-fence-counter-that-never-stops-counting-eats-the-body"
type: lesson
status: active
title: "A fence counter that never stops counting eats the body"
created: "2026-10-20"
---

# A fence counter that never stops counting eats the body

## Context

`compile-harness.sh`'s `render_skill` strips a record's YAML frontmatter for
agy's flat prompts with `awk '/^---[[:space:]]*$/{fm++; next} fm>=2{print} fm==0{print}'`.
The PowerShell twin, `Convert-SkillRecord`, does the same with a regex that
matches the first frontmatter block only.

## Problem

The awk rule's `next` fires on every `---` line, not only the two fences, so
every horizontal rule in a prompt's body was dropped: 70 rules in 19 of the 35
committed records, on Linux and macOS only. Windows kept them. Nothing
noticed, because no test compared the two twins' output, and each twin's own
tests only covered records without a rule in the body. It surfaced the first
time both engines deployed the same records side by side (CLI-026 slice 1).

## Solution

The Go render (`harness.RenderSkillRecord`) stops counting after the second
fence. `tests/harness-skills-parity.bats` diffs the Go deploy against the bash
one over the committed records and declares this one difference explicitly.

## Rule

A frontmatter or region parser counts its delimiters only until it has the ones
it needs; past that, a delimiter is content. When porting an engine, run old and
new over the real inputs and diff before trusting either: a difference between
two twins is a defect in at least one of them, and the port is where it shows.
