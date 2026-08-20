---
tags: [spec, verification]
created: "2026-08-15"
---

# Verification - REFACTOR-012-entries-tagged-union

## Evidence

| AC | Proof |
|---|---|
| **AC1** `Entry.SourceID()` + both consumers | `TestEntrySourceID_DistinguishesBackendSources` (6 cases), `TestEnvSourceMap_RejectsTwoBwSecretsExposingOneVar`. Guard observed RED against the pre-fix comparison — see "Guards observed failing" below. |
| **AC2** selection on `validate: github-token`, `Entries()` carries `Validate` | `TestCheckPATExpiry_SelectsBwBackedPAT`, `TestCheckPATExpiry_DistinctBwPATsProbedSeparately`, `TestCheckPATExpiry_ProbesEachSourceOnce`. Both observed RED against the pre-fix predicate and dedupe key. |
| **AC3** resolution via the Loader seam, no ambient-env read | `TestCheckPATExpiry_IgnoresAmbientEnvironment` — an ambient `GITHUB_PERSONAL_ACCESS_TOKEN` is set to a sentinel and the probe must carry the *resolved* token instead. `grep -rn 'Getenv' cli/internal/doctor/checks_pat.go` returns only the `DOTF_PAT_EXPIRY_WARN_DAYS` threshold read. No occurrence of `secrets_refresh` remains in the repo. |
| **AC4** severity taxonomy, 401 still the only FAIL | `TestCheckPATExpiry_Classification` — 12 rows, two of them the new resolution branches, both asserting `Failures() == 0`. `TestCheckPATExpiry_BwLockedSkipsWithoutResolving` asserts a locked/absent vault produces no failure, no resolution attempt, no probe. |
| **AC5** backend SSOT + resolver coverage | `TestResolversCoverEveryValidBackend`, `TestParseRegistry_AcceptsExactlyValidBackends`. |
| **AC6** live | Below. |

## Guards observed failing (Standing Order: a check never observed red is not evidence)

Each fix was reverted in isolation and the guard confirmed to fail, then restored:

1. Selection predicate reverted to `!strings.HasPrefix(e.File, "github.")` →
   `TestCheckPATExpiry_SelectsBwBackedPAT` FAILs with
   `[SKIP] no secrets marked 'validate: github-token' in the registry`.
2. Dedupe key reverted to `id := e.File` →
   `TestCheckPATExpiry_DistinctBwPATsProbedSeparately` FAILs: 1 probe, not 2 —
   the second bw PAT collapses into the first.
3. `render.go` guard reverted to `prev.File != e.File` →
   `TestEnvSourceMap_RejectsTwoBwSecretsExposingOneVar` FAILs: the
   non-deterministic-registry guard does not fire.

## Test status

```
go build ./...   OK
go vet ./...     OK
go test ./...    OK (all packages)
golangci-lint run   0 issues   (v2.12.2 — matches versions.conf's GOLANGCI_LINT_VERSION pin, per BUG-071)
```

## AC6 — live, on this machine

`dotf doctor --verbose`, built from this branch:

```
[PAT expiry]
  [ OK ] github.token: valid, expires in 40 day(s) (2026-09-25)
  [FAIL] github-bitacora-pat/api-token: token invalid or expired (HTTP 401) — rotate it
```

Two secrets probed. Before this change the section probed **none**: the age-backed
one SKIPped because the ambient environment is empty by design, and the bw-backed
one was never selected at all.

**The FAIL is a real finding, not a test artefact.** Confirmed by an independent
path that does not share code with the check:

```
$ dotf secrets run --only BITACORA_PAT -- sh -c 'curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $BITACORA_PAT" https://api.github.com/user'
401
```

Filed separately — see "Decisions made during implementation".

**Daemon-down latency:** NOT verified live. Taking the shared `bw serve` daemon
down would break the three parallel sessions using it on this machine, so the
claim rests on `TestCheckPATExpiry_BwLockedSkipsWithoutResolving`, which asserts
that a `locked` or `absent` state produces **zero** resolution attempts — the
latency risk (BUG-080's ~1.5s-per-secret shellout) is structurally unreachable
rather than merely fast. Stated as a skip, per Definition of Done.

## Decisions made during implementation

- **Selection moved to `validate: github-token`, not a widened filename match.**
  The marker is already the registry's declared "this is a probeable GitHub
  token" and already gates `secrets sync ci`'s pre-upload check. Matching bw item
  names by convention would have reproduced the original defect one layer up.
  `GITHUB_PERSONAL_ACCESS_TOKEN` was missing the marker and gained it here;
  without that, switching predicates would have silently narrowed coverage.
- **No new FAIL branch (AC4).** An unresolvable declared secret is arguably a
  setup defect, but `doctor` is the last step of `setup-linux.sh` and a new
  non-zero branch would fail the bootstrap of every mid-migration machine.
  Changing a fleet-wide exit contract is a policy decision, not a side effect of
  repairing a tagged-union consumer. The concern that motivated the stricter
  option — an unmonitored token passing in silence — is met in the message
  instead: both resolution branches say outright that the expiry is **not** being
  monitored, so a SKIP cannot be read as "nothing to do".
- **`type Backend string` rejected for this PR.** It buys type-safety at
  boundaries but not exhaustiveness: the repo has no `.golangci.yml` and runs
  golangci-lint with default linters, so `exhaustive` is not enabled and would
  have to be adopted first. Constants + the resolver-coverage test are the honest
  guard. Recorded in the proposal's Out of scope.
- **Serve-only Bitwarden reader in doctor.** The CLI's own wiring
  (`BWFallbackReader`) falls back to shelling out to `bw`; doctor does not, and
  gates on `BWServeStatus` before resolving at all.
- **`SelectCI`'s manual `e.Validate = s.Validate` removed.** The flattening now
  carries it, which is the actual fix — that hand-assignment was the only reason
  any consumer saw a populated `Validate`, and the reason no other did.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md` — **yes**: a health check whose
      precondition the architecture forbids reports SKIP forever, and a SKIP is
      indistinguishable from "nothing to check". The check was not weakened by
      the migration; it was already unreachable and the migration only widened
      the blind spot.
- [ ] ADR-worthy decision — no. ADR-028 already governs; this conforms to it.
- [ ] New `00_meta/patterns/` candidate — no. The tagged-union-consumer class is
      already covered by the existing verification-fails-toward-unproven family
      and has not recurred outside this repo.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/REFACTOR-012-entries-tagged-union/`
- [ ] Bitácora #972 closed with the PR link (ADR-018)
- [ ] `/adversarial-review` run and `review.md` signed by a pool model (never the implementer)
- [ ] Lesson written to `docs/lessons.md`
