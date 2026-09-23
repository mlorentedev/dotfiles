---
tags: [spec, verification, templates]
created: "2026-09-22"
---

# Verification - CLI-080-secrets-reconcile

## Evidence

| AC | Proof |
|---|---|
| AC1 | `TestReconcilePlanWritesNothing` — zero writes, every operation printed, sync observed |
| AC2 | `TestPlanOrdersOperationsByDependency`, `TestApplyReconcileConvergesAndASecondPlanIsEmpty`, `TestApplyReconcileAddsAFieldToAnExistingItem` |
| AC3 | `TestApplyReconcileNeverPutsAValueInItsOutput`, `TestReconcileApplyConvergesAndPrintsNoValue` — planted values, asserted absent |
| AC4 | `TestPlanNeverOverwritesAndReportsASatisfiedFrom` |
| AC5 | `TestPlanBlocksWhatItCannotSource` (5 cases), `TestPlanDefersADormantItemWithNoSource`, `TestApplyReconcileRefusesABlockedPlan`, `TestApplyReconcileRefusesAnEmptySource` |
| AC6 | `TestReconcilePlanWritesNothing` (sync observed), `TestReconcileApplyFailsWhenTheStoreDoesNotConverge` |
| AC7 | `TestRegistryRejectsAMalformedFrom` (5 cases), `TestRegistryValidatesFromOnADormantBlock`, `TestBWDeclarationsCarryFromForFileExposedSecrets` |
| AC8 | `TestSetItemFolderChangesOnlyTheFolder`, `TestSetItemFolderEmptyUnfiles`, `TestBWServeWriter_MoveItem_MatchesBWPutShape` |
| AC9 | Live, 2026-09-22, operator-authorized — see *Live apply* below |
| AC10 | `reconcile_retire_test.go` (11 tests) — pure core, planner, apply against the daemon fake, parity; live below |

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` — exit 0
- `golangci-lint run` at the pinned v2.12.2 — 0 issues
- `features.json` f1–f8 executed — 8/8 pass
- **Live plan, read-only, 2026-09-22**, this branch's registry against the real store:

  ```
  ~ move-item      dockerhub                (no folder) -> Dotfiles/apps
  + create-item    github-cli-pat           field "GITHUB_PERSONAL_ACCESS_TOKEN" in Dotfiles/apps, copied from GitHub/"Personal Access Token"
  + create-item    github-release-pat       field "RELEASE_TOKEN" in Dotfiles/apps, copied from GitHub/"release-token"
  - deferred       zoho                     … -> dotf secrets migrate ZOHO_APP_PASSWORDS
  - deferred       zoho                     … -> dotf secrets migrate ZOHO_RECOVERY_CODE

  Plan: 3 to apply, 0 blocked, 2 deferred.
  ```

### Mutation

16 mutations; 15 killed, 1 equivalent.

| Area | Mutation | Result |
|---|---|---|
| planner | create an absent shared item twice | killed |
| planner | block a dormant declaration instead of deferring | killed |
| planner | accept an ambiguous source | killed |
| planner | drop the dependency sort | **survived first**, killed after the fixture's names were chosen against alphabetical order |
| planner | report any unseen `from:` as satisfied | **equivalent** — see below |
| registry | accept `from:` on a multi-var secret | killed |
| registry | drop `from:` from a file-exposed declaration | **survived first**, killed by `TestBWDeclarationsCarryFromForFileExposedSecrets` |
| apply | resolve a folder twice per run | killed (against a fake whose folder listing is stale until sync) |
| apply | accept an empty source | killed |
| apply | apply a blocked plan | killed |
| apply | move-item as a no-op | killed |
| apply | add-field as a no-op | **survived first**, killed by `TestApplyReconcileAddsAFieldToAnExistingItem` |
| seam | unfile with `""` instead of null | killed |
| seam | daemon MoveItem as a no-op | killed |
| registry write | keep the age line on activation | killed |
| drift→plan | (covered by #1600's three) | — |

**The equivalent mutant.** `satisfied()` re-checks that the destination field exists.
Every `from:` still unseen after the findings are processed has it by construction —
an absent item or field would have produced a finding. The check stays because it
states the definition directly rather than by inference from `LayoutDrift`'s
completeness; if drift ever stops reporting a case, this is the line that keeps
reconcile from calling it satisfied.

## Live apply (AC9)

Run from this branch before merge, with the operator's authorization:

```
applied  move-item      dockerhub
applied  create-item    github-cli-pat
applied  create-item    github-release-pat

Converged: 3 operation(s) applied, and a second plan is empty.
note: bw.from on GITHUB_PERSONAL_ACCESS_TOKEN is satisfied; it can be removed from the registry
note: bw.from on RELEASE_TOKEN is satisfied; it can be removed from the registry
```

- **Independent re-plan:** `Plan: 0 to apply, 0 blocked, 2 deferred` (the deferrals are
  `zoho`, which is `migrate`'s).
- **`drift`:** 4 findings → 1 (`zoho`, dormant). 187 items, 23 governed.
- **By consequence:** `GET https://api.github.com/user` with each token, injected through
  `dotf secrets run` from the new items → **HTTP 200** for both. The age blob they used
  to read answered 401.
- **CI consumer:** `dotf secrets sync ci --repo mlorentedev/dotfiles RELEASE_TOKEN` →
  the repo secret moved from 2026-06-27 to 2026-09-22T23:44. `release-please` re-run:
  the error changed from `Bad credentials` to the account's GraphQL rate limit, so the
  token now authenticates. The green re-run after the quota resets is recorded on the PR.

### Second apply: retiring the sources (operator-authorized)

After the operator reported the resulting duplication — searching "release" returned
both `GitHub/release-token` and `github-release-pat` — both `from:` records gained
`retire: true`:

```
- retire-source  GitHub   remove GitHub/"Personal Access Token", once verified equal to github-cli-pat/"GITHUB_PERSONAL_ACCESS_TOKEN"
- retire-source  GitHub   remove GitHub/"release-token", once verified equal to github-release-pat/"RELEASE_TOKEN"
Converged: 2 operation(s) applied, and a second plan is empty.
```

Both tokens still HTTP 200 afterwards, from their new items. The satisfied `from:`
records were then deleted from the registry; a re-plan against the clean registry is
`Plan: 0 to apply, 0 blocked, 2 deferred` with no pending notes.

Retire mutations: 6, all killed against a tree that builds (lesson 284) —
drop the equality check, retire without the flag, accept an ambiguous source, accept
an absent field, remove nothing, run retire first. The last survived until
`TestPlanPutsRetireAfterEveryOtherOperation` pinned why retire runs last.

### Found during the live apply: `sync ci` could not be scoped

A full `sync ci --dry-run` would have **created** `HIVE_WORKER_API_KEY` in this repo:
`NAN_API_KEY`'s entry exposes two vars and `consumers` is declared per entry, so every
var goes to every CI consumer. Pushing it would have put a second copy of the credential
where no workflow reads it. Two responses:

- **In this PR:** `sync ci [SECRET_NAME...]` scopes the upload to the named GitHub
  secrets (env vars, not registry ids, since an id would carry both vars). A name
  outside the selection fails before any upload. `TestSecretsSyncCi_ScopedToNamedSecrets`,
  `TestSecretsSyncCi_UnknownScopedNameFailsBeforeUpload`.
- **Ticketed:** the model itself, #1603. A bare `sync ci` still over-uploads.

## Decisions made during implementation

- **Findings carry their declaration.** Reconcile maps drift's findings to operations
  rather than re-deriving the comparison, so the two commands cannot disagree about
  what has drifted.
- **Two-phase apply.** Every source is read before the first write, so a bad source
  costs nothing — not even the folder the plan would have created first.
- **Folders resolved once per run.** `ResolveFolder` creates on miss and the daemon
  lists from a cache; resolving a folder made moments earlier could duplicate it.
- **The idempotence check is the command's, not only the tests'.** `--apply` syncs,
  re-plans and fails unless the second plan is empty.
- **The bw serve fake was made more faithful.** It answered listings with id+name
  only, gave every created object the same id, and stored created items without
  one — each a way for a test to pass against a daemon that does not exist.
- **Deferred, not blocked, for dormant declarations.** Otherwise the pending `zoho`
  migration would have stopped every other operation from ever applying.

## Adversarial review, round 1 — disposition

`review.md` round 1: **FAIL**, `agy/gemini-3.1-pro-high`, reviewed `f16fb2e`, committed
verbatim. #1600 merged before this disposition was written. The fixes land in the
follow-up PR on `fix/secrets-reconcile-review`, and round 2 reviews that state.

| Severity | Finding | Disposition |
|---|---|---|
| Blocker (REAL) | A declaration that copies and retires fails the first `--apply`'s own convergence check | **Applied.** `--apply` runs up to `maxApplyPasses = 2` passes (`applyUntilConverged`). A retire is planned only once its copy exists, and it compares the copy as the store holds it after a sync, so a pass can unlock a retire but never perform one. That invariant is kept: the retire runs in the second pass, with the full two-phase protocol. Anything other than a retire left after a pass fails where it appears, so a store that ignores writes, or a flap, cannot earn another pass. The bound is in the loop header (lesson 286). `TestReconcileApplyCopiesAndRetiresInOneRun` (red before the fix) and `TestReconcileSecondPassIsOnlyForRetires`. AC10 amended. |
| Minor (THEORETICAL → REAL) | `ResolveFolder` creates a folder without syncing | **Applied.** Graded REAL here: the daemon's folder list does not show a folder created since its last sync, so the next resolve of the same name (the next `dotf secrets set` into it) created a **second folder with the same name**. The test double already modelled that cache (`staleFolders`), and the new subtest was red with two `Dotfiles/infra` folders. The create now calls `syncAfterWrite`, the same rule as every other write. |

Mutations, on trees that build, each under a memory-capped scope: remove the retire-only
condition, cap passes at 1, make `onlyRetires` always true, drop the folder sync.
**4/4 killed.** (The first of these, run against the earlier body-bounded loop,
never terminated. It is the cause of lesson 286.)

## Promotion candidates

- [ ] Pattern for `00_meta/patterns/`: "declare a data migration as a record the tool
      applies and then reports satisfied" (Terraform `moved`, applied to credentials).
      Candidate if a second store adopts it.
- [ ] ADR? No — this implements ADR-028 §2/§6 and records its decisions here.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-080-secrets-reconcile/`
- [ ] Independent adversarial review passed (reviewer != implementer)
