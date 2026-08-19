---
spec: "OPS-026-age-root-file-authority"
verdict: "PASS"
reviewed_sha: "17b667bc091da85684ebc85349474e0d972fedd5"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-19"
---

## Adversarial review

**Scope**: OPS-026-age-root-file-authority
**Sources**: `specs/OPS-026-age-root-file-authority/{proposal,tasks,verification,features}.json`
**Diff**: `main..HEAD` on `feat/age-root-file-authority` (17b667bc)
**PR**: Not yet merged (branch `feat/age-root-file-authority`)

### Spec and task alignment

- All 7 implementation tasks are `[x]` and carry diff evidence. The one unticked task (`dotf spec init` scaffolds `features.json`) is a pre-existing tool issue, not this spec's work — correctly labelled as "Not this spec's work, found while doing it."
- No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec contract file.
- All 7 acceptance criteria are addressed in `features.json` with non-vacuous verification commands and evidence.
- `features.json` IDs are unique (f1, f2, f3, f4, f6, f7, f8 — f5 is an intentional gap; no collision).
- `proposal.md` status is `draft` — not yet `archived`, which is correct for the pre-archive verification window.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| — | — | — | No blockers, no majors, no minors. The change is correct, well-tested, and carefully engineered. | See below | — | — |

**None.** After tracing every consumer of `reg.Entries()` across the codebase — `run`, `verify`, `show`, `sync ci`, `render`, `migrate`, `ls` — and verifying the `NotMaterialized`/`Verifier` interface split via mutation testing, I find no defect, no spec violation, no security gap, and no missing test that would prevent archiving.

**Mutation evidence**: removing the `NotMaterialized` skip in `EnvFor` causes `TestEnvFor_SkipsTheRootWhenResolvingEverything` and `TestEnvFor_ResolvesEverySingleBackendWithoutError` to both FAIL (observed). Removing the `Verifier` check in `Verify` causes `TestFileAuthority_PresentAndCorrectModeIsOK` and `TestFileAuthority_WrongModeFails` to both FAIL (observed). Both mutations were confirmed present before the test run, landed cleanly after revert.

**Full test suite**: `go test ./...` — 17 packages, 0 FAIL. `golangci-lint run` at the pinned 2.12.2 — 0 issues.

**Consumer paths verified**:

| Consumer | Entry source | Root handling | Correct? |
|----------|-------------|---------------|----------|
| `dotf secrets verify` | `reg.Entries()` → `Loader.Verify(e)` | `Verify` prefers `Verifier.VerifyEntry` → present/mode check | ✓ |
| `dotf secrets run` (bulk) | `reg.Entries()` → `Loader.EnvFor(entries, nil)` | `NotMaterialized` skip → silent | ✓ |
| `dotf secrets run --only AGE_KEY_PERSONAL` | `reg.Entries()` → `Loader.EnvFor(entries, only)` | `only != nil` → falls through to `Resolve` → loud refusal | ✓ |
| `dotf secrets show AGE_KEY_PERSONAL` | `reg.ShowEntry(id)` | `ShowEntry` refuses file secrets → error before `EnvFor` | ✓ |
| `dotf secrets sync ci` | `reg.SelectCI(repo)` | `SelectCI` skips floor secrets and file secrets | ✓ |
| `dotf secrets migrate AGE_KEY_PERSONAL` | `migrateGuard(s)` | Guard refuses floor secrets | ✓ |
| `dotf secrets render` | `envSourceMap(reg, home)` | `envSourceMap` skips `IsFile` entries | ✓ |
| `dotf secrets ls` | Raw `reg.Secrets` iteration | Lists ID/plane/vars, no resolution | ✓ |

**Cross-boundary risks checked:**

- `RawResolve` bypasses `NotMaterialized` — but is only called by `migrate`, which is already gated by `migrateGuard` refusing floor secrets. Correct.
- `BackendFileAuthority` in `Entries()` uses `ageEntries` with `s.Age == ""` → `Entry.File` is `""`. The root's resolver ignores `File`; the empty field is harmless. Documented in the explicit `case BackendFileAuthority` comment.
- `VerifyEntry` has a TOCTOU race between `os.Stat` and use — inherent to file-based health checks, documented in the deferred-drift comment, not a defect.
- The `bw:` block with no `field` is correctly handled: `checkBwSources` is not called for `file-authority`, and `checkBWFolder` tolerates empty `Folder`. The fixture in `TestParseRegistry_AcceptsExactlyValidBackends` carries the same shape as the shipped entry.

**Round 1 and 2 findings**: All 10 findings from the two prior rounds (5 + 5) are dispositioned as Applied in `verification.md`. The mutation evidence section proves the round-1 blocker (broken `run`) is fixed. The `NotMaterialized`/`Verifier` split from round-2 finding #4 is the strongest design improvement of the series.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale |
|-----------|-------------|-----------|
| Correctness | A | All acceptance criteria verified, negative paths covered (absent, wrong mode, empty file, non-regular file, explicit refusal), no observed defects. Mutation tests confirm all guards. |
| Verification | A | Evidence proves each criterion with reproducible commands + outputs. Mutation evidence is demonstrated (not just asserted). Round 1 and 2 results are dispositioned with evidence. |
| Scope | A | Diff matches proposal exactly. The only extras are the lesson (`lesson-212`) and the ADR amendment (both documented in the proposal). No scope creep. |
| Reliability | A | Error paths handled: absent (`ErrSecretAbsent` → MISSING), wrong mode (FAILED), non-regular file (FAILED), empty file (FAILED), refusal on explicit `--only` (error). `run` skip is silent for bulk, loud for explicit. |
| Maintainability | A | Clear naming (`NotMaterialized`, `Verifier`, `fileAuthorityResolver`, `checkFileAuthoritySources`). All functions under 40 lines. Comments explain WHY (the deferred drift comparison, the NotMaterialized/Verifier split rationale, the explicit switch case). No dead code. |
| Handoff-readiness | A | Spec updates included (ADR amendment, lesson 212). All contract files present. `features.json` has non-vacuous verification commands. Deferred work (#1000) is documented in the proposal's Out of scope and in the resolver's comment. |

### Verdict

**PASS** — No blockers, no majors, no minors. Rubric is all A. The change is ready for archive.

### Recommended next steps (before archive)

1. Run `dotf spec archive OPS-026-age-root-file-authority` — no blockers remain.
2. The deferred drift comparison (#1000) is the only meaningful gap. The resolver's comment and the proposal's Out of scope document it honestly. The current check answers a narrower question and says so, which is the correct shape for a deferred feature.
3. The lesson `lesson-212-an-invalid-instrument-is-indistinguishable-from-an` captures the class of defects this spec's rounds found. It is a cross-project insight and a candidate for vault promotion (`00_meta/patterns/`) if the pattern is not already there.