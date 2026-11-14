---
id: "lesson-600-bsd-sed-reads-a-gnu-alternation-as-no-match-and-says-nothing"
type: lesson
status: active
title: "BSD sed reads a GNU alternation as no match and says nothing"
created: "2026-11-14"
---

# BSD sed reads a GNU alternation as no match and says nothing

## Context

#3437 made the lane guard warn when a lease has expired (#3438). The guard is POSIX sh and the lease
holds an RFC 3339 instant in the machine's own zone, so the guard parses it with `sed` and does the
date arithmetic in the shell: `date -d` is GNU-only and `date -j` is BSD-only. The first cut matched
the zone with a basic regular expression, `\(Z\|z\|[+-]HH:MM\)`.

## Problem

`\|` is a GNU extension to basic regular expressions. BSD sed, which is what macOS ships, does not
read it as alternation, so the pattern matched nothing and `sed -n ... /p` printed nothing. No error,
no warning: the parse yielded an empty string, the helper treated an empty instant as "no expiry",
and every lease looked valid. The test that wrote an expired lease failed on the Mac with an empty
stderr, which reads exactly like a guard that works and has nothing to say. On the Linux leg the same
script passed.

## Solution

Use extended expressions, `sed -nE`, with `|`, `{n}` and `(...)?` instead of the `\|` and `\{n,m\}`
spellings: `-E` is understood by GNU, BSD, BusyBox and Git for Windows sed. And read an empty parse as
a defect to test, not as a negative: the test now writes expired and valid leases in UTC and in zones
twelve and fourteen hours away, and asserts the warning appears for the expired ones, so a parser that
silently returns nothing cannot pass. The `.claude/CLAUDE.md` rule against BSD-incompatible `sed` forms
(`\s`, `\b`, the `I` flag) covers the same ground; this is one more spelling of it.

## Related

- #3437, #3438, the guard change this came out of.
- Lesson 268: a refused question and a negative answer print the same thing.
