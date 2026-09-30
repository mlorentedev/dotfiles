---
id: "CLI-091e-dotf-as-a-product"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-29"
issue: "mlorentedev/dotfiles#1843"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-091e: dotf as a product (track E of #1843)

## Why

<!-- from issue #1843: CLI-091: [EPIC] Fixes reach machines and repos by themselves, and dotf becomes usable outside this repo -->

`dotf` is meant to be useful to developers and SREs outside this repository, but it is not yet. 49 lines of its `--help` text, across 89 commands, cite internal ticket and decision ids (`ADR-028`, `GUARD-017`, `#1381`) or migration jargon ("twins"). Its README describes it as a port of shell scripts. Configuration is scattered across environment variables. Track E makes the CLI read as a product without changing what it does.

## What

Track E lands row by row (#1843). Each row adds its acceptance criteria here before its PR starts.

Row E1, in this PR:

- Every piece of text `--help` prints describes behaviour. That covers descriptions, examples and flag usage, and none of it cites an internal id or the word "twin". A test walks the whole command tree and fails on one.
- `cli/README.md` tells a new user what `dotf` does, how to install it, and where the commands are.

Already landed without this spec, because each was under the spec-gate threshold: E4 (#1844, the build-info version) and the first part of E3 (#1847, SECURITY.md).

Later rows, whose criteria are added when they start: E2, E5 to E10.

## Out of scope

- Error messages. Some still cite an ADR, for example the refusal to print a secret in an agent environment. They are a later row, because their wording is part of what tests and hooks match on.
- Code comments. They keep their ids; that is where the ids are useful.
- Hiding commands behind a profile (E6).

## Risks / open questions

- Tests or scripts could match the old help strings. A search of `tests/`, `scripts/` and the setup scripts found none.
- The guard's id pattern is a list of known prefixes. A new prefix would pass unnoticed, so the list sits next to the test for a reviewer to extend.

## Acceptance criteria

- [ ] **AC1:** `TestHelpTextHasNoInternalReferences` walks every command from the root and fails when a short or long description, an example or a flag usage line contains an internal id, an issue number or "twin". It passes on this tree.
- [ ] **AC2:** `cli/README.md` covers what `dotf` does, how to install it (the release installer and `go install`), a table of the commands, and how to build and test it. It cites no internal ids.

## References

- EPIC #1843, track E; #1844 (E4); #1847 (E3, first part)
