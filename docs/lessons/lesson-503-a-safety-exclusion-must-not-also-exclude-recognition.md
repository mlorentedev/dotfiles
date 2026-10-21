---
id: "lesson-503-a-safety-exclusion-must-not-also-exclude-recognition"
type: lesson
status: active
title: "A safety exclusion must not also exclude recognition"
created: "2026-10-21"
---

# A safety exclusion must not also exclude recognition

## Context

`dotf herdr explorer` toggles a docked yazi pane. Its #2815 review found that the
toggle could close a stranger's yazi, and the fix excluded the invoking pane from
every candidate list, with a test asserting the invoking pane is never closed.

## Problem

The explorer is a pane people work in, so the key is pressed from inside it as
often as from beside it. With the invoking pane excluded from recognition too, the
explorer it owns was neither "ours" nor a stranger, and a second press docked a
second explorer instead of closing the first. The round 2 review reproduced it
against the released 0.76.0, which is what the new `prefix+shift+e` binding runs.
No test caught it: the one that existed asserted the exclusion that caused it,
and none drove the toggle from inside the thing it toggles.

## Solution

Ownership is the label, so `classify` reads the label before it skips the
invoking pane: a labelled invoking pane is ours and gets closed; an unlabelled one
is still never touched. The test of the bug now presses the key with the explorer
as the active pane.

## Rule

An exclusion added so a command cannot hurt something must exclude it from being
acted on, not from being recognised. And a guard test that asserts the exclusion
proves the fix, not the feature: drive the command from every position the user
can press it from, including inside the object it manipulates.
