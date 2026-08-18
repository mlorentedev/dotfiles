---
id: lesson-141-a-cli-s-help-long-strings-are-untested-literals-a-
type: lesson
status: active
created: "2026-07-01"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 141: A CLI's `--help`/`Long` strings are untested literals — a dangling doc ref ships green

**Context**: The DR-escrow slice (#661) shipped `dotf secrets backup` whose `Long` help referenced a `guide-secrets-recover.md` that was never created — the recover protocol was (correctly) hardened into `guide-secrets-governance.md` instead, so the referenced file never existed. Every behaviour test was green and the command worked; the dangling reference was caught only because a human read the real `--help` output during review.

**Problem**: A Cobra command's `Short`/`Long`/`Example` are plain string literals. Behaviour tests invoke `RunE` and assert on the command's *effects*; they never render `--help`, so the help text is a user-facing surface that **no test exercises**. A broken flag description, a stale path, or a reference to a file that does not exist sails through `go test` untouched and only surfaces when a user (or a lucky reviewer) reads `--help`. The escrow's dangling `guide-secrets-recover.md` was exactly this class: invisible to the whole suite, visible only to a human.

**Solution**: Treat help text as testable output (`cli/internal/cmd/help_smoke_test.go`). (1) `TestEveryCommandHelpRenders` walks root + every subcommand and runs the real `--help`, asserting a `Usage:` block renders with no error — catching panics and broken templates. (2) `TestHelpDocReferencesExist` scans each command's `Short`/`Long`/`Example` for `docs/….md` references and asserts each file exists in the repo — the precise guard for the dangling-ref class. Red-teaming the guard itself (injecting a bogus `docs/…` ref and confirming it goes red) exposed a gap in the first cut: it scanned only *subcommands*, so a dangling ref in the **root** command slipped through — scanning root + subcommands closed it.

**Rule**: Help and usage literals are user-facing output, so test them like output — render `--help` for every command in CI, and assert any concrete repo file a help string names actually exists. A guard you have never watched fail may be blind: inject the exact fault it targets, confirm it goes red, then revert — here that step is what revealed the root command was unscanned. Scope reference checks to unambiguous repo paths (`docs/….md`) so the guard stays precise and false-positive-free; vault paths (`MEMORY.md`) and template patterns (`specs/<id>/…`) are deliberately excluded.
