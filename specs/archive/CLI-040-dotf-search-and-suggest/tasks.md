---
id: "CLI-040-dotf-search-and-suggest-tasks"
type: spec-tasks
status: active
created: "2026-08-18"
template_version: "1.0"
---

# Tasks — CLI-040 Fast Knowledge Search & Dynamic Harness Suggest

## Implementation

- [x] **T1: Harness Suggest Engine**
  - Implement `Suggest(triggers []TriggerRule, prompt string, paths []string) Suggestion` in `cli/internal/harness/triggers.go`.
  - Wire `newHarnessSuggestCmd()` in `cli/internal/cmd/harness.go` supporting `--prompt`, `--diff`, and `--json`.
  - Add unit test coverage in `cli/internal/harness/triggers_test.go`.
- [x] **T2: Vault Search & Ranking Package**
  - Implement `cli/internal/search/search.go` with `ParseDocument`, `IndexVault`, `ScoreDocument`, and `Search`.
  - Token-weighted relevance scoring (ID, Title, Tags, Keywords, Description, Body).
  - Add unit tests in `cli/internal/search/search_test.go`.
- [x] **T3: Search CLI Subcommands**
  - Implement `newSearchCmd()` in `cli/internal/cmd/search.go`.
  - Register `dotf search` in `cli/internal/cmd/root.go` and `dotf vault search` in `cli/internal/cmd/vault.go`.
- [x] **T4: Integration Test Suites**
  - Create `tests/harness-suggest.bats` testing prompt matching, file arguments, and `--json` format.
  - Create `tests/dotf-search.bats` testing keyword search, `--type` filtering, and `--json` outputs.
