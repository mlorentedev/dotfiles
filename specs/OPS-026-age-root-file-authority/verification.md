---
tags: [spec, verification, templates]
created: "2026-08-19"
---

# Verification - OPS-026-age-root-file-authority

## Evidence

Produced in the session of 2026-08-19, not recalled.

- AC1 -> `go test ./internal/secrets/ -run 'FileAuthority|AcceptsExactlyValidBackends'` — 8/8
- AC2 -> `dotf secrets ls` — `AGE_KEY_PERSONAL  floor  AGE_KEY_PERSONAL`
- AC3 -> `dotf secrets verify` — **34 ok, 0 missing, 0 failed** (baseline before the
  change: 33 ok, 0 missing, 0 failed)
- AC4 -> mutations observed live, real root never touched (scratch registry, throwaway key):

  ```
  mode 0600            -> 1 ok, 0 missing, 0 failed
  chmod 0644 (stat: 644 confirmed before the run)
                       -> 0 ok, 0 missing, 1 failed   (non-zero exit)
  file removed (absent confirmed)
                       -> 0 ok, 1 missing, 0 failed   (MISSING, not FAILED)
  ```

- AC6 -> `TestFileAuthority_ResolveRefuses` + `TestResolversCoverEveryValidBackend`
  (the drift comparison moved to #1000; see the proposal's Out of scope)
- AC7 -> `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md:146`

## Test status

- `go build ./... && go vet ./... && go test ./...` — 17 packages ok, 0 FAIL
- `golangci-lint run` at the pinned 2.12.2 (matches `versions.conf`) — 0 issues
- No regressions: the 33 pre-existing secrets report exactly as before

## Decisions made during implementation

- **The root is its own backend, not an exemption in the validator.** A named
  carve-out in `checkAgeSources` was the smaller diff and was rejected: the next root
  (a second machine, a rotated identity) would ask for the same special case again.
- **`Resolve` refuses rather than returning the key.** Materializing the root through
  the same facade as the secrets it protects widens the blast radius. The refusal is
  pinned by a test, because it is the kind of thing a later change "fixes" helpfully.
- **Absent is MISSING, wrong mode is FAILED.** A fresh checkout has no key yet and
  must not look broken; a present key readable by others is a real defect.
- **`bw:` without a folder.** The ratified taxonomy has no `floor` folder and its
  comment says floor secrets carry no `bw:` block at all. ADR-028 distinguishes
  authority from convenience copy, and the comment does not. Rather than amend a
  ratified taxonomy unilaterally, the block is unfoldered and the tension is recorded.

## Promotion candidates

- The shape "a check that cannot answer the real question yet must say so where the
  check lives" — `fileAuthorityResolver`'s comment about the deferred drift
  comparison. Same family as lesson 212.

## Round 1 adversarial review — disposition

`nan/deepseek-v4-flash`, verdict **FAIL** on `43e351f`. Five findings, all five applied.
Recorded here rather than only in the transcript, because a verdict nobody dispositioned
is the silent green this repository spent 2026-08-17/18 cataloguing.

| # | Severity | Finding | Disposition |
|---|---|---|---|
| 1 | **Blocker** (REAL) | `EnvFor` resolves every entry, so the root's refusing resolver broke `dotf secrets run` outright for every invocation without `--only`. The `Verifier` seam was consulted by `Verify` and not by `EnvFor`. | **Applied.** Reproduced first: branch binary errors, main's exits 0. `run` now SKIPS a self-verifying entry when `only == nil` and REFUSES it when named explicitly — a bulk request nobody aimed at the root gets silence, an explicit one gets a loud refusal. |
| 2 | Major (THEORETICAL) | `TestResolversCoverEveryValidBackend` asserts a resolver EXISTS, never that it works inside the loop it lives in — which is why a green suite shipped the Blocker. | **Applied.** `TestEnvFor_ResolvesEverySingleBackendWithoutError` puts one entry of every declared backend through `EnvFor`. |
| 3 | Major (THEORETICAL) | `EnvFor` had zero coverage for `file-authority`. | **Applied.** Two cases, skip and explicit refusal. |
| 4 | Minor (THEORETICAL) | `Entries()` routed the backend by fallthrough; a future backend would inherit `ageEntries` silently. | **Applied.** An explicit `switch` case with the reason, citing REFACTOR-012. |
| 5 | Minor (REAL) | The `bw:` block has no `field` and nothing reads it; a reader could take it for a live source. | **Applied.** Stated in the proposal's *What* and at the point of use in the registry. |

**Mutation evidence for the new guards:** with the `EnvFor` skip removed (confirmed
present: `grep -c` of the guard line returns 0), `TestEnvFor_SkipsTheRootWhenResolvingEverything`
and `TestEnvFor_ResolvesEverySingleBackendWithoutError` both FAIL; both pass after revert.
They would have caught the Blocker.

**What the round teaches, beyond this spec:** `Verify` was green, the unit suite was
green, and the shipped command was broken — because a seam was added to one consumer of
the resolver map and not the other. That is REFACTOR-012's shape, which this very file
cites, arriving in the change that cites it. Static coverage of "does a resolver exist"
could not see it; only exercising the loop could.
