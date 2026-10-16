---
id: "lesson-449-a-tee-through-a-pipe-reorders-output-written-around-it"
type: lesson
status: active
title: "A tee through a pipe reorders output written around it; order needs a barrier through the same pipe"
created: "2026-10-16"
owner: manu
tags: [converge, output, pipes, concurrency]
---

# A tee through a pipe reorders output written around it; order needs a barrier through the same pipe

**Context:** `dotf converge` keeps a transcript of each applied run (CLI-116). It swaps `os.Stdout` and `os.Stderr` for pipes, and a goroutine per pipe copies what arrives to the console and to the transcript. A step's own output, and every child's, goes through those pipes. The printer's `[ OK ] <step>` result line is written straight to the console and the transcript.

**Problem:** A step's last lines could land after its result line, and after the next step's start line. The pump was still copying them when the printer wrote. #2486 recorded the symptom as "not reproduced". A step that prints more than one pipe buffer reproduces it every time: with 4000 lines of 38 bytes, between 2620 and 3483 of them reached the console before the result line, across runs.

**Solution:** Before the printer writes a start or result line, it calls a barrier. The barrier writes a marker into both pipes and waits until each pump has reached its marker. The pump strips the marker and counts it. Everything ahead of the marker has been copied by then. The wait is bounded by the same `drainTimeout` as the end of the run, so a console that stopped reading cannot hang the run. The marker is shorter than `PIPE_BUF`, so a concurrent writer cannot interleave inside it. A marker split across two reads is held back until the next read completes it.

**Why:** Two writers that reach the same sink by different routes have no order between them. The only order a pipe gives is the order of bytes inside it. A barrier that travels through the pipe turns "has the pump finished?" into "has the pump read this byte?", which it can answer. Sleeping, flushing `os.Stdout`, or waiting for the pipe to look empty cannot answer it: the bytes may be in the kernel buffer, in the pump's read buffer, or in its write.
