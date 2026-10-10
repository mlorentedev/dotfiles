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
- [x] **AC7:** `TestRotate_PushCIUploadsTheNewValueToEveryCIConsumer`, `TestRotate_WithoutPushCIUploadsNothing`, `TestRotate_PushCIWithNoCIConsumerSaysSo`, `TestRotate_PushCIRefusesAnUnpushableCIConsumerBeforeRotating` (a malformed slug and a `GITHUB_*` var both fail before the vault is written), `TestRotate_PushCIFailureOnOneRepoStillPushesTheOthers`. Mutations: no push, dry-run pushes, push without the flag, every consumer read as CI, stop at the first failed repo, skip the unpushable check, validate after the write; all killed. *(Restated after the archive review: "push without the flag" means the two-sided mutation, which drops the flag guard and resolves the targets unconditionally. The one-sided mutation, which drops only `!pushToCI` from the guard, is not observable, because `repos` stays empty without the flag.)*

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

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-037-secrets-rotate/` -> `specs/archive/CLI-037-secrets-rotate/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

## Archive review dispositions (2026-10-10)

The archive review (`nan/qwen3.8-flash`, `PASS-WITH-GAPS`, reviewed `bade1483`) is committed unchanged with this PR.

| # | Finding | Disposition |
|---|---|---|
| 1 | `--push-ci` uploads every var of the secret to every `ci:` consumer | **Applied as an interim guard, root ticketed in #2306.** The premise was corrected: `NAN_API_KEY`'s two names are one credential (registry comment), so "a var it never rotated" does not hold. The effect is still real. Listing only names, `dotfiles`, `hive` and `kubelab` hold `NAN_API_KEY` and no `HIVE_WORKER_API_KEY`, so a push would create 13 copies that nothing reads. `ciTargets` now refuses a multi-var push before any write. Test: the `two vars` case of `TestRotate_PushCIRefusesAnUnpushableCIConsumerBeforeRotating`; the mutation was killed. `sync ci` without names shares the widening, which is the registry-model fix in #2306. |
| 2 | Two AC7 mutation claims overstate what the tests protect | **Applied.** `TestRotate_PushCIProbesTheNewValueOnce` pins one probe per rotation, and flipping `skipVerify` is killed. AC7's "push without the flag" is restated above as the two-sided mutation. |
| 3 | The empty-value refusal has no test | **Applied.** `TestRotate_RefusesAnEmptyValue` checks `setCalls == 0`; the mutation was killed. |
| 4 | The note for an unimplemented probe has no test | **Applied.** `TestRotate_ReportsAnUnimplementedProbeAsUnverified` uses a `validate: dockerhub` fixture; the mutation was killed. |
| 5 | `pushCI` zips `EnvFor` with the selection by position | **Applied.** `checkAligned` refuses a dropped or reordered resolution before any upload (`TestCheckAligned_RefusesAMisalignedResolution`). |
| 6 | A repeated `ci:` consumer uploads twice | **Declined.** `gh secret set` overwrites, so this is noise without corruption, and the registry holds no duplicate pair today. |
| Q | The push re-resolves through the read path instead of uploading the value that was just probed | **Answered.** This is safe because the reader, the writer and the syncer come from one pinned backend (`bwBackend()`), and the confirmation step has already proved that the read path serves the new value. Should `secretLoader()` ever resolve through a different backend from `bwRead()`, the push would no longer be covered by that proof. That case belongs with #2306's rework of `SelectCI`. |

