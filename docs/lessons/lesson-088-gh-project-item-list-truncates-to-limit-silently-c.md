---
id: lesson-088-gh-project-item-list-truncates-to-limit-silently-c
type: lesson
status: active
created: "2026-06-13"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 088: `gh project item-list` truncates to `--limit` silently — check `totalCount` before asserting absence

**Context**: Verifying whether issues #344/#347/#350 had landed on the bitácora Projects v2 board after an `item-add`. Listed the board's items and grepped for the issue numbers; they were absent from the output, so the working diagnosis became "the add failed, the issues are missing from the board".

**Problem**: `gh project item-list` paginates with a default `--limit` of 30 and returns only that page — items beyond the limit are omitted with no warning and no non-zero exit. The board already held more than 30 items, so the issues were present but off the returned page. A grep over the truncated list "proved" an absence that was really just pagination. The JSON form exposes a `totalCount` that exceeded the returned `items | length`, but the naive list never compared them.

**Solution**: Before concluding an item is absent from a Projects v2 board, reconcile counts: `gh project item-list <n> --owner <o> --format json --limit 1000 | jq '{returned: (.items|length), total: .totalCount}'` and only trust an absence when `returned == total` (page exhausted). Pass a `--limit` >= `totalCount`, or page through, rather than grepping the default page.

**Rule**: Any CLI whose list command has a default `--limit` can answer an existence/absence question *wrongly* once the collection outgrows one page. Treat "not in the output" as "not on this page" until `totalCount` (or exhaustive paging) confirms the page was complete. Silent truncation reads as "covered everything" when it didn't.
