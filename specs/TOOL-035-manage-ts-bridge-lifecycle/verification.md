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

- [ ] Criterion 1 -> shared health foundation complete; status classifier and command pending
- [ ] Criterion 2 -> start manager and no-duplicate tests pending
- [ ] Criterion 3 -> exact process identity complete; owned stop pending
- [ ] Criterion 4 -> strict state store and platform process seams complete; bounded logs pending
- [x] Criterion 5 -> `docs/runbooks/ts-bridge-client.md` recovery guard

## Test status

- Pre-implementation consistency check: PASS; every criterion has tagged tasks and a non-vacuous feature command.
- State foundation: focused `Store`, `LaunchDescriptor` and `ProcessRecord` tests -> PASS.
- Shared health foundation: `TestProbeHealth*` and `TestCheckTsBridge*` -> PASS without changing TOOL-034 doctor behavior.
- Exact identity regression: `TestInspectProcessRequiresExactStartIdentity`, `TestInspectProcessRequiresExactExecutable`, `TestInspectProcessDistinguishesMissingFromInspectionFailure` and `TestCaptureAndInspectCurrentProcess` -> PASS on Windows.
- Linux CI: the Ubuntu CLI test job passes the bridge package at head `46e589f4` ([job](https://github.com/mlorentedev/dotfiles/actions/runs/42412126870/job/128522345004)).
- Platform compilation: `GOOS=linux GOARCH=amd64 go test -c ./internal/bridge` and `GOOS=darwin GOARCH=amd64 go test -c ./internal/bridge` -> PASS. This proves macOS compilation only; a live macOS current-process run remains pending.
- TLS guard: `TestProbeHealthUsesTLSAndRemoteDNSWithoutWeakening` asserts the exact listener and destination curl argument vectors, so `-k`, `--insecure` or an alternate CA cannot enter unnoticed.
- Feature predicates: f1-f4 each exit non-zero while their exact future named tests are absent; f5 exits zero against the existing runbook recovery guard.
- Copilot recovery guard: the `features.json` f5 verification predicate -> PASS; the runbook contains remove/re-add recovery and no lifecycle code references `data.db`.
- Focused vet: `go vet ./internal/bridge ./internal/doctor` -> PASS.
- Race detector: not run locally because the installed Go has CGO disabled; this foundation adds no goroutines.
- Manual smoke test: pending synthetic start/status/stop and attended live consequence check
- No regressions in existing test suite: pending CI

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- ADR-059 selects explicit operator commands and rejects an unattended scheduler for this scope.
- The first successful start stores only a non-secret descriptor; later sessions reuse it without dotfiles owning profile content.
- The original state/status PR exceeded the atomic review cap after exact ownership became mandatory. The user approved splitting it into state, process-identity and status PRs before start/stop.
- The state foundation owns only strict non-secret descriptors, process records and the shared health interpretation; it exposes no command.
- Status and later stop share one exact identity seam: PID liveness alone is never ownership.

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
