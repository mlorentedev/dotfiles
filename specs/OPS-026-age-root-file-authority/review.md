---
spec: "OPS-026-age-root-file-authority"
verdict: "FAIL"
reviewed_sha: "43e351f2ede472321eb4a230f928652f444f6c4b"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-19"
---

## Adversarial review

**Scope**: OPS-026-age-root-file-authority
**Sources**: `specs/OPS-026-age-root-file-authority/{proposal,tasks,verification}.md`, `features.json`, `cli/internal/secrets/{resolve.go,registry.go,secrets.go,backend_test.go,fileauthority_test.go}`, `secrets/registry.yaml`, `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`

### Spec and task alignment

The implementation matches the spec's intent: `file-authority` is a new backend, `ValidBackends()` includes it, `validateSecret` accepts the right shapes and rejects the wrong ones, the registry entry exists, and `dotf secrets verify` reports it. Tasks are all ticked `[x]`.

However, there is a critical gap between the spec's "Out of scope: `dotf secrets run` exposing the root" — which is a deliberate design choice — and the implementation's side effect of breaking `dotf secrets run` entirely for every invocation without `--only`. The spec's out-of-scope statement describes a *feature limitation* (the root won't be handed to child processes), but the implementation creates a *regression* (the command fails outright). These are not the same thing.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | run | `dotf secrets run` (without `--only`) fails because `EnvFor` iterates ALL entries from `reg.Entries()`, hits the `AGE_KEY_PERSONAL` entry with `Backend: BackendFileAuthority`, and calls `fileAuthorityResolver.Resolve()` which returns an error. The `Verifier` interface added for `Verify` is never checked in `EnvFor`. | `DOTFILES_REPO_DIR="" dotf secrets run -- true` → `Error: AGE_KEY_PERSONAL is the age root: it is not materialized through this facade` (exit 1). The spec's "Out of scope: `dotf secrets run` exposing the root" describes a feature limitation, not a regression. This breaks every `run` invocation without `--only`. | UNTESTED — no test verifies `EnvFor` skips or handles `file-authority` entries | code: `EnvFor` must skip entries whose backend implements `Verifier` (same pattern `Verify` already uses) |
| Major | THEORETICAL | test-coverage | `TestResolversCoverEveryValidBackend` only checks *presence* of a resolver in the map, not *correctness*. `fileAuthorityResolver` is present, so the test passes — but its `Resolve` returns an error, which is the root cause of the Blocker. The test cannot catch the `EnvFor` breakage. | Code read of `backend_test.go:14-25`. The test checks `got[b]` key existence only. | `TestResolversCoverEveryValidBackend` | tests: add a test that `EnvFor(reg.Entries(...), nil)` succeeds for every backend (or at minimum does not return an error when called with all entries) |
| Major | THEORETICAL | test-coverage | `EnvFor` has zero test coverage for `file-authority` entries. The func is the primary consumer of the resolver map, and the gap that lets the Blocker ship. `resolve_test.go` has no `EnvFor` test with `BackendFileAuthority`. | `grep -rn 'EnvFor' cli/internal/secrets/resolve_test.go` returns empty. | UNTESTED | tests: add `TestEnvFor_SkipsFileAuthority` or equivalent |
| Minor | THEORETICAL | maintainability | `Entries()` routes `BackendFileAuthority` through `ageEntries()` as a fallthrough (any non-BW backend hits `ageEntries()`). This works only because `ageEntries()` preserves the original backend in the returned `Entry`. A future backend that is neither BW, age, nor file-authority would silently inherit `ageEntries`'s behavior, which may be wrong. | Code read of `registry.go:426-431` — `if s.Backend == BackendBW { ... ageEntries }` with no explicit case for `BackendFileAuthority`. | `TestResolversCoverEveryValidBackend` (passes) | code: add an explicit `s.Backend == BackendFileAuthority` case in `Entries()` (or document why the fallthrough is safe) |
| Minor | REAL | documentation | `bw:` block on `AGE_KEY_PERSONAL` has no `field` key. The registry.yaml entry works because `checkFileAuthoritySources` does not call `checkBwSources`, so `bw:` is purely informational. This is correct per the spec but is not obvious from reading the registry entry alone — a reader could assume the `bw:` block is a live source. | `secrets/registry.yaml:310`: `bw: { item: AGE-SECRET-KEY-PERSONAL }` — no `field`. Registry comment says "Nothing reads this block yet; it names where the drift comparison of #1000 will look." | UNTESTED | spec (proposal.md): the `bw:` block's purpose is already documented in the registry comment, but the proposal's "What" section could mention that the `bw:` block is tolerated not required. |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | C | `verify` and `ls` work correctly, but `run` without `--only` is broken — a regression on existing functionality. |
| Verification | B | Evidence covers most acceptance criteria with reproducible commands, but the `run` regression is untested and unmentioned. |
| Scope | A | Diff matches the proposal precisely; no scope creep. The run regression is an unintended side effect, not a scope violation. |
| Reliability | C | `run` without `--only` fails entirely. The fix is small (one interface check in `EnvFor`), but the gap shipped. |
| Maintainability | B | Clean structure, clear naming, well-commented. The `Entries()` fallthrough is a minor smell. |
| Handoff-readiness | B | Spec artifacts are present and well-written. The `run` regression is the gap that prevents archive. |

### Verdict
**FAIL** — one **REAL Blocker** (the `run` regression) that is **UNTESTED**. The checker's own rubric says: a single Blocker forces FAIL regardless of rubric grades. The fix is small (add a `Verifier` check to `EnvFor`), but it must be applied, tested with a named test, and verified before this spec can archive.

### Recommended next steps (before archive)

1. **Fix the blocker** — in `EnvFor` (`resolve.go:101`), add a `Verifier` check before calling `r.Resolve(e)`:
   ```go
   if v, isVerifier := r.(Verifier); isVerifier {
       continue // not resolvable through this facade
   }
   ```
   This matches the pattern `Verify` already uses (line 164) and is the minimal, correct fix.

2. **Add a named test** — `TestEnvFor_SkipsFileAuthority` (or extend an existing table) that calls `EnvFor` with a `file-authority` entry and verifies it is silently skipped rather than causing an error. This is the regression test that prevents re-introduction.

3. **Add a test for `EnvFor` with all entries** — a test that `EnvFor(reg.Entries(...), nil)` does not error on any backend, catching the same class of regression for future backends.

4. **Re-run verification** — after fix, confirm `dotf secrets run -- true` succeeds (exit 0) and still injects non-root secrets correctly.

5. **Archive** — once the fix is applied and tested, `dotf spec archive OPS-026-age-root-file-authority` should pass.