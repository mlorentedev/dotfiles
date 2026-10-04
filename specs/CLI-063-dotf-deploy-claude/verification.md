---
tags: [spec, verification, templates]
created: "2026-09-04"
---

# Verification - CLI-063-dotf-deploy-claude

## Evidence

Every criterion maps to a `features.json` entry whose `verification` command was run on
`origin/main` at `66da55b5` on 2026-10-03 and exited 0.

- [x] AC1 (as amended) -> f1, f9. `TestClaudeEntriesDeclared`, `TestDeployCmd_BareDeployRegistersMissingMCPServers`,
      `TestDeployCmd_BareDeployInstallsMissingClaudePlugins`, `TestRegister*`. Commits `614ae4ed` (#1992),
      `0f67099a` (#1994), `944993e2` (#1996)
- [x] AC2 -> f2. `tests/claude-plugins.bats`, commit `614ae4ed`
- [x] AC3 -> f3. `TestPerKeyMergePolicy`, commit `944993e2`
- [x] AC4 (as inverted) -> f4. `TestClaudeEntriesDeclared` (every template key reaches the box),
      `TestClaudeSettingsTemplateHasNoHooks`, commit `944993e2`
- [x] AC5 (as amended) -> f5. `TestManifestVersion_FreezesTheFieldSet`, `TestParseManifest_RefusesWhatItCannotFullyRead`
- [x] AC6 -> f6. `TestGuardNeverAddsAHooksKey`, `TestClaudeSettingsTemplateHasNoHooks`, `TestClaudeEntriesDeclared`
- [x] AC7 -> f7. `TestSyncCountsOnlySuccessfulInstalls`, `TestDeployCmd_AFailedPluginInstallIsNamedAndFailsTheRun`;
      #1491 recorded as `divergences.md` row 1
- [x] AC8 -> f8. No commit that touched this spec changed `setup-*.{sh,ps1}` or `scripts/`. The check now walks
      every such commit; the old `origin/main...HEAD` form is empty by construction on a closing branch.

## Test status

- Go: `go build ./... && go vet ./... && go test ./...`, `GOOS=windows go vet ./...`, pinned `golangci-lint` -> all green; `golangci-lint` 2.12.2 (the pin) on a cleaned cache: 0 issues
- Shell: `bats tests/*.bats` -> 1775/1776; the one failure is #1641 (test 1364, the oh-my-zsh snapshot against the local install), environmental and unrelated
- Manual smoke test on msi:
  - `dotf deploy --dry-run` -> `claude-mcp 0 added, 4 already present`.
  - Deploying `claude-settings` onto a copy of `~/.claude/settings.json` adds `$schema` and the
    template's 17 `permissions.deny` rules and changes nothing else (`hooks` identical, 45 allow
    entries unchanged).
  - A second run reports `in sync`. So does a re-plan after the twin's jq merge has run over the
    result.
- No regressions in the existing suite: yes.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **The command home moved from `dotf agent claude sync` to `dotf deploy` before any code was
  written.** #1339's framing rested on three premises that measurement falsified: ADR-032:122-124
  reserves `dotf agent` for the quota-spending run side; `strategy: merge` already exists (AI-039,
  in production on three configs); and `ai/deploy.json`'s own `$comment` names MCP registration as
  slice 3 of CLI-039. Owner decision 2026-09-05.
- **The scope shrank as a result.** The "largest and subtlest" increment is not a 100-line jq port:
  `mergeInto` already implements 3 of Claude's 6 key policies. What is missing is three per-key
  behaviours (`env`, `enabledPlugins` nested merge; `permissions.allow` union) the manifest cannot
  yet express.
- **Golden characterization capture was ruled out deliberately**, not skipped. Four capabilities
  running at four points of a setup, two of them shelling out to `claude`, have no single
  capturable stream. See `tasks.md` for the per-capability oracle chosen instead.

- **Increment 3 needed no new manifest vocabulary.** AI-042 had already made `deepMerge` granular
  after this proposal measured it as a top-level replace, so `strategy: merge` expresses Claude's
  policy as is. AC1, AC4 and AC5 were amended (2026-10-03); the version stays 3 and no release is
  needed before the entry takes effect. Lesson 336 records how the measurement went stale.
- **MCP and plugins are converge steps, not declared entries**: declaring an action needs vocabulary
  the re-scope avoided adding.
- **`prerequisite_command` is not run by the Go path** (#1993). That issue must land before any
  cutover deletes the twins, since the twins' MCP loop is hive's only installer today.
- **24 behaviour differences against the twins** are recorded in `divergences.md`. Rows 18 and 19
  change existing boxes: the template's deny rules arrive, and `attribution` merges instead of being
  replaced whole.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? yes: lesson 336, a spec measurement goes stale when another change moves the engine (landed in #1996)
- [x] ADR-worthy decision? no: the command home follows ADR-032 and the converge-step slot already exists (CLI-093)
- [x] New pattern candidate? no: lesson 336 is local to this engine

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-063-dotf-deploy-claude/` -> `specs/archive/CLI-063-dotf-deploy-claude/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
