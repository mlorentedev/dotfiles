---
tags: [spec, verification, templates]
created: "2026-09-27"
---

# Verification - SEC-006-vault-curation

## Evidence

- [x] AC1 (dry run writes nothing, one line per row): `TestCurateDryRunThenApplyWithTheDigest` asserts zero writes and the `Plan:` summary. Live evidence: the dry run below ran against the real vault through the bw serve daemon, and made no write.
- [x] AC2 (every blocking rule, whole-plan refusal): `TestCurateBlocks` has 17 table cases. Each one asserts the row blocks, and that `ApplyCurate` then refuses with zero writes. Also covered by `TestApplyMutationRefusesAPasskeyItem` (every mutation kind), `TestCurateBlocksRegistryAndPasskeyItems` (through the command) and `TestCurateDropURIsLetsAPasskeyKeeperStandUntouched` (the only way past the keeper-passkey rule is a declared drop, which leads to a delete alone).
- [x] AC3 (apply, re-plan to done, second run empty): `TestCurateAppliesEveryOpAndConverges` covers all 8 ops. They apply, then re-plan as all `done` with zero further writes. `TestCurateDryRunThenApplyWithTheDigest` checks the same through the command.
- [x] AC4 (digest): `TestCurateDigestTracksRevisions` checks that the digest is deterministic, that editing an untouched item leaves it unchanged, and that editing a touched item changes it. `TestCurateApplyRefusesAStaleOrMissingDigest` checks that a missing digest and a stale one both refuse, with zero writes.
- [x] AC5 (no value in any output): `assertNoValue` runs over dry-run, apply and error-path output for the sentinel password, username, token-bearing URI and the fixture item names. `TestDecodeItemsCountsWhatCurateNeedsWithoutKeepingIt` plants TOTP, passkey and attachment key material and asserts that the projection keeps the counts only.
- [x] AC6 (passkey count): `TestCurateApplyFailsOnAPasskeyCountChange` makes a write drop a passkey elsewhere in the vault, and asserts `PASSKEY COUNT CHANGED: 3 before, 2 after`.

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...`: exit 0. `GOOS=windows go vet ./...`: ok. `golangci-lint run ./...` (pinned 2.12.2): 0 issues.
- Mutation check (rerun after `drop=uris`): each of 12 guards was disabled in turn, and every mutant was killed by a failing test. The guards were: the passkey check in `applyMutation`; the passkey check in the planner; the attachments check on a delete; the duplicate's passkeys, attachments and TOTP on a merge; the keeper-passkey rule; `drop=uris`; registry ownership; the revision in the digest; the `--digest` comparison; and the before/after passkey count. The first run found one survivor: no test covered a merge duplicate with attachments. `TestCurateBlocks` gained that case (17 cases now), and the mutant was killed on the rerun.
- Live dry run, 2026-09-27, against the owner's vault (read-only: sync and list), with the private plan (91 rows on the first run, 89 after the correction below):
  - First run: `Plan: 82 to apply, 0 done, 9 blocked.` All 9 blocks were real precondition failures, and none was a false positive:
    - a merge whose keeper carries a passkey and would have needed a URI carried to it;
    - a merge whose usernames differ;
    - five rows gated on revocations at their providers;
    - two edit rows on an item that had gone to the trash since the inventory.
  - After the plan was corrected (trashed item's rows removed; `drop=uris` on the passkey-keeper merge, whose extra URI is the `www.` host of the keeper's own URI, both at the default base-domain match): `Plan: 83 to apply, 0 done, 6 blocked.` The 6 blocks that remain are owner decisions: 5 gates, and one `alias` claim.
  - No output line carried a value, a URI or an item name.
- No regressions: the full `go test ./...` passes.

## Decisions made during implementation

- **`hide` and `reprompt` may touch registry-declared items.** The registry governs where an item lives and which fields it has, not how they are displayed. The live plan must hide two secret-named fields on a registry item.
- **Refusal instead of a measurement** for passkey items (see lesson 312). The measurement risked the unrecoverable loss it was meant to rule out.
- **`drop=uris` is a declared loss, not an exception to the passkey rule.** It was added after the first live dry run.
- **Id prefixes resolve through `resolvePrefix`**, not `resolve`, because `render.go` already owns that name.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-312-refuse-by-construction-what-you-cannot-measure-without-losing-it.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: curate applies ADR-028 (Bitwarden SSOT, reconcile's plan/apply shape) to the items the registry leaves out; it adds a command, not a new architectural decision
- [x] New pattern candidate for `00_meta/patterns/`? no: the saved-plan digest and the refuse-what-you-cannot-measure rule are recorded in lesson 312; they occur in one project so far

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SEC-006-vault-curation/` -> `specs/archive/SEC-006-vault-curation/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
