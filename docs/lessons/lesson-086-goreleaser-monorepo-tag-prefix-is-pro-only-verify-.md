---
id: lesson-086-goreleaser-monorepo-tag-prefix-is-pro-only-verify-
type: lesson
status: active
created: "2026-06-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 086: goreleaser monorepo.tag_prefix is Pro-only — verify paywalled features empirically

**Context**: CLI-001 scaffold (ADR-020): configuring goreleaser for the nested `cli/` Go module with `cli/vX.Y.Z` release tags.

**Problem**: Trained memory and most blog posts present `monorepo.tag_prefix` as a goreleaser feature; it is GoReleaser **Pro**-only. OSS silently treats a prefixed tag as the literal version (`cli/v0.0.1`), and the slash corrupts artifact paths (`dist/dot_cli/v0.0.1_...`).

**Solution**: Exercised the release pipeline locally with a throwaway tag + the OSS binary BEFORE the first real release; switched to plain `v*` tags (the CLI is the repo's only released artifact) and documented the revisit condition in the spec (CLI-001 R2).

**Rule**: Before designing around any third-party tool feature, check the OSS/paid feature split in current docs (Context7) AND exercise the pipeline empirically with a throwaway run. Feature paywalls invalidate trained memory silently.
