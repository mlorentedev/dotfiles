---
spec: "CLI-091b-machine-convergence"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "3f13e199f1dced7f7bdf9295bedba745f0314d80"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: CLI-091b-machine-convergence — `git diff 6d1393d7ec4fc3a2c0b8d2c7454354f63fc0f3d2...HEAD`
(18 commits / 165 files; the CLI-091b change is commit `a94ae346`, the only one in the range that
touches this contract). The rest of the range belongs to other specs already carrying their own
archived reviews (CLI-091e, HARNESS-111, OPS-042, WIN-014, AI-034, HARNESS-041, TOOL-023); I
verified they do not alter the CLI-091b surfaces and did not re-review them.

**Sources**: `specs/CLI-091b-machine-convergence/{proposal,tasks,verification}.md`, `features.json`,
`review-request.json`; `cli/internal/cmd/tools.go`, `cli/internal/cmd/tools_test.go`,
`cli/internal/env/env.go`, `cli/internal/env/env_test.go`, `cli/internal/tools/install.go`,
`cli/internal/tools/install_test.go`, `cli/internal/tools/catalog.go`.

**Evidence gathered in this session** (all re-run, not read):

- `cd cli && go build ./... && go vet ./...` → exit 0.
- `GOOS=windows go vet ./...` → exit 0.
- `go test ./...` → all packages ok, exit 0 (targeted `TestInstallerPlan`, `TestResolveCatalogPath*`,
  `TestTools*` all pass).
- `golangci-lint run ./internal/...` → `0 issues` (one stale cross-worktree cache warning, unrelated).
- `go run ./cmd/dotf tools install --dry-run` → five rows (`sops/opencode/copilot/bw/yarn`, all
  `skip` at their pins), exit 0, and no `~/.local/bin` entry created by the run.
- **Mutation testing** (each mutation reverted; `git status --porcelain` clean afterwards apart from
  the launcher's `review-request.json`). Every guarded property was caught by a named test:
  1. `Plan` always returns `install` → `TestInstallerPlan` FAIL.
  2. `ResolveCatalogPath` always returns the mirror → `TestResolveCatalogPathPrefersRepoCheckout`
     and `TestToolsList_CheckoutCatalogWins` FAIL.
  3. `Plan` reports `unsupported` before probing (the bug the commit's second patch fixed) →
     `TestInstallerPlan/no build for this platform, but installed: the probe still runs` FAIL.
  4. `--dry-run` delegates to the applying path → `TestToolsInstall_DryRun` FAIL (and it really did
     download and write, confirming the test catches a write, not just output text).
  5. `Plan` reaches the `Run` seam for npm → `TestInstallerPlan/npm below the pin` FAIL.
- Launcher digests verified: `spec.ContractDigests(specs/CLI-091b-machine-convergence)` reproduces
  `review-request.json`'s three `contract_digests` exactly, so this review is fresh against the
  contract on disk (`reviewed_sha` = `git rev-parse HEAD` = `3f13e19…`).

### Spec and task alignment

- **AC1** — `Installer.Plan` (`cli/internal/tools/install.go`) runs `in.current(t)` (the same probe
  `Install` uses, dispatching github-release→`Dest/bin`, npm→PATH) and the same `decideAction`, and
  reaches neither `Fetch` nor `Run`; `dotf tools install --dry-run` prints
  `NAME INSTALLED PIN ACTION` from `planToolsInstall`. The "changes nothing / exit 0" half is proven
  by `TestToolsInstall_DryRun` (Dest absent check) and by mutation 4. Met, with the caveats below.
- **AC2** — `env.ResolveCatalogPath` is used by both `tools list` and `tools install`
  (`loadToolsCatalog`). The cwd-checkout-wins and mirror-without-checkout halves are met and
  mutation-verified. One clause is not literally met (finding 1).
- **tasks.md** — every `[x]` maps to the diff: `ResolveCatalogPath`, `Installer.Plan`, the flag, and
  all five named tests exist and run.
- **features.json** — both entries non-vacuous; verification commands resolve to real test names;
  `state: pending` with empty `evidence`, which is correct (the harness, not the author, sets
  `passing`). No `passing` with empty evidence.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec vs code (AC2) | The resolver has a **third** tier AC2 does not describe. AC2: "the checkout that contains the working directory when it has one, and from the deploy mirror otherwise". Code: cwd-containing checkout → **the `DOTFILES_REPO_DIR` checkout** → mirror. When the cwd is not in a checkout but `DOTFILES_REPO_DIR` is set (setup exports it: `setup-linux.sh:1638`), `tools list`/`install` read that checkout, not the mirror AC2 names. | `env.ResolveCatalogPath` calls `RepoDir()`; `RepoDir` falls back to `DOTFILES_REPO_DIR` when the walk-up finds no `.git` (`cli/internal/env/env.go:170-180`). `TestResolveCatalogPathPrefersRepoCheckout` sets `DOTFILES_REPO_DIR` and `t.Chdir`s to a temp dir outside any checkout, then asserts the repo copy — the test *encodes* the third tier. | `TestResolveCatalogPathPrefersRepoCheckout` (proves it), `TestToolsList_CheckoutCatalogWins` (cwd tier) | spec artifact — and note that `proposal.md`'s own "What" already says the checkout is found "cwd first (`env.RepoDir`)", so the code follows the more specific contract text and ADR-030's resolver idiom. Do not edit the contract for this; record the divergence in `verification.md` (non-contract). |
| Major | THEORETICAL | correctness / plan honesty | `Plan` does not mirror the entry-level pre-conditions `Install` enforces **before** `decideAction`, so a plan can promise an action the apply refuses. (a) A github-release tool whose asset exists but whose `checksums` template is empty: `Plan` reports `install`/`upgrade`/`skip`, while `installRelease` returns `no checksums file declared — refusing to install unverified`. (b) An npm tool with an empty `source.package`: `Plan` reports an action, while `installNpm` returns `npm source declares no package`. No entry in today's `packages.json` triggers either (`sops` declares checksums; all four npm tools declare a package), so this is latent, not live. | `cli/internal/tools/install.go`: `installRelease` computes `sumsName` and errors before `decideAction`; `installNpm` errors on `pkg == ""`; `Plan` checks only `AssetName` and the source type. | UNTESTED (no case exercises either pre-condition through `Plan`) | code + tests — extract the shared pre-flight both call, or add catalog-validation rows to `TestInstallerPlan` |
| Minor | REAL | verification artifact | `verification.md` states "`Plan` reports `unsupported` **before probing**, for a release tool with no asset for the platform." That is false for the code on disk — the probe runs first and the row keeps its installed version. The sentence describes the pre-fix behavior that the same commit's second patch (`fix(tools): make the dry run refuse what install refuses…`) reversed. | `Plan`: `Installed: in.current(t)` precedes the `switch`; `TestInstallerPlan/no build for this platform, but installed: the probe still runs` asserts a non-empty `Installed` and passes. | `TestInstallerPlan/no_build_for_this_platform,_but_installed:_the_probe_still_runs` | spec artifact — `verification.md` is excluded from the staleness check and is free to edit |
| Minor | REAL | AC1 semantics | `unsupported` is documented in AC1 as "the tool has no build for this platform", but `Plan` also emits it for a source type `Install` refuses (`homebrew`, anything not `github-release`/`npm`). The row gives the reader no way to tell "no asset for this OS/arch" from "this catalog entry is uninstallable everywhere". | `Plan`'s `default:` branch → `PlanUnsupported`; `TestInstallerPlan/a source type Install refuses` expects exactly that. | `TestInstallerPlan/a_source_type_Install_refuses` | code (a reason column/message) or spec (AC wording); low risk |
| Minor | SPECULATIVE | test coverage | AC1's "exits 0" is proven end-to-end only for the `install`-action path. No command-level test drives a row whose action is `unsupported` (also exit 0) or the `--dry-run <name>` single-selection form. | `TestToolsInstall_DryRun` runs with no name against a one-tool fixture and matches an `install` row; the unsupported branch is unit-tested at the `Installer` level only. | UNTESTED (unsupported/exit-0 and single-name paths) | tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Both ACs met on their primary paths and every guarded property is mutation-verified; the plan/apply pre-flight divergence (finding 2) is a latent negative-path gap. |
| Verification       | B | Commands are reproducible and I re-ran all of them; `verification.md` carries one claim the code contradicts (finding 3). |
| Scope              | A | The CLI-091b change is 6 code/test files plus its spec folder; no unrelated edits, no dependency or contract-surface change beyond one new flag and one new resolver. |
| Reliability        | B | Dry-run provably never reaches `Fetch`/`Run` and writes nothing; the plan/apply honesty gap is the one unhandled path. |
| Maintainability    | A | `Plan` and `planToolsInstall` are short, single-purpose, documented with *why*; no dead code; CC well under 10. |
| Handoff-readiness  | B | Spec, tasks, features and the lesson-321 promotion are in place; the stale decision line in `verification.md` (finding 3) should be corrected before archive. |

### Verdict
PASS WITH GAPS

No Blocker and no **REAL** Major: the one Major (finding 2) is THEORETICAL — no catalog entry in the
shipped `packages.json` can trigger it — so it is tracked, not blocking. Findings 1, 3 and 4 are
REAL but Minor; finding 5 is SPECULATIVE and is surfaced only. No rubric dimension is C or D, and
the two real misbehaviours a reviewer would most fear (a dry run that writes, a plan that ignores
`decideAction`) are both named-test-caught.

`dotf spec archive` is **advisable** in this state: the frontmatter above is valid, `reviewed_sha`
matches the launcher's requested head, the reviewer id is in `harness/reviewer-pool.json`, and the
recorded `contract_digests` still match the files on disk. No contract-set edit is requested by this
review.

### Recommended next steps

All of these are outside the contract set (`proposal.md` / `tasks.md` / `features.json`), so none
invalidates this verdict. Disposition each in `verification.md` — applied, ticketed, or declined
with a reason, per the Definition of Done.

- **Finding 3 (do this before archive)** — correct the `verification.md` "Decisions made during
  implementation" line: `Plan` probes *before* it decides `unsupported`. As written it documents the
  behavior the same commit reverted.
- **Finding 2 (ticket it)** — file a follow-up for the plan/apply pre-flight divergence: either give
  `Plan` and `Install` one shared entry-validation helper, or add `TestInstallerPlan` rows for a
  release tool with no checksums template and an npm tool with no package. Track it with the rest of
  the Track-B rows so it lands before a catalog entry can hit it.
- **Finding 1 (note it, do not edit the contract)** — record in `verification.md` that
  `ResolveCatalogPath` has the `DOTFILES_REPO_DIR` tier in addition to the two AC2 names, and that
  `proposal.md`'s "What" already declares `env.RepoDir`. If a future round wants AC2 to be literal,
  that is a contract edit and needs its own review.
- **Findings 4 and 5 (optional, low priority)** — add an `unsupported` reason to the `Plan` row or
  correct AC1's wording, and add an `unsupported`-action + single-name case to
  `TestToolsInstall_DryRun`.
