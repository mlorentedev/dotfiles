---
id: "lesson-569-generate-test-fixtures-from-the-writer-not-by-hand"
type: lesson
status: active
title: "Generate a parser's test fixture from the writer, never by hand"
created: "2026-11-07"
---

# Generate a parser's test fixture from the writer, never by hand

## Context

HARNESS-186 moved the pre-commit lane guard's policy out of the shell and into a
committed marker, `.dotf/lanes.json`, so that adding a session host would be a
data change rather than a code change. The Go side writes the marker with
`json.MarshalIndent`; the guard reads it with `sed`, which is line-based.

`json_list()` matched `"canonical_lanes": [ ... ]` on a single line. The Go
writer puts every array element on its own line. The two shapes never meet.

## Problem

The test fixture was hand-written and kept each array on one line, because that
is easier to read in a test file. Against that fixture the `sed` worked. Against
the real file `json_list()` returned nothing, and the guard fell back to a
hardcoded list of lane names and one hardcoded host — the exact duplication the
marker had been introduced to remove.

So the feature was simultaneously "tested" and broken: the guard still blocked
the right paths, but only because its fallback happened to equal the default. The
first operator who added a session host to `third_party_roots` would have watched
the guard ignore it, with a green suite and no error anywhere. A silent fallback
turns a parser bug into a policy bug, and the fallback is what hid it.

## Solution

Derive the fixture from the writer. The guard's fixture is now built with
`json.MarshalIndent` over `lane.CanonicalLanes` and
`lane.ThirdPartyWorktreeRoots` — the same values, the same encoder, the same
indentation as production — so a change in either side breaks the test instead of
drifting quietly. A companion assertion checks the written file is still indented
one element per line, because that shape is now a contract the guard depends on
and not an aesthetic choice.

Delete the fallback at the same time. A reader whose default equals the value it
was supposed to read cannot report a read failure. With the fallback gone, a
marker that declares no `third_party_roots` leaves that path unblocked — which is
correct, because no host should be privileged in code — and a scenario asserts
exactly that, so the agnosticism claim is a test rather than a comment.

Two habits follow. When a component writes a file another component parses, the
parser's fixture must be produced by the writer, not typed by hand; and any
`if empty then use default` in a reader must be justified as policy, because its
usual effect is to convert "I could not read this" into "the default applies".
