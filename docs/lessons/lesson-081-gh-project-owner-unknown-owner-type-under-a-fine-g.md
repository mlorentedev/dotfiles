---
id: lesson-081-gh-project-owner-unknown-owner-type-under-a-fine-g
type: lesson
status: active
created: "2026-06-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 081: `gh project --owner` → "unknown owner type" under a fine-grained PAT; a green CI is not a green workflow

**Context:** HARNESS-010's `bitacora-status.yml` used `gh project item-add/item-edit --owner mlorentedev` to move an assigned issue to In Progress. `actionlint`, `test`, and `spec-gate` were all green, and the same commands worked when run with my local `gh` auth.
**Problem:** On the first real assignment the workflow failed at runtime with `unknown owner type`. The `gh project` CLI resolves the owner (user vs org) via an API call the fine-grained `BITACORA_PAT` cannot satisfy — so it works with a local token but not with the workflow's secret. No CI job exercises the workflow with the real secret, so CI was green while the workflow was broken.
**Solution:** Drive Projects v2 from `actions/github-script` (or raw `gh api graphql`) — `addProjectV2ItemById` + `updateProjectV2ItemFieldValue`, the same path `actions/add-to-project` uses — which does not depend on owner-type resolution. And validate any secret-dependent workflow end-to-end with the **real secret** (a throwaway trigger), not local credentials: "mechanism proven locally" ≠ "workflow proven". (Corollary: do not silence a linter false-positive — e.g. `SC2016` on GraphQL `$vars` in single quotes — with scattered `# shellcheck disable`; pick a form the linter understands, like `github-script` for GraphQL.)
**Tags:** `#github-actions` `#projects-v2` `#fine-grained-pat` `#gh-cli` `#verify-with-real-secret` `#no-kludges`
