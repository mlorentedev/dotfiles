---
spec: "CLI-038-secrets-probe"
verdict: "PASS"
reviewed_sha: "39689e1a6314e52c2868a869029e78becc3c8298"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-16"
---

## Adversarial review

**Scope**: CLI-038-secrets-probe
**Sources**: `specs/CLI-038-secrets-probe/{proposal,tasks,verification}.md`, `cli/internal/secrets/{probe,bwserve_probe,bwserve}.go`, `cli/internal/cmd/secrets_probe.go`, `docs/lessons.md` (diff against `718c8958…main`)

### Spec and task alignment

All 8 acceptance criteria are addressed in the implementation:

| AC | Status | Evidence |
|---|---|---|
| AC1 — reports status, shape, field names, lengths, fingerprints | [x] | `TestShapeProbe_ReportsShapeNotContent`, `ProbeReport.String()` |
| AC2 — sentinel never appears in output | [x] | `TestShapeProbe_NeverEmitsAValue` (both `raw` states) |
| AC3 — 2xx body never printed, including with `--raw` | [x] | `TestShapeProbe_RawNeverShowsA2xxBody` |
| AC4 — `--raw` prints non-2xx bodies only, capped | [x] | `TestShapeProbe_RawShowsNon2xxBody` + `TestShapeProbe_RawBodyIsCapped` (512-byte cap) |
| AC5 — probe goes through `BWServeClient` | [x] | `TestProbe_UsesTheSameTransportAsCall` |
| AC6 — `--count N` reports distribution, no value | [x] | `secrets_probe.go` `printDistribution`, `TestProbeCmd_NonPositiveCountIsRefused` |
| AC7 — read-only | [x] | No unlock/sync/set/rotate called on probe path; verified by code audit |
| AC8 — `docs/lessons.md` names this as sanctioned probe | [x] | Resolution entry added at `docs/lessons.md` line 2422 |

All tasks are checked `[x]`. No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| Minor | REAL | maintainability | `is2xx` / `isSuccess` duplicated in two packages — identical predicate, two copies | `probe.go:197` and `secrets_probe.go:155` define `status >= 200 && status < 300`. Comment at the cmd copy says "Duplicated as a tiny predicate rather than exported", which is a conscious choice rather than drift, but creates the latent pattern for a third copy. | UNTESTED — no test asserts both are the same function; a change to one would only be caught at build if the types diverged | code — export `is2xx` from `secrets` or inline the status check; not gating |
| Minor | REAL | maintainability | `ProbeItemID` (probe.go:104) loads full credential-bearing items via `c.Probe()` on the search endpoint — the body carries every `fields[].value` and `login.password` for matching items in memory | Code audit of `ProbeItemID`: receives `ProbeResult.Body` containing full items, unmarshals only `Name` and `ID`, returns only the id. Body goes out of scope on return. Safe by scoping, not by structural exclusion — a future refactor adding a debug print in that function would leak. | UNTESTED — `TestProbe_ErrorsNeverCarryBodyBytes` covers error paths but no test asserts `ProbeItemID` never holds secrets longer than its call frame | tests — add a regression test pinning that body bytes from the search response are never rendered |
| Minor | THEORETICAL | correctness | `collectValues` silently drops non-string `value` entries in the field-label exception path (e.g. `{"name":"enabled","value":true}` — the boolean `true` has no case in the walk) | Code audit: `fieldLabel` returns `(label, true)` because `m["value"]` exists (as `bool`). Then `collectValues(path+"["+label+"]", true, out)` receives a `bool`, which hits none of the `switch` clauses. No value leak, but the field is invisible — a potential diagnostic gap. Bitwarden field values are always strings in practice, so this is THEORETICAL. | UNTESTED — no test exercises a non-string custom field value | code — add a `default` branch in `collectValues` that logs the type name or marks the field as "unexpected type" |
| Minor | THEORETICAL | safety | Field label exception prints the field `name` verbatim in output paths (e.g. `data.fields[0][my-secret-rotation-key]`). The spec states "labels are not credentials", which is true for Bitwarden's data model, but the exception is scoped at the schema level rather than verified. | Code audit of `collectValues` lines 163–183: a user who names a custom field "gitlab-token-2026-08" will see that string printed. Safe by domain knowledge, not by any validation. | UNTESTED — no test asserts a field label is safe to print | spec — explicitly document the assumption in `proposal.md` risks section; code — optional: validate no label looks like a potential secret (heuristic only) |
| Minor | SPECULATIVE | cmd-testing | The cmd-layer test coverage for probe is limited to refusal paths (`TestProbeCmd_NonPositiveCountIsRefused`, `TestProbeCmd_NonBWSecretIsRefused`, `TestProbeCmd_UnknownSecretIsRefused`). No positive test exercises `--raw`, `--count`, or distribution output from the cmd wiring. | All three cmd probe tests verify `err != nil`. The shape tests in the `secrets` package cover the output safety, so a regression in cmd wiring (e.g. `isSuccess` being wrong, or flags not reaching `ShapeProbe`) would not be caught by unit tests. | UNTESTED — no cmd-level positive test | tests — add an httptest-based cmd test that starts a test server and asserts distribution output under `--count` with both 200 and 500 responses |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All acceptance criteria verified; negative paths for value leakage are thorough. Minor gaps: non-string values silently dropped, cmd-level wiring has no positive-path tests. |
| Verification | A | Evidence is complete and reproducible — live runs against the real daemon, commands with output captured, mutation tests demonstrated. The spec's own verification found 2 real defects in the tool. |
| Scope | A | Diff matches proposal exactly (11 files, 1167 LOC). No unrelated changes. The `docs/lessons.md` update and spec artifacts are the only non-code changes. |
| Reliability | B | Error paths handled: transport failure (`ErrBWServeUnreachable`), read failure (body cleared), non-2xx (observed, not errored), non-JSON (stated plainly), empty values (fingerprinted as "(empty)"). Minor: transport failure in `--count` aborts without partial distribution. |
| Maintainability | B | Consistently clear naming, functions under 40 lines, excellent comments explaining WHY (the `collectValues` field-label docstring is exemplary). One duplication (`is2xx`/`isSuccess`). |
| Handoff-readiness | A | Spec artifacts complete and current. `docs/lessons.md` updated. No dangling tags. The seam agreement and falsified early hypotheses are recorded in the tasks. |

### Verdict

**PASS** — No Blocker or Major findings. Evaluator rubric grades are all B or above (mechanical aggregation rule: all ≥ B → PASS). The change is solid: the core safety property ("structurally unable to print a secret") is enforced by type design (`ProbeReport` has no value-bearing field) and triple-pinned by tests (`TestShapeProbe_NeverEmitsAValue`, `TestShapeProbe_RawNeverShowsA2xxBody`, `TestProbe_ErrorsNeverCarryBodyBytes`, `TestProbe_ReadFailureCarriesNoPartialBody`). The field-label exception is the one design assumption that could someday be wrong, but it is explicitly documented and scoped to Bitwarden's data model.

### Recommended next steps (before archive)

1. (Optional, low priority) Export `is2xx` from the `secrets` package and use it in `secrets_probe.go` instead of duplicating the predicate. Not blocking — worth a drive-by before the third copy appears.
2. (Surface only) The `ProbeItemID` search response carries full credential bodies in memory. A short doc-comment on `ProbeItemID` that explicitly says "the search body is held during resolution but never rendered" would protect against a future refactor adding a debug print there. Currently that fact is implicit.
3. (Surface only) The non-string custom field value gap (`collectValues` silently drops `{"name":"x", "value": true}`) is theoretical for Bitwarden but could surprise a future backend adapter. Not gating — note against the `collectValues` switch to say "value must be string for field-label branch; non-string values are dropped."

`dotf spec archive` is **advisable** in the current state — the review passes, all gates are clear, and the minor findings are documentation/maintenance items that do not block archive.