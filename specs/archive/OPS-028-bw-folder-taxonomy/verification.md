---
tags: [spec, verification, templates]
created: "2026-08-14"
---

# Verification - OPS-028-bw-folder-taxonomy

## Evidence

- [x] AC1 (schema + validation) -> `TestBWSource_Folder`, `TestBWSource_Folder_NoneDeclared`, `TestParseRegistry_BwFolder_RejectsUnratified`
- [x] AC2 (writer places item in declared folder) -> `TestNewItemBody_Folder`, `TestBWFolderResolver_EmptyNameIsNoop`, `TestSecretsSet_CreateAbsent_UsesDeclaredFolder`, `TestSecretsMigrate_UsesDeclaredFolder`, `TestSecretsSet_DryRun_NeverResolvesFolder`
- [x] AC3 (registry populated) -> `secrets/registry.yaml`: 21 `plane: app` entries carry `folder: Dotfiles/apps`, 5 `plane: infra` entries carry `folder: Dotfiles/infra` (script-verified count, `dotf secrets ls` confirms the registry still parses)
- [x] AC4 (move `openai-api-key`, verify unchanged) -> folder `Dotfiles/apps` created (id `5f1985f7-9d84-45c1-bd18-b4a60012a18f`), item `openai-api-key` (id `c028dd20-6b07-4d2f-9db5-b4a50032c202`) moved via `bw edit item` touching only `folderId`; SHA-256 of the `api-key` field value identical before/after (`fa7146f04fe351c914bd3291884f31efb43dd15ffb2364429c5457f52bdf7938`); `dotf secrets verify OPENAI_API_KEY` -> `OK ... bw` (run against `feat/secrets-bw-migration`, the branch holding the canary's `bw` flip)
- [x] AC5 (no regression) -> `TestSecretsSet_CreateAbsent_NoFolderDeclared`; full suite below

## Test status

- Test suite: `cd cli && go build ./... && go vet ./... && go test ./... -count=1` -> all packages `ok`, zero failures. `-count=1` is load-bearing: a plain `go test ./...` gave a false all-green once during this spec (stale build-cache hit against `TestSetBackendBW_RealRegistry_OnlyTargetChanges`, which reads `registry.yaml` at runtime — a file the cache doesn't track), caught only by the adversarial review re-running with `-count=1`.
- Lint: `golangci-lint run ./...` (pinned v2.12.2 per `versions.conf`) -> `0 issues`
- Manual smoke test: folder created, item moved, and re-verified live against the real Bitwarden vault (see AC4 above) — no longer fake-only.
- No regressions in existing test suite: yes — full `go test ./... -count=1` green before and after, `TestSecretsSet_CreateAbsent_NoFolderDeclared` pins the unfoldered path unchanged

## Adversarial review

Two passes, both `nan/deepseek-v4-flash` via `pi` (the reviewer-pool primary, HARNESS-071/#955 — Anthropic models are disqualified as reviewers in this repo per that gate, since Claude implements nearly every change including this one):

1. **`reviewed_sha c13a8b0`** -> **PASS-WITH-GAPS**. Confirmed both findings from an earlier, informal same-model-as-implementer pass (discarded, not used to gate anything) were genuinely fixed: dormant `bw.folder` validation, and the stale golden-string test. Found two NEW Minor/THEORETICAL findings of its own: a TOCTOU race in `ResolveFolder` under concurrent callers, and no enforcement that a declared folder matches its secret's `plane` (an app-plane secret could declare `Dotfiles/infra` and pass).
2. **`reviewed_sha e5973a7`** (after fixing both) -> clean **PASS**. Verified the `planeFolder` check's ordering, its handling of planes with no required folder (personal/floor), and judged the TOCTOU documentation-only fix as sufficient for a single-operator CLI. Full findings in `review.md`.

## Decisions made during implementation

## Decisions made during implementation

- Folder resolution (`BWFolderResolver.ResolveFolder`) is a **separate interface** from `BWCreator.CreateItem`, not folded into it — `CreateItem` trusts an already-resolved `folderID` and never resolves a name itself. Keeps the JSON-body test (`TestNewItemBody_Folder`) free of any bw-CLI fake, and keeps `createAbsent` the one place a folder name turns into an id.
- `ResolveFolder` is called **after** the dry-run/confirm gate in `createAbsent`, not before — folder creation is itself a write (`bw create folder`), so `--dry-run` must never trigger it. Pinned by `TestSecretsSet_DryRun_NeverResolvesFolder`.
- Canonical casing resolved as `Dotfiles/apps` / `Dotfiles/infra` (matching ADR-028's own "Bitwarden folder taxonomy" section) over issue #951's own worked-example quote (`"dotfiles/apps"`, lowercase) — the latter is a transcription slip, not a second valid form. `ParseRegistry` rejects the lowercase form explicitly (test case in `TestParseRegistry_BwFolder_RejectsUnratified`).
- `plane: personal` entries (6, already carrying a dormant `bw:` block) were left with no `folder:` — ADR-028's ratified taxonomy has no `Dotfiles/personal`, and validation rejects it if declared. Deferred to #586, noted in `proposal.md` Out of scope.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? Yes — validate dormant/pre-declared config fields unconditionally, not gated on the state that activates them (the `bw.folder` gap only fired at migrate time, handing a typo straight to a folder-creating side effect).
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No — this implements ADR-028's already-ratified taxonomy, it doesn't decide anything new.
- [x] New pattern candidate for `00_meta/patterns/`? No — single occurrence in this repo so far; revisit if the dormant-validation gap recurs elsewhere.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/OPS-028-bw-folder-taxonomy/` -> `specs/archive/OPS-028-bw-folder-taxonomy/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
