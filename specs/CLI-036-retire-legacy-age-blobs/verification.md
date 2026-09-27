---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - CLI-036-retire-legacy-age-blobs

## Evidence

- [x] AC1 -> `TestCommittedAgeBlobsAreClaimed` (`cli/internal/secrets/committed_blobs_test.go`): red with exactly 31 names, green after `git rm` of the 31; `git ls-files 'sensitive/*.secret.age'` prints only `sensitive/id_ed25519.secret.age`.
- [x] AC2 -> `TestCheckSecrets_BwBackedEntriesAreNotAgeAsserted`: the unclaimed blob is a FAIL named `orphan: chatgpt.api-key.secret.age`; the output carries no `#971`.
- [x] AC3 -> `TestCheckSecrets_FixPrunesUnclaimedMirrorBlob`: without `--fix`, 2 FAILs and no disk change; with it, 0 FAILs and 2 `[FIX ]` lines; the second run prints no `[FIX ]`.
- [x] AC4 -> `TestCheckSecrets_FixRefuses`, four rows (blob still in the checkout, no escrow, no registry, mirror == checkout). Case (d), the files that are not blobs, is asserted in the AC3 test: `id_ed25519.secret.age`, `env-mapping.conf`, `README.md` and `dr/` survive `--fix`. Mutation: with the escrow gate dropped, the "checkout without the DR escrow" row fails; restored.
- [x] AC5 -> ADR-028 amendment (2026-09-26). `reportUnreferencedBlobs` no longer exists; its replacement `pruneOrReportOrphans` carries the corrected comment.
- [ ] AC6 -> pending: `go run ./cmd/dotf doctor --fix` on msi from the worktree, then the owner runs `dotf secrets drift`.

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...`: green except `TestEmbeddedTemplatesMatchVault` and `TestIDPatternProseMatchesCode`, which compare against the local vault. #1769 (open) patched the vault ahead of its merge, so they are red in every worktree without it, this change included; they pass once #1769 lands.
- `golangci-lint run` (pinned 2.12.2): 0 issues.
- `shellcheck`, `bash -n`, `zsh -n` on `scripts/bitacora-rollout.sh`: clean.
- bats: see the PR body.

## Decisions made during implementation

- The prune lives in `dotf doctor --fix`, not in setup: the mirror must keep `id_ed25519.secret.age` (the resolver reads `SecretsDir/<file>.secret.age`), so the deploy cannot simply stop copying `sensitive/`; and #802 decided that doctor prunes and setup only copies.
- Three consumers of the retired blobs had to move with them: `scripts/bitacora-rollout.sh` decrypted `github.bitacora.secret.age` as a fallback (now it requires `dotf secrets run --only BITACORA_PAT`, with a bats test); the Windows CI job planted a dummy `nan.api-key.secret.age` (removed: `NAN_API_KEY` is bw-backed, and the blob would now be a FAIL); and the NaN, pi and `sensitive/` READMEs told the reader to create or expect a blob.
- `docs/runbooks/secrets-management.md` gets a corrected banner only: its full rewrite is #600, and its USB section is rewritten by #1770.
- The shell-era partial write `*.secret.age.tmp.*` counts as an orphan, so the prune removes it too.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson: yes, lesson-308 (a comment that grants a safety exemption outlives the ADR it cites).
- [x] ADR: yes, the ADR-028 amendment (2026-09-26).
- [x] Pattern: no; the rule is specific to this repo's secrets layout.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-036-retire-legacy-age-blobs/` -> `specs/archive/CLI-036-retire-legacy-age-blobs/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
