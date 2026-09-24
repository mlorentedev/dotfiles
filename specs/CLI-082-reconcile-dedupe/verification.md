---
tags: [spec, verification, templates]
created: "2026-09-23"
---

# Verification - CLI-082-reconcile-dedupe

## Evidence

| AC | Proof |
|---|---|
| AC1 | `TestVerifyRetiresGivesEachRetireAVerdict` + `TestReconcilePlanShowsEachRetireVerdict` |
| AC2 | `TestVerifyRetiresNamesHowToSettleEachBlocker` + `TestApplyRetireRefusalNamesTheExits` (new), and `TestApplyRetireRefusesWhenTheValuesDiffer` (pre-existing from CLI-080, in `reconcile_retire_test.go`; unchanged, and it now runs through the rebuilt `sameValue`) |
| AC3 | `TestRegistryValidatesRetiredItems` |
| AC4 | `TestPlanRetiredItems` + `TestApplyDeletesARetiredItem` + `TestBWServeWriter_DeleteItem` + `TestBWPut_DeleteItemDeletesByID` + `TestReconcileDeletesARetiredItem` + `TestLockHintWriterForwardsEveryMethod` |
| AC5 | `TestPlanRetiredItems` (the ambiguous name) |
| AC6 | planted values asserted absent in the AC1, AC2 and AC4 command and core tests |

## Test status

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...`: clean
- `golangci-lint run` at the pinned v2.12.2: 0 issues
- `go test -count=1 ./...`: green across the module
- `features.json`: 6/6 pass. The archived CLI-080 (10/10) and CLI-078 (9/9)
  features still pass on this tree.
- Every run above under a memory-capped scope (lesson 286).

### Mutation

21 mutations, one at a time, each under a 2 GB cap, restored from
`git checkout HEAD`, and counted only on a tree that vets clean. **21 killed**,
each by the named test. They covered each verdict turned into another, a
non-equal retire kept as an operation, a remedy missing an exit, the apply
re-verification removed, each of the four `retired:` rules dropped, an
ambiguous name deleted anyway, a gone item not reported, delete-item ranked
first, apply skipping the delete, the daemon delete not syncing, the CLI
deleting by name, the plan skipping verdicts or retired items, the plan line
hiding the shape, and the shape ignoring notes.

One **survived** the first run: the lock-hint wrapper swallowing `DeleteItem`.
Nothing tested that the wrapper forwards any of its six methods.
`TestLockHintWriterForwardsEveryMethod` now calls every `BWWriteClient` method
through it by reflection, so a method added later is covered without anyone
remembering to, and it kills that mutant.

### Live, read-only (2026-09-23)

A plan against the real store, with a scratch registry that declares eight
legacy pairs as `from` + `retire` and lists `github-cli-pat` under `retired:`.
Nothing was applied.

- 3 retires **verified equal**: `OPEN ROUTER API KEY`/notes,
  `POLLEX_API_KEY`/notes, `pypi.org`/"API token".
- 1 `delete-item`: `github-cli-pat`, shown as `fields GITHUB_PERSONAL_ACCESS_TOKEN; login`
  with its reason.
- 4 blocked, each naming its way out:
  - `Hetzner`: two items carry the name.
  - `login.tailscale.com`, `Gmail` and `Stripe`: their values **differ** from the
    canonical copy, so the owner has to say which is current.
- No value, length or digest appeared in the output.

## Decisions made during implementation

- **The plan reads values, and says so.** It is the only way to know a retire's
  outcome before applying. The read goes through the same pinned reader as the
  apply, and only the verdict leaves `compareRetire`. The command's help says it.
- **A static conflict is the registry's job, not the plan's.** An item listed
  under `retired:` while a declaration still names it is refused at parse time, so
  CI catches it. The plan only decides what depends on the store: gone, or
  ambiguous.
- **Deletion is declared, never inferred.** An item a retire empties is shown by
  the plan (`fields …; notes; login`) but not deleted until the registry lists it.
  Same reason `retire` is a flag and not a default.
- **Delete by id, never by name.** Both backends resolve the name through the
  lookup every edit already uses; `bw delete item <name>` would act on whichever
  item bw picked.

## Found, not in scope

- **`bw.from` is refused on a multi-var secret.** So `cloud.nan.builders`/"api-key",
  a copy of `NAN_API_KEY`, cannot be retired this way. `NAN_API_KEY`'s two vars read
  the *same* field, so a single source fills it; the refusal ("one source cannot
  fill 2 fields") is stricter than the case needs. Recorded on #1624.

## Promotion candidates

- [ ] Lesson? The survivor above is the reusable part: a wrapper over an interface
      is only tested for the methods someone remembered, and a reflection test
      over the interface removes the remembering. A narrow case of lesson 279's
      family; recorded here rather than as a new lesson.
- [ ] ADR? No. It extends CLI-080 within ADR-028.

## Adversarial review — disposition

The first run (`nan/qwen3.8-flash`, launched against `3672a0d`) ended after about
64 minutes on HTTP 429 from NaN (`max_parallel_requests` 7/7). It wrote no verdict
and left its transient mutation harness in the worktree, which was moved out of
the tree unread before the relaunch (#1642). The relaunch used
`agy/gemini-3.1-pro-high`, the one pool member not on NaN, against `6d5fcf5`.

`review.md`: **PASS**, `agy/gemini-3.1-pro-high`, reviewed `6d5fcf5`, committed
verbatim. **Disclosed: it is a thin review.** It took about 3 minutes, and its text
records no test or mutation run of its own. The mutation evidence for this spec
is the 21/21 battery above and the read-only live plan.

| Severity | Finding | Disposition |
|---|---|---|
| Minor (THEORETICAL) | `checkRetired` accepts a whitespace-only item name | **Ticketed** (#1624): it touches code, and code is not changed after a passing verdict. The effect is benign: the plan reports the item gone. |
| Minor (THEORETICAL) | An empty source with an empty destination blocks as `destination empty` | **Declined.** An empty destination must never let a source be deleted, and an item left with nothing is removed through `retired:` (`delete-item`), not through a retire. |

**Not archived in this PR, deliberately.** The spec gate accepts an archive only
from a PR that closes the spec's issue (SDD-038), and #1624 stays open for its
other items. CLI-082 archives with the PR that closes #1624.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-082-reconcile-dedupe/`
- [ ] Independent adversarial review passed (reviewer != implementer)
