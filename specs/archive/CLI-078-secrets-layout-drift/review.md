---
spec: "CLI-078-secrets-layout-drift"
verdict: "PASS"
reviewed_sha: "0c10f0bfcf9d4339af22b24e4209a788a90d48c7"
reviewer: "nan/mimo-v2.5"
date: "2026-09-23"
---
## Adversarial review

**Scope**: CLI-078-secrets-layout-drift, round 5 — `git diff c52e637cd2e014d5c897853f74841dbdc1061f66...HEAD` (80 files, +5836/−390), base as resolved by the launcher. Every round-4 finding was re-verified against this HEAD, not against the author's claims.
**Sources**: `specs/CLI-078-secrets-layout-drift/{proposal,tasks,verification}.md`, `features.json`, round 4's `review.md`, `cli/internal/secrets/{layout,bwserve_list,registry,reconcile}.go`, `cli/internal/cmd/secrets_{drift,reconcile}.go` and their tests, `secrets/registry.yaml`, `docs/runbooks/guide-secrets-governance.md`, `docs/secrets-inventory.md`.

### Spec and task alignment

Mechanical, run at HEAD in this session:

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` — clean.
- `go test -count=1 ./...` — green across the module (`rc=0`).
- All nine `features.json` commands executed verbatim — **9/9 pass**.
- `golangci-lint run` at the pinned v2.12.2 — **0 issues**.

**Every round-4 blocker is closed, and I proved each one dead with mutations against a tree that builds:**

| Round-4 claim | Mutation reopened | Result |
|---|---|---|
| AC6: username value can leak (no assertion for `login.username`) | add `Username string` to `ItemSummary`, populate from `w.Login.Username` | **killed** — `TestDecodeItemsCannotCarryAValue` fails: `"USERNAME-must-not-survive-7e41" found in [{"Name":"loaded",...,"Username":"USERNAME-must-not-survive-7e41",...}]` |
| AC8: grep lists 4 of 5 writer methods; `MoveItem`/`RemoveField` pass | n/a — the grep was **replaced** with `TestDriftSourceCallsNoWriter` which derives the forbidden set from `BWWriteClient` via reflection and walks the AST | **closed** — I verified `MoveItem` is in the derived set (`t.Fatalf` confirms it); the test is structurally immune to interface growth |
| Field-dedupe key missing the field half | drop field from key: `w.once(DriftFieldMissing, d.Item)` | **killed** — `TestLayoutDriftReportsOneFindingPerProblemNotPerVar` fails: `want 2 findings, got 1` |
| `ratifiedFolders()` derivation unpinned | empty the `validBWFolders` set | **killed** — `TestParseRegistry_BwFolder_RejectsUnratified` fails: error says `(())` instead of naming the ratified folders |
| Unresolved `folderId` reads as `Folder==""` | n/a — **fixed by `FolderUnresolved` flag** in `ItemSummary` | **closed** — `TestDecodeItemsMarksAFolderIDTheListDoesNotCarry` asserts `FolderUnresolved` for items with unknown folder ids; `TestLayoutDriftSaysSoWhenItCannotNameAnItemsFolder` asserts the `item-folder-unknown` finding |
| Plane↔folder guard covers only 2 of 4 planes | n/a — **fixed**: `planeFolder` now denies any folder on a plane absent from the map | **closed** — I tested: `plane: floor` + `bw:{folder: Dotfiles/apps}` → refused; `plane: personal` + `bw:{folder: Dotfiles/infra}` → refused |
| `drift` never syncs before reading | n/a — **fixed**: `readInventory()` syncs first | **closed** — `TestDriftSyncsBeforeItReads` asserts `sync` is the first call; `TestDriftRefusesAStoreItCannotSync` asserts no reads follow a failed sync |
| Stale comments (`bwLister` "pinned backend", drift "two declared items") | n/a | **closed** — `bwLister` comment now says "Nil in production, where the inventory is read from the daemon directly (secrets.BWServeClient{})", which is accurate; drift comment says "three declared items missing entirely", matching `verification.md` |
| Stale `docs/secrets-inventory.md` item names | n/a | **closed** — `dockerhub-token`→`DockerHub`, `x-twitter`→`X_*`, `beehiiv-dns`→`beehiiv.dns-records`; migration section notes "Done for every one listed here in June" |

AC-by-AC refutation attempt at HEAD: AC1 through AC9 each **hold** — I could not break any of them without a committed test failing. The projection (`AC6`) is the strongest defence: a compound mutation adding a value-bearing field to `ItemSummary` is killed by a test that marshals the entire result and asserts no distinctive secret string survives.

### Findings

No findings. All round-4 findings are resolved with evidence.

Minor observations (not gateable — surfaced for the author's discretion):

1. **`ItemSummary.Revised` is carried, documented, and never read by any caller** (only `IsZero` assertions in tests). It is substrate for CLI-080's rotation-age slice. Worth a follow-up ticket naming the consumer, or is it intentionally substrate?

2. **The "age secrets NOT yet in bw" section header in `docs/secrets-inventory.md` is slightly misleading** — the body immediately says "Done for every one listed here in June", so the to-do framing is historical, not current. A rename to "Formerly age-only, now bw" would remove the ambiguity.

3. **Nothing runs `drift` in CI or a hook.** The spec's exit-code design presumes a gate, but `drift` needs an unlocked daemon which CI does not have. "can gate" is the capability; a scheduled run belongs with #1596's staleness slice. This is by design and stated at the seam.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 9 ACs verified with passing tests; compound mutation for AC6 kills a value leak; plane guard enforced for all planes; sync-before-read enforced; no round-4 defect survives. |
| Verification       | A | Every AC has named tests, 9/9 features pass, mutations against a building tree kill all security and correctness pins; the AC6 projection test marshals the full result and asserts absence of 6 distinct secret values. |
| Scope              | A | Diff matches the proposal exactly — drift command, projection, registry fix-ups, and fixes for round-4 findings. No creep; CLI-080 is a separate spec. |
| Reliability        | A | Error paths handled (sync failure refused, unreadable store refused), idempotent (read-only, no state mutation), fix-ordered findings, one per problem, non-zero exit. |
| Maintainability    | A | Functions under 40 lines (driftWalk split into 5 methods, max 18 exec lines), clear naming, the deliberate `hasField` duplication is named and pinned by an agreement test, no dead code. |
| Handoff-readiness  | A | `verification.md` records every round's dispositions with the test that proves them, the mutation table matches what I measured, deferred items name their tickets (#586, #1603, #1621). |

### Verdict
PASS

### Recommended next steps

- `dotf spec archive CLI-078-secrets-layout-drift` is advisable in its current state.
- The three minor observations above can be dispositioned in `verification.md` (applied / ticketed / declined with a reason) or carried into follow-up tickets — none is gateable.
