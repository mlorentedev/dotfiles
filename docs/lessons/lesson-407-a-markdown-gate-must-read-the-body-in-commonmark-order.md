---
id: "lesson-407-a-markdown-gate-must-read-the-body-in-commonmark-order"
type: lesson
status: active
title: "A markdown gate must read the body in CommonMark order"
created: "2026-10-13"
---

# A markdown gate must read the body in CommonMark order

## Context
`scripts/check-knowledge-gate.sh` drops what GitHub does not render before it looks for the
`## Knowledge` section: code fences and HTML comments. It did that in two passes, fences first and
comments second, with a fence or comment opener recognised wherever its characters appeared.

## The Trap
Each construct can hold another's opener, and every pass that ignores that refuses a valid body.
Fixing #2310 turned up four cases, all in the same PR:

- A fence marker quoted inside a comment opened a fence that the comment pass never closed (#2310).
- A `` ```js `` line inside a fence counted as its closer, so the next bare marker opened a new
  fence that swallowed the rest of the body.
- A backtick line whose info string holds a backtick, such as `` ```code``` rest ``, counted as an
  opener. CommonMark reads it as inline code.
- A `<!--` inside an inline code span opened a comment. The PR's own description tripped this one:
  it mentions `` `<!--` `` in prose, and the gate reported its `## Knowledge` section absent.

Every case passed the suite, because each test wrote one construct at a time.

## The Solution
One left-to-right pass that keeps the state GitHub keeps: whether the line is inside a fence, or
inside a comment. On each line, the earliest of the competing openers wins: a code span or a
`<!--`. A closer has to be the opener's own kind:
- A fence closes on a bare marker of the same character, at least as long as the opener.
- A code span closes on a backtick run of the same length.

An independent review compared the rewrite with cmark-gfm (GitHub's renderer) on 12,000 generated
bodies, and the five awks it runs on against each other.

## Takeaways
- When a gate parses a grammar, test the constructs nested inside each other, not only one at a
  time. The one-at-a-time suite was green throughout.
- Feed the gate the description of the PR that changes it. A PR about markdown parsing quotes the
  very syntax it parses.
- Two gates that strip markdown should share one implementation. The spec gate still carries its
  own copy, with the closer defect fixed here: #2407.
