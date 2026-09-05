---
tags: [spec, tasks, templates]
created: "2026-09-04"
---

# Tasks - CLI-063-dotf-deploy-claude

> TDD order. One task = one focused commit. Tick as you go.
> `[P]` = no dependency on another unchecked task. `[AC<n>]` = satisfies acceptance criterion n.

## Settled before implementation

**Golden characterization capture does NOT transfer from CLI-021. Decided, not open.**

CLI-021 captured a single script's stdout and diffed it byte-for-byte. That worked because
`vault-maintenance-weekly.sh` was one process with one output stream. This is not that shape:

- The four capabilities run at **four different points** of a setup, not as one invocation.
- Their observable effect is on **`$HOME` and on `claude` itself**, not on stdout — a snapshot
  guard's whole job is that nothing is printed and nothing is lost.
- Two of them **shell out to `claude`**, so a capture on this box records that box's plugin
  registry, not the behaviour.

The honest instruments, per capability:

| Capability | Oracle |
|---|---|
| `settings.json` merge | **Table tests over the policy table.** Input: a template + a box state. Expected: the merged object. Each of the six keys gets a case that FAILS under top-level replace. |
| Plugin list | **A drift guard**, ~15 lines of bats: grep both twins' hardcoded ids and assert set-equality with `ai/claude/plugins.json`. This is AC2, and it is the only thing that can catch the twins diverging. |
| Snapshot guard | **Unit test on the threshold predicate**, reusing `claude_json_min_bytes`. No file I/O needed to test "is this truncated". |
| MCP registration | **Fake `claude` on PATH** recording argv. Asserts skip-if-present and error-surfacing without touching a real registry. |

Nothing here is a golden file. Do not add one.

## Setup

- [x] `proposal.md` complete, acceptance criteria testable
- [x] Command home settled against ADR-032 — `dotf deploy`, not `dotf agent` (owner decision 2026-09-05)
- [x] Oracle question settled (above)
- [ ] Branch off `main`, spec folder committed, #1339 set to In Progress

## Increment 1 — snapshot guard + plugin sync (first PR, ≤300 LOC)

- [ ] [P] [AC2] Failing bats: both twins' hardcoded plugin ids equal `ai/claude/plugins.json`
- [ ] [AC2] Add `ai/claude/plugins.json` with the ids read out of the twins verbatim
- [ ] [P] [AC7] Failing Go test: truncation predicate at the `claude_json_min_bytes` boundary
- [ ] [AC7] Implement the predicate by **calling** the existing threshold in
      `cli/internal/mem/session_start_adapter.go:83` — a second literal `10240` is the defect
- [ ] [AC1] Plan/apply for the plugin capability; counts a plugin only on install success (#1491
      divergence recorded in `divergences.md`, NOT fixed in the `.ps1`)
- [ ] [AC6] **Negative test: the Go path emits no `hooks` key under any input.** Written in
      increment 1 even though increment 1 does not write settings — the proposal names this as the
      single most likely way to cause an incident, so the assertion predates the code that could
      violate it
- [ ] Refactor; `go build ./... && go vet ./... && go test ./...`, `GOOS=windows go vet ./...`,
      pinned `golangci-lint`, `bats tests/*.bats`

## Increment 2 — MCP registration

- [ ] [P] Failing test with a fake `claude` on PATH: an entry already registered is skipped
- [ ] Failing test: `claude mcp add` non-zero is surfaced, not swallowed
- [ ] Implement reading `mcp-servers.json`; carry HIVE-118's remove-then-re-add for `uvx hive-vault`

## Increment 3 — per-key merge policy

> **Coordination point, not a blocker:** this increment edits `harness/manifest.json`'s neighbour
> `ai/deploy.json` and bumps its `version`. `harness/manifest.json` itself is another session's
> declared surface — if this increment ends up touching it, coordinate before editing.

- [ ] [P] [AC3] Failing table test: `env` nested merge preserves a box-only key
- [ ] [AC3] Failing table test: `enabledPlugins` nested merge preserves a box-only plugin
- [ ] [AC3] Failing table test: `permissions.allow` unions and dedupes
- [ ] [AC4] Failing test: a template key absent from the policy **errors**, not skipped silently
      (this is the `outputStyle` defect; a passing test here is the whole point of the increment)
- [ ] [AC3] Implement per-key policy declaration — recommended shape (a) from `proposal.md`:
      `strategy: merge` plus a sibling `keys:` map
- [ ] [AC5] Bump `ai/deploy.json` `version` to 4; failing test first that an older decoder refuses
- [ ] Declare Claude's `settings.json` entry against the new policy

## Closing

- [ ] Every acceptance criterion covered by ≥1 test and ≥1 `features.json` entry with a
      non-vacuous verification command
- [ ] Lint + both layers green; `GOOS=windows go vet ./...` clean
- [ ] `verification.md` filled in
- [ ] **Independent adversarial review** (reviewer ≠ implementer) before `dotf spec archive`
- [ ] No twin deleted, no caller repointed — verified by `git diff --stat` showing nothing under
      `setup-*.{sh,ps1}` or `scripts/`

## Machine-readable features

Emits a sibling `features.json` per [[pattern-feature-list-as-primitive]]. The agent CANNOT write
`"state": "passing"` — only the harness, after running `verification` and capturing exit 0, may set
that terminal state.
