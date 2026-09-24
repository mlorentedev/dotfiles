---
tags: [spec, verification, templates]
created: "2026-09-21"
---

# Verification - CLI-078-secrets-layout-drift

## Evidence

| AC | Proof |
|---|---|
| AC1 | `TestLayoutDriftReportsAFolderNameNothingCarries` |
| AC2 | `TestLayoutDriftChecksDormantDeclarations` + `TestBWDeclarationsIncludesDormantBlocksOnAgeBackedSecrets` |
| AC3 | `TestLayoutDriftReportsAMisfiledItem` + `TestParseRegistry_BwFolder_RefusedOnAPlaneWithNoFolder` |
| AC4 | `TestLayoutDriftResolvesFieldsTheWayTheReaderDoes` (all four paths, both directions) |
| AC5 | `TestLayoutDriftReportsOneFindingPerProblemNotPerVar` + `TestLayoutDriftDedupesAMisfiledItemAcrossVars` |
| AC6 | `TestDecodeItemsCannotCarryAValue` |
| AC7 | `TestDecodeItemsErrorNeverQuotesTheBody` |
| AC8 | `TestDriftSourceCallsNoWriter` (the writer surface derived from `BWWriteClient`) + `TestDriftExitsNonZeroOnAFindingAndNamesTheSecret` |
| AC9 | `TestDriftSyncsBeforeItReads` + `TestDriftRefusesAStoreItCannotSync` + `TestDecodeItemsMarksAFolderIDTheListDoesNotCarry` + `TestLayoutDriftSaysSoWhenItCannotNameAnItemsFolder` + `TestPlanBlocksAnItemWhoseFolderItCannotName` |

## Test status

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` — clean
- `go test -count=1 ./...` — green across the module
- `golangci-lint run` at the pinned **v2.12.2** — 0 issues
- All nine `features.json` commands executed: **9/9 pass** (round 4)
- **Live run against the real store** (185 items, 27 declared targets): four
  findings, each independently confirmed — `dockerhub` unfoldered, and
  `github-cli-pat`, `github-release-pat`, `zoho` absent.

### Mutation

Five mutations, each killed:

| Mutation | Killed by |
|---|---|
| a password reaches `ItemSummary` | `TestDecodeItemsCannotCarryAValue` |
| skip dormant declarations | `TestBWDeclarationsIncludesDormantBlocksOnAgeBackedSecrets` |
| report a misfiled item per var | `TestLayoutDriftDedupesAMisfiledItemAcrossVars` |
| `hasField` forgets the login cases | `TestLayoutDriftResolvesFieldsTheWayTheReaderDoes` |
| drop the per-var field fallback | `TestBWDeclarationsResolvesPerVarFieldOverrides` |

**Three of those five survived the first round**, and the tests that kill them now
were written in response. The dormant-declaration hole is the one worth recording:
the original test handed `LayoutDrift` a struct with `Dormant: true` and passed
happily while `BWDeclarations` — the walk that has to *discover* dormancy — was
never exercised at all. A fixture built past the code under test proves nothing
about it.

## Decisions made during implementation

- **The projection is a type, not a rule.** `itemWire` cannot name a value, so
  `encoding/json` drops passwords during decoding. A redaction step would have
  been a line someone could delete; this is one nothing can reorder past.
- **The `Dotfiles/` prefix lives in the declaration, not in code.** Prepending it
  in `ResolveFolder` would mean the declared value differs from the real one, so
  the report would have to undo the translation to avoid lying.
- **Dormant declarations are checked.** They are the migration target, and three
  of them pointed at items that do not exist. Left unchecked they surface only
  during the migration that needs them.
- **`hasField` mirrors `fieldFromItem` deliberately**, against lesson 279's
  general rule, because the alternative puts a value-resolving path one refactor
  away from a value-free report. Named in the code and pinned by a test.
- **Read-only is structural.** No `--fix`, no writer passed in, and a
  committed check that greps the source (comments stripped) for any write call.

## Defects found after merge (2026-09-22, first run of the merged binary)

Found by running `drift` against the live store while designing `reconcile`,
not by the tests above: all three depend on the real shape of the store or the
registry, which the fixtures did not reproduce.

- **Unfoldered items read as filed in "No Folder".** `ListFolders` dropped the
  null-id pseudo-folder; the id index `ListItems` built did not, so `""` mapped
  to `No Folder`. Fixed by one `folderIndex` both paths share the rule of.
- **File-exposed secrets were never checked.** `BWDeclarations` walked
  `expose.env` only, so 8 secrets — `KUBECONFIG`, `SSH_KEY`, the recovery
  codes — contributed no declaration. AC2 held only for env secrets. Declared
  targets went from 27 across 16 items to 34 across 21 once fixed.
- **`folder: ""` was read as "must be unfoldered".** The first two defects
  masked it: fixing either alone would have reported every hand-filed personal
  item as misfiled. `""` now means placement is not governed, which is what the
  taxonomy says (app and infra only; personal deferred to #586).
- **`ItemSummary.Revised` was documented as an upper bound on age; it is a
  lower bound.** The value existed at or before the last edit. No code read it
  yet, so only the comment changed — but the rotation slice builds on it.

Each fix is pinned by a test, and each test was confirmed red with its fix
reverted (3 mutations, 3 killed).

## Promotion candidates

- [ ] Lesson for `docs/lessons/`? **Deferred, deliberately.** The candidate is
      "a fixture built past the code under test proves nothing about it", and it
      is a restatement of lesson 279's family rather than a new class. Recorded
      in this file instead; DoD §2 skip, stated.
- [ ] ADR-worthy? No — this implements ADR-028 §6, it does not amend it. The
      folder NAMES gained a prefix, which §6 left unspecified.
- [ ] Pattern for `00_meta/patterns/`? Not yet. "Project at the producer so the
      type system enforces redaction" is a strong candidate if it recurs in a
      second project.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved to `specs/archive/CLI-078-secrets-layout-drift/`
- [ ] Bitácora `#1596` closed with the PR link (ADR-018). **Deliberately not
      closed.** This spec and CLI-080 deliver the layout half of #1596. Its
      rotation-age and stale-CI problems are still open, so the issue was reopened
      on 2026-09-23 and the archive PR references it without closing it.
- [x] Independent adversarial review passed (reviewer != implementer): round 5,
      `nan/mimo-v2.5`, reviewed `0c10f0b`

## Adversarial review, round 1 — disposition

`review.md` round 1: **FAIL**, `nan/glm5.3-flash`, reviewed `c7a8adc`. Committed
verbatim; round 2 reviews this branch's head.

| Severity | Finding | Disposition |
|---|---|---|
| Blocker | File-exposed declarations never enter the walk (7 of 33) | **Applied** — the same defect found independently on the live run; fixed and pinned above |
| Major | Live canary resolves the stale literal `"apps"`, so a live run would create a stray folder | **Applied** — the canary derives its folder from `planeFolder["app"]`, so it cannot go stale again |
| Minor | `ListItems` keeps the null-id "No Folder" row | **Applied** — `folderIndex`, tested against the null-id shape bw serve emits |
| Minor | AC8's non-zero exit has no command-level test | **Applied** — `TestDriftExitsNonZeroOnAFindingAndNamesTheSecret`, confirmed red with the exit removed |
| Minor | `LayoutFinding.Secret` documented for the reader but never printed | **Applied** — each line ends `[<registry id>]`, confirmed red with it removed |
| Minor | `tasks.md` "PR opened" box unticked | **Applied** — ticked |

The reviewer found two of the three post-merge defects without a live vault
(the file-expose walk and the pseudo-folder); it did not find the third
(`folder: ""` read as "must be unfoldered"), which only shows once the other
two are fixed.

## Adversarial review, round 2 — disposition

`review.md` round 2: **FAIL**, `agy/gemini-3.1-pro-high` (a different provider
family from round 1), reviewed `c7b912b`. Committed verbatim.

| Severity | Finding | Disposition |
|---|---|---|
| Blocker (REAL) | Duplicate item names: the by-name index kept the last one, so a declared item could be judged by a same-named personal item | **Applied** — new finding `item-ambiguous`, raised instead of any judgement about either item, matching the reader's refusal. `TestLayoutDriftReportsAnAmbiguousItemInsteadOfJudgingEither` |
| Minor | `field-missing` not deduped across vars (AC5) | **Applied** — deduped per item and field. `TestLayoutDriftDedupesAMissingFieldAcrossVars` |
| Minor | An empty note or username reads as missing, while `fieldFromItem` returns `""` without error | **Declined, with reason** — an empty value is not a usable secret; `verify` would resolve it to nothing. Reporting it is the useful outcome. The detail says "does not carry it", which is accurate for a field with no value |

Mutation for this round: 2 mutations, 2 killed — **against a tree that builds**. The
first run of these two reported both killed while the package failed to compile,
which proves nothing (lesson 284).

## Adversarial review, round 3 — disposition

`review.md` round 3: **FAIL**, `nan/deepseek-v4-flash`, reviewed `f16fb2e`, committed
verbatim. #1600 merged before this disposition was written. The fixes land in the
follow-up PR on `fix/secrets-reconcile-review`, and round 4 reviews that state.

| Severity | Finding | Disposition |
|---|---|---|
| Major (REAL) | A declaration naming no field is counted and never checked (`hasField("")` true, `fieldFromItem("")` refuses) | **Applied.** It is now a `field-missing` finding whose detail says no field is declared, and `hasField("")` returns false, like the reader. `TestLayoutDriftReportsADeclarationThatNamesNoField`, red before the fix. **Live:** drift went from 1 finding to 2 (zoho + `AGE-SECRET-KEY-PERSONAL`), confirming the review's probe. The registry now declares `field: notes`, verified value-free (the item is a secure note: `fields=[] notes=true login=false`). Back to 1 finding, zoho, deferred; `verify` 35/35. The finding was chosen over a parse-time rule because drift reports what the reader refuses, and the one offender is fixed in data. |
| Major (THEORETICAL) | AC7 pinned on the outer-envelope error path only | **Applied.** `TestDecodeItemsErrorNeverQuotesTheBody` covers both paths. The review's mutation M13 (quote `inner.Data`) is now **killed**. |
| Minor (REAL) | AC4 "matches `fieldFromItem` exactly" is false for empty values, and its test never ran the reader | **Applied, with the operator-level call made explicit.** An empty value counts as missing where the projection can see it (notes, username), because an empty secret is not usable. `TestLayoutDriftAgreesWithTheReaderOnEveryItemShape` hands one raw item to both sides across 13 shapes and requires agreement. Two divergences are declared in the table (an empty password or custom-field value reads as present: the projection never decodes those values). AC4 is reworded to match. The review's surviving mutations (invert the notes dispatch, `HasNotes: true`) and a third (`HasUsername` ignoring empty) are all **killed**. |
| Minor (THEORETICAL) | Misfiled dedupe keyed on the item, so two declarations disagreeing on a folder yield one finding and reconcile flaps | **Applied at the source.** The registry refuses an item declared in two folders (`checkOneFolderPerItem`, registered only on success like `seenVar`), so the item-keyed dedupe cannot hide a disagreement. `TestRegistryRefusesTwoFoldersForOneItem`. The live registry parses clean. Independently, reconcile's second pass admits only retires, so a flap would fail rather than loop. |
| Minor (REAL) | `LayoutDrift` 71 lines, CC≈18 | **Applied.** Split into a `driftWalk` with one method per question (`checkFolder`, `resolveItem`, `checkPlacement`, `checkField`); `LayoutDrift` is 10 lines. Behaviour unchanged: the full suite passes unmodified. |
| Minor (REAL) | Comments assign `reconcile` to CLI-078; unmanaged counts unexplained; `docs/secrets-inventory.md` still `apps/<item>` | **Applied.** Both comments say CLI-080. The unmanaged count is restated as "164 of 187 on 2026-09-23, items whose name no declaration uses", with why the old "161 of 185" was wrong. The inventory uses `Dotfiles/apps/`, `Dotfiles/infra/`, and personal items name no folder (#586), with a legend line. |

Mutation for this round, **on trees that build, each under a memory-capped scope**
(lesson 286): 11 mutations, 11 killed. Four earlier sessions lost this battery to
the OOM killer, and the cause is lesson 286.

## Adversarial review, round 4 — disposition

`review.md` round 4: **FAIL**, `nan/qwen3.8-flash`, reviewed `4ef2d85`, committed
verbatim (`7c50182`). All four round-3 fixes held against its mutations. The fixes
below are on `chore/archive-secrets-specs`, and round 5 reviews that head. The
contract set moved once, in one commit: AC3, AC6 and AC8 amended, AC9 added, and
f3, f8 and f9 in `features.json`.

| Severity | Finding | Disposition |
|---|---|---|
| Major (REAL) | AC6's test plants no username, so a `Username` field on `ItemSummary` passes the suite | **Applied.** It plants a distinctive username and asserts it absent. The reviewer's compound mutation is now **killed** by `TestDecodeItemsCannotCarryAValue`. AC6's wording says what is planted: every member the projection reads, and the values it must drop. |
| Major (THEORETICAL) | An unresolved `folderId` reads as unfoldered, and `drift` never syncs although it is gated on | **Applied, both halves.** `ItemSummary.FolderUnresolved` marks a folder id the list does not carry; drift reports it as `item-folder-unknown`, never "(no folder)", and reconcile blocks on it with its own remedy instead of planning a move against a stale list. drift now syncs first, through `readInventory`, which reconcile shares; a store that cannot sync is refused. The lister and its syncer are one seam, so the cache synced is the cache read (`rotate` records what the other way costs). AC9 added. **Live:** drift synced and reported only the known, deferred `zoho` finding. |
| Major (THEORETICAL) | `floor`/`personal` may declare any ratified folder, and reconcile would move the item there | **Applied as a guard, not a documented permission.** A plane with no `planeFolder` entry has no legal folder: absence denies, and a plane gains a folder by being given one. The live registry still parses (none of its 7 floor/personal `bw:` blocks declares a folder). The comments on `validBWFolders` and `planeFolder`, and the runbook's ADD step 2, now say what `AGE_KEY_PERSONAL` shows: a floor secret may carry a `bw:` block, as a convenience copy with no placement. AC3 amended. |
| Minor (REAL) | AC8's grep lists four writer methods; `MoveItem` and `RemoveField` pass it | **Applied**, as question 1 suggests: the guarantee is now a Go test. `TestDriftSourceCallsNoWriter` derives the forbidden names from `BWWriteClient` by reflection, walks drift's AST (comments are not in it), proves its scanner finds a planted `bwWrite().MoveItem`, and asserts the inventory seam has no writer method and drift has no `--fix` or `--apply`. f8 runs it. |
| Minor (REAL) | The `field` half of the missing-field dedupe key is unpinned | **Applied.** Two distinct fields missing from one item are two findings, in `TestLayoutDriftReportsOneFindingPerProblemNotPerVar`. |
| Minor (REAL) | The derived ratified-folder message is unpinned | **Applied.** `TestParseRegistry_BwFolder_RejectsUnratified` asserts it names both folders. |
| Minor (REAL) | `docs/secrets-inventory.md` has drifted from the registry | **Applied, by removing the copy rather than correcting it.** The drift was wider than measured: besides the three names reported, rows 4, 19 and 22–26 also named items the registry does not use. The column now names the registry id, and the registry alone says where each lives. The "migrate TO bw" section is a dated record of what was done. Generating the table from `BWDeclarations()` is declined: with no item repeated, nothing is left to generate. |
| Minor (REAL) | Two comments contradict the code | **Applied.** `bwLister`'s says the inventory is read from the daemon directly, and why the pinned backend has no lister half: the projection has one implementation to audit. drift's says three items were missing, not two. |

Questions, answered:

1. *Is a source grep the right home for AC8?* No, and it is gone (finding 4).
2. *`ItemSummary.Revised` is never read.* It is substrate for #1596's rotation-age
   slice, which stays open; that issue is its consumer, so no new ticket. The
   proposal still called it an upper bound, which round 1's post-merge fix had
   already corrected in the code; it now says lower bound.
3. *Nothing runs drift in CI or a hook.* Deliberate for now. drift needs an
   unlocked daemon, which CI does not have (CI receives secrets through
   `sync ci`). "can gate" is the capability the exit code gives; a scheduled
   run belongs with #1596's staleness slice.
4. *Three denominators in one summary line.* **Applied.** It said "across 23
   item(s)" for the vault items a declaration names; it now says "naming 24
   item(s), 23 of them in the vault", so an absent item shows as the gap.
   `TestDriftSummaryCountsDeclaredItemsAndThoseInTheVault`. A count of
   placement-ungoverned declarations is declined: #586 will change it, and the
   plane rule above already makes it a property of the plane.
5. *drift reads through `BWServeClient{}`, not the pinned backend.* Deliberate.
   `bw list items` could list through the CLI too, but it answers with the same
   whole plaintext, and `BWLister` exists so the value-free projection has
   exactly one implementation to audit. A CLI lister would be a second decoder
   of every value in the vault. Now stated at the seam (finding 8).

Mutation for this round, **one mutant at a time, each under a 2 GB memory cap,
restored from `git checkout HEAD`, and counted only on a tree that vets clean**
(lessons 284 and 286): 12 mutations, **12 killed**, each by the named test —
the username leak, the field-less dedupe key, the empty ratified list, a folder
on an ungoverned plane, `FolderUnresolved` never set, drift ignoring it,
reconcile losing its case, the inventory read unsynced, a sync failure ignored,
drift calling `MoveItem`, drift growing `--fix`, and `item-folder-unknown`
raised for a declaration that governs no placement. The whole module, the pinned
linter (v2.12.2, 0 issues), `GOOS=windows go vet`, CLI-078 f1–f9 (9/9) and
CLI-080 f1–f10 (10/10) are green.

## Adversarial review, round 5 — disposition

`review.md` round 5: **PASS**, `nan/mimo-v2.5` (a fifth model family across five
rounds), reviewed `0c10f0b`, committed verbatim (`c8a1acc`). It re-ran the module,
the linter, `GOOS=windows go vet` and f1–f9 (9/9), and reopened three round-4
defects by mutation, all killed. Its other round-4 closures were read from the
tests rather than mutated; this file's round-4 battery (12/12) is the mutation
evidence for those. No findings; three observations:

| Observation | Disposition |
|---|---|
| `ItemSummary.Revised` is never read | **Declined, no ticket.** Its consumer is #1596's rotation-age slice, which is open. The review attributes it to CLI-080; that spec does not read it either. |
| The inventory's "NOT yet in bw" heading contradicts its body | **Applied.** Now "age secrets that were not in bw (June 2026) → migrated". Docs only, after the verdict. |
| Nothing runs `drift` in CI or a hook | **Declined**, as in round 4 (question 3): CI has no unlocked daemon, and a scheduled run belongs with #1596's staleness slice. |
