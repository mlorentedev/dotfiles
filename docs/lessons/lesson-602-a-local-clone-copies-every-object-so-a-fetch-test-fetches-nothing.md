---
id: "lesson-602-a-local-clone-copies-every-object-so-a-fetch-test-fetches-nothing"
type: lesson
status: active
title: "A local clone copies every object, so a fetch test fetches nothing"
created: "2026-11-14"
---

# A local clone copies every object, so a fetch test fetches nothing

## Context

HARNESS-101's replay checkout fetches a case's reviewed commit when the clone lacks it: by SHA,
then through the case's `fetch_ref`. The test built a bare remote, pushed a commit only to
`refs/pull/7/head`, cloned the remote with `git clone <path>`, and asserted that the checkout
reached the commit.

## Problem

The test passed with the `fetch_ref` removed from the case, and again with the fetch code
disabled. `git clone` of a local path hardlinks the source's whole object store, including
objects that only `refs/pull/*` reaches, so the commit was already in the clone. The resolver
found it locally and never fetched. A test of fetch logic was green on a fixture that could not
exercise it.

## Solution

Clone fixture remotes with `git clone --no-local`, which goes through the pack transport and
copies only what the advertised branches reach. To reach a fallback path, make the first path fail
for real: under protocol v0 a server refuses a want for an unadvertised commit, so a reviewed
commit below the tip of the pull ref fails by SHA and resolves through the ref.

## Prevention

Before trusting a green test of a fallback, delete the code path it claims to cover and run it
again. A test that still passes is testing the fixture.
