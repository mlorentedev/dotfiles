---
tags: [spec, verification, templates]
created: "2026-09-29"
---

# Verification - CLI-091b-machine-convergence

## Evidence

- [x] AC1 -> tests `TestInstallerPlan`, `TestToolsInstall_DryRun`
- [x] AC2 -> tests `TestResolveCatalogPathPrefersRepoCheckout`, `TestResolveCatalogPathFallsBackToDeployed`, `TestToolsList_CheckoutCatalogWins`, `TestToolsList_MirrorWithoutCheckout`

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...`: green. `GOOS=windows go vet ./...`: green. golangci-lint 2.12.2: 0 issues.
- Manual smoke test: `go run ./cmd/dotf tools install --dry-run` in a worktree printed five rows, all `skip` at their pins, and exited 0.
- No regressions in the existing suite. The three existing command tests now run from a directory outside any checkout, because the resolver would otherwise find this repository's catalog.

## Decisions made during implementation

- `Plan` reports `unsupported` before probing, for a release tool with no asset for the platform. `Install` errors in that case, so a plan row is more useful than an error.
- The doctor side of #1381 is unchanged. It already reads the checkout first. Its own checkout resolution differs from `env.RepoDir` (`DOTFILES_REPO_DIR` before the cwd), and that is #1418.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-091b-machine-convergence/` -> `specs/archive/CLI-091b-machine-convergence/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
