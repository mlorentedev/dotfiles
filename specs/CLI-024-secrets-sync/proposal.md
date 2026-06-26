---
id: "CLI-024-secrets-sync"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#612"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, sync, cli, go, ci, github, bitwarden, age]
template_version: "1.0"
---

# CLI-024-secrets-sync

## Why

<!-- from issue #612: secrets lifecycle: fail-loud hardening + reproducible CLI write/migrate/rotate (audit backlog) -->

Headless consumers (CI runners, containers, agents) cannot resolve secrets at runtime —
there is no interactive `bw unlock` — so ADR-028 specifies a `sync` step that materializes
a scoped secret set **ahead of time**. Today only the CI path exists and it is shell:
`scripts/github-secrets-manager.sh` consumes `dotf secrets ls --pairs` (which emits
`VAR<TAB>age-source`, **age only**), decrypts each age file itself, and uploads via
`gh secret set`. The instant a `ci:*` secret flips to `backend: bw` (the ADR-028 cutover,
C4 shipped in #627), `ls --pairs` excludes it (no age source to emit) and it **silently
disappears** from the Actions upload. Migrating any `ci:*` secret is therefore gated on
making this materialization backend-agnostic — this slice is what unblocks the rest of
#612 Phase C. ADR-029 ratifies the design (`dotf secrets sync <target>`, slice 1 = `ci`).

## What

A new Go command **`dotf secrets sync ci [--repo OWNER/REPO] [--dry-run]`** that pushes a
scoped, backend-agnostic secret set to a GitHub repository's Actions secrets:

1. **Consumer routing — `ci:<owner>/<repo>`.** The `consumers:` tag carries the target
   repo (DECIDED 2026-06-26): an entry feeds repo X's CI iff its `consumers` contains
   `ci:<owner>/<repo>` (e.g. `ci:mlorentedev/dotfiles`). `gh secret set` is per-repo, so
   the repo — not a workflow "purpose" — is the routing key. This replaces the current
   purpose tags (`ci:release`, `ci:bitacora`, `ci:image-push`); see Risks for the
   migration of existing entries.
2. **Select.** `--repo` defaults to the current repo's origin slug
   (`initrepo.OriginRepo`, validated by `initrepo.ValidRepoSlug` — the same resolution
   `spec init` uses); selection = every entry whose `consumers` contains `ci:<--repo>`.
3. **Resolve backend-agnostically.** Each selected entry's value is resolved through the
   existing `secrets.Loader.EnvFor` (`cli/internal/secrets/resolve.go`), which dispatches
   per-entry through the age/bw resolver map. The CI boundary never branches on backend
   and never asks for an age *source* — only the *value*. This is the leak ADR-029 removes.
4. **Upload behind a mockable seam.** A new `GitHubSecretSetter` interface (mirror of the
   `BWReader`/`BWWriter` pattern) with `SetSecret(repo, name, value string) error`:
   production shells out to `gh secret set <name> --repo <repo>` with the value on **stdin**
   (no argv leak, no temp file — C4); a package-level seam var (`ghSecretSetter`, like
   `bwReader`) lets command tests inject a fake with no `gh`, no network, no secrets.
   **The Actions secret name is the exposed env var name, 1:1** — the flat
   consumer-boundary convention (`${{ secrets.<VAR> }}` reads exactly the registry's
   `expose.env` name). This is deliberate: name-matching belongs at the *consumer*
   boundary, while the *storage* layout stays decoupled (the `bw:` block groups by
   service/account + purpose, never mirrored from the VAR — ADR-028's A1 model). The
   registry is the auditable indirection between the two; the VAR↔secret match is not a
   naming coincidence but the contract `sync ci` enforces.
5. **Exclude with a specific reason** (mirroring `migrate`'s guard vocabulary): file
   secrets (not Actions secrets), `floor`/`age-offline` secrets, and `GITHUB_*`-prefixed
   vars (Actions reserves the prefix — they cannot be created as repo secrets). Each
   exclusion is reported, never silent.
6. **Idempotent + `--dry-run` (C5).** `gh secret set` overwrites, so re-running is a no-op
   in effect. `--dry-run` reports the intended `VAR → repo` set (names + byte lengths,
   **never values**) and performs no upload.
7. **Retire the shell twin (ADR-020 strangler-fig), parity-gated.** Once `sync ci`'s
   selected set matches the legacy script's uploaded set for this repo, delete
   `scripts/github-secrets-manager.sh`, `tests/github-secrets-manager.bats`, the
   `dotf secrets ls --pairs` flag (the backend leak), and its test
   `TestSecretsLs_Pairs_EnvOnly`.

After this PR `sync ci` works **before, during, and after** a `ci:*` secret's age→bw flip,
so `migrate <ci-var>` no longer drops it — the `migrateGuard`'s `ci:*` refusal
(`cli/internal/cmd/secrets_migrate.go`) can then be lifted.

## Out of scope

- **`sync container` / `sync agent`** — `0600` `.env` materialization for non-CI headless
  consumers; designed in ADR-029, deliberately deferred (file-at-rest concerns differ).
- **The legacy `.env`-file mode** of `github-secrets-manager.sh` (reading `.env`/`.env.local`
  and the `SSH_PRIVATE_KEY_BASE64 → SSH_PRIVATE_KEY` decode special-case) — `sync ci` is
  registry-driven only. Confirm no `ci:*` registry secret relies on the base64-SSH decode
  before deleting the script (see Risks); if one does, model it in the registry, not a
  command special-case.
- **The `github.token` 1→2 split** (#321 / #612 C9, `migrate --split`) — distinct PATs per
  purpose is its own slice; `sync` just pushes whatever the registry resolves.
- **`retire` / `backup` / rotation** (#612 C6/C7).
- **Bulk `sync ci --all-repos`** — one `--repo` per invocation here; a multi-repo loop is a
  follow-up.

## Risks / open questions

- **Consumer-tag migration is a registry edit with cross-repo reach.** This slice migrates
  **dotfiles' own** CI tags (`ci:release`, `ci:bitacora`, `ci:image-push`) to
  `ci:mlorentedev/dotfiles` — the entries C5 actually unblocks and can verify. The
  remaining purpose tags belonging to *other* repos (`ci:payments`, `ci:newsletter`,
  `ci:social`, `ci:publish`, `ci:yt-metrics`) are swept to `ci:<owner>/<repo>` **with the
  operator confirming each owner/repo** — tracked as an explicit task here, never silently
  dropped (a half-migrated registry is tolerated by the parser and by `migrateGuard`'s
  `HasPrefix(c, "ci:")` check, so the sweep can land incrementally without breakage).
- **`GITHUB_*` vars cannot become Actions secrets.** GitHub reserves the prefix; the
  legacy script skips them. `sync ci` must exclude them with a clear message — and the real
  fix for any such CI need is to rename the var at the workflow, not push it.
- **Live smoke is Windows-empirical → deferred (C6).** `sync ci` against bw needs
  `bw unlock` + `gh auth`; that end-to-end run is the canary (#612 C8), not CI. The command
  core is fully unit-testable on Linux by mocking the resolver seams (`Decryptor`/`BWReader`
  already injectable) and the new `GitHubSecretSetter`.
- **Parity before deletion (the interlock).** Do not delete `github-secrets-manager.sh` /
  `ls --pairs` until `sync ci --dry-run`'s VAR set for dotfiles equals the script's current
  uploaded set. Gate the deletion on that one-time check.
- **Value delivery.** The value reaches `gh` via stdin (base64 not required for env values),
  never an argv element or a temp file — the plaintext-at-rest minimization of C4.

## Acceptance criteria

- [ ] **AC1 — selects + uploads the repo's CI set.** Given registry entries tagged
  `ci:OWNER/REPO`, `sync ci --repo OWNER/REPO` resolves each value via `Loader.EnvFor` and
  calls the setter once per var with `(repo, VAR, value)`; entries tagged for *other* repos
  are not selected. *Verify:* Go test with a fake resolver + fake `GitHubSecretSetter`;
  assert the exact `(repo, name, value)` calls.
- [ ] **AC2 — backend-agnostic (the unblock).** An age-backed and a bw-backed entry both
  tagged for the repo upload identically; flipping one age→bw does not change the uploaded
  set. *Verify:* Go test with one age + one bw entry through fakes; assert both reach the
  setter.
- [ ] **AC3 — exclusions are specific + silent-free.** A file secret, a `floor`/`age-offline`
  secret, and a `GITHUB_*`-prefixed var each are skipped with a distinct reported reason and
  produce no setter call. *Verify:* Go test per exclusion.
- [ ] **AC4 — `--dry-run` inert + non-leaking.** Reports `VAR → repo` with byte lengths,
  never the value, and makes zero setter calls. *Verify:* Go test asserts the fake setter
  was never called and the output contains no secret value.
- [ ] **AC5 — `--repo` defaults to origin; bad slug errors.** With no `--repo`, the target
  is the current repo's origin slug; an invalid `owner/name` fails loud with a clear message.
  *Verify:* Go test (injected repo-root fixture / invalid slug).
- [ ] **AC6 — parity gate before retirement.** `sync ci --dry-run`'s selected VAR set for
  dotfiles equals the legacy script's uploaded set; only then are the script, its bats,
  `ls --pairs`, and `TestSecretsLs_Pairs_EnvOnly` removed. *Verify:* documented parity check
  in `verification.md` + the deletions present in the diff.
- [ ] **AC7 — clean + additive.** `cd cli && go test ./... && go vet ./... && golangci-lint
  run --exclude-use-default=false ./internal/...` all clean; the diff is the new command +
  the seam + the registry tag migration + the retirements — no unrelated behaviour change.

## References

- Bitácora board: `mlorentedev/dotfiles#612` (Phase C, item C5).
- ADR: `docs/adr/adr-029-secrets-sync-headless-materialization.md` (the design this implements);
  `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` (the `sync` step is specified there);
  `docs/adr/adr-020-tooling-cli-go-convergence.md` (strangler-fig: absorb the shell twin).
- Reuse: `cli/internal/secrets/resolve.go` (`Loader.EnvFor`, the resolver map),
  `cli/internal/secrets/bw.go` (the `BWReader`/`BWWriter` seam this mirrors),
  `cli/internal/initrepo/github.go` (`OriginRepo`, `ValidRepoSlug`).
- Retires: `scripts/github-secrets-manager.sh`, `tests/github-secrets-manager.bats`,
  `dotf secrets ls --pairs` + `TestSecretsLs_Pairs_EnvOnly`.
- Sequencing: lifts the `ci:*` refusal in `cli/internal/cmd/secrets_migrate.go`'s
  `migrateGuard` once landed; `sync container`/`sync agent` are the deferred ADR-029 targets.
