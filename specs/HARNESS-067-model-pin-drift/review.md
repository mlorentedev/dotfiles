---
spec: "HARNESS-067-model-pin-drift"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "7a018fcdaf9b2c73ce24405f3ade2c8e152f5395"
reviewer: "nan/glm5.3-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: HARNESS-067-model-pin-drift (round 2 — the last under the two-round cap)
**Sources**: `specs/HARNESS-067-model-pin-drift/{proposal,tasks,verification}.md` + `features.json`; diff `cba6be2f...HEAD` as resolved by the launcher. That base is a long-lived worktree base carrying many other merged specs, so the spec's own change was isolated to its commits (`d7e5ddcc` initial implementation, `a18981e2` persona-gate fix, `0a584d4e` mimo-v2.5 retirement, `21c3a0f0` round-1 fixes, `1cf4414c`/`2f816422`/`7a018fcd` spec docs) and the pin-scope files: `cli/internal/harness/model_pins*.go`, `cli/internal/doctor/checks_model_pins*.go`, `harness/model-pins.json`, `harness/model-map.json`, the declared pin-site files, and the round-1 review artifacts (`review-round-1.md`, `review-request-round-1.json` — confirmed byte-preserved).

### Spec and task alignment

- All 12 ACs ticked in `proposal.md`; round-1 findings 1–7 have a disposition table in
  `verification.md` with named proofs. Finding 8 (hardcoded `retiredProvider` list) is
  explicitly declined — acceptable; the list is two entries with recorded decisions behind
  both, and the message still reports the value verbatim.
- Every verification command in `features.json` names a test that exists (21 test funcs
  checked by name — `go test -run` with a matching-nothing name still exits 0, so I grepped
  each `func Test…` rather than trusting exit codes). All pass.
- Tasks "Round-2 independent review" and "Independent adversarial review before archive" are
  correctly **unticked** — this review is that item.
- Round-1 fix verification reproduced independently (see findings/evidence below): prefix-drop
  mutation and map-entry deletion both fail the repo guard; an emptied registry fails loudly
  in both sweeps and never renders as a clean result.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | extraction | `toml-key` extraction takes the **first** `key = "value"` match in the whole file regardless of TOML section, while `regex` kind errors on multiple matches. If a second `model` key is ever added in `.pr_agent.toml`'s `[pr_reviewer]` or `[pr_code_suggestions]` section (PR-Agent supports per-section model overrides), it is silently unchecked — and the AC9 sweep covers TOML keys **by name only** (`located[file+" "+m[1]]`), so the new key would read "located" while no value of it is ever checked. Today the file has exactly one `model` key (verified), so nothing is currently wrong. | `extractTOMLKey` uses `FindStringSubmatch` (first match) vs `Extract`'s regex branch, which errors on >1 match for `kind: regex`; `TestEveryModelKeyInATOMLSiteIsDeclared` keys coverage by key name, not by section/value | UNTESTED | code + tests (make `toml-key` error on multiple matches like `regex` does, or anchor locators to a section) |
| Minor | THEORETICAL | doctor | A deployed site file that exists but is **unreadable** (permissions, EIO) is `Warn`ed but not counted as a site or a finding; if it is the only deployed site, the run then ends `Skip "no deployed pin sites present"`, which understates a read failure as an absence — the same confusion C15 bans for the registry. | code read of `checkModelPins` (non-`IsNotExist` branch continues without `sites++` or `findings++`); `TestModelPinsSkipsWhenNotDeployed` covers the absent case only | UNTESTED | code (count the Warn as a finding, or fail on a present-but-unreadable site) |
| Minor | SPECULATIVE | catalog | `staleSnapshotOf` reports any catalog id of the form `<declared-id>-<digits/dashes>` as a frozen snapshot of the shorter declared id. A *distinct* live model whose id extends a declared one with a numeric tail (e.g. a hypothetical `qwen3.8-flash-2`) would be misreported as a stale snapshot of `qwen3.8-flash`. Provider `-MMDD` naming makes collisions plausible but none exists in the current map. | code read of `staleSnapshotOf`; no repro against current ids | UNTESTED | — (surface only; a WARN on a catalog row, not gating) |
| Question | — | verification | AC5/AC6's **live-machine** reproduction was not independently re-run this session: a review session deliberately did not execute full `dotf doctor` against the live `$HOME` (doctor performs other operations, incl. a vault sync path, that a reviewer should not trigger). The behaviors are covered by the named unit tests, which pass; the live findings (four WARNs on `~/.pi/agent/settings.json`) are accepted as previously recorded, not re-measured. | this session's constraint; `TestModelPinsReportsAFrozenSnapshot`, `TestModelPinsDistinguishesARetiredProvider` pass | those two named tests | none (recorded decision to accept unit-level proof) |

**Independent mutations run this session (all reverted, tree clean after):**

1. `.pr_agent.toml` `model = "openai/mimo-v2.6-flash"` → `"mimo-v2.6-flash"` (round-1 finding 4's shape): `TestEveryRepoRoutingPinResolvesInTheMap` FAILs with the misspelling diagnostic. Confirms the round-1 fix landed.
2. Deleted `tiers.mid.gemini` from `harness/model-map.json`: the same test FAILs naming `agy-model`'s normalized id as undeclared. Confirms the map additions are load-bearing, not decorative.
3. Emptied `sites: []` in `harness/model-pins.json`: both AC9 sweeps FAIL loudly ("declares no sites — refusing to report a clean sweep over nothing"). Confirms AC7/C15 in the live path, not just in fixtures.

**Security/code checklist over the diff:** no injection (locators are compiled as regexes with `QuoteMeta` for TOML keys; file paths come from the committed registry), no secrets, no writes from the check (`TestModelPinsNeverWrites` compares content *and* mtime), no blocking I/O in async paths (n/a), functions small and single-purpose. The JSONC comment stripper only removes whole-line `//` and block comments — trailing `//` inside URL strings is preserved, which is the correct conservative choice.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All ACs verified, guard proven non-vacuous by injection and by my own mutations; the one negative-path gap is the multi-section TOML key (finding 1). |
| Verification       | A | Every `features.json` command names a real, existing, passing test; RED-then-GREEN evidence recorded per round-1 finding. |
| Scope              | B | No creep in the spec's own commits; launcher base spans shared history, so scope was judged on the spec's commits and pin-scope files. |
| Reliability        | A | Fail-loud on absent/unparseable/empty registry and rotted locators, never-writes asserted, zero-match is an error not a clean pass. |
| Maintainability    | A | Small functions, two-sets-not-one and fail-loud decisions documented in place with their incident history. |
| Handoff-readiness  | A | Disposition table for all round-1 findings, phase-2 and out-of-scope items recorded on tickets, round-1 artifacts preserved. |

### Verdict
PASS WITH GAPS

No Blocker and no REAL Major: the single Major is THEORETICAL (requires a future config edit;
today's `.pr_agent.toml` has exactly one `model` key), and the Minors are THEORETICAL/SPECULATIVE.
All rubric dimensions B or above. The Major is tracked with the disposition below and does not gate.

### Recommended next steps

Disposition these in `verification.md` (applied / ticketed / declined) or carry them into a
follow-up ticket — the contract set is closed by this verdict, and none of these require a
contract edit:

- **Finding 1 (toml-key first-match)** — ticket it. Preferred fix: make `kind: toml-key` error
  when the key matches more than once, mirroring `kind: regex`, plus a named regression test
  that adds a `model = "…"` line under `[pr_reviewer]` in a fixture and expects the guard to
  fail or error loudly. This is the same "sweep says covered, extractor sees nothing" class
  the AC9 sweep exists to close.
- **Finding 2 (present-but-unreadable deployed site)** — fold into the same follow-up: count it
  as a finding so the run cannot end on a Skip.
- **Finding 3 (snapshot false positive)** — leave as-is; if a numeric-tail model ever lands in
  a catalog, the WARN names the base id it assumed, which makes the misclassification visible.
- **Round-1 finding 8 disposition stands** (declined, hardcoded two-provider list) — no action.
