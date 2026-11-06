---
id: "lesson-565-test-exact-archive-members-against-the-release"
type: lesson
status: active
title: "Test exact archive members against the release"
created: "2026-11-06"
---

# Test exact archive members against the release

## Context

The generic archive installer passed zip and tar.gz tests that built synthetic
archives with a binary at the declared path. The ts-bridge catalog therefore
looked complete with `ts-bridge.exe` and `ts-bridge` as exact members.

## Problem

The published assets wrapped each binary in an OS-and-architecture directory.
Checksum verification succeeded, but the first live install failed because the
declared member did not exist. Synthetic extraction tests proved the extractor,
not that catalog data matched the release it described.

## Solution

For exact-member archive installs, verify every supported asset's real member
path before accepting the catalog entry. Keep synthetic tests for malformed,
duplicate, link and atomicity cases, and add a catalog regression that asserts
the observed member path for every OS/architecture pair.
