---
spec: "BUG-173-converge-step-dependencies"
verdict: "PASS"
reviewed_sha: "acfa1df97ac4dedbc19b3dea68ba73358a443885"
reviewer: "agy/gemini-4-hydrogen-high"
date: "2026-11-16"
---
## Adversarial review

**Scope**: `BUG-173-converge-step-dependencies` (`git diff ccf90b3b8cd9ebfe5c5e2bbda03284acd8a0ccfc...HEAD`, primarily commit `d0c98f7d6546` — PR `#3560`, checked for interaction with merged commits `d1dee106` and `2060f629`)
**Sources**:
- `specs/BUG-173-converge-step-dependencies/{proposal,tasks,verification}.md`
- `specs/BUG-173-converge-step-dependencies/features.json`
- `docs/adr/adr-064-a-failed-converge-step-holds-back-only-its-dependents.md`
- `cli/internal/converge/{needs,needs_registry,converge,records,checkout,harness,tools,sysupgrade,cachegc,pwshmodules,pisettings,configs,orcahooks,shellcompletions,bind,gitconfig,tsbridge,gitlfs,githooks,lanes,envpersist,envpersist_user,selfdeploy,hiveupgrade,workstation,extensions,legacy}.go`
- `cli/internal/cmd/{converge,update,deploy_agy}.go`, `cli/internal/doctor/checks_self_deploy.go`

### Spec and task alignment
- `proposal.md` defines AC1–AC9, scope, out-of-scope, all 24 step dependency declarations with code references, and risks; no `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in `specs/BUG-173-converge-step-dependencies/`.
- `tasks.md`: all Implementation tasks are `[x]` and backed by diff evidence in `d0c98f7d6546`. Two Closing checkboxes remain `[ ]` ("Lint passes" due to local CRLF checkout `gofmt` noise, and "PR opened referencing this spec folder" though `#3560` already merged into the branch).
- `features.json` lists 9 entries (`f1`–`f9`) in `state: "pending"` with empty `evidence` and non-vacuous `go test` verification commands; `dotf spec digest BUG-173-converge-step-dependencies` matches `review-request.json` `contract_digests` (`features.json`: `3c230f44919ab3a8f8c94997e3c5685daf7916d6ac3cfb6cb4a831ef2c95fd88`, `proposal.md`: `90a9caa927c48222b7ffb7f860ca8c73fa1b15c2c626b62a6300a848b9607264`, `tasks.md`: `400a5c2c4aa1d8cfd62d2a7ee334f2ed4fdb8c866f2ccd3ee1369816db42b626`).
- Every row of `Needs()` in `cli/internal/converge/needs_registry.go` matches `proposal.md` and aligns with actual filesystem/env/binary reads across all 24 reconcilers, including post-merge `env/resolve.go` (`d1dee106`) and `sysupgrade.go` (`2060f629`).
- Fresh verification runs executed in this review session:
  - `go build ./...`, `go vet ./...` on `windows`, `linux`, and `darwin`: clean (exit 0).
  - `go test ./internal/converge/ -count=1 -timeout 5m`: `ok` (`243.328s`).
  - All `features.json` commands (`TestRun_*`, `TestRegistry_*`, `TestRunObserved_ReportsEachStepItRuns`, `TestCheckSelfDeploy_ByState`) and `TestConvergeConsole_Streams`: `ok`.
  - Mutation check: removing `nGitHooks` from `(lanes).Needs()` turned `TestRegistry_WhatAFailureHoldsBack` red (`git-hooks fails: not reached [legacy-setup], want [lanes legacy-setup]`), then reverted cleanly.
  - `golangci-lint run ./internal/converge/ ./internal/doctor/ ./internal/cmd/` (v2.12.2 with clean temp cache): 0 findings on new/modified code; 3 `gofmt` warnings on untouched CRLF files (`bind.go`, `cachegc.go`, `catalog.go`).
  - `go run ./cmd/dotf converge --plan --repo ..`: demonstrated live in-place skipping when `configs-deploy` failed (due to `DOTFILES_REPO_DIR` pointing at the main checkout without `PROJECTS_PATH`), holding back `records-bind`, `git-config`, `agent-extensions`, and `legacy-setup` as `not reached: configs-deploy failed` while `orca-hooks`, `ts-bridge-client`, `git-lfs`, `git-hooks`, `lanes`, `env-persist`, `self-deploy`, `hive-upgrade-retire`, and `workstation-target` still ran.
  - **UNVERIFIED locally**: full repo `go test ./...` (bounded per-package here; CI is reference) and a non-dry-run failing `dotf converge` apply on a real host.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | tests / AC3 | `TestRunObserved_ReportsEachStepItRuns` and `TestConvergeConsole_Streams` place the `not reached` step at the end of the slice rather than before an independent step that still runs, so in-place interleaving in `Observer`/`convergePrinter` is verified by code read and `--plan --repo ..` execution rather than an explicit interleaved unit assertion | `cli/internal/converge/observe_test.go:19-45`; `cli/internal/cmd/converge_console_test.go:265-290`; `cli/internal/converge/needs_test.go:123-142` runs `free` after `needsHarness` via `Run` without asserting `Observer` or `rep.Entries` order | `TestRunObserved_ReportsEachStepItRuns` (partial — tail only) | tests (optional follow-up) |
| Minor | THEORETICAL | converge / revisit | `revisit()` skips steps already in `down` (`converge.go:201`) but does not evaluate `down.blocks(needsOf(...))` between two steps that both revisit after the same trigger step; if a second `Revisiter` ever depended on an earlier `Revisiter` that failed during revisit, it would still run | `cli/internal/converge/converge.go:195-214`; currently only `records-harness` implements `Revisiter` (`rg -n RevisitAfter cli/`) | UNTESTED | code + tests (optional if a second `Revisiter` is ever added) |
| Minor | REAL | tests / AC9 | `TestCheckSelfDeploy_ByState` exercises the 2-line error case (`"and 1 more line in last.json"`), leaving the plural (`>= 3` lines -> `"lines"`) and `\r\n` branches of `errorLine` unexercised by a named test | `cli/internal/doctor/checks_self_deploy.go:120-130`; `cli/internal/doctor/checks_self_deploy_test.go:62` | `TestCheckSelfDeploy_ByState` (covers 1-line and 2-line only) | tests (optional follow-up) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 9 ACs verified against code and tests; 24-step dependency graph matches actual file/env/tool reads with no missing edges found. |
| Verification       | B | All `features.json` tests, `internal/converge/`, `go build`, cross-OS `go vet`, mutation check, and `--plan` run verified; minor gaps in interleaved observer and plural `errorLine` tests, and full `go test ./...` unrun locally. |
| Scope              | A | `d0c98f7d6546` stays strictly within converge step dependencies, `doctor` single-line error display, ADR-064, docs, and tests. |
| Reliability        | A | Undeclared steps safely default to all-prior needs, absent/skipped/unsupported needs do not block, errors join without loss, and failed steps are excluded from revisits. |
| Maintainability    | A | Small single-purpose functions (`needsOf`, `blocks`, `Unreached`, `errorLine`), low complexity, 0 non-CRLF linter issues, clear rationale comments. |
| Handoff-readiness  | A | ADR-064 recorded, ADR-041/ADR-045/runbook/README updated, `verification.md` and `features.json` complete. |

### Verdict
PASS

### Recommended next steps
- None required to unblock `dotf spec archive BUG-173-converge-step-dependencies` (all contract artifacts are intact and verified; do not edit `proposal.md`, `tasks.md`, or `features.json` prior to archive).
- Optional follow-up (or record disposition in `verification.md` / a follow-up ticket): add an interleaved `Observer` case (running step after a `not reached` step) in `cli/internal/converge/observe_test.go` and a 3-line `errors.Join` case in `cli/internal/doctor/checks_self_deploy_test.go`.
