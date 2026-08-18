---
id: lesson-189-an-apostrophe-in-a-comment-inside-an-open-awk-bloc
type: lesson
status: active
created: "2026-08-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 189: An apostrophe in a comment inside an open `awk '...'` block reopens bash's own parser

**Context**: HARNESS-069 (#917), extending `scripts/compile-harness.sh`'s `render_skill()` with a rationale comment explaining why a new awk rule strips pre-existing `generated_*` fields before re-injecting fresh ones. The comment read, in part, "...describing `$HOME's` relationship to the record...". `bash -n` on the whole script failed with `syntax error near unexpected token '_from'`, pointing at a completely different line — an unrelated regex three lines below the comment.

**Problem**: the comment lived *inside* an already-open bash single-quoted string (the `awk -v ... '...'` block spans several lines as one token). Bash single quotes have no escape mechanism and no concept of a nested comment — the very next `'` character, wherever it occurs and for whatever reason, closes the string. The apostrophe in `$HOME's` was exactly that character. Everything after it — several more lines, including the next awk rule's regex `/^generated(_from|_sha)?:/` — became unquoted bash source, and `(_from|_sha)` is not valid bash syntax outside a string. The reported error location (the regex) was three lines downstream of the actual defect (the apostrophe), which is the general shape of this failure: the parser doesn't fail where the quote broke, it fails wherever it next hits something it can't parse as bare bash.

The blast radius was also wider than the one function: because `compile-harness.sh` is sourced as a whole script, the syntax error broke *every* code path in the file, not just `render_skill`. The bats run showed dozens of unrelated tests failing (`ENGINE-002`, `HARNESS-054`, the plain `do_refresh` smoke tests) with the unhelpful `run_refresh; [ "$status" -eq 0 ]' failed` — nothing in that output pointed at an awk comment three functions away.

**Solution**: `bash -n scripts/compile-harness.sh` immediately isolated the actual syntax error and its line number, which was the fast path to the fix — reword the comment to avoid the contraction ("`$HOME's relationship`" → "the deploy target's relationship", still avoiding a literal `'`) rather than trying to escape the apostrophe (`'\''` inside a multi-line awk-in-bash comment is its own hazard, and not worth it for prose).

**Rule**: inside any `awk '...'`, `sed '...'`, or other multi-line single-quoted bash block — including its comments — a literal apostrophe closes the string exactly like anywhere else; bash does not distinguish "comment" from "code" while scanning for the matching quote. When a script written across several such blocks throws a syntax error whose line number doesn't match anything obviously wrong, check every open single-quoted region *above* that line for a stray apostrophe before assuming the reported line is where the bug is. `bash -n` is cheap and finds this in one shot — run it after any edit that touches text inside a quoted block, not only after edits to code.

**Tags**: `shell`, `bash`, `awk`, `debugging`
