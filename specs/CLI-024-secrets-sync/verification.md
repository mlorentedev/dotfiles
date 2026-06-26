---
tags: [spec, verification, secrets, sync, cli, go, ci, github]
created: "2026-06-26"
---

# Verification - CLI-024-secrets-sync

> Status: **PR-A implemented + verified** (the `sync ci` command). AC6/AC7-retirement land in
> PR-B (the parity-gated removal of the shell twin). No evidence is fabricated.

## Evidence

- [x] **AC1 — selects + uploads the repo's CI set** -> `TestSecretsSyncCi_SelectsRepoSet`
  (fake resolver + fake setter; asserts the exact `(repo, VAR, value)` calls; other-repo
  entries not selected) + `TestSelectCI` (pure-helper coverage).
- [x] **AC2 — backend-agnostic** -> `TestSecretsSyncCi_AgeAndBwUploadIdentically`
  (one age + one bw entry both reach the setter with their resolved values).
- [x] **AC3 — exclusions specific + silent-free** -> `TestSecretsSyncCi_Exclusions`
  (file / floor|age-offline / `GITHUB_*` each skipped with a distinct reason, no setter call).
- [x] **AC4 — `--dry-run` inert + non-leaking** -> `TestSecretsSyncCi_DryRunInert`
  (zero setter calls; output has byte lengths, asserts neither secret value appears).
- [x] **AC5 — `--repo` default + bad slug** -> `TestSecretsSyncCi_RepoResolution`
  (origin default via injected `repoOriginResolver`; invalid `owner/name` errors, no upload).
- [ ] **AC6 — parity gate before retirement** -> **PR-B**: parity check logged + the diff
  removes the script, its bats, `ls --pairs`, and `TestSecretsLs_Pairs_EnvOnly`.
- [x] **AC7 — clean + additive (PR-A scope)** -> full suite green; `go vet` + `golangci-lint
  --exclude-use-default=false` exit 0; gofmt-clean. (Retirements are PR-B.)

## Test status

- `cd cli && go test ./...` -> all packages `ok` (secrets + cmd suites incl. the 7 new tests).
- Lint: `golangci-lint run --exclude-use-default=false ./internal/...` exit 0; `go vet ./...`
  clean; `gofmt -l` clean on the new files.
- Smoke: `go run ./cmd/dotf secrets sync ci --help` renders the command + flags.
- Parity (AC6, PR-B): `dotf secrets sync ci --repo mlorentedev/dotfiles --dry-run` VAR set vs
  the legacy `github-secrets-manager.sh --from-mapping --list` set -> _to record in PR-B_.
- Live `gh secret set` + a real `bw`-backed resolve are the canary smoke (#612 C8) with the
  operator's `bw unlock` + `gh auth` — Windows-empirical, not CI.

## Decisions made during implementation

- Consumer routing keyed on `ci:<owner>/<repo>` (DECIDED 2026-06-26): the repo, not a
  workflow "purpose", is the routing key (`gh secret set` is per-repo).
- Actions secret name == exposed env VAR (1:1, flat consumer-boundary convention); storage
  layout (`bw:` item/field) stays decoupled and grouped by service/account.
-

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? <decide at close — candidate: "name-match at the consumer
  boundary, decouple at the storage boundary" if it proves non-obvious in review>
- [ ] ADR-worthy? no — within ADR-029 (this spec implements it).
- [ ] New pattern for `00_meta/`? no — repo-specific (the registry indirection generalizes,
  but it is already captured by ADR-028/secrets-security).

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-024-secrets-sync/` -> `specs/archive/CLI-024-secrets-sync/`
- [ ] Bitácora board ticket (#612 C5) updated with the PR link (ADR-018)
- [ ] Promotions above executed (if any)
