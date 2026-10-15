---
id: "lesson-431-a-comment-stripper-that-ignores-strings-can-still-produce-valid-json"
type: lesson
status: active
title: "A comment stripper that ignores strings can still produce valid JSON"
created: "2026-10-14"
owner: manu
tags: [go, json, jsonc, testing, silent-failure]
---

# A comment stripper that ignores strings can still produce valid JSON

**Context:** #2504 added `"{HOME}/Projects/worktrees/*/*"` to agy's `trustedWorkspaces`. Two Go tests in `cli/internal/harness` failed on every OS with `pin "agy-model": unexpected end of JSON input`, although `jq` read the file fine.

**Problem:** `stripJSONComments` removed `/* … */` with the regex `(?s)/\*.*?\*/`, which does not know about string values. In `ai/agy/settings.json` it had always cut from the `/*` of `"{HOME}/Projects/*"` to the `*/` of `"**/node_modules/**"`, deleting the end of one array, a key and the start of another. What remained happened to be valid JSON, so the pin still extracted and nobody saw it. The added glob moved the cut to a place that broke the syntax. The sweep in `model_pins_coverage_test.go` skips a file that does not parse, so a file the stripper corrupted could also have hidden its routing keys silently.

**Solution:** the stripper scans byte by byte and tracks whether it is inside a string, with escapes; comment markers inside a string are text. That also makes trailing `//` comments safe to strip, which the regex version avoided because of URLs. A table test covers globs, URLs, an escaped quote, block, whole-line and trailing comments, and an unterminated block comment, which is kept so the parser reports it.

**Why:** a pre-processor that rewrites input before a strict parser can turn a malformed result into a well-formed wrong one, and the parser's success then certifies the corruption. Text containing the delimiters (`/*` in a glob, `//` in a URL) is ordinary data in a config file. Test the transform on its own, with inputs that contain the delimiters as data, not only through the parser downstream.
