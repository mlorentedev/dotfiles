---
id: "lesson-436-to-share-a-layout-with-another-tool-measure-its-naming-rule"
type: lesson
status: active
title: "To share a layout with another tool, measure its naming rule"
created: "2026-10-14"
owner: manu
tags: [herdr, worktree, go, testing, interop]
---

# To share a layout with another tool, measure its naming rule

**Context:** CLI-118 (#2503) makes `dotf worktree add` create worktrees where herdr does, `<root>/<repo>/<branch-slug>`, so that each tool finds what the other made. herdr's config reference says only "<repo>/<branch-slug> checkouts", and herdr 0.9.3 has no option to change the shape.

**Problem:** "branch-slug" reads as "the branch with `/` turned into `-`", and that version would have passed for `feat/foo`. herdr does more. In an isolated session (`herdr --session <name> server` with `HERDR_CONFIG_PATH` pointing at a scratch config), `worktree create --branch` produced `feat-up-case-x` for `Feat/Up_Case.x` and `fix-x-y` for `fix/x--y`. It lowercases, turns every run of non-alphanumerics into one `-` and trims the ends. A formula derived from the docs would agree on the easy rows and diverge on the rest: `done <slug>` would look in the wrong directory and adoption would miss the worktree, with no error.

**Solution:** `ResolveWorktreePath` implements the measured rule, and `TestResolveWorktreePath` pins exactly the four measured rows, no invented ones. The probe used throwaway sessions (`herdr session delete` afterwards), never the operator's default server.

**Why:** when two tools must agree on a path, the other tool's output is the specification, not its prose. Choose probe inputs that separate plausible rules (case, dots, underscores, repeated separators, more than one `/`), and keep the test table to what you measured, so a new release that changes the rule is caught by a re-probe rather than hidden by guesses.
