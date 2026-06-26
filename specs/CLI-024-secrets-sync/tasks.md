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

## Implementation

- [ ] **`GitHubSecretSetter` seam** — interface `SetSecret(repo, name, value string) error`
  in `cli/internal/secrets` (next to `BWWriter`). Production `GHSecretSet` shells out to
  `gh secret set <name> --repo <repo>` with the value on stdin. Unit-test the fake; the
  shell-out is the live canary (#612 C8), not CI — mirror `BWPut`'s test split.
- [ ] **CI selection helper** — given a registry + an `owner/repo`, return the entries whose
  `consumers` contains `ci:<owner/repo>`, excluding file / `floor` / `age-offline` /
  `GITHUB_*`-prefixed vars, each with a specific skip reason. Pure function, table-driven test.
- [ ] **`sync` command skeleton + `sync ci`** — `newSecretsSyncCmd()` with a `ci` subcommand;
  `--repo` (default `initrepo.OriginRepo`, validate `initrepo.ValidRepoSlug`) and `--dry-run`
  flags; package-level `ghSecretSetter` seam var (like `bwReader`). Wire into `newSecretsCmd`.
- [ ] **Resolve + upload** — resolve each selected var via `secretLoader().EnvFor(entries, only)`
  (age|bw transparent); for each `VAR=value`, call `ghSecretSetter.SetSecret(repo, VAR, value)`.
  Fail-fast on a resolution error (never a partial upload).
- [ ] **`--dry-run`** — report `VAR → repo` with byte lengths, never the value; zero setter calls.
- [ ] **Registry tag migration (dotfiles' own)** — rewrite `ci:release` / `ci:bitacora` /
  `ci:image-push` → `ci:mlorentedev/dotfiles` in `secrets/registry.yaml` (comment-preserving).
- [ ] **Tests** — AC1 selects+uploads only the repo's set; AC2 age+bw upload identically;
  AC3 three exclusions; AC4 `--dry-run` inert + non-leaking; AC5 origin default + bad-slug error.
- [ ] **Parity gate, then retire** — confirm `sync ci --dry-run` VAR set == the legacy script's
  uploaded set for dotfiles; then delete `scripts/github-secrets-manager.sh`,
  `tests/github-secrets-manager.bats`, the `ls --pairs` flag, and `TestSecretsLs_Pairs_EnvOnly`.
- [ ] **Lift the migrate guard** — remove the `ci:*` refusal in `secrets_migrate.go`'s
  `migrateGuard` (now that `sync` is the backend-agnostic upload path) + drop/adjust its test.

## Follow-ups (tracked, not in this slice)

- [ ] Sweep the remaining purpose tags (`ci:payments` / `ci:newsletter` / `ci:social` /
  `ci:publish` / `ci:yt-metrics`) to `ci:<owner>/<repo>` — operator confirms each owner/repo.
- [ ] (Optional) naming-convention lint in `Registry.validate()` (service prefix / field vocab).

## Closing

- [ ] Every AC covered by ≥1 test (live `gh`/`bw` is the canary smoke, #612 C8)
- [ ] `features.json` carries non-vacuous verification commands
- [ ] `cd cli && go test ./... ` green; `go vet ./...` + `golangci-lint run
  --exclude-use-default=false ./internal/...` clean; `go build ./...` ok
- [ ] Additive: new command + seam + registry tag migration + retirements only
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec + #612 C5

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following
[[pattern-feature-list-as-primitive]]. Each acceptance criterion maps to ≥1 feature with
`id`, `behavior`, `verification` (executable command), `state`, and `evidence`.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after
running `verification` and capturing exit 0, may set that terminal state.
