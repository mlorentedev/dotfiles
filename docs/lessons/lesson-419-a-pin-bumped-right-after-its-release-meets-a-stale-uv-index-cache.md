---
id: "lesson-419-a-pin-bumped-right-after-its-release-meets-a-stale-uv-index-cache"
type: lesson
status: active
title: "A pin bumped right after its release meets a stale uv index cache"
created: "2026-10-14"
---

# A pin bumped right after its release meets a stale uv index cache

## Context
hive 4.4.1 reached PyPI, and the catalog pin in `packages.json` was bumped from 4.3.2 to 4.4.1 within the hour. The Mac mini had resolved `hive-vault` shortly before the release.

## Problem
`uv tool install --force hive-vault==4.4.1` failed with "no version of hive-vault==4.4.1". The publish job had succeeded, and the version was on PyPI. uv answered from the package index it had cached before the release, so a pin that was correct failed on exactly the machines that had resolved the package most recently. Only `--refresh` made the install succeed.

## Solution
dotf's uv-tool install passes `--refresh-package <package>`. That flag revalidates this one package's index entry and leaves the rest of the cache alone. The install runs only when the tool is absent or below its pin, so a converge at the pin never pays for the refresh.

The doctor's fix for a missing hive now names `dotf tools install hive`, the pinned install, instead of `uv tool install --upgrade`. The `--upgrade` form installs whatever PyPI's latest is.

## Why
A version pin and a package-manager cache answer two different questions: what to install, and what exists. A pin is bumped right after a release, which is exactly when the cache is most likely to predate it. Every installer that resolves a pinned version through a cached index has this window. The fix belongs in the one install path, not in a note to retry with `--refresh`.
