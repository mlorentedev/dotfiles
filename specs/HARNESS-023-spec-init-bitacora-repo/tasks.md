---
tags: [spec, tasks, templates]
created: "2026-06-16"
---

# Tasks - HARNESS-023-spec-init-bitacora-repo

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `fix/spec-init-bitacora-repo`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] `spec.Gate(num, repo)` — append `--repo <slug>` when repo non-empty; update Gate tests to new signature.
- [x] `spec.Render(id, date, repoSlug, …)` / `spec.Scaffold(…, repoSlug, …)` — drop the `const repoSlug = "dotfiles"`, thread the slug through; frontmatter writes full `owner/repo#N`.
- [x] `cmd/spec.go` — resolve slug (flag `--bitacora-repo` > `$DOTF_BITACORA_REPO` > `initrepo.OriginRepo`); validate with `ValidRepoSlug`; error if unresolvable while gating; wire into Gate + Scaffold; `[INFO]` line names the repo.
- [x] Regression guards: `TestRenderSubstitutesAndFixesIssueFrontmatter` (full slug), `TestSpecInitBitacoraRepoFlagOverrides` (flag/cross-repo), `TestSpecInitUnresolvableRepoFails` (error path), env path in `TestSpecInitWithOpenIssueSetsFrontmatter`.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Type checks / build pass (`go build`, `go vet`)
- [x] Lint passes (`gofmt -l` clean)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-023-spec-init-bitacora-repo/features.json`):

```json
[
  {
    "id": "HARNESS-023-spec-init-bitacora-repo-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
