---
id: "lesson-369-read-shifts-fields-left-when-the-first-one-is-empty"
type: lesson
status: active
title: "read shifts fields left when the first one is empty"
created: "2026-10-08"
---

# read shifts fields left when the first one is empty

## Context
The PR-Agent publication guard receives every attempt as one `<outcome> <model>` line, built by
the workflow from `${{ steps.<id>.outcome }}`, and judges the last attempt that ran. It parsed
each line with `while read -r line_outcome line_model` and treated any outcome other than empty
or `skipped` as an attempt (#1923).

## The Trap
An outcome that expands to nothing leaves the line as ` anthropic/claude-haiku-5-5`. With the
default `IFS`, `read` strips leading whitespace before splitting, so the first variable gets the
MODEL and the second gets nothing. The guard then took `anthropic/claude-haiku-5-5` for an
outcome, skipped the "no attempt ran" branch and went on to measure. No error appears: the fields
are well-formed, they are just the wrong fields. A test that fed three empty outcomes caught it.

## The Solution
Accept only the known values (`success|failure|cancelled`) and ignore everything else, so a
shifted field can never pass for an outcome. More generally: when a positional line can have an
empty leading field, either match the field against its closed set of values, or use a separator
that is not whitespace (`IFS='|'`), which keeps an empty first field in place.
