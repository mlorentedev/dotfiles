---
id: "AI-023-orca-ade-and-iac-doctrine-tasks"
type: spec-tasks
status: implementing
created: "2026-08-22"
spec: "specs/AI-023-orca-ade-and-iac-doctrine/proposal.md"
tags: [spec, tasks]
---

# Tasks: AI-023-orca-ade-and-iac-doctrine

- [x] Create `ai/orca/` overlay directory with `ORCA.md`, `orca.yaml`, and `governance.json` [AC1]
- [x] Add `orca.yaml` template to `cli/internal/initrepo/templates/` [AC2]
- [x] Update `harness/manifest.json` with `iac-and-idempotence` and `ai/orca/ORCA.md` target [AC3]
- [x] Run `compile-harness.sh --refresh` to inject updated doctrine [AC3]
- [x] Run full bats and Go test suites [AC4]
