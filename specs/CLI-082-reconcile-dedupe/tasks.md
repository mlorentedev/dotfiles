---
tags: [spec, tasks, templates]
created: "2026-09-23"
---

# Tasks - CLI-082-reconcile-dedupe

## Setup

- [x] Branch created from main: `feat/reconcile-retire-verdict-and-delete-item`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] Failing tests for the retire verdict, the blocker remedies and
      the apply refusal's exits; then `compareRetire` and `VerifyRetires`, with
      `sameValue` rebuilt on the same comparison
- [x] [AC3] Failing test for `retired:` validation; then `RetiredItem` and
      `checkRetired`, which read every declaration, defective ones included
- [x] [AC4] [AC5] Failing tests for planning, applying and transporting a
      delete; then `PlanRetiredItems`, `itemShape`, `OpDeleteItem` and `DeleteItem`
      on both backends and the lock-hint wrapper
- [x] [AC1] [AC4] [AC6] Command-level tests; then `planFromStore` verifies retires
      and plans retired items, and the plan prints both
- [x] Runbook CONVERGE: plan-time verdicts (step 3), the exits (step 6), retiring
      an item (step 8)

## Closing

- [x] Every AC covered by a named test; every AC has a `features.json` entry
- [x] `go build` / `go vet` / `GOOS=windows go vet` clean; `golangci-lint` at the pin, 0 issues
- [x] Whole module green
- [x] Read-only plan against the live store
- [ ] Mutation battery, one mutant at a time under a memory cap
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
- [ ] Independent adversarial review passed

## After merge (data, per-write authorization)

- [ ] Apply the three verified-equal retires (`OPEN ROUTER API KEY`, `POLLEX_API_KEY`,
      `pypi.org`), then retire the two items they empty
- [ ] Retire `github-cli-pat` (after #1632)
- [ ] Settle the four blocked pairs with the owner: the two `Hetzner` items, and the
      tailscale, Gmail and Stripe pairs whose values differ
