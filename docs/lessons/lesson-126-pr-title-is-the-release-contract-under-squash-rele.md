---
id: lesson-126-pr-title-is-the-release-contract-under-squash-rele
type: lesson
status: active
created: "2026-06-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 126: PR title is the release contract under squash + release-please

**Context**: Shipping ADR-028 secrets work as squash-merged PRs; release-please (release-type: simple) cuts releases from conventional-commit subjects.

**Problem**: #584 (registry + `dotf secrets ls/show`) squash-merged with a non-conventional PR title ("Secrets registry: ..."). Squash promotes the PR title to the merge-commit subject, which release-please parses. It logged `unexpected token ' ' at 1:8` and counted 0 releasable commits -> no 0.19.0 release ever opened, even though a user-facing feat had landed. The feature sat on main, unreleased and undeployed, silently.

**Solution**: Landed the next `feat(secrets):` PR to re-trigger release-please (it swept #584 into the 0.19.0 tag). Opened #589 to add a conventional-commit PR-title gate.

**Rule**: With squash-merge the PR TITLE is the release-parsed subject -- it must be a valid Conventional Commit. A non-conventional title doesn't error; it silently drops the change from versioning. When a feature merged but no release PR appears, check the merge-commit subject first.
