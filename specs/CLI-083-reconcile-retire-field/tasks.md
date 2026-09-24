---
tags: [spec, tasks]
created: "2026-09-24"
---

# Tasks - CLI-083-reconcile-retire-field

## Setup

- [x] Branch created from main: `feat/reconcile-retire-field`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [ ] [P] [AC1] Failing test for field entries in `retired:` and blank names; then
      `RetiredItem.Field` and a field-aware `checkRetired`
- [ ] [AC2] [AC5] Failing tests for planning and applying a retired field; then
      `OpDeleteField` in `PlanRetiredItems` and `ApplyReconcile`, and the plan line
- [ ] [P] [AC3] Failing test for `bw.from` on a multi-var secret over one field;
      then `checkBWFrom` refuses only when the variables resolve to different fields
- [ ] [P] [AC4] Failing test for a name given twice to `sync ci`; then
      `scopeUpload` deduplicates
- [ ] [AC2] [AC5] Command-level test: plan and apply of a retired field through
      `dotf secrets reconcile`
- [ ] Runbook CONVERGE step 8: retiring a field, and that only the escrow recovers it

## Closing

- [ ] Every AC covered by a named test; every AC has a `features.json` entry
- [ ] `go build` / `go vet` / `GOOS=windows go vet` clean; `golangci-lint` at the pin, 0 issues
- [ ] Whole module green
- [ ] Read-only plan against the live store
- [ ] Mutation battery, one mutant at a time under a memory cap
- [ ] `verification.md` filled in
- [ ] Independent adversarial review passed
- [ ] PR opened: archives CLI-082 and CLI-083, closes #1624

## After merge (data, per-write authorization)

- [ ] Retire the dead fields `Stripe`/"backup-codes" and `login.tailscale.com`/"auth-key"
- [ ] Retire the `cloud.nan.builders`/"api-key" copy through `bw.from` on `NAN_API_KEY`
