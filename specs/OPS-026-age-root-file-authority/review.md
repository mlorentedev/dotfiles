---
spec: "OPS-026-age-root-file-authority"
verdict: "PASS WITH GAPS"
reviewed_sha: "07650264443fb189810e6a557ba51734f737e238"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-19"
---

## Adversarial review

**Scope**: OPS-026-age-root-file-authority
**Sources**: `specs/OPS-026-age-root-file-authority/{proposal,tasks,features}.json`, `verification.md`, `cli/internal/secrets/{resolve.go,registry.go,secrets.go,backend_test.go,fileauthority_test.go}`, `secrets/registry.yaml`, `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`

### Spec and task alignment

The implementation matches the spec's intent: `file-authority` is a new backend, `ValidBackends()` includes it, `validateSecret` accepts the right shapes and rejects the wrong ones, the registry entry exists, and `dotf secrets verify` reports it. All tasks are ticked `[x]`. The five findings from the round 1 adversarial review have all been applied and dispositioned with evidence in `verification.md`.

Two spec-artifact inconsistencies were found (see findings below): the proposal's AC6 wording is stale ("no Loader entry" → the implementation has a resolver that refuses), and the `features.json` has duplicate IDs.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec (proposal.md) | AC6 wording is stale: the proposal says "the resolver-coverage test states in its own text why `file-authority` has **no Loader entry**" but the implementation has a Loader entry — a `fileAuthorityResolver` whose `Resolve` refuses. The `features.json` f6 entry correctly reflects the actual design ("file-authority has a Loader entry, and it is a resolver that REFUSES rather than a missing one"), but the proposal was not updated to match. | `specs/OPS-026-age-root-file-authority/proposal.md` line 37: "no Loader entry" vs `features.json` f6: "a resolver that REFUSES". The intent (AC6 is satisfied) is met; the wording is what is wrong. | UNTESTED — this is a documentation issue, not a code test | spec (proposal.md): update AC6 wording to match the actual implementation |
| Minor | REAL | spec (features.json) | Duplicate feature ID: two entries carry ID `f7`. The first covers AC7 (ADR amendment), the second covers AC8 (EnvFor skip behavior, added by the round 1 review). The second should be `f8` or a distinct slug. `dotf spec archive` may not check for unique IDs, but duplicate IDs in a data file that `contractFiles` includes are a data integrity issue: a reader or tool processing the array by ID sees only one of the two. | `python3 -c 'import json; [print(i["id"]) for i in json.load(open("specs/OPS-026-age-root-file-authority/features.json"))]'` shows two `f7` entries. | UNTESTED — no test validates unique feature IDs | spec (features.json): change the second `f7` to `f8` |
| Minor | THEORETICAL | code (resolve.go) | `VerifyEntry` checks for directories (`fi.IsDir()`) but not for other non-regular file types (FIFO, device, socket, etc.). A special file at the key path with mode 0600 would pass the mode check and report OK, even though `age --decrypt` would fail on it. | Code read of `VerifyEntry` in `resolve.go:280-307`. Only `os.Stat` + `fi.IsDir()` guards the file type — no `fi.Mode().IsRegular()` check. | `TestFileAuthority_PresentAndCorrectModeIsOK` (creates a regular file, the only path exercised) | code: add `!fi.Mode().IsRegular()` check to `VerifyEntry` |
| Minor | THEORETICAL | code (resolve.go) | The `EnvFor` skip (`resolve.go:122`) uses `r.(Verifier)` as a proxy for "should not be resolved." This is correct for the current single implementer (`fileAuthorityResolver`), but if a future backend implements `Verifier` for a different reason (e.g., a read-only health check that also resolves), the skip would silently drop that backend's secret from bulk resolution. | Code read of `resolve.go:122-124`. The `Verifier` interface doc says it is "the optional half of Resolver, for a backend whose health question is not 'did it resolve'" — not "for a backend that should not be resolved." | `TestEnvFor_SkipsTheRootWhenResolvingEverything` (passes, but targets the specific backend, not the interface) | code: either check `BackendFileAuthority` specifically, or add a method to `Verifier` (e.g., `SkipEnvFor() bool`) that makes the intent explicit |
| Minor | THEORETICAL | tests | The test fixture for `file-authority` in `TestParseRegistry_AcceptsExactlyValidBackends` does not include a `bw:` block. The actual registry entry (`AGE_KEY_PERSONAL`) has `bw: { item: AGE-SECRET-KEY-PERSONAL }` without a `field` — this is by design (the spec says `bw:` is tolerated, not required, and not a source), but there is no unit test that explicitly verifies that a `bw:` block without a `field` on a file-authority secret is accepted by the parser. | `backend_test.go:65`: the fixture for `BackendFileAuthority` has no `bw:` block. The actual registry parses correctly (verified via `dotf secrets verify`), but the unit trip-wire is absent. | `TestParseRegistry_AcceptsExactlyValidBackends` (the file-authority fixture has no `bw:` block) | tests: add a `bw:` block to the file-authority fixture in `TestParseRegistry_AcceptsExactlyValidBackends`, or add a dedicated test case |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All acceptance criteria met; two spec-artifact quality issues (stale AC6 wording, duplicate feature ID) |
| Verification | B | Reproducible evidence for all criteria; features.json has duplicate IDs, proposal AC6 stale |
| Scope | A | Diff matches proposal precisely; no scope creep |
| Reliability | B | Error paths handled; `VerifyEntry` misses regular-file type guard; `EnvFor` skip is interface-based rather than backend-specific |
| Maintainability | A | Clear naming, well-commented, explicit switch cases, clean structure |
| Handoff-readiness | B | Spec artifacts present; stale AC6 wording and duplicate feature ID need cleanup before archive |

### Verdict
**PASS WITH GAPS** — no Blocker or Major findings; all rubric dimensions are B or above. The Minor findings are real but do not prevent archive — they are documentation and quality issues rather than correctness or security defects. The duplicate feature ID and stale AC6 wording should be fixed before archive to produce a clean artifact trail.

### Recommended next steps (before archive)

1. **Fix the duplicate feature ID** — change the second `f7` to `f8` in `features.json`. The entry covers AC8 (EnvFor skip behavior, added by round 1 review).
2. **Update AC6 wording in proposal.md** — change "no Loader entry" to "has a Loader entry whose resolver refuses" to match the actual implementation.
3. **Add a regular-file type guard to `VerifyEntry`** — add `!fi.Mode().IsRegular()` after the `fi.IsDir()` check in `resolve.go`. This is a theoretical concern but is a legitimate hardening step.
4. **Add a `bw:` block to the file-authority fixture** in `TestParseRegistry_AcceptsExactlyValidBackends` to explicitly test that the parser accepts a `bw:` block without a `field` on a `file-authority` secret.
5. **`dotf spec archive` is advisable** after the above spec-artifact fixes are applied (items 1 and 2). The code-level findings (items 3 and 4) are hardening recommendations, not archive gates.