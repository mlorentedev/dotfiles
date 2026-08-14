---
id: "BUG-074-doctor-bw-reach"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-13"
issue: "mlorentedev/dotfiles#944"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-074-doctor-bw-reach

## Why

<!-- from issue #944: BUG-074: doctor proves the age floor by behaviour but the live SSOT only by presence -->

`dotf doctor` proved the two secret tiers of ADR-028 with opposite rigour, and
applied the weaker one to the tier that matters more. The age floor — the
*backup* — was proven by behaviour: derive the recipient, encrypt a sentinel,
decrypt it back, compare bytes. Bitwarden — the tier ADR-028 designates the
**live SSOT** — was proven by `sys.has("bw")`, i.e. a binary on `PATH`. So doctor
printed a green `bw (Bitwarden CLI — live secrets SSOT) found` for the 45 days
its refresh token was dead, and the outage surfaced only when an operator ran
`bw unlock` by hand and got `invalid_grant` (HTTP 400). This is the class already
named by #898 (a check never observed failing is not evidence) and #852 (verify
behaviour, not representation).

## What

A new `Bitwarden reach (live secrets SSOT)` doctor section that proves the vault
is *reachable*, tiered by what can be established without an operator present:

1. `bw status` reports `unauthenticated` — and the report names `bw login` as the
   recovery, explicitly ruling out `bw unlock`, whose master-password prompt
   makes an expired token read as a credential problem.
2. `lastSync` older than 30 days warns. This is the tier that catches the real
   failure: local status keeps reporting `locked` long after the server has
   expired the token, so elapsed time since a successful sync is the only
   observable that moves.
3. With an unlocked vault, `bw sync` proves the token against the server. `bw
   list` would not — it is served from the local cache and passes on a dead
   token.

Severity is keyed to real exposure (count of `backend: bw` registry entries), not
to a flat policy: advisory while everything is still on age, FAIL from the first
migrated secret.

## Out of scope

- Automating login/unlock (API-key or `--passwordenv` flows). The check reports
  and instructs; it never handles credentials.
- `bw serve` as the resolution transport — named in #585's AC but not implemented
  anywhere today; tracked there, not here.
- The migration itself (#585) and the `retire` path (#938).

## Risks / open questions

- **`bw sync` mutates state inside a diagnostic.** Accepted deliberately and
  documented in the code: it advances `lastSync` and renews the token. The
  alternative probes are all local-cache reads that pass on a dead token, so
  they prove nothing.

  An earlier draft justified the side effect by claiming it "makes a periodic
  `dotf doctor` the keep-alive that would have prevented this incident
  outright". The adversarial review asked whether any cadence actually
  satisfies that, and the answer is **no** — so the claim is withdrawn rather
  than defended. Tier 3 runs only on an *unlocked* vault, which this same
  document calls the normal resting state, and nothing schedules doctor. The
  renewal is therefore opportunistic: it happens when an operator already has a
  session open, and not at all otherwise.

  The prevention claim belongs to **tier 2**, which needs no session: on a
  locked vault the 30d staleness warning still fires, and 30d < the observed
  45d, so it lands while the token is still renewable. That reassignment is
  what makes the offline tier load-bearing rather than a nicety.
- **Severity source must not be the deployed registry.** Counting through
  `cfg.DotfilesDir` (`~/.dotfiles`) would read the copy that lags the checkout
  during exactly the migration this guards, holding severity at advisory as
  exposure begins — the #635 drift class rebuilt inside the fix for a drift bug.
  Resolved: the seam reads `env.ResolveRegistryPath`, which prefers the checkout.
- **CI must not go red for a headless container.** The first draft of this risk
  claimed "no workflow runs `dotf doctor`", from a grep of `.github/workflows`
  and `verify-setup.bats`. That claim was **false**, and the adversarial review
  caught it. The real chain is `ci.yml` (`integration` job) →
  `tests/Dockerfile.integration:55` (`RUN bash setup-linux.sh`) →
  `setup-linux.sh:1505` (`dotf doctor`). doctor **does** run in CI.

  The conclusion survives, but on two safeguards that had to be verified rather
  than assumed:
  1. The invocation is `dotf doctor || log_warning …` — non-fatal by
     construction, so a FAIL cannot fail the job.
  2. `bw` is not installed in the integration image, so the reach check `Skip`s
     before it can evaluate anything.

  Both are load-bearing. Removing either — making doctor fatal in setup, or
  adding `bw` to the image — turns a migrated registry into a red CI, so
  whichever of those changes lands first must also add a headless escape hatch.

- **Network subprocesses must be bounded.** `bw status` and `bw sync` are the
  only network-bound `CommandOutput` callers in doctor, and plain `CommandOutput`
  has no deadline while the `HTTPGet` seam is capped at 5s. Since doctor is the
  last step of `setup-linux.sh`, an unbounded hang there hangs a bootstrap.
  Resolved: both calls go through a new `CommandOutputBounded` seam
  (`bwStatusTimeout` 15s, `bwSyncTimeout` 45s).

- **The severity producer is the part that can rot silently.** Every check test
  injects the `BWBackedSecrets` seam, and the real predicate is unreachable on
  today's machine because all registry entries are still `age` — so neither CI
  nor a live smoke exercises it. Resolved: `bwBackedSecrets` has its own tests
  against a temp registry, mutation-verified to fail when the predicate breaks.

## Acceptance criteria

- [x] `unauthenticated` is reported, with severity keyed to whether the registry
      actually holds `backend: bw` entries.
- [x] A stale `lastSync` warns before the token dies (30d < the observed 45d).
- [x] With a session available, reachability is proven by an authenticated call,
      not by `PATH` presence.
- [x] Mutation-tested in both directions: each tier observed failing when broken,
      and the passing direction proven live against a real unlocked vault.

## References

- Bitácora board: mlorentedev/dotfiles#944
- Related ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` (the two-tier
  model and the live-SSOT designation), `docs/adr/adr-030` (registry read-side
  resolution, via #635)
- Related issues: #898 (a check never observed failing is not evidence), #852
  (verify behaviour, not representation), #585 (the migration this guards)
