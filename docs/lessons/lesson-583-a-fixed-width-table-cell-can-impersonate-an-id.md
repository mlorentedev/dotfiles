---
id: "lesson-583-a-fixed-width-table-cell-can-impersonate-an-id"
type: lesson
status: active
title: "A fixed-width table cell can impersonate an id"
created: "2026-11-11"
---

# A fixed-width table cell can impersonate an id

## Context

The `system-upgrade` converge step (TOOL-036) asks `winget upgrade` which catalog packages are outdated. winget has no
machine-readable listing for this, so the step parses its console table: `Name`, `Id`, `Version`, `Available`,
`Source`, padded to fixed widths.

## Problem

The first parser (#3287) took a line as outdated when any whitespace-separated field equalled a catalog id. A `Name`
cell can hold text that looks like an id, and winget truncates long names with an ellipsis, so a package outside the
catalog could match by its name and be upgraded. That breaks AC3 (only catalog entries are upgraded). The fix (#3295)
read only the `Id` column, sliced at the start index taken from the header line. Review then found the slice was
counted in runes. winget pads by display cells, and a CJK or fullwidth character fills two cells, so a wide name
shifts every later column and the slice cuts into the id (`it.Git` for `Git.Git`).

## Solution

Locate the column from the header and count display cells, not runes or bytes (`golang.org/x/text/width`: wide and
fullwidth runes are two cells). Read the id from that column only, and compare it exactly against the catalog. Tests
cover a name cell that contains a catalog id, an ellipsis-truncated name, and a wide-character name
(`TestSystemUpgradeWindowsMatchesOnlyTheIdColumn`, `TestSystemUpgradeWindowsIdColumnCountsDisplayWidth`).

## Takeaway

When a tool offers only a human table, parse by column position, never by "some field equals". Take the positions
from the header line, measure them in the unit the tool pads by (display cells for a terminal table), and match exact
values in the one column that carries the identifier. Lesson 113 is the same class for winget: its output shape must
be observed, not assumed. The English text of its messages is no safer, because winget localizes them (#3343).
