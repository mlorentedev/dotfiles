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

- `Plan` reports `unsupported` for a release tool with no asset for the platform. `Install` errors in that case, so a plan row is more useful than an error. The probe still runs first, so the row keeps the installed version (`TestInstallerPlan/no build for this platform, but installed: the probe still runs`); an earlier draft decided before probing, and the second patch of #1848 reversed it.
- The doctor side of #1381 is unchanged. It already reads the checkout first. Its own checkout resolution differs from `env.RepoDir` (`DOTFILES_REPO_DIR` before the cwd), and that is #1418.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: the one non-obvious point, that a resolver test needs the checkout and the mirror to disagree, is lesson 321's, written for #1863
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-041 (#1845) already sets the convergence order this implements
- [x] New pattern candidate for `00_meta/patterns/`? no: repository-specific CLI behaviour

## Review (2026-09-30)

`review.md`, `nan/deepseek-v4-flash`, reviewed `3f13e19`: **PASS-WITH-GAPS**.

| Finding | Disposition |
|---|---|
| Major, theoretical: `Plan` skips the checksums and npm-package checks `Install` makes before deciding, so a dry run could promise what install refuses | Ticketed as #1892. No entry in today's `packages.json` triggers it. |
| Minor: this file said `Plan` decides `unsupported` before probing | Fixed above. |
| Minor: the catalog resolver has a third tier, `DOTFILES_REPO_DIR`, between the cwd checkout and the mirror, which AC2 does not name | Recorded here, no code change. `proposal.md`'s What names `env.RepoDir`, which is where that tier comes from, and `TestResolveCatalogPathPrefersRepoCheckout` asserts it. AC2 is contract text and stays as reviewed. |
| Minor: `unsupported` also covers a source type `Install` refuses, and the row does not say which | Ticketed with the Major, #1892. |
| Speculative: no command-level test for an `unsupported` row or `--dry-run <name>` | Ticketed with the Major, #1892. |

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-091b-machine-convergence/` -> `specs/archive/CLI-091b-machine-convergence/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
