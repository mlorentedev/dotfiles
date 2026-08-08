---
id: "BUG-042-closing-keyword-markdown"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#773"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-042-closing-keyword-markdown

## Why

<!-- from issue #773: check-spec-gate.sh closing-keyword scan is markdown-unaware -->

`_closing_issue_numbers` scanned the raw PR body with no awareness of markdown, so
it fired on any occurrence anywhere. #767's own body tripped its own check twice:
once on a fenced dogfooding transcript, once on a sentence describing a
different, future PR.

The structural point is worse than the instance: **a PR documenting a
text-scanning gate is the likeliest thing in the repo to contain a realistic
example of the exact string that gate hunts for.** Every future PR touching this
machinery is a candidate to trip it, and the escape hatch (`skip-archive`)
records a false claim in the durable record when used to route around a false
positive.

Separately, the pattern required whitespace between the keyword and `#`, so
`Closes: #N` was missed entirely — a bypass in the opposite direction.

## What

1. Strip fenced code blocks and inline code spans from the body before scanning.
2. Accept the colon form.

## The criterion, and what was rejected

The issue also proposed anchoring the keyword to a line or list-item start.
**Rejected.** GitHub matches a closing keyword anywhere in the body, so an
anchored gate would stay silent while GitHub closed the issue and left the spec
un-archived. For a check whose entire job is to demand the archive, a miss is far
worse than a false alarm.

The criterion adopted instead is alignment: **fire exactly when GitHub would
close the issue.** GitHub does not linkify — and so does not close from — a
reference inside code, which makes code the one region where "this is not a
directive" is provable rather than guessed. Ordinary prose is left alone
deliberately, because GitHub reads it too.

## Out of scope

- The Discipline Gate / archive-on-merge deadlock (#800) — same file, landed
  separately as #808.
- `dotf spec archive`'s own markdown-unaware tag scan (#769) — same root pattern,
  different language, its own PR.

## Acceptance criteria

- [x] **AC1** A closing keyword only inside a fenced block does not trigger.
- [x] **AC2** A closing keyword only inside an inline code span does not trigger.
- [x] **AC3** A genuine closing keyword in ordinary prose still triggers.
- [x] **AC4** Fence handling is delimiter-aware: prose after a closed fence is
      still scanned.
- [x] **AC5** `Closes: #N` triggers.

## Risks / open questions

- **Could a real declaration hide in a code span?** In principle, but a closing
  declaration written as code would not close the issue on GitHub either, so the
  gate stays aligned with the thing it mirrors.
