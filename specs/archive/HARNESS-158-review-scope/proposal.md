---
id: "HARNESS-158-review-scope"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-09"
issue: "mlorentedev/dotfiles#1727"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# HARNESS-158: review a spec over its own commits

## Why

<!-- from issue #1727: HARNESS-158: a spec whose slices merge separately is reviewed over every commit merged in between, not over its own -->

`dotf spec review` hands the reviewer `git diff <base>...HEAD`, where the base is the parent of the commit that first added the spec folder. A spec whose slices merge to main one at a time is then reviewed over everything other work merged in between. HARNESS-142's closing review ran over 14 commits, of which 5 were the spec's own; MEMORY-009's ran over 11 unrelated commits including a release. Only a note in `verification.md` kept those reviewers on scope, and a reviewer that judges unrelated commits either wastes its budget or files findings against work the spec never touched.

## What

The launcher derives the spec's own commits from git and names them in the review prompt, so the reviewer knows which commits in the range are the spec's and which arrived from other work:

- a commit in `<base>..HEAD` is the spec's own when it touches the spec folder under any name the base resolver followed (renames included), or when its message references the spec's issue (the `issue:` frontmatter, `#N` not followed by a digit);
- merge commits are left out (`--no-merges`); the commits a merge brought in are judged on their own content;
- `--dry-run` and the launch line report the count: `git diff <base>...HEAD (N of M commits are the spec's own)`.

Nothing is recorded at merge time, so a hand-merged slice and an already-archived spec answer the same way.

## Out of scope

- The archive gate and `review-request.json`: the base SHA stays the provenance the gate checks. The commit list is advice to the reviewer, not a new contract field.
- Option B of #1727 (recording each slice's merge commit in the spec).
- `--initial` reviews (CLI-132): a spec born in a root commit keeps its empty-tree scope.

## Risks / open questions

- A slice that neither touches the folder nor names the spec's issue is missed. Measured on HARNESS-142: every slice up to the review touched the folder, so the folder rule alone recovered all five. Mitigation: the prompt still gives the whole range, and says the list is the starting point, not a fence.
- A commit that names the issue in passing ("unrelated to #1727") is included. Accepted: inclusion only asks the reviewer to read it first.
- A spec without a parseable `issue:` falls back to the folder rule alone, and the prompt says so.

## Acceptance criteria

- [ ] AC1: In a fixture history with two slices merged at different times and unrelated commits between and after them, the derived list holds exactly the slices' commits.
- [ ] AC2: A slice that does not touch the spec folder but whose message references the spec's issue is in the list; a commit referencing an issue whose number only starts with the same digits (`#17270` for `#1727`) is not.
- [ ] AC3: A spec folder renamed after it was added still recovers the commits made under its earlier name.
- [ ] AC4: The review prompt names each own commit (short SHA and subject) and `spec review --dry-run` prints the `N of M` count; an `--initial` scope is unchanged.
- [ ] AC5: Run against the real HARNESS-142 history (base `adf49c93`, head `c4d0e6c8`), the derivation returns exactly #3178, #3195, #3196, #3205 and #3213 out of the 14 commits in the range.

## References

- Issue: mlorentedev/dotfiles#1727 (decision A recorded there, 2026-11-09)
- Base resolver: `cli/internal/spec/review_request.go` (`ResolveReviewBaseDetail`, BUG-108 rename tracing)
- Prompt: `cli/internal/spec/review_launch.go` (`writeReviewScope`)
