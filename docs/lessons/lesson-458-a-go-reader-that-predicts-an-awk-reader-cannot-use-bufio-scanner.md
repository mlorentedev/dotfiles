---
id: "lesson-458-a-go-reader-that-predicts-an-awk-reader-cannot-use-bufio-scanner"
type: lesson
status: active
title: "A Go reader that predicts an awk reader cannot use bufio.Scanner"
created: "2026-10-16"
owner: manu
tags: [doctor, harness, awk, crlf, cli]
---

# A Go reader that predicts an awk reader cannot use bufio.Scanner

**Context:** `dotf doctor` checks that each agent record's `model:` tier resolves, to predict whether the next deploy can render it (GUARD-006). The render reads the tier with awk (`skill_field` in `scripts/compile-harness.sh`). The check reads it with its own Go reader (GUARD-020, #1740).

**Problem:** The two readers disagreed on three record shapes, and the check reported OK where the render failed. Two were plain choices: the Go reader trimmed the key, so it accepted `model : top`, and it trimmed the fence, so it accepted an indented `---`. The third did not look like a choice. `bufio.Scanner` with its default `ScanLines` drops the carriage return before each newline, so a CRLF record read as `top` in Go and as `top\r` in awk. `resolve-tier "top\r"` exits 1. No trim on the Go side could make the two agree, because the byte was already gone before any trim ran.

**Solution:** The check now reads the record the way awk does. It splits on `\n` only, so a `\r` survives. A fence is `---` at column 0, optionally followed by whitespace. A key matches only when it sits at column 0 directly before its colon. A value loses the whitespace after the colon and keeps everything else. A differential test, `TestRecordModelAgreesWithTheRender`, runs the real `skill_field` beside the Go reader on the same bytes for each shape, the way `TestRecordTargetsAgreesWithTheRender` already did for `targets:`.

**Why:** A check that exists to predict another program must take that program's reading, defects included, not a more reasonable one. That covers the input layer too: Go's default line scanner normalises the very byte the prediction turns on. Hold such a check to its target with a test that runs the target, never with a hand-written list of the target's rules.
