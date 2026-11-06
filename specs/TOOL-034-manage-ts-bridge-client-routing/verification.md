---
tags: [spec, verification, templates]
created: "2026-11-05"
id: "TOOL-034-manage-ts-bridge-client-routing"
type: spec-verification
status: active
owner: manu
---

# Verification - TOOL-034-manage-ts-bridge-client-routing

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criteria 1-3 -> PR 1 generic archive source and rollout-safety tests
- [x] Criteria 4-6 -> PR 2 role/catalog/drop-in tests
- [x] Criteria 7-8 -> PR 3 doctor and live consequence evidence

## Test status

- PR 1: focused archive/catalog/role tests -> PASS; affected package tests and vet -> PASS; changed-file `gofmt -l` -> clean.
- Independent code review: all three Medium findings applied.
  - Final placement now writes/syncs/chmods a sibling temp file and renames it over the destination.
  - Unsupported archive extensions are refused by both Plan and Install before download/extraction.
  - Catalog load requires asset/member resolution parity for every supported GOOS/GOARCH pair.
- Rollout review: a PR 1 binary would recognize `github-archive` but silently ignore a later `roles` field. The generic role schema and no-argument install/converge filters now land in PR 1, before PR 2 publishes the ts-bridge entry; explicit named install remains an override.
- PR 2: full `go test ./internal/env ./internal/tools ./internal/converge` -> PASS; `go test -timeout 20m ./internal/cmd` -> PASS in 582.381s; exact f3/f4 selectors and focused vet -> PASS.
- PR 2 live install: the release checksum matched, all six archive member paths were inventoried, and the managed Windows install placed ts-bridge v1.20.0 successfully.
- PR 3 focused tests: `go test ./internal/doctor -run '^TestCheckTsBridge' -count=1` -> PASS in 4.573s; `go test ./internal/tools -run '^TestInstallGitHubArchiveSecondRunSkipsWithoutFetch$' -count=1` -> PASS in 2.594s; focused vet -> PASS.
- PR 3 live doctor: pinned ts-bridge v1.20.0 -> PASS; listener -> PASS; TLS-valid Gitea destination through SOCKS -> PASS.
- Live routing: global Gitea URL proxy resolved to `socks5h://127.0.0.1:1081`; GitHub resolved direct; Copilot App's bundled Git completed `ls-remote` against both origins.
- Copilot App recovery: an origin cached as `github_owner/repo` while Gitea was unreachable remained rejected after Git recovered. Removing and re-adding the project created neutral forge metadata and workspace initialization succeeded.
- Runtime lifecycle remains operator-controlled and is tracked by TOOL-035/#3244. The running bridge later proved why: its SSH bootstrap exited and the listener disappeared rather than healing itself.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- ADR-058 selects a generic archive source rather than a ts-bridge-specific downloader.
- Tool role filtering lands before the first role-bound archive entry: pre-PR-1 binaries skip the unknown source, while PR-1-and-newer binaries understand and enforce `roles`.
- Git routing uses an entirely dotf-owned include file, not writes into the user/gh-owned `~/.gitconfig`.
- PR 1 streams only the exact member into a temp file after the release checksum passes; no archive path is expanded onto disk.
- PR 2 filters role-bound tools for no-argument install/converge while an explicit named install remains an operator override.
- The proxy is a complete dotf-owned drop-in. A foreign file at that path is refused, not overwritten or removed.
- Catalog tests assert the observed member path in every published OS/architecture asset; synthetic archives alone do not prove release layout.
- Doctor treats listener-down as an explicit warning and listener-ready/destination-failed as a failure; readiness alone is not destination health.
- Health response bodies are parsed only for status and never printed, so profile or credential-adjacent fields cannot enter the report.
- Copilot App project metadata is repaired through remove/re-add, never by writing its internal database.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: `docs/lessons/lesson-565-test-exact-archive-members-against-the-release.md`
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: `docs/adr/adr-058-role-gated-userspace-bridge-client.md`
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: existing setup-idempotence and per-machine-override patterns cover the reusable method.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-034-manage-ts-bridge-client-routing/` -> `specs/archive/TOOL-034-manage-ts-bridge-client-routing/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
