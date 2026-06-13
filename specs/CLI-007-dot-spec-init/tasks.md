---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - CLI-007-dot-spec-init

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/dot-spec-init`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (drift-guard decision recorded)

## Implementation

> TDD order. Domain in `internal/spec/`; Cobra wiring in `internal/cmd/spec.go`. `cmd/dot` stays main-only (CLI-002 R2).

- [x] `internal/spec`: `ValidateID` (grammar incl. sub-id `SDD-012b`, date form, rejects junk) — `TestValidateID`
- [x] `internal/spec`: embed templates via `//go:embed`; `Render` substitutes placeholders, sets frontmatter `issue: "dotfiles#N"`, injects the `## Why` comment — `TestRender*`
- [x] `internal/spec`: `Scaffold` writes 3 files, refuses to clobber, warns on archive/ id — `TestScaffold*`
- [x] `internal/spec`: `Gate(issueNum)` shells to `gh issue view --json state,title` (stubbed on PATH; open→ok, closed/missing→error) — `TestGate*`
- [x] `internal/spec`: drift-guard test — embedded == vault SSOT; `t.Skip` when `$VAULT_PATH` absent (ADR-013), non-fragile — `TestEmbeddedTemplatesMatchVault`
- [x] `internal/cmd/spec.go`: Cobra `spec` + `init` with `--issue` / `--force-no-gate` — `TestSpecInit*`
- [x] Wire `root.AddCommand(newSpecCmd())` in `root.go`; `TestSpecListedInRoot`
- [x] Error messages mirror the shell's gate semantics/wording for parity

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] Type checks pass (`go vet ./...` clean)
- [x] Lint passes (`gofmt -l .` empty)
- [x] No unrelated changes in the diff (no scope creep — only `cli/` + this spec folder)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-007-dot-spec-init/features.json`):

```json
[
  {
    "id": "CLI-007-dot-spec-init-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
