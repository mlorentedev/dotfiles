---
id: lesson-135-name-match-at-the-consumer-boundary-decouple-at-th
type: lesson
status: active
created: "2026-06-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 135: Name-match at the consumer boundary, decouple at the storage boundary

**Context**: `dotf secrets sync ci` (CLI-024-secrets-sync) uploads registry secrets to a repo's GitHub Actions secrets.

**Problem**: A secret crosses two boundaries with different naming pressures — the consumer boundary (`gh secret set` / a workflow's `${{ secrets.X }}` need an exact name match) and the storage boundary (Bitwarden's own item/field organization, grouped by service or account for a human browsing the vault). Forcing one naming scheme across both either breaks the consumer's exact-match requirement or fights the storage layer's own organizing logic.

**Solution**: The Actions secret name is always identical to the exposed env var — a flat 1:1 convention at the consumer boundary. Bitwarden storage (`bw: {item, field}`) stays decoupled and is free to group related secrets by service/account; the registry (`secrets/registry.yaml`) is the only place that maps between the two.

**Rule**: At a consumer boundary — anywhere a caller's literal expectation must match exactly (env var names, API param names) — name-match precisely. At a storage boundary — anywhere only the system itself reads the layout — organize for the storage's own convenience. Never let storage-layer naming leak into a consumer contract, and keep exactly one seam (here, the registry) that translates between the two.
