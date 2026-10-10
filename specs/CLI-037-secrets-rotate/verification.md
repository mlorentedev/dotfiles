---
tags: [spec, verification, templates]
created: "2026-08-15"
---

# Verification - CLI-037-secrets-rotate

## Evidence

- [x] **AC1:** `TestRotate_WritesSyncsAndProvesTheChange` (sync called once, value written and re-read). #1003, `36448473`.
- [x] **AC2:** the same test asserts `rotated … ->` and that neither value appears; `TestRotate_FailsWhenTheReadPathStillServesTheOldValue`.
- [x] **AC3:** `TestRotate_RefusesANoOp`; `TestRotate_RefusesToCreate` for the provisioning boundary.
- [x] **AC4:** `TestRotate_ProbesTheNewValueAndFailsWhenItDoesNotAuthenticate` (added 2026-10-10; mutations "drop the probe call" and "swallow its error" both fail it).
- [x] **AC5:** `TestRotate_DryRunWritesNothing`, `TestRotate_DryRunPushCINamesTheReposAndUploadsNothing`.
- [x] **AC6:** #1007 (`fe2f1913`) put the write path on `bw serve`: `TestSelectBWBackend_ReadAndWriteAlwaysAgree`, `TestBWServeWriter_SetField_UpdatesAndPreserves`. #993 closed 2026-08-16.
- [x] **AC7:** `TestRotate_PushCIUploadsTheNewValueToEveryCIConsumer`, `TestRotate_WithoutPushCIUploadsNothing`, `TestRotate_PushCIWithNoCIConsumerSaysSo`, `TestRotate_PushCIRefusesAMalformedCIConsumerBeforeUploading`. Mutations: no push, dry-run pushes, push without the flag, every consumer read as CI; all killed.

## Test status

- `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...`: exit 0 on 2026-10-10.
- `golangci-lint run ./...` (2.12.2, the pin), after `cache clean`: 0 issues.
- The `secrets sync ci` tests pass unchanged after `pushCI` was extracted from it.
- Not run live: a real rotation with `--push-ci` uploads to GitHub. The seam is `ghSecretSetter`, the same one `sync ci` uses in production.

## Decisions made during implementation

- **`--push-ci` reuses `sync ci`'s upload half rather than shelling out to it or copying it.** `pushCI` now carries the skip reporting, resolve-before-upload and liveness gate for both commands.
- **The push skips its own liveness gate.** It runs only after the rotation was read back and, where declared, probed live; probing the same value twice adds a network call and no information.
- **Every `ci:` consumer is validated before the first upload** (`initrepo.ValidRepoSlug`, the guard `sync ci` applies to `--repo`), so a registry typo cannot leave some repos rotated and the rest on the old credential.
- **A secret with no CI consumer is a note, not an error,** because the rotation itself succeeded.
- **Every `ci:` consumer is pushed, not the current repo's only.** A rotated credential that stays old in one consuming repo is the half-rotation this command exists to remove.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the rotation lessons (fingerprint vs liveness, sync is load-bearing) are in the command's own doc comment and #996; nothing new was learned closing it.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-029 already names rotation as C7; this implements it.
- [x] New pattern candidate for `00_meta/patterns/`? no: a single-repo command.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-037-secrets-rotate/` -> `specs/archive/CLI-037-secrets-rotate/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
