---
id: lesson-323
type: lesson
status: active
created: "2026-09-30"
owner: manu
tags: [lesson, go, windows, testing]
---

# 323 — A path quoted with %q doubles every Windows backslash, so a test that looks for the raw path passes only on Linux

## What happened

#1902 made `dotf doctor` warn when a `packages.json` exists but cannot be read. The WARN carries the error from `tools.Load`, which formats the path with `%q`: `parse package catalog "C:\\Users\\...\\packages.json": unexpected end of JSON input`.

The test checked `strings.Contains(out, filepath.Join(repo, "packages.json"))`. On Linux a path has no backslash, so the quoted and raw forms contain the same characters and the test passed. On `windows-latest` the raw path has single backslashes and the output has doubled ones, so the test failed. The behaviour was correct on both.

Every local check was green: `go test`, `go vet`, `GOOS=windows go vet` and the linter. `GOOS=windows go vet` compiles the Windows build but runs nothing, so it cannot see a string mismatch.

## Rule

- When the text under test comes from an error that quotes a path (`%q`, `strconv.Quote`), assert on the quoted form: `fmt.Sprintf("%q", path)`. It matches on every OS.
- A test that asserts a path inside output is a Windows test, even when nothing in it looks platform-specific. Linux CI cannot tell a raw path from a quoted one.
- In the message itself, quoting is still right. It keeps a path with spaces readable and unambiguous, and changing `tools.Load`'s format to suit a test would move the bug to its other readers.

## References

- `cli/internal/doctor/checks_catalog_test.go`, `TestCheckShadowedCatalogTools_NamesAnUnreadableCatalog`
- `cli/internal/tools/catalog.go`, `Load`
- #1902
