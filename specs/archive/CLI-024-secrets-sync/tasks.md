---
tags: [spec, tasks, secrets, sync, cli, go, ci, github]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-sync

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in
> `draft` state; freeze once you start `implementing`.
>
> **Gating:** unblocks #612 C5. Independent of an open #621/#624 — reuses only what is on
> main (`Loader.EnvFor`, the bw seam, `initrepo.OriginRepo`). The live cutover smoke is
> Windows-empirical (#612 C8), deferred; everything below is Linux-doable with fakes.

## Setup

- [x] Branch from main: `feat/secrets-sync-spec`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Consumer-routing decision resolved (`ci:<owner>/<repo>`, DECIDED 2026-06-26)

## Implementation (PR-A — this PR)

- [x] **`GitHubSecretSetter` seam** — `SetSecret(repo, name, value string) error` in
  `cli/internal/secrets/github.go` (next to `BWWriter`). Production `GHSecretSet` shells out
  to `gh secret set <name> --repo <repo>` with the value on stdin. Fake unit-tested; the
  shell-out is the live canary (#612 C8), not CI — mirrors `BWPut`'s test split.
- [x] **CI selection helper** — `Registry.SelectCI(repo)` returns the entries whose
  `consumers` contains `ci:<owner/repo>`, excluding file / `floor` / `age-offline` /
  `GITHUB_*`-prefixed vars, each with a specific skip reason. Pure, table-driven test.
- [x] **`sync` command skeleton + `sync ci`** — `newSecretsSyncCmd()` with a `ci` subcommand;
  `--repo` (default via injectable `repoOriginResolver` → `initrepo.OriginRepo`, validated by
  `initrepo.ValidRepoSlug`) + `--dry-run`; package-level `ghSecretSetter` seam (like
  `bwReader`). Wired into `newSecretsCmd`.
- [x] **Resolve + upload** — resolves each selected var via `secretLoader().EnvFor` (age|bw
  transparent); calls `ghSecretSetter.SetSecret(repo, VAR, value)`. Fail-fast (no partial upload).
- [x] **`--dry-run`** — reports `VAR → repo (N bytes)`, never the value; zero setter calls.
- [x] **Registry tag migration (dotfiles' own, evidence-scoped)** — rewrote `ci:release`
  (RELEASE_TOKEN) + `ci:bitacora` (BITACORA_PAT) → `ci:mlorentedev/dotfiles`. Scope confirmed
  by grepping `.github/workflows/` — `ci:image-push` (DOCKERHUB_*) is NOT a dotfiles secret,
  so it is excluded here and handled in the sweep below.
- [x] **Tests** — AC1 selects+uploads only the repo's set; AC2 age+bw upload identically;
  AC3 three exclusions; AC4 `--dry-run` inert + non-leaking; AC5 origin default + bad-slug error.

## PR-B (next — retirement, parity-gated)

- [x] **Retire the legacy path** — deleted `scripts/github-secrets-manager.sh`,
  `tests/github-secrets-manager.bats`, the `ls --pairs` flag + `TestSecretsLs_Pairs_EnvOnly`,
  and the dangling references (test.sh, setup-linux.sh, verify-setup.bats, pat-expiry.yml,
  runbooks/troubleshooting docs). ⚠ **Parity gate is a pre-merge step (operator):** confirm
  `sync ci --dry-run` VAR set == the legacy script's uploaded set before merging (#612 C8,
  Windows-empirical — needs `gh auth`).
- [x] **Lift the migrate guard** — removed the `ci:*` refusal in `secrets_migrate.go`'s
  `migrateGuard` (now that `sync` is the backend-agnostic upload path) + dropped its test case.

## Follow-ups (tracked, not in this slice)

- [ ] Sweep the remaining purpose tags (`ci:image-push` / `ci:payments` / `ci:newsletter` /
  `ci:social` / `ci:publish` / `ci:yt-metrics`) to `ci:<owner>/<repo>` — operator confirms
  each owner/repo (none are dotfiles' own per the workflow grep).
- [ ] (Optional) naming-convention lint in `Registry.validate()` (service prefix / field vocab).

## Closing (PR-A)

- [x] Every PR-A AC covered by ≥1 test (live `gh`/`bw` is the canary smoke, #612 C8)
- [x] `features.json` carries non-vacuous verification commands
- [x] `cd cli && go test ./...` green; `go vet ./...` + `golangci-lint run
  --exclude-use-default=false ./internal/...` clean; `go build ./...` ok
- [x] Additive: new command + seam + registry tag migration only (retirements are PR-B)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec + #612

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following
[[pattern-feature-list-as-primitive]]. Each acceptance criterion maps to ≥1 feature with
`id`, `behavior`, `verification` (executable command), `state`, and `evidence`.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after
running `verification` and capturing exit 0, may set that terminal state.
