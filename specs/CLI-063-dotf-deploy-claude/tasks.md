---
tags: [spec, tasks, templates]
created: "2026-09-04"
---

# Tasks - CLI-063-dotf-deploy-claude

> TDD order. One task = one focused commit. Tick as you go.
> `[P]` = no dependency on another unchecked task. `[AC<n>]` = satisfies acceptance criterion n.

## Settled before implementation

**Golden characterization capture does NOT transfer from CLI-021. Decided, not open.**

CLI-021 captured a single script's stdout and diffed it byte-for-byte. That worked because
`vault-maintenance-weekly.sh` was one process with one output stream. This is not that shape:

- The four capabilities run at **four different points** of a setup, not as one invocation.
- Their observable effect is on **`$HOME` and on `claude` itself**, not on stdout — a snapshot
  guard's whole job is that nothing is printed and nothing is lost.
- Two of them **shell out to `claude`**, so a capture on this box records that box's plugin
  registry, not the behaviour.

The honest instruments, per capability:

| Capability | Oracle |
|---|---|
| `settings.json` merge | **Table tests over the policy table.** Input: a template + a box state. Expected: the merged object. Each of the six keys gets a case that FAILS under top-level replace. |
| Plugin list | **A drift guard**, ~15 lines of bats: grep both twins' hardcoded ids and assert set-equality with `ai/claude/plugins.json`. This is AC2, and it is the only thing that can catch the twins diverging. |
| Snapshot guard | **Unit test on the threshold predicate**, reusing `claude_json_min_bytes`. No file I/O needed to test "is this truncated". |
| MCP registration | **Fake `claude` on PATH** recording argv. Asserts skip-if-present and error-surfacing without touching a real registry. |

Nothing here is a golden file. Do not add one.

## Setup

- [x] `proposal.md` complete, acceptance criteria testable
- [x] Command home settled against ADR-032 — `dotf deploy`, not `dotf agent` (owner decision 2026-09-05)
- [x] Oracle question settled (above)
- [x] Branch off `main`, spec folder committed, #1339 set to In Progress

## Increment 1 — snapshot guard + plugin sync (first PR, ≤300 LOC)

- [x] [P] [AC2] Failing bats: both twins' hardcoded plugin ids equal `ai/claude/plugins.json`
- [x] [AC2] Add `ai/claude/plugins.json` with the ids read out of the twins verbatim
- [x] [P] [AC7] Failing Go test: truncation predicate at the `claude_json_min_bytes` boundary
- [x] [AC7] Implement the predicate by **calling** the existing threshold in
      `cli/internal/mem/session_start_adapter.go:83` — a second literal `10240` is the defect
      *Done as:* `mem.ClaudeJSONFloor(configPath)` returns the configured `claude_json_min_bytes`
      (override included) with `mem.ClaudeJSONMinBytes` as its default, and the session-start
      canary uses that same constant. The predicate itself is new (`claude.Truncated`): the canary
      asks "below an absolute floor", the guard asks "snapshot ≥ floor and more than half lost".
- [x] [AC1] Plan/apply for the plugin capability (plan/apply only; the declaration is increment 3); counts a plugin only on install success (#1491
      divergence recorded in `divergences.md`, NOT fixed in the `.ps1`)
      **Sequencing (2026-10-03):** increment 1 ships plugin sync as a converge step of a bare
      `dotf deploy`, in the slot CLI-093 (#1953) opened for the Orca hooks after this spec was
      written. Declaring it in `ai/deploy.json` needs new entry vocabulary, and the strict decoder
      refuses an unknown field before it reads `version`, so every installed `dotf` older than the
      change stops deploying *all* configs (and the Orca heal) until a release ships. That break is
      paid once: the declaration moves to increment 3, next to the version bump it requires. AC1
      closes there, not here.
- [x] [AC6] **Negative test: the Go path emits no `hooks` key under any input.** Written in
      increment 1 even though increment 1 does not write settings — the proposal names this as the
      single most likely way to cause an incident, so the assertion predates the code that could
      violate it
- [x] Refactor; `go build ./... && go vet ./... && go test ./...`, `GOOS=windows go vet ./...`,
      pinned `golangci-lint`, `bats tests/*.bats`

## Increment 2 — MCP registration

- [x] [P] [AC1] Failing test with a fake `claude`: an entry already registered is skipped
      *Done as:* the fake answers through the `claude.Runner` seam increment 1 introduced, not a
      script on PATH; the real argv is pinned separately (`TestMcpAddArgvMatchesTheTwins`)
- [x] [AC1] Failing test: `claude mcp add` non-zero is surfaced, not swallowed
- [x] [AC1] Implement reading `mcp-servers.json`; carry HIVE-118's remove-then-re-add for `uvx hive-vault`
      **Decision (2026-10-03):** the Go path honours `prerequisite_binary` (skip and report, the
      `requires` idiom) and does **not** run `prerequisite_command`. The only such command is
      `uv tool install --upgrade hive-vault`. That is a tool install, not a registration, and an
      upgrade on every run is not idempotent. uv owns hive on msi (`uv tool list` → `hive-vault
      v4.2.2`), but `packages.json` does not declare it, so the twins' MCP loop is hive's only
      installer. Filed as #1993; it must land before the cutover deletes the twins. Nothing
      regresses meanwhile, because the shell twins still run the command.
      **Shipped (2026-10-03)** as the `claude-mcp` converge step, before `claude-plugins`, as in
      both twins. The two steps are joined, so one failing still runs the other. The npx gate the
      twins put around the whole loop is now `prerequisite_binary: npx` on `sequential-thinking`,
      which both twins already honour per server.

## Increment 3 — per-key merge policy

**Re-scoped (2026-10-03): no new manifest vocabulary, no version bump, no release dependency.**
The proposal (2026-09-05) measured `mergeInto` as a *top-level* replace. AI-042's review round 3
later made it granular: `deepMerge` recurses into objects and `unionLists` unions lists. That is
already the policy `env`, `enabledPlugins` and `permissions.allow` need, so Claude's
`settings.json` is declared with the vocabulary that exists (`strategy: merge`). Every released
`dotf` reads the manifest, so the 3a/3b split and its release in between are not needed.

What that does to the criteria:

- **AC1** — amended: the capabilities are reachable through `dotf deploy`. `settings.json` is a
  declared entry; MCP and plugins stay converge steps, because declaring an action would need the
  vocabulary this re-scope avoids adding. The proposal records the amendment.
- **AC3** — already satisfied by the engine. The tests are still written, as the regression guard.
- **AC4** — inverted. There is no allow-list in code any more: every template key is written, so
  `outputStyle`'s defect class cannot recur. The template is now the allow-list, and the guard
  is that it never carries `hooks` (also AC6's settings-writer half).
- **AC5** — not needed: no field is added, and `TestManifestVersion_FreezesTheFieldSet` already
  fails if one ever is without a bump.

Two behaviour changes against the twins, measured on msi by deploying onto a copy of
`~/.claude/settings.json`, are divergences 18 and 19: the template's 17 `permissions.deny` rules
now reach existing boxes (msi had 0), and `attribution` merges instead of being replaced whole.

- [x] [P] [AC3] Table test: `env` and `enabledPlugins` merge nested and `permissions.allow` unions, each
      keeping a box-only key (`TestPerKeyMergePolicy`); turning off the recursion or the union is killed
- [x] [AC4] [AC6] The template carries no `hooks` (`TestClaudeSettingsTemplateHasNoHooks`)
- [x] [AC1] Declare `claude-settings` in `ai/deploy.json`: `strategy: merge`, into
      `{CLAUDE_CONFIG_DIR}/settings.json`, requiring `claude` (`TestClaudeEntriesDeclared` deploys
      the shipped entry over a box holding hooks, a status line and box-only keys)
- [x] Fix the manifest `$comment`, which still described merge as a top-level replace
- [x] Parity on msi: the merge onto a copy of the real file adds `$schema` and 17 deny rules and
      changes nothing else (`hooks` identical, 45 allow entries unchanged); a second run is
      `in sync`, and so is a re-plan after the twin's jq merge ran over the result (no flapping)

## Closing

- [x] Every acceptance criterion covered by ≥1 test and ≥1 `features.json` entry with a
      non-vacuous verification command
- [x] Lint + both layers green; `GOOS=windows go vet ./...` clean
- [x] `verification.md` filled in
- [ ] **Independent adversarial review** (reviewer ≠ implementer) before `dotf spec archive`
- [x] No twin deleted, no caller repointed — verified by `git diff --stat` showing nothing under
      `setup-*.{sh,ps1}` or `scripts/`

## Machine-readable features

Emits a sibling `features.json` per [[pattern-feature-list-as-primitive]]. The agent CANNOT write
`"state": "passing"` — only the harness, after running `verification` and capturing exit 0, may set
that terminal state.
