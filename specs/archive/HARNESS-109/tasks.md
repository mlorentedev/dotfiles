---
tags: [spec, tasks, templates]
created: "2026-09-03"
---

# Tasks - HARNESS-109

> TDD order. One task = one focused commit. `[P]` marks a task with no dependency on another
> unchecked task; `[AC<n>]` maps it to an acceptance criterion in `proposal.md`.

## Setup

- [x] Branch created from main: `fix/harness-109-agent-type` (off `origin/main` at `eaf8a91`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the one that blocked the
      ticket ("what else does the payload contain") was answered by measurement before writing it,
      and the answer refuted the ticket's first candidate fix

## Implementation

- [x] [P] Measure the payload's real field set out of the installed Claude Code executable, and
      settle whether a second type-carrying field exists. **It does not** — this is what makes the
      dispatch-map design the only one available, so it comes first
- [x] [P] Measure the scale from the decision journal, and check its provenance before quoting it
      (schema uniformity; resolving vs unresolved records from the same binary era)
- [x] [P] [AC1] [AC5] `dispatch.go` — the per-session map: `DispatchPath`, `LoadDispatched`,
      `RecordDispatch`, `ValidDispatchName`, with the entry cap
- [x] [AC1] [AC5] Its unit tests: round-trip, latest-wins, the cap, the three ways the file can be
      lost, and the key-by-session decision pinned so nobody "makes it consistent" with the ledger
- [x] [AC1] Surface `DispatchName`/`DispatchType` on `ToolCall`, extract them in
      `normaliseToolCall` via `dispatchArgs`, and record the dispatch in the gate's `RunE`
- [x] [AC2] [AC3] [AC4] Rewrite `loadGatePersona` as the three-step resolution, adding
      `roleNotAPersona` — and map it onto the EXISTING `no-role` outcome rather than a ninth
- [x] [AC2] [AC4] End-to-end tests through the real command, including the control that fails open
      without a preceding dispatch (without it the map would not be what does the work)
- [x] [AC5] Assert the security property on the BYTES: a secret-shaped value in the dispatch's
      `prompt` reaches neither the map file nor any journal record
- [x] [AC7] Bound the gate's hook timeout in `harness/manifest.json`, and write the guard to assert
      the CLASS — it immediately found a second unbounded hook (agy's `BeforeTool`), now also bound
- [x] Refactor: the AC7 assertion had to run through `MergeHooks`, not marshal `HookCommand`, or it
      would pass on a timeout that never reaches the settings file
- [x] [AC6] Live probe: one owner-authorised named dispatch, evidence recorded in
      `verification.md`, shared binary restored byte-identically afterwards

## Closing

- [x] Every acceptance criterion is covered by at least one test
- [x] Every acceptance criterion has a `features.json` entry with a non-vacuous verification
      command — **each was executed**, and the check caught a real defect in one of them: an
      unanchored `grep -c -- '--- PASS: Test'` counted indented subtests too, so `f5` expected 4 and
      got 18. Anchored to `^`. A command that can return a plausible number without measuring what
      it claims is exactly the shape this repo has catalogued
- [x] Type checks pass — `go build`, `go vet`, and `GOOS=windows go vet`
- [x] Lint passes — `golangci-lint run` at the pinned `2.12.2`, 0 issues
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Independent adversarial review before archive (not the implementer)

## Machine-readable features

`features.json` holds one entry per acceptance criterion. `state` stays `pending`: only the harness
may write `passing`, after running `verification` and capturing exit 0. All seven commands were run
locally and exit 0; the evidence for the owner-gated AC6 is the journal excerpt in
`verification.md`, which its command asserts is present.
