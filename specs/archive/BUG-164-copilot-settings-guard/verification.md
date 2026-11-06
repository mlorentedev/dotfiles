---
tags: [spec, verification, templates]
created: "2026-11-05"
id: "BUG-164-copilot-settings-guard"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-164-copilot-settings-guard

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 -> synthetic `COPILOT_HOME` fixtures in `copilotsettings`, doctor, command and converge tests.
- [x] Criterion 2 -> `TestLoadPolicy*`, including the committed policy and invalid contract table.
- [x] Criterion 3 -> `TestCheckCopilotSettingsReportsFileAndKeyWithoutValue`.
- [x] Criterion 4 -> `TestRepairRemovesOnlyRetiredLeavesAndIsIdempotent`, sibling preservation and doctor fix/backup.
- [x] Criterion 5 -> `TestDeployCmd_CopilotRetiredSettingsAreRepairedForBareAndNamedRuns` and `TestConfigsDeploy_CopilotRetiredSettingsPlanApplyAndSecondPlan`.
- [x] Criterion 6 -> package repair, command deploy and converge second-run assertions.
- [x] Criterion 7 -> missing, malformed, non-object, unreadable settings and malformed/missing policy tests.
- [x] Criterion 8 -> feature f6 static source regression; no matching invocation exists.

## Test status

- Baseline: full `internal/doctor` is not a valid local baseline on this host because unrelated symlink-privilege and domain-account tests fail on unchanged `origin/main`; focused BUG-164 selectors will be used.
- Exact feature commands f1-f5: PASS; f6 static source scan: no matches.
- Adjacent suites: `go test ./internal/copilotsettings ./internal/deploy ./internal/converge` -> PASS; `go test ./internal/cmd -run '^TestDeployCmd_'` -> PASS.
- Doctor registration contracts and all `TestCheckCopilot*` selectors -> PASS.
- Vet: `go vet ./internal/copilotsettings ./internal/deploy ./internal/doctor ./internal/cmd ./internal/converge` -> PASS.
- Formatting and lesson hygiene -> PASS.
- Manual smoke: a built development binary repaired a synthetic Copilot home, created `settings.json.pre-dotf`, preserved unrelated keys, and reported the second named deploy in sync.
- Independent code review: one Medium finding applied — doctor now fails on a missing policy; only the deploy callback ignores it to keep old checkouts compatible.

## Review disposition

- **Applied:** added `TestCurrentSourceDeclaresNoRetiredPath`, so the tracked
  `ai/copilot/settings.json` cannot reintroduce a path declared retired.
- **Recorded, no contract edit:** `tasks.md` still shows the lint/PR closing
  boxes unchecked because changing a contract file after the passing review
  would stale it. PR #3218 is open; final CI state is recorded on the PR.
- **Declined for this PR:** the theoretical rendered-config ordering gap.
  `RepairConfig` returns false for every entry except `copilot-settings`, whose
  manifest entry is `render: false`. Any future rendered repair must add an
  `ErrRenderIncomplete` regression before wiring that entry.
- **Accepted as verification-only gap:** f6 uses `! rg`; this session ran it
  with ripgrep present and captured exit 1/no matches. The runtime guard is
  independently covered by `TestCheckCopilotSettingsReportsFileAndKeyWithoutValue`.

## Adjacent issue disposition

- #3208 is separate: it concerns `cli/internal/vm/run.go` timeout log capture,
  not `cli/internal/deploy/run.go`.
- #2455 is separate: it changes identity semantics for merged list elements;
  BUG-164 adds an optional pre-deploy repair and does not alter merge behavior.
- #1948 is separate: it proposes new seed/replace-key manifest strategies and
  a deployment-order rollout; BUG-164 deliberately leaves manifest version 6.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Keep policy outside `ai/deploy.json` so the currently installed `dotf 0.79.0` keeps parsing and deploying main before the next release.
- Reuse `deploy.EditJSON` for the one-time `.pre-dotf` backup, staged write, atomic rename, and declared mode instead of adding a second writer.
- The optional `deploy.RunOptions.Repair` seam is nil-compatible and executes only after platform/command gates, so skipped entries and old callers are unchanged.
- Missing policy is strict in direct inspection/doctor. Only `RepairConfig` treats it as no-op because a new binary may operate briefly against an older checkout.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: `docs/lessons/lesson-559-a-config-listing-that-omits-invalid-keys-is-not-a-validator.md`
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-020 already governs the cross-OS Go ownership and no new architecture boundary is introduced.
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the false-green behavior is currently measured only for GitHub Copilot.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/BUG-164-copilot-settings-guard/` -> `specs/archive/BUG-164-copilot-settings-guard/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
