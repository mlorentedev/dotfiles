---
id: "lesson-424-a-golden-file-with-an-extension-escapes-the-extensionless-eol-guard"
type: lesson
status: active
title: "A golden file with an extension escapes the extensionless eol guard"
created: "2026-10-14"
owner: manu
tags: [windows, gitattributes, golden, testing, ci]
---

# A golden file with an extension escapes the extensionless eol guard

**Context:** CLI-114 (#2466) added Go golden files (`cli/internal/*/testdata/*.golden`) that `consoletest.Golden` compares byte-for-byte with the console's output. Every local check passed on macOS.

**Problem:** `test (windows-latest)` failed every golden test, in `doctor` and `cmd` alike. Each `want` ended its lines in `\r\n` and each `got` in `\n`. `* text=auto` converts text files to CRLF on a Windows checkout. `tests/gitattributes-eol.bats` (BUG-039) guards this class, but only for *extensionless* files, and `tests/golden/**` has a path rule. A `.golden` file elsewhere matched neither.

**Solution:** `*.golden text eol=lf` in `.gitattributes`, plus a guard test in the same file: every tracked `*.golden` resolves `eol=lf`. Deleting the rule makes that test fail.

**Why:** a guard keyed on a file's *shape* (no extension) protects that shape, not the property that matters (compared byte-for-byte). Each new byte-compared fixture type needs its own rule, or the guard has to be keyed on the comparison. Normalising line endings inside `Golden` would have hidden the checkout defect rather than fixed it.
