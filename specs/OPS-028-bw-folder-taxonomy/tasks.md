---
tags: [spec, tasks, templates]
created: "2026-08-14"
---

# Tasks - OPS-028-bw-folder-taxonomy

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/OPS-028-bw-folder-taxonomy`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (casing pinned to the ADR's `Dotfiles/apps`/`Dotfiles/infra`; name→id + create-folder mechanism resolved by task below)

## Implementation

- [x] [P] [AC1] Write failing `TestBWSource_Folder` (registry.go): parse a `bw: { folder: Dotfiles/apps, item: x, field: y }` block, assert `Folder == "Dotfiles/apps"`
- [x] [AC1] Add `Folder string `yaml:"folder"`` to `BWSource`; add `ParseRegistry` validation rejecting any `folder:` outside `{"Dotfiles/apps", "Dotfiles/infra"}` (floor/personal never declare one)
- [x] [P] [AC2] Write failing test for folder name→id resolution (`fakeFolderList`, `TestBWFolderResolver_EmptyNameIsNoop`)
- [x] [AC2] Implement `BWPut.ResolveFolder(name string) (string, error)` in `bw.go`: `bw list folders`, match by name, `bw create folder` if absent, idempotent on repeat calls
- [x] [AC2] Write failing test asserting `CreateItem`'s JSON body carries `folderId` when the secret declares a folder, and omits it (unchanged behavior) when it doesn't (`TestNewItemBody_Folder`)
- [x] [AC2] Thread the resolved folder id through `CreateItem`/`newItemBody`; wire `applySet`/`createAbsent` (shared by `set` and `migrate`) to resolve `s.BW.Folder` AFTER the dry-run/confirm gate, never before (folder creation is itself a write)
- [x] Refactor: confirmed `BWPut.SetField` (update-only path, no create) untouched — folder placement only applies at creation
- [x] [AC5] Ran the existing bw writer test suite unchanged — zero regressions for entries with no `folder:` declared (`go test ./...` full CLI suite green)
- [x] [AC3] Populated `folder: Dotfiles/apps` on all 21 `plane: app` registry entries with a `bw:` target, `folder: Dotfiles/infra` on all 5 `plane: infra` ones (floor has no `bw:` block; the 6 `plane: personal` entries stay undeclared per Out of scope)
- [ ] [AC4] Once Bitwarden is unlocked: resolve/create `Dotfiles/apps`, move the existing `openai-api-key` item into it via the `bw` CLI, verify the `api-key` field value is byte-identical before/after — **blocked on `bw unlock`, vault reports `"status":"locked"` as of this session**
- [ ] [AC4] `dotf secrets verify OPENAI_API_KEY` (DOTFILES_REPO_DIR pointed at this worktree) still reports OK post-move

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/OPS-028-bw-folder-taxonomy/features.json`):

```json
[
  {
    "id": "OPS-028-bw-folder-taxonomy-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
