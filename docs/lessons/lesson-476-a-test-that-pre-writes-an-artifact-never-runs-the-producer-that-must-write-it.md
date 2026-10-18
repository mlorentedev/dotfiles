---
id: "lesson-476-a-test-that-pre-writes-an-artifact-never-runs-the-producer-that-must-write-it"
type: lesson
status: active
title: "A test that pre-writes an artifact never runs the producer that must write it"
created: "2026-10-17"
---

# A test that pre-writes an artifact never runs the producer that must write it


**Context:** CLI-094 added a stall detector that watches the reviewer's transcript file and names the last tool call in it. Its tests passed, 12 of them, and a real round of the spec's own archive review found the feature broken in production.

**Problem:** Every test created the transcript with `os.WriteFile` before starting the fake runner. In production the file is written by `transcript-sink`, which buffered events in 4 KB and flushed at EOF, so while a reviewer ran the file stayed empty: a wedge named no call, and a healthy run with sparse output looked silent. The tests fed the consumer a file the real producer would not have written at that moment.

**Solution:** Flush the sink per settled event, and add a test that connects the real pieces: `RunWithLimits` writing into `SinkTranscript` through a pipe, asserting the detector names the call the sink stored. A second test pins the sink alone: an event is on disk before the stream ends.

**Why:** When one component reads what another writes, the contract is the timing and the form of the write, not only the content. A fixture that skips the producer tests the consumer against an input that may never occur. The check that finds it is cheap: for every consumer test, name which production code wrote its input, and if the answer is "the test", add one case that runs that code.
