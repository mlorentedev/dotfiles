---
tags: [spec, verification]
created: "2026-09-24"
---

# Verification - CLI-083-reconcile-retire-field

## Evidence

- [x] AC1 (a retired field is validated) -> `69558d0`, `e89b8aa`.
  - `TestRegistryValidatesRetiredFields` covers: blank field, `username`,
    `password`, a destination field (single-var, multi-var override, multi-var
    default), a `bw.from` source field, the same field twice, whole and field in
    either order, a blank reason, and two fields of one item accepted.
  - `TestRegistryValidatesRetiredItems` gained the blank-item row.
- [x] AC2 (delete-field is planned and applied) -> `69558d0`.
  - `TestPlanRetiredFields`: planned last, with its reason and what the item
    keeps; an absent field or item reported gone; an ambiguous name blocked.
  - `TestApplyDeletesARetiredField`: removed through the writer seam, and the
    applied line names item and field.
  - `TestReconcileDeletesARetiredField`: command level. Plan, apply, the item and
    its other fields left alone, the next plan reports it gone.
- [x] AC3 (`bw.from` on a multi-var secret over one field) -> `69558d0`.
  - `TestBWFromOnAMultiVarSecret`: one field accepted, different fields refused,
    a shared override that is its own destination refused.
  - `TestPlanMultiVarFromPlansOnce`: one `retire-source`, then one satisfied record.
- [x] AC4 (`sync ci` dedupes names) -> `69558d0`, `TestScopeUploadDedupesNames`.
- [x] AC5 (no value in output) -> `TestPlanRetiredFields` and
  `TestReconcileDeletesARetiredField`, whose vault values all carry `PLANTED`.

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean.
- `golangci-lint run ./internal/secrets/... ./internal/cmd/...` at the pin (2.12.2): 0 issues.
- `go test -p 1 ./...` under `MemoryMax=3G`: every package ok.
- **Mutation battery**, one mutant at a time, restored with `git checkout HEAD`,
  under `MemoryMax=3G`: **21/21 killed by a test**. M17 first died on a build
  error (an unused map), which proves nothing, so it was rewritten to compile and
  re-run; a test killed it. The mutants: blank item and field accepted, login
  fields accepted, read fields unchecked, per-var and `bw.from` fields
  unregistered, whole/field conflict allowed, duplicates keyed by item, an absent
  field still planned, shape not reduced, notes not cleared, apply dropping
  delete-field, rank moved first, target naming the item, multi-var refused
  again, the self check on the base field, `bwFields` not deduplicated, `sync ci`
  names not deduplicated, the plan line missing, the ambiguity threshold raised,
  and the planner planning per var.
- **Read-only plan against the live store** (branch binary, a scratch registry
  declaring the field retires and `bw.from` on `NAN_API_KEY`):

  ```
  - retire-source  cloud.nan.builders   remove cloud.nan.builders/"api-key", verified equal to nan-api-key/"api-key"
  - delete-field   Stripe               field "backup-codes", keeps: login; retired: ...
  - delete-field   login.tailscale.com  field "auth-key", keeps: login; retired: ...
  ! blocked        Hetzner              cannot delete "Hetzner/key": the name matches 2 items, ...
  Plan: 3 to apply, 1 blocked, 0 deferred.
  ```

  The multi-var copy plans once, and the ambiguous `Hetzner` blocks, as the
  proposal says.

## Decisions made during implementation

- **`username` and `password` are refused as retired fields.** They are an
  account's login, not a legacy copy. Their presence cannot be read without the
  value (`ItemSummary` never decodes a password), so a plan could not honestly
  show whether there is anything to remove.
- **Map keys are not display labels.** Item names in the store contain `/`
  (`grafana/status.kubelab.live`), so `item/field` keys could collide. Keys use a
  NUL separator (`fieldKey`); `label()` is only printed.
- **`bwDestField` was replaced by `bwFields()[0]`.** Once `bw.from` reaches a
  multi-var secret, the destination is the one field its vars share, which may be
  a common override rather than the secret-level field.
- **The satisfied note says `retired X is gone`, not `retired item X is gone`**,
  because it now names fields too. CLI-082's command test was updated to match.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? No. Lesson 288 already covers probing
      a dead value by consequence.
- [ ] ADR-worthy decision? No. It extends ADR-028's reconcile without changing it.
- [ ] New pattern candidate for `00_meta/patterns/`? No.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-083-reconcile-retire-field/` -> `specs/archive/CLI-083-reconcile-retire-field/`
- [ ] Bitácora board ticket closed with the PR link (ADR-018)
- [ ] CLI-082 archived in the same PR, which closes #1624

## Review disposition

Adversarial review: **PASS**, `nan/mimo-v2.5`, on `2fc33d3` (`review.md`).

| Finding | Disposition |
|---|---|
| Minor, THEORETICAL: the blocked detail for an ambiguous field entry names `item/field` (`cannot delete "Hetzner/key"`) though the ambiguity is the item's | **Declined.** The detail names the declared entry the operator has to act on, and the same sentence gives the cause ("the name matches 2 items"). The live plan printed exactly that, and it read correctly. `PlanNote.Item` carries the bare item for anything structured. Changing the code after the verdict would stale the review for a wording preference. |
