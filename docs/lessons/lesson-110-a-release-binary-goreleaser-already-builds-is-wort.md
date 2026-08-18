---
id: lesson-110-a-release-binary-goreleaser-already-builds-is-wort
type: lesson
status: active
created: "2026-06-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 110: A release binary goreleaser already builds is worthless until each OS's setup script actually downloads it

**Context**: WIN-006 wired Windows setup to fetch a prebuilt `dotf` release binary instead of requiring a local Go toolchain.

**Problem**: `dotf`'s cross-platform binaries were already produced by `goreleaser` on every tagged release — but `setup-windows.ps1` had no step that downloaded them. That made "dotf doesn't work on a fresh Windows box" read as a CLI-porting problem ("needs Go"), when the actual gap was one missing fetch-and-verify step in setup.

**Solution**: Add `install-dotf.ps1` (a PowerShell mirror of the existing `install-dotf.sh`), dot-sourced by `setup-windows.ps1` non-fatally before anything needs `dotf`.

**Rule**: Before treating "doesn't work on OS X" as a build/porting problem, check whether the artifact already exists and the gap is purely a missing fetch step in that OS's setup path. Producing a release artifact and consuming it in every deploy path are two separable deliverables — verify both exist before assuming a bigger rewrite is needed.
