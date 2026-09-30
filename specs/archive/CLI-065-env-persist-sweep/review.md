---
spec: "CLI-065-env-persist-sweep"
verdict: "PASS"
reviewed_sha: "2a0f3d2aa8cdd12568ddccdf9f4c68f10a36d1a6"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: CLI-065-env-persist-sweep / mlorentedev/dotfiles#1363 (round 3). Stated diff range: `git diff 1f9e195b1fc3afd4d87c359e3f9b96eaca62cc5b...HEAD`.
**Sources**: `specs/CLI-065-env-persist-sweep/{proposal,tasks,verification,features}.md|json`, `review-request.json`, `review-round-1.md`, `review-round-2.md`; `cli/internal/env/{persist,persist_windows,persist_other,resolve,env}.go`; `cli/internal/cmd/env_persist.go`; `cli/internal/doctor/checks_env_persist.go`; lesson 244.

### Scope note (read this with the findings)

The launcher-resolved base `1f9e195` is the parent of the spec's implementation commit `8de1cd1` (#1378). The range carries 234 commits: the spec's own change (`8de1cd1`), the round-1/2 fixes as rebased (`0dded160`, `2a0f3d2a`), the lesson update (`515da32a`), and ~230 unrelated commits that landed on `main` in between. I reviewed the CLI-065 change in full — the whole of `8de1cd1` judged together with the two late fixes, plus the current state of every file the spec owns at `HEAD` — and did not review the unrelated mainline bulk; it is outside this spec's declared scope and not covered by this verdict. Note round 2's `reviewed_sha` (`684f210a`) no longer exists in history: the fixes were rebased (`1ece408`→`0dded160`, `684f210`→`515da32a`), so the whole change was re-read at `HEAD` rather than as a delta.

### What I ran this session (evidence, not assertion)

- `cd cli && go build ./...` → exit 0; `go vet ./...` → clean; `GOOS=windows go vet ./...` → clean; `GOOS=linux go vet ./internal/env/` → clean.
- `go test ./... -count=1` → every package `ok`, 0 failures (fresh, this session).
- `golangci-lint run ./internal/env/... ./internal/cmd/... ./internal/doctor/...` → `0 issues` (v2.12.2, the pinned version).
- **All five runnable `features.json` verification commands (f1–f5) re-run → green**; f6 is Windows-box-only (`uname` guard) — **UNVERIFIED** on this Linux runner, as in rounds 1–2 (box transcript accepted as evidence).
- **Contract freshness**: computed `ContractDigests` with the repo's own function (temp test, removed) → `features.json cba26200…`, `proposal.md b813feef…`, `tasks.md 51efaa76…` — all three match `review-request.json` exactly; this review will not be stale at archive. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tag in any spec file (only hits are the transcript's echo of round 2's own prose).
- **Mutation battery, each reverted, tree clean afterwards:**
  1. Remove round 2's empty/padded-name refusal from `ValidateNames` → `TestPersist_RefusesANameTheMarkerCannotRoundTrip` **FAIL**. Round 2's REAL Major fix is load-bearing and named-tested.
  2. Remove the `ValidateNames` call from `checkPersisted` (cmd `--check` path) → `TestEnvPersist_CheckRefusesTheMarkerNameAsPersistDoes` **FAIL**.
  3. Remove the `ValidateNames` call from `checkPersistedEnv` (doctor path) → `TestCheckPersistedEnv_ByStatus/reserved_contract_name_→_WARN_naming_the_refusal` **FAIL**.
  All three entry points are pinned; all three refusals (round 1's reserved marker name, round 2's empty and padded names) are inside the one guarded function.
- **Probe against shipped code** (temp test, removed): `Leftovers(["I","ı"], nil)` → `["I"]` — see F1. Also confirmed refused: `"A;B"` (separator); accepted: `"FOO BAR"` (internal space — correct).

### Spec and task alignment

- `proposal.md` AC1–AC6 all map to code; every task row `[x]` and, where checkable, matches the diff — including the round-2 row ("`ValidateNames` also refuses … an empty or whitespace-padded name … `persist`, `--check` and doctor all call it"), which the mutation battery proves from both ends.
- `verification.md` dispositions all five round-1 findings and both round-2 Majors with reasons and named tests; provenance cites the real object `8de1cd1`.
- Round 2's two REAL Majors (empty name, whitespace-padded name) are **fixed exactly as demanded**: one guard in `ValidateNames`, covered by `TestPersist_RefusesANameTheMarkerCannotRoundTrip` (`""`, `" "`, `" FOO"`, `"FOO "`, `"\tFOO"`, no store operation before refusal), with all three call sites re-verified by mutation above. Round 1's findings stand as round 2 confirmed them.
- Declared non-goals (pre-marker values, `PATH`, machine scope, rc layer) are honoured: no marker → `owned` empty → nothing deleted; `Delete` maps `ErrNotExist` → nil in `persist_windows.go` and broadcasts `WM_SETTINGCHANGE` like `Set`; off Windows the command is a no-op (f5 green).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | THEORETICAL | sweep set (`Leftovers`) | The dedup key (`strings.ToUpper`) and the equality predicate (`strings.EqualFold`) can disagree for non-ASCII names: two marker entries that upper-case to the same key but are not fold-equal — e.g. `"I"` and `"ı"` — cause the second to be skipped without ever being checked, so it is neither swept nor reported, and the marker rewrite drops it (silent orphan + false-clean `--check`, the round-2 F2 harm class). Requires a contract holding such a pair — no shipped contract does, no plausible typo produces one, and whether the real HKCU lookup treats the pair as one value (which would self-heal the sweep) is unverifiable on this runner. ASCII case-variant pairs (`FOO`/`foo`) behave correctly on the real store because the registry shares one value and the record is rebuilt from the contract. | Probe this session: `Leftovers(["I","ı"], nil)` → `["I"]` (the `"ı"` entry vanished); `Leftovers(["SS","ß"], nil)` → `["SS","ß"]` (control — simple case mapping, no collision) | UNTESTED | code (align dedup with the equality predicate, e.g. fold both through the same function) or decline with a reason in `verification.md` — not gating |
| Minor | SPECULATIVE | spec tasks.md | Setup row still says "Branch created from main: `feat/env-persist-sweep`" while the work sits on `fix/env-persist-reserved-marker` (round 2 flagged this too). | `tasks.md` Setup; `git branch` | n/a (doc nit) | spec contract file — **do not edit now**: an edit would stale this verdict for zero behavioural gain; carry as a note |
| Minor | THEORETICAL | tests | The read-path tests exercise `ValidateNames` only through the reserved-marker-name case; no `--check`/doctor test feeds it a padded or empty name (mutation 1 left cmd/doctor green). Composition is still sound — one guarded function, each call site pinned by mutations 2–3 — but a table case in `cmd`/`doctor` would pin the new refusals at the read paths too. | mutation 1–3 this session | `TestEnvPersist_CheckRefusesTheMarkerNameAsPersistDoes` (call site), `TestPersist_RefusesANameTheMarkerCannotRoundTrip` (guard) | tests (optional hardening) |

Not findings, recorded for completeness: `--check` non-snapshot reads — declined in `verification.md` with a stated single-writer argument, accepted in round 2 and unchanged; AC6 box transcript — accepted as box evidence, **UNVERIFIED** on this runner; validation at three call sites rather than inside the readers — round-2 Minor, declined with a stated reason, and the mutations show all three sites are pinned today.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | AC1–AC5 verified against named tests re-run this session; AC6 has box transcript; the round-1/2 negative classes (reserved, empty, padded, separator) all refuse before any store write. |
| Verification       | B | Every runnable feature command green here, three mutations prove each guard and each call site load-bearing, digests match; AC6 remains box-only (UNVERIFIED on Linux). |
| Scope              | B | The spec's own commits match the proposal exactly; the launcher range additionally carries ~230 unrelated mainline commits (the base choice, not the implementer's). |
| Reliability        | B | Errors name the variable, sweep-before-write order pinned, delete-of-absent honoured, refusals precede any store op; one exotic non-ASCII dedup edge remains (F1). |
| Maintainability    | B | Small functions, one shared `Leftovers`, lint/vet clean; validation lives at three call sites instead of one seam — declined with a stated reason, mutation-pinned. |
| Handoff-readiness  | A | `verification.md` dispositions both prior rounds with named tests, lesson 244 updated in-session, every task ticked and traceable. |

Aggregation: all B or above, no C, no D. Severity axis: no Blocker, no Major — round 2's two REAL Majors are fixed and mutation-verified. Both axes agree.

### Verdict
PASS

Round 2's FAIL is cleared: both REAL Majors are fixed in one guard (`ValidateNames` refusing empty and whitespace-padded names), covered by a named regression test that fails when the guard is removed, with every read/write entry point re-pinned by mutation this session. No new Blocker or Major was found; the one mechanism-level edge found this round (F1) is non-ASCII, contract-unreachable in practice, and tagged THEORETICAL. Minors listed above stand as tracked gaps, not gates.

### Recommended next steps

- **`verification.md`** (outside the contract set, safe to edit): disposition F1 — either apply the dedup-key alignment in `code` with a named test, or decline it with a reason ("contract names are ASCII identifiers; the pair is unreachable"); also note this round's mutations as evidence for the call-site pinning.
- **Contract set (`proposal.md` / `tasks.md` / `features.json`)**: no edit — closing them as recorded is what keeps this verdict valid; the branch-name nit in `tasks.md` rides along as a known cosmetic note.
- **`tests`** (optional, non-blocking): one `cmd`/`doctor` table case feeding a padded name through `--check`/doctor to pin the round-2 refusals at the read paths (F3).
- **Vault**: no new promotion needed; lesson 244 already carries both rules the three rounds produced ("bounded by the writer's record", "a read path must share what the write refuses").

**Verdict: PASS. `dotf spec archive` is advisable in the current state** — contract digests match `review-request.json`, no draft tags remain, and AC6's box evidence is already recorded in `verification.md`.
