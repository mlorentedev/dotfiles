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

**Mutation test** — each tier broken deliberately, the guarding test observed
going red, then reverted:

| Mutation | Result |
|---|---|
| severity forced to always-advisory | detected (test red) |
| staleness warning never fires | detected (test red) |
| claims reach without running `bw sync` | detected (test red) |

**Live smoke**, built binary against the real vault — both states observed:

```
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
  failed. Its side effect is treated as a feature and documented in the code: a
  periodic `dotf doctor` becomes the keep-alive that prevents the expiry.
- **Severity source is the checkout registry, not `cfg.DotfilesDir`.** The first
  draft used the existing `loadRegistry(cfg)` helper, which reads the deployed
  `~/.dotfiles` copy. That copy demonstrably lags the checkout (doctor has a
  section for the drift), so it would have reported zero bw-backed secrets while
  the checkout was actively flipping entries — pinning severity at advisory
  exactly as exposure began, i.e. rebuilding the #635 drift bug inside the fix
  for a drift-class bug. Introduced a `BWBackedSecrets` seam reading
  `env.ResolveRegistryPath` instead.
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
