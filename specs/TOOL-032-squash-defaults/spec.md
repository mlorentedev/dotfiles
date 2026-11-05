---
id: spec-TOOL-032-squash-defaults
type: spec
status: implementing
created: "2026-11-05"
owner: manu
issue: "mlorentedev/dotfiles#3169"
---
# TOOL-032: squash with the PR title and no body

## Intent

A squash commit's body is the branch's commit messages (`COMMIT_MESSAGES`), and those carry
`Refs owner/repo#N`. release-please's default changelog renders each reference as `closes`, so merging a
release PR closes issues (CI-005). ci-workflows avoided that with `changelog-type: github`. release-please
skips a release only when its notes are empty, and GitHub's notes never are, so every docs or ci push
opened a patch release PR (ci-workflows#23, #24). If the squash body is blank, the commit carries no
reference to render, and the default changelog can again decide what a release is: `feat`, `fix` or a
breaking change.

## Scope

`dotf init github`, and the GitHub step of `dotf init`, also ensure `squash_merge_commit_title=PR_TITLE`
and `squash_merge_commit_message=BLANK`. Both work the way `delete_branch_on_merge` already does. They
read the current values first. When the values already hold, nothing changes and the result is
`already-set`. `--dry-run` reports what would change and makes no PATCH. A missing `gh` or an unreadable
repo is a skip, not an error. A PATCH the server rejects is an error.

Out of scope:
- Disabling merge commits or rebase merges. Those still carry each commit's body; that is the owner's
  decision.
- Removing the CI-005 sanitizer from dotfiles' release-please workflow. That gets its own ticket.

## Consequences

- A `BREAKING CHANGE:` footer in a commit body no longer reaches release-please. A breaking change is
  `!` in the PR title (`feat!:`).

## Acceptance criteria

1. Unit tests with a fake `gh` cover each case: values already set means no PATCH, differing values are
   PATCHed, `--dry-run` makes no PATCH, and a missing `gh` is a skip.
2. `dotf init github --repo mlorentedev/ci-workflows` leaves `PR_TITLE`/`BLANK`, and a second run reports
   `already-set`.
