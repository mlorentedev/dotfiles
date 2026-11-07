---
tags: [spec, verification, templates]
created: "2026-11-06"
id: "TOOL-035-manage-ts-bridge-lifecycle"
type: spec-verification
status: active
owner: manu
---

# Verification - TOOL-035-manage-ts-bridge-lifecycle

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> PR4 bridge status/state, exact process-identity and command tests
- [ ] Criterion 2 -> PR5 start manager and no-duplicate tests
- [ ] Criterion 3 -> PR5 owned stop and PID-reuse tests
- [ ] Criterion 4 -> PR4/PR5 state, log and platform-process tests
- [x] Criterion 5 -> `docs/runbooks/ts-bridge-client.md` recovery guard

## Test status

- Pre-implementation consistency check: PASS; every criterion has tagged tasks and a non-vacuous feature command.
- PR4 identity regression: `go test ./internal/bridge -run '^(TestInspectProcess.*|TestCaptureAndInspectCurrentProcess|TestProcessRecordRejectsInvalidIdentity|TestServiceStatus.*)$' -count=1` -> PASS on Windows.
- PR4 command regression: `go test ./internal/cmd -run '^TestBridgeStatusCommand.*$' -count=1` -> PASS on Windows after isolating inherited `GIT_CONFIG_*` test configuration.
- Platform compilation: `GOOS=linux GOARCH=amd64 go test -c ./internal/bridge` and `GOOS=darwin GOARCH=amd64 go test -c ./internal/bridge` -> PASS.
- Copilot recovery guard: the `features.json` f5 verification predicate -> PASS; the runbook contains remove/re-add recovery and no lifecycle code references `data.db`.
- PR4 vet: `go vet ./internal/bridge ./internal/doctor ./internal/cmd` -> PASS.
- Broad local package run: `internal/bridge` passed; `internal/cmd` timed out in the unrelated existing `TestReviewRefusesDirtyTreeWithoutAllowDirty`, and `internal/doctor` hit existing Windows symlink-privilege/domain-account fixture failures. Focused changed behavior passes; CI remains authoritative for the repository-wide Windows baseline, tracked as #3247 / `BUG-166-isolate-windows-test-state`.
- Race detector: not run locally because the installed Go has CGO disabled; CI remains authoritative and PR4 adds no goroutines.
- Manual smoke test: pending synthetic start/status/stop and attended live consequence check
- No regressions in existing test suite: pending CI

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- ADR-059 selects explicit operator commands and rejects an unattended scheduler for this scope.
- The first successful start stores only a non-secret descriptor; later sessions reuse it without dotfiles owning profile content.
- PR4 is read-only status/state; PR5 adds process mutation so each review holds one control-flow surface.
- Exact executable and platform-native process-start identity moved into PR4 because a read-only status cannot truthfully report `healthy` or `unowned` from PID liveness alone. PR5 reuses this seam before signalling.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: `docs/lessons/lesson-566-pid-liveness-is-not-process-ownership.md`
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: `docs/adr/adr-059-operator-controlled-userspace-bridge-lifecycle.md`
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-035-manage-ts-bridge-lifecycle/` -> `specs/archive/TOOL-035-manage-ts-bridge-lifecycle/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
