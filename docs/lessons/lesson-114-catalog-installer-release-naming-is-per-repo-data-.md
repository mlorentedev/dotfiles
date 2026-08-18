---
id: lesson-114-catalog-installer-release-naming-is-per-repo-data-
type: lesson
status: active
created: "2026-06-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 114: Catalog installer: release naming is per-repo data, not a convention (CLI-029)

**Context**: `dotf tools install` (the declarative `packages.json` catalog's installer) reuses the `install-dotf` download→checksum→place pattern, generalised from one CLI to any github-release tool. First tool: sops.

**Problem**: Two assumptions baked into `install-dotf` do NOT generalise. (1) **Archive shape**: `install-dotf` extracts `dotf` from a `.tar.gz`/`.zip`; sops ships **raw binaries** (`sops-v3.13.1.linux.amd64` is the executable itself), so an extraction step would fail. (2) **Checksum filename**: `dotf` ships `checksums.txt`; sops ships `sops-v3.13.1.checksums.txt`. A hardcoded name (or a single asset template) silently 404s or mis-resolves. Both only surface against the *live* release — a unit test over a fixture happily passes a wrong assumption.

**Solution**: Treat the irregularities as **catalog data**: per-OS `asset` map (already in PR-A) plus a `Source.Checksums` template, both expanded from `packages.json`. Drop the extraction step (place the raw binary, rename to the command name, chmod). Reconcile is **pin-as-minimum** (`decideAction`: install/upgrade/skip, never downgrade — REFACTOR-011/013). Verified the real chain with one live `gh release view` + an end-to-end smoke (`dotf tools install` → `sops --version` → idempotent skip), not just the hermetic `Fetcher`-seam tests.

**Rule**: Before wiring a downloader for a new release source, verify the **exact** asset names, archive-vs-raw shape, and checksum-manifest filename against the **live** release (`gh release view <tag> --repo <r>`) — release naming is per-project data, never a safe convention. Keep those facts in the catalog (templates), not in installer code, so the next tool (CLI-028) is a data edit, not a code change. Hermetic seam tests prove the *logic*; only a live smoke proves the *facts*.
