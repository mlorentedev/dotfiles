---
tags: [spec, verification]
created: "2026-08-28"
---

# Verification - AI-039-copilot-settings-merge

## Evidence

Run on the Windows work box, 2026-08-28, worktree `dotfiles-wt-copilot-settings`,
branch `feat/copilot-settings-merge`, `dotf` built from the branch (Copilot CLI 1.0.81).

- [x] **AC1** → `TestDeploy_MergePreservesUnmanagedKeys`, `TestDeploy_MergeCreatesAnAbsentDestination`,
  `TestDeploy_MergeToleratesTheCLIHeaderAndDoesNotChurnIt`, `TestDeploy_MergeIsIdempotentAndDoesNotRewrite`,
  `TestDeploy_MergeRejectsANonObjectDestinationByName`. Mutation: merge starting from an empty
  object (drops unmanaged keys) → the first and third go red; restored.
- [x] **AC2** → `TestPlanConfig_And_DryRun_CreateNoDestinationDirectory` (both strategies),
  `TestPlanConfig_RefusesARenderedConfig`, `TestParseManifest_ValidatesStrategyByName`. Mutation:
  `MkdirAll` before the compare in the non-rendered path → red; restored.
- [x] **AC3** → `tests/copilot-config.bats` 11/11 (documented-key set frozen from `copilot help config`
  1.0.81; manifest shape; explicit copy in both setups); `TestDeployCmd_SkipsAnEntryWhoseRequiredCommandIsAbsent`
  (the `requires` gate that keeps the integration guard `verify-setup.bats` "copilot config NOT
  deployed when the binary is absent" true).
- [x] **AC4** → `TestCheckDeployManifest_ByStatus` (PASS with counts; WARN naming the merge entry, the
  replace entry, absent destinations; gated entry compared once its command is on PATH; SKIP without a
  repo; no directory created in any case).
- [x] **AC5** → box, in order:
  1. `~/.copilot/settings.json` before: keys `allowedUrls, contextTier, effortLevel, includeCoAuthoredBy, model, renderMarkdown`.
  2. `dotf deploy --dry-run` → `would deploy copilot-settings`; `copilot-config` (CLI-managed, `//` header)
     and `copilot-mcp` **in sync** — the header tolerance measured on the real file.
  3. `dotf deploy` → `deployed  copilot-settings`; after: the six keys plus `autoUpdate=false`,
     `model=gpt-5.6-sol`, `includeCoAuthoredBy=false`, `effortLevel=max`, `contextTier=long_context`,
     `allowedUrls` intact (16 entries).
  4. `dotf deploy` again → all five entries `in sync`; `config.json` still opens with the CLI's two `//` lines.
  5. `dotf doctor` → `[Deployed agent configs (ai/deploy.json)] (1 checks, all ok)`.
  6. `copilot -p "Reply with exactly: OK"` → exited with code 0.

## Test status

```text
go build ./... && go vet ./... && GOOS=linux go vet ./... && go test ./internal/deploy/ ./internal/cmd/ ./internal/doctor/   -> ok
golangci-lint run ./...   -> 0 issues
bats tests/copilot-config.bats tests/env-contract.bats tests/setup-linux.bats tests/setup-windows.bats   -> all ok
shellcheck setup-linux.sh; bash -n setup-linux.sh   -> ok
setup-windows.ps1: parse errors 0; non-ASCII bytes 32 = origin/main; CRLF 2347, bare LF 0
```

- No regressions in the existing suite: yes.

## Decisions made during implementation

- **Semantic in-sync for merge, textual for replace.** The CLI's `//` header on `config.json` would
  make a byte-compare call every setup run drift; the header is dropped on read only and never
  rewritten unless a managed key differs (measured: `copilot-config` in sync on the first dry-run).
- **`requires` came into scope.** `verify-setup.bats` asserts `~/.copilot` never appears on a box
  without copilot (#1312); a bare `dotf deploy` writing Copilot's JSON everywhere would have broken
  it and left files nobody reads (#843). Deploy and doctor honour the same field.
- **Compare hoisted above staging for non-rendered entries.** `Deploy(dryRun)` used to create the
  destination directory and a temp file before comparing; a diagnostic built on it would deploy
  by asking. `PlanConfig` is the read-only half doctor calls.
- **`{HOME}/.copilot`, not `{COPILOT_HOME}`.** `env.ResolvePath` needs `machine.json`, which a
  first setup has not written when `dotf deploy` runs; the contract's default is the same path.
- **Drift is WARN.** Copilot's `/model` legitimately rewrites a managed key; the remedy is one
  command and the doctor row names it.

## Promotion candidates

- [ ] Lesson: no (the manifest comment and this spec carry the measurement).
- [ ] ADR-worthy decision: no — it extends CLI-039's manifest under ADR-020 C7.
- [ ] Pattern: no.

## Archive checklist

- [ ] `dotf spec review AI-039-copilot-settings-merge` PASS
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/AI-039-copilot-settings-merge/`
- [ ] Bitácora #1322 closed with the PR link
