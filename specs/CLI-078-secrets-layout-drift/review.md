---
spec: "CLI-078-secrets-layout-drift"
verdict: "FAIL"
reviewed_sha: "c7a8adca25b9e151340d68b15d56502378c3a7bd"
reviewer: "nan/glm5.3-flash"
date: "2026-09-22"
---

## Adversarial review

**Scope**: CLI-078-secrets-layout-drift — `dotf secrets drift`, the shape-only inventory (`bwserve_list.go`), `LayoutDrift`/`BWDeclarations` (`layout.go`), the `Dotfiles/` folder-taxonomy move, and the spec triad.
**Sources**: `specs/CLI-078-secrets-layout-drift/{proposal,tasks,verification,features.json}`; `git diff c52e637cd2e014d5c897853f74841dbdc1061f66...HEAD` (3 commits: #1597 plus two dependency bumps).

### Spec and task alignment

- Proposal and tasks read as a coherent contract; every AC has a named test in `layout_test.go`, and `features.json` carries 8 non-vacuous entries — all 8 re-run green in this review.
- Independently re-executed: `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` clean; `go test -count=1 ./...` green across the module; `golangci-lint run` at the pinned v2.12.2 → 0 issues.
- Three mutations applied and reverted by this review, each killed by the named test: (a) `itemWire` naming `login.password` + `ItemSummary` carrying it → `TestDecodeItemsCannotCarryAValue` red; (b) `BWDeclarations` skipping non-BW backends → `TestBWDeclarationsIncludesDormantBlocksOnAgeBackedSecrets` red; (c) `hasField` dropping the login dispatch → `TestLayoutDriftResolvesFieldsTheWayTheReaderDoes` red. The projection claim (AC6) is genuinely mutation-verified, not asserted.
- **The alignment breaks on one axis: the walk only covers env-exposed declarations.** `BWDeclarations` iterates `s.Expose.Env.Vars` exclusively (`layout.go`), so any secret whose `bw:` block is paired with a **file** expose contributes zero declarations. Seven of the 33 bw-carrying secrets in the real `secrets/registry.yaml` fall in that hole — reproduced this session by parsing the real registry through `ParseRegistry().BWDeclarations()`: 26 of 33 secrets represented, with `KUBECONFIG`, `ZOHO_RECOVERY_CODE`, `GMAIL_BACKUP_CODE`, `CHATGPT_BACKUP_CODE`, `CHATGPT_RECOVERY_CODE`, `STRIPE_BACKUP_CODE` and `AGE_KEY_PERSONAL` reported absent from the walk.
- The proposal's Why argues that "a declaration nobody checks is not a declaration; it is a comment that happens to be YAML." That sentence still describes seven live declarations after this change, including five actively-read bw-backed targets (`kubelab-kubeconfig`/notes, `gmail-backup-code`/notes, `openai-account`/backup-code, `openai-account`/recovery-code, `stripe-backup-code`/notes) and the field `recovery-code` on the dormant `zoho` item. Nothing in the proposal's Out-of-scope list excludes file exposes; the ACs are written universally.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | coverage / AC2 | File-exposed `bw:` declarations never enter the drift walk: 7 of 33 bw-carrying secrets produce zero `BWDecl`s, so a missing or misfiled item for `kubelab-kubeconfig`, `gmail-backup-code`, `stripe-backup-code`, `openai-account` (×2 fields) — and the dormant `zoho`/`recovery-code` declaration — would go unreported, violating AC1–AC4 as written (AC2 claims every declared absent item is reported, including dormant ones) | Reproduced this review: a throwaway test parsing `secrets/registry.yaml` through `ParseRegistry().BWDeclarations()` fails with the 7 named secrets; 27 declarations vs 33 bw-carrying secrets — the same 27 `verification.md` reports as if complete | UNTESTED — `TestBWDeclarationsIncludesDormantBlocksOnAgeBackedSecrets` and `TestLayoutDriftChecksDormantDeclarations` use env-exposed fixtures only | code (`layout.go` walk must also flatten `Expose.File.Var`) + tests; re-review follows |
| Major | THEORETICAL | store-splitting residue | The opt-in live canary `TestLiveBWServeWriter_CanaryRoundTrip` still calls `ResolveFolder("apps")` (`bwserve_writer_live_test.go:55`); after the taxonomy move, no folder named `apps` exists in the vault, so a deliberate live run would CREATE a stray `apps` folder — the exact split this spec exists to prevent — and cleanup deletes only the item, so the folder persists | Code read: `BWPut.ResolveFolder`/`BWServeWriter.ResolveFolder` create on miss; registry rename left this call site behind | UNTESTED (opt-in live test, skipped by default) | code (one line: `"apps"` → `"Dotfiles/apps"`) |
| Minor | THEORETICAL | folder resolution | `ListFolders` filters the null-id pseudo-folder, but `ListItems` builds `byID` without filtering — if `bw serve` really emits a null-id "No Folder" row (as the new comment in `bwserve_list.go` asserts), every unfoldered item resolves to `Folder:"No Folder"` instead of `""`, contradicting the `ItemSummary.Folder` doc and the "(no folder)" detail text. One of the two consumers of `decodeFolders` is wrong; no recorded fixture pins which | Code read of `bwserve_list.go`; both tests for unfoldered resolution pass a hand-built map, never the production path | UNTESTED | code + tests (recorded fixture of a real `/list/object/folders` body) |
| Minor | THEORETICAL | AC8 exit code | The "exits non-zero on any finding" half of AC8 has no named test: `features.json` f8 only greps the source for writer calls; no cmd-level test runs `drift` with findings and asserts a non-zero exit. The LayoutDrift→`RunE` error wiring is unverified | `features.json` f8 command is a grep; no test in `cli/internal/cmd/` exercises the drift command | UNTESTED | tests (cmd-level test with a stubbed `bwLister`) |
| Minor | REAL | reporting | `LayoutFinding.Secret` is documented as "registry id, so the reader knows which line to edit", but the command never prints it (`secrets_drift.go` prints Kind/Item/Detail only). With dedupe across secrets, the printed line can lose which registry entry declared the item | Code read of `secrets_drift.go` vs `layout.go` struct doc | UNTESTED | code (print the id, or amend the claim) |
| Question / assumption | THEORETICAL | AC4 wording | `hasField` matches `fieldFromItem`'s dispatch, not its empty-value behavior: the reader resolves an empty `notes`/`username` without error (refusal happens later in EnvFor), while `hasField` reports them missing. The direction is toward honesty (EnvFor refuses empties anyway) and the password case is documented in the proposal — but AC4's "matches fieldFromItem exactly" is true of the dispatch only. Confirm this reading is intended | Code read of `bw.go:fieldFromItem` (empty value returns no error) vs `resolve.go` (EnvFor refuses empty) | `TestLayoutDriftResolvesFieldsTheWayTheReaderDoes` covers absence only, not the empty-value axis | spec artifacts note in `verification.md` (outside the contract set — safe to edit now) |
| Minor | REAL | contract staleness | `tasks.md`'s last box "[ ] PR opened referencing this spec folder" is unticked while PR #1597 is merged (commit 799ca66 is in the reviewed range). Stale contract artifact. **Not edited by this review** — editing a contract file would invalidate this verdict | `git log c52e637..HEAD` vs `tasks.md` | n/a | disposition in `verification.md`; tick the box in the next round before re-review |

Scope note: the launcher-stated range includes two commits unrelated to the spec (`cli/go.mod`/`go.sum` bump #1590, `pr-agent.yml` action bump #1591). Documented here as a range artifact, not creep by the spec.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Env-path ACs are genuinely met, but the comparison silently covers only 26 of 33 bw-carrying secrets — criteria partially met for the registry the command actually runs against |
| Verification       | B | Evidence is reproducible and mutation-backed, but the live-run count ("27 declared targets") is presented as complete without noting the seven excluded secrets |
| Scope              | B | Diff matches the proposal; the two dependency bumps are launcher-range artifacts, not spec creep |
| Reliability        | B | Error paths handled well (unreadable store, malformed JSON, byte-count-only errors, bad dates); the pseudo-folder ambiguity is the one unclear path |
| Maintainability    | A | Small functions, deliberate duplication named and mutation-pinned, docs explain why |
| Handoff-readiness  | B | Spec triad complete and honest (mutation survivors recorded); lesson capture explicitly deferred with a stated DoD skip |

### Verdict
FAIL

One **REAL** Blocker (file-exposed declarations outside the walk — reproduced against the real registry this session, UNTESTED). Per the archive gate, a change closing this spec cannot archive on this verdict; the contract set may be edited freely in the next round since this review is invalidated by any contract edit anyway.

### Recommended next steps

Minimum set to flip to PASS (all in the code/tests set except where noted; re-review required either way):

1. **Fix the walk** — flatten `Expose.File.Var` alongside `Expose.Env.Vars` in `BWDeclarations` (this also closes the dormant file-exposed class, e.g. `zoho`/`recovery-code`), with named tests asserting a file-exposed bw secret produces a declaration and that the real-registry walk reaches all 33 (a registry-shape test or a fixture mirroring the file-expose pattern).
2. **Fix the canary** — `ResolveFolder("Dotfiles/apps")` in `bwserve_writer_live_test.go`; otherwise the next deliberate live run mints the stray folder this spec exists to prevent.
3. **Pin the pseudo-folder question** — record a real `/list/object/folders` body as a fixture and make `ListItems` and `ListFolders` agree on the null-id row.
4. **Name the AC8 exit-code test** — cmd-level test with a stubbed `bwLister` returning findings, asserting non-zero exit.
5. **Correct `verification.md`** (outside the contract set — may be applied now): state that the live run covered env-exposed declarations only, and disposition the `tasks.md` unticked PR box and the empty-value reading of AC4.
6. After 1–4 land, run a fresh review against the new HEAD; `dotf spec review CLI-078-secrets-layout-drift` resolves the pool.
