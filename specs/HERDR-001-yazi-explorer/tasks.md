---
tags: [spec, tasks]
created: "2026-10-18"
---

# Tasks - HERDR-001-yazi-explorer

> TDD order. One task = one focused commit. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`.

## Setup

- [x] Branch: `feat/herdr-optimization` (a worktree the owner created for this theme; the spec folder landed on it rather than a new branch)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC6] (landed in `216fa210`, not in this change's diff) Failing test: each wrapper's line is compared whole, so `HERDR_AGENT`
      cannot be swapped between pi and opencode (`tests/pi-config.bats`).
      The old regex broke on the prefix and was widened to assert the hint.
- [x] [AC6] Wrappers take `HERDR_AGENT=<canonical id>` on both shells, with the
      comment that says why it cannot be an `export` (lesson 493).
- [x] [AC6] Live check: `HERDR_AGENT=pi dotf secrets run … -- pi` in a pane →
      `herdr agent get` reports `agent: "pi"`, `agent_session`, `screen_detection_skipped: true`.
- [x] [AC1] Failing test: `prefix+e` binding must be a popup running yazi
      (`tests/herdr-config.bats`), plus a test that the tool it runs has a pin.
- [x] [AC1] `[[keys.command]] prefix+e` in `ai/herdr/config.toml`; yazi pinned in
      `versions.conf`; `herdr config check` green on source and deployed copy.
- [x] [AC5] `YAZI_VERSION` reaches the rendered mise config (`yazi = "26.9.1"`),
      second `dotf tools sync` reports nothing to do, `yazi --version` runs.
- [x] [AC2] Failing test: `dotf herdr explorer` splits right with the pane's cwd
      and ratio, then runs yazi in the new pane.
- [x] [AC2] `cli/internal/cmd/herdr.go` implements split + run, reading
      `HERDR_ACTIVE_*` first and `HERDR_PANE_ID`/`HERDR_TAB_ID` second.
- [x] [AC3] Failing test: a second call with an explorer already running closes
      it and opens nothing; another tab's explorer is never touched.
- [x] [AC3] Toggle detection by foreground process (`yazi`), not by a pane label
      that any rename clears; verified live: opened `w10:p9`, then closed it.
- [x] [AC4] Failing test: no yazi on PATH refuses and names `dotf tools sync`.
- [x] [AC4] `herdrLookPath` preflight before any herdr call.
- [x] [AC1] Ordering guard: a test fails if a binding names `dotf herdr explorer`
      while `DOTF_VERSION` predates it (lesson 373). Narrowed to one direction in
      #2827 (lesson 498), then replaced by the binding test when the binding landed.
- [x] `ui.agent_panel_sort = "priority"` + a static test.
- [x] Collateral: the `herdr config check` canary appended a second `[ui]`, which
      is a duplicate-table error — it now appends `[terminal]`, a table this file
      does not define, and names the key it expects.

- [x] [AC1] `[[keys.command]] prefix+shift+e` -> `dotf herdr explorer`, once
      `DOTF_VERSION` reached 0.76.0; the guard became the binding's own test.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go vet ./...` and `go test ./...` pass
- [x] `bats tests/herdr-config.bats tests/pi-config.bats tests/versions-conf.bats` pass
- [x] No unrelated changes in the diff (the deploy of the owner's machine is the
      provisioning step this change exists for, not a second concern)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. The evidence collected by hand during implementation lives in `verification.md`, not here.
