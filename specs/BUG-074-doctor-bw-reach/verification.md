---
tags: [spec, verification, templates]
created: "2026-08-13"
---

# Verification - BUG-074-doctor-bw-reach

## Evidence

- [x] **Severity keyed to exposure** -> commit `145516c`, test
      `TestBWReach_SeverityFollowsExposure` (same vault state asserted in both
      directions: `live=0` advisory, `live=3` exactly one FAIL naming the count).
- [x] **Stale `lastSync` warns before the token dies** -> test
      `TestBWReach_StaleSyncWarnsWhileStatusLooksHealthy`, which fixes the vault
      at `status: locked` with a 45d-old sync — the incident's own shape, and the
      state that read as healthy for 45 days.
- [x] **Reach proven by an authenticated call, not PATH presence** -> test
      `TestBWReach_UnlockedVaultProvesReach` (pass) and
      `TestBWReach_UnlockedButSyncFailsIsAFail` (fail, including the assertion
      that a Node stacktrace collapses to one line).
- [x] **Mutation-tested in both directions** -> see below; plus the live run.

## Test status

- Test suite: `go test ./...` -> every package `ok`, no regressions.
- Lint: `golangci-lint run` -> `0 issues`, run with the **pinned** 2.12.2 from
  `versions.conf` (BUG-071: a local binary on another major reports 0 issues on
  code CI rejects).
- Spec gate: `./scripts/check-spec-gate.sh --base-ref origin/main --head-ref HEAD
  --explain` -> 220 production LOC counted, tests correctly excluded (205 + 5).

**Mutation test** — each behaviour broken deliberately, the guarding test
observed going red, then reverted.

Round 1 mutated only the *consumers*. The adversarial review showed that was not
enough: mutating the severity **producer** (`s.Backend == "bw"` →
`"bitwarden"`, a predicate that can then never match) left the entire `cli/`
suite green — 13 packages ok, exit 0. The seam introduced for testability had
left the real implementation with no coverage at all, and the row that claimed
"severity forced to always-advisory → detected" had mutated `if live > 0`, not
the code that produces `live`. Round 2 adds the producer tests and re-runs the
full battery, including the reviewer's own mutant:

| Mutation | Round 1 | Round 2 |
|---|---|---|
| severity consumer forced to always-advisory | detected | detected |
| staleness warning never fires | detected | detected |
| claims reach without running `bw sync` | detected | detected |
| **producer: `s.Backend == "bitwarden"`** (reviewer's) | **SURVIVED** | detected |
| **producer: counts every entry** | not tested | detected |
| **producer: silent fallback to the deployed copy** | not tested | detected |
| **bounded exec: deadline ignored** | not tested | detected |
| **clock: no negative-skew guard** | not tested | detected |

Two of those round-2 mutants describe defects the fixes introduced or exposed,
not merely untested paths:

- **Silent fallback to the deployed registry.** Writing the producer test
  revealed that `env.ResolveRegistryPath` falls back to `~/.dotfiles` when the
  checkout registry is absent — so the counter could read the stale mirror, the
  exact failure the design section claims to prevent. Fixed by reading
  `env.RepoRegistryPath` (checkout-only, fails loud).
- **Unbounded network exec.** `bw status` / `bw sync` ran through a seam with no
  deadline inside a diagnostic that terminates `setup-linux.sh`. Fixed by
  `CommandOutputBounded`, tested against the production closure rather than the
  fake.

**Live smoke**, built binary against the real vault — both states observed:

```text
# locked vault (no BW_SESSION):
[Bitwarden reach (live secrets SSOT)]
  [ OK ] Bitwarden synced 0d ago
  [INFO] vault locked — token not exercised; `export BW_SESSION=$(bw unlock --raw)` before re-running to prove reach

# unlocked vault (operator session exported):
[Bitwarden reach (live secrets SSOT)]
  [ OK ] Bitwarden synced 0d ago
  [ OK ] Bitwarden reach verified (authenticated sync round-trip)
```

The originating failure was also captured live before the fix: with the vault
logged out, the exact call the resolver makes returns
`exit 1 / stdout empty / stderr "You are not logged in."`.

## Decisions made during implementation

- **`bw sync` as the deep probe, accepting that it mutates state.** Every
  local-cache read (`bw list`, `bw get`) passes against a dead token, so none of
  them proves reach. `sync` exercises the token refresh — the exact path that
  failed.
- **The keep-alive claim was withdrawn, not defended** (round-1 review's open
  Question). Justifying the side effect as "a periodic `dotf doctor` is the
  keep-alive that would have prevented this outright" required a cadence that
  does not exist: tier 3 runs only on an unlocked vault, the resting state is
  locked, and nothing schedules doctor. The renewal is opportunistic. The
  prevention claim was reassigned to tier 2, which fires on a locked vault with
  no session at all — which is also why 30d had to be chosen below the observed
  45d rather than at some round number.
- **Severity source is the checkout registry, not `cfg.DotfilesDir`.** The first
  draft used the existing `loadRegistry(cfg)` helper, which reads the deployed
  `~/.dotfiles` copy. That copy demonstrably lags the checkout (doctor has a
  section for the drift), so it would have reported zero bw-backed secrets while
  the checkout was actively flipping entries — pinning severity at advisory
  exactly as exposure began, i.e. rebuilding the #635 drift bug inside the fix
  for a drift-class bug. Introduced a `BWBackedSecrets` seam reading
  `env.RepoRegistryPath` instead — checkout-only, and deliberately not
  `env.ResolveRegistryPath`, whose deployed-copy fallback would have restored
  the same stale count on any machine whose checkout registry is missing (see
  the round-2 mutant "producer: silent fallback to the deployed copy").
- **Separate section rather than extending `checkSecretsTooling`.** Keeps the
  existing check's tests untouched and lets a reader see at a glance that reach
  is a distinct claim from presence.
- **`firstLine` reused from `checks_vault_hooks.go`** rather than redeclared;
  the new `bwFailDetail` wraps it to fall back to the exec error, because `bw`
  fails in two shapes (a clean line when logged out, a Node stacktrace on an
  expired token).

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — a health check that
      reads local state proves liveness of nothing: `bw status` reported a
      healthy-looking `locked` for 45 days after the server had expired the
      token, so the check must exercise the remote path or measure elapsed time
      since one succeeded.
- [ ] ADR-worthy decision? **no** — implements ADR-028's existing model; decides
      nothing new about it.
- [ ] New pattern candidate for `00_meta/patterns/`? **no** — the general form
      ("verify behaviour, not representation") is already #852's subject; this is
      one application of it, not a new pattern.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-074-doctor-bw-reach/` -> `specs/archive/BUG-074-doctor-bw-reach/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
