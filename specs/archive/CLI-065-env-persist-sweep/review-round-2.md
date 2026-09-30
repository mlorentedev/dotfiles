---
spec: "CLI-065-env-persist-sweep"
verdict: "FAIL"
reviewed_sha: "684f210abe3db3d536761dd819ed064b20a99a53"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: CLI-065-env-persist-sweep / mlorentedev/dotfiles#1363 (round 2). Stated diff range: `git diff 1f9e195b1fc3afd4d87c359e3f9b96eaca62cc5b...HEAD`.
**Sources**: `specs/CLI-065-env-persist-sweep/{proposal,tasks,verification,features}.md|json`, `review-request.json`, round-1 `review-round-1.md`; `cli/internal/env/{persist,persist_windows,resolve,env}.go`; `cli/internal/cmd/env_persist.go`; `cli/internal/doctor/checks_env_persist.go`; lesson 244.

### Scope note (read this with the findings)

The launcher-resolved base `1f9e195` is the parent of this spec's implementation commit `8de1cd1` (#1378). The range now carries 232 commits: the spec's own change (`8de1cd1`, the round-1 fix `1ece408`, the lesson `684f210`, plus the spec files) and ~229 commits that landed on `main` afterwards from other specs. I reviewed the CLI-065 change in full — the whole of `8de1cd1`, the late fix `1ece408` judged against it, and the current state of every file the spec owns at `HEAD` — and did not review the unrelated bulk. That bulk is outside this spec's declared scope and is not covered by this verdict.

### What I ran this session (evidence, not assertion)

- `cd cli && go build ./...` → exit 0; `go vet ./...` → clean; `GOOS=windows go vet ./...` → clean; `GOOS=linux go vet ./internal/env/` → clean.
- `go test ./... -count=1` → every package `ok`, 0 failures (7.6 s).
- `golangci-lint run ./internal/env/... ./internal/cmd/... ./internal/doctor/...` (v2.12.2, the pinned version) → `0 issues`.
- All five runnable `features.json` verification commands → green (f6 is Windows-box-only, `uname` guard — **UNVERIFIED** on this Linux runner, as in round 1).
- **Mutation battery, each reverted, tree left clean:**
  1. Remove the `EqualFold(v.Name, ManagedMarker)` guard from `ValidateNames` → `TestPersist_RefusesTheMarkerNameAsAVariable`, `TestEnvPersist_CheckRefusesTheMarkerNameAsPersistDoes`, `TestCheckPersistedEnv_ByStatus/reserved_contract_name` all FAIL. → round-1's Major fix is load-bearing and named-test covered.
  2. Remove the `ValidateNames` call from `checkPersisted` (cmd read path) → `TestEnvPersist_CheckRefusesTheMarkerNameAsPersistDoes` FAIL.
  3. Remove the `ValidateNames` call from `checkPersistedEnv` (doctor read path) → `TestCheckPersistedEnv_ByStatus/reserved_contract_name` FAIL.
- **Contract freshness**: computed `ContractDigests(specDir)` with the repo's own function — matches all three digests recorded in `review-request.json` (`features.json` cba26200…, `proposal.md` b813feef…, `tasks.md` b31f342e…) exactly, so this review will not be stale at archive.
- **Probes against the shipped code** (synthetic inputs, temp tests, removed): see findings F1 and F2.

### Spec and task alignment

- `proposal.md` AC1–AC6 all map to code; all boxes `[x]`; no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tag in any contract file (the sole grep hit is round-1's own prose). Declared non-goals (pre-marker values, `PATH`, machine scope, rc layer) are honoured: with no marker `owned` is empty, nothing is deleted.
- `tasks.md`: every row `[x]` and, where I could check it, the claim matches the diff — including the new round-1 row (`ValidateNames` refuses the marker name; `persist`, `--check` and doctor all call it), which the mutation battery proves. Nit: the Setup row still names the branch `feat/env-persist-sweep` while the work sits on `fix/env-persist-reserved-marker` — trivial, not gated.
- `verification.md`: all five round-1 findings dispositioned with reasons (four applied, one declined with a stated argument); the unresolvable `a1878e0` provenance was replaced by `8de1cd1`, a real object; the two combined drift+retired doctor rows exist (`checks_env_persist_test.go:39-40`).
- Round-1's REAL Major (marker-name collision) is fixed exactly as demanded, and both round-1 Minors about read paths are fixed at every current entry point.
- **But** the fix treats the symptom class, not the stated rule. `ValidateNames`'s own doc says it "refuses a contract name the marker could not round-trip"; two name shapes still round-trip wrongly, and both break acceptance criteria — findings F1/F2.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | name validation / write path (AC1) | An empty contract name passes `ValidateNames` (no separator, not equal to the marker). `Persist` then writes an unnamed store value, while `MarkerValue` explicitly skips empty names — so the marker does not name what the run persisted, violating AC1 ("writes an ownership marker naming every contract variable it persisted"), and the value can never be swept: the record never lists it, so it is a permanent dotf-written orphan in the store. `loadContract`/`Resolve` perform no name validation, so nothing else rejects `""`. | Probe this session against the shipped code (fake store): `Persist([{Name:"",Value:"v"}])` → `results=[{Name: "" Value: v Changed: true} {Name: DOTF_MANAGED_ENV …}]`, `store=map[:v DOTF_MANAGED_ENV:]`, marker=`""` — persisted but unnamed. Same trigger class as round 1's accepted REAL Major (a contract edit nothing rejects); no shipped `env-contract.json` name is empty today (13 names, checked). | UNTESTED | code + tests (`ValidateNames` refuse `""`; table case beside `TestPersist_RefusesTheMarkerNameAsAVariable`) |
| Major | REAL | name validation / sweep + `--check` (AC2, AC3) | A whitespace-padded name (`" FOO"`) breaks the marker round-trip: `MarkerValue` records it raw, `ParseMarker` trims it to `FOO`. On retirement the sweep looks up `FOO`, misses the stored `" FOO"`, **skips the delete, and rewrites the marker without it** — the ownership record loses the name while the value persists. From then on nothing can sweep it (it is not in the marker) and `Retired`/`--check` report **clean** while a retired dotf-written value sits in the store: AC2's delete and AC3's detection both fail, silently, for this input class. | Probe this session: run 1 with `{" FOO","v"}` → `after run1: map[ FOO:v DOTF_MANAGED_ENV: FOO]`; run 2 with the name retired → `store=map[ FOO:v DOTF_MANAGED_ENV:]`, results contain only the marker rewrite (`SWEEP MISSED`); `Retired(store, nil) = []` (`CHECK BLIND`). Registry value-name lookup is exact for whitespace, so the real store behaves the same; no shipped contract name is padded. | UNTESTED | code + tests (`ValidateNames` also refuse `name != strings.TrimSpace(name)` — one guard; all three read/write entry points already call it) + a cmd/env table case |
| Minor | THEORETICAL | validation placement | The guarantee is enforced at three call sites (`Persist`, `checkPersisted`, `checkPersistedEnv`) rather than inside the read functions themselves; a future fourth reader of `Drift`/`Retired`/`MarkerStale` would inherit round 1's bug by construction. Not a live gap: grep shows exactly three production call sites, all guarded, each mutation-verified today. | grep of `cli/` non-test callers; mutations 2–3 above. | `TestEnvPersist_CheckRefusesTheMarkerNameAsPersistDoes` (cmd), doctor case "reserved contract name → WARN naming the refusal" | code (optional; defense-in-depth) or vault pattern note |

Not findings, recorded for completeness: `--check` non-snapshot reads — declined in `verification.md` with a stated single-writer argument, accepted; AC6 box transcript — accepted as box evidence, unverifiable on this runner.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | AC1–AC5 hold on the shipped contract (all probes and suites green), but two name classes nothing rejects break AC1/AC2/AC3 with a check that reports clean — substantial negative-path gaps in the spec's central guarantee. |
| Verification       | B | Rich named, reproducible evidence; every runnable features command re-ran green here and three mutations proved the late fix load-bearing; AC6 is box-only and both new gaps are UNTESTED. |
| Scope              | B | The spec's own commits match the proposal exactly; the launcher range additionally carries 229 unrelated main commits (not the implementer's doing). |
| Reliability        | C | Errors name the variable, order is load-bearing and pinned, delete-of-absent honoured — but a degenerate name produces a silent skip and a lost ownership record instead of a refusal. |
| Maintainability    | B | Small functions, one shared `Leftovers`, lint/vet clean; the name guard is duplicated across entry points instead of enforced at one seam. |
| Handoff-readiness  | A | `verification.md` dispositions every round-1 finding with reasons, lesson 244 updated in-session with the read-path rule, tasks ticked and traceable. |

Aggregation: two Cs, no D → rubric alone would read PASS WITH GAPS. The severity path is worse: two REAL Majors → **FAIL** (the skill joins the axes by taking the more severe).

### Verdict
FAIL

Two **REAL Majors**, both reproduced by probe against the shipped code this session and both **UNTESTED**: the marker-name reservation from round 1 was closed for one spelling, but the guarantee `ValidateNames` claims — "a name the marker could not round-trip" — still has two holes (empty, whitespace-padded), each of which breaks a stated acceptance criterion (AC1; AC2+AC3 with a false-clean `--check`). No Blocker. Round 1's Major is confirmed fixed and named-test covered.

### Recommended next steps

- **Code + tests (fixes the FAIL)**: extend `ValidateNames` in `cli/internal/env/persist.go` with two refusals — an empty name, and a name whose `TrimSpace` differs from itself — with table cases beside `TestPersist_RefusesTheMarkerNameAsAVariable` (both probes above are the red-green script). All three entry points already call `ValidateNames`, so `--check` and doctor inherit the fix without new call sites; add one read-path assertion only if you want caller coverage pinned beyond the existing tests. **Then re-review (round 3)** — a REAL Major in code is fix, then re-review.
- **Contract set**: no edit needed or wanted — AC1–AC3 already require this behaviour, so `proposal.md` / `tasks.md` / `features.json` stay as recorded in `review-request.json` (their digests match disk; an edit would only add staleness).
- **`verification.md`** (outside the contract set, safe to edit): disposition F1/F2 when the fix lands, and keep the round-1 decline of the snapshot reading as is — its single-writer argument is sound.
- **Vault**: the round-1 lesson addition ("a read path must share what the write refuses") is the right promotion; optionally extend it to "and the writer must refuse what its own record cannot represent" when the fix lands.

**Verdict: FAIL. `dotf spec archive` is NOT advisable in the current state.** Minimum to flip: the two `ValidateNames` refusals plus their named regression tests (one focused commit), and a round-3 review confirming them — the rest of the change, including every round-1 disposition, already stands.
