---
id: "CLI-039-dotf-deploy"
type: spec
status: draft
created: "2026-08-16"
issue: "mlorentedev/dotfiles#1023"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — CLI-039-dotf-deploy

Slice 1 of 3. Every intermediate merge must be functional, so this slice ships a
working command AND removes the duplication it replaces — a command with no
caller would be uncalled code, which the atomic-PR rule explicitly forbids.

## 1. Package (TDD)

- [x] Manifest parse + validation, naming the offending entry.
- [x] `{VAR}` expansion through the env seam; an unresolvable token is an ERROR,
      never an empty path segment.
- [x] Stage → render → compare → install, atomically.
- [x] Test: declared mode is applied (0600 for a config that may carry a secret).
- [x] Test: idempotent, asserted on **mtime** — "in sync" that rewrites the file
      has told the truth about intent and not about the filesystem.
- [x] Test: `--dry-run` writes nothing.
- [x] Test: the renderer runs on the STAGED copy, never the repo source —
      rendering the source would write a credential into the checkout.
- [x] Test: each error names its config; "deploy failed" sends a reader to code.
- [x] Test: the shipped manifest parses and declares pi as 0600.

## 2. Command

- [x] `dotf deploy [name]`, `--dry-run`, registered on root.
- [x] Render seam CALLS `secrets.Render`; no second substitution implementation.

## 3. Shrink the twins

- [x] `setup-linux.sh`: pi block 21 → 9 lines.
- [x] `setup-windows.ps1`: pi block 25 → 12 lines.
- [x] Both degrade with a warning when `dotf` is absent, as the old blocks did —
      a bootstrap must not hard-stop because one config could not deploy.
- [x] ASCII-only in the PowerShell file (PSScriptAnalyzer).

## 4. Verification

- [x] Go: build, vet, test, `golangci-lint` at the pin.
- [x] Shell: `shellcheck`, `bash -n`, `zsh -n`.
- [x] Live `--dry-run` against the real machine.
- [ ] `bats tests/*.bats` for the setup-script regression suite.

## Out of scope

Slices 2 (opencode, gated on testing whether it self-resolves `{env:}`) and 3
(MCP registration); `secrets render`'s contract; a macOS bootstrap.
