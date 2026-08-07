---
id: "BUG-026-setup-no-checkout-writes"
type: spec
status: implementing
created: "2026-07-09"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — BUG-026-setup-no-checkout-writes

- [x] T1. Remove the `.github/copilot-instructions.md` sync block from
  `setup-linux.sh`; replace with an invariant comment pointing at
  `tests/docs-drift.bats`.
- [x] T2. Remove the parallel sync block from `setup-windows.ps1`; same
  invariant comment, ASCII-only.
- [x] T3. Add the checkout-hygiene guard to `tests/verify-setup.bats`
  (`git -C "$REPO_DIR" status --porcelain` empty; skip if no `.git`).
- [x] T4. Split the dirty-status branch in `cli/internal/update/update.go`:
  unreadable status → fail-safe message; dirty → message names the porcelain
  paths (#694).
- [x] T5. Extend `cli/internal/update/update_test.go` to assert the dirty
  message names the offending path (`wantMsgSubstr`).
- [x] T6. Local verification: `go build ./... && go vet && go test`,
  `bash -n setup-linux.sh`, bats parse, ps1 ASCII check, docs-drift parity.
- [ ] T7. CI green (unit `test` incl. docs-drift, `integration` incl. new
  guard, `lint-powershell`, Go tests).
