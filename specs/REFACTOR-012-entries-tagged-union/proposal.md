---
id: "REFACTOR-012-entries-tagged-union"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-15"
issue: "mlorentedev/dotfiles#972"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, doctor, refactor]
template_version: "1.0"
---

# REFACTOR-012-entries-tagged-union

<!-- from issue #972: REFACTOR-012: Entries() is a tagged union and three consumers still read File without checking Backend -->

## Why

`Registry.Entries()` returns a **tagged union**: each `Entry` carries `Backend`,
and only the age backends populate `File`. A consumer that reads `File` without
checking the tag gets `""` — Go's zero value, so the mistake is silent rather
than a type error. BUG-077 (#969, #973) was one such consumer and produced 56
false FAILs the day 28 secrets migrated to Bitwarden; half its output told the
user to delete the ADR-028 DR floor. That fix repaired the instance. This repairs
the class.

Sweeping the other callers found the same assumption in two more, plus a third
defect that only became visible while reading them: the PAT-expiry check has not
monitored a single token since the login-time loader was retired, because it
looks for the token in the ambient environment that ADR-028 exists to keep empty.

## What

Four changes in one PR — one logical change (every `Entries()` consumer
dispatches on the tag), with the guard for each instance beside it.

**1. `Entry` gains a backend-qualified source identity.** Two call sites compare
or key on `File` alone to mean "same underlying secret":
`render.go`'s non-deterministic-registry guard (`prev.File != e.File`) and
`checks_pat.go`'s dedupe (`idx[e.File]`). For bw entries both operands are `""`,
so the first never fires and the second collapses every bw secret into one. A
single `Entry.SourceID()` — derived from the union's own fields — fixes both,
and puts the identity where the union is defined rather than in each consumer.

**2. The PAT-expiry check selects on a declared marker, not on a filename
convention.** Today: `!e.IsFile && strings.HasPrefix(e.File, "github.")`, which
is the naming convention of the *age blob*. A bw entry has no `File` at all, so
`BITACORA_PAT` — migrated in #961 — is invisible to it. The registry already
carries the right marker for this: `validate: github-token`, declared on
`RELEASE_TOKEN` and on `BITACORA_PAT`. Selecting on it is backend-neutral and
deletes the dependency on blob naming. Two consequences, both part of this
change: `Entries()` must carry `Validate` (today only `SelectCI` sets it, by
hand — the same producer/consumer asymmetry in a third place), and
`GITHUB_PERSONAL_ACCESS_TOKEN` must gain the `validate: github-token` line it is
missing, or the switch would silently narrow the probed set.

**3. The PAT-expiry check resolves its token through the Loader.** It currently
reads `sys.Getenv(...)`. Under ADR-028 secrets are never exported into the
ambient shell — that is the architecture, not a misconfiguration — so on a
correctly configured machine every PAT resolves to `""` and the check SKIPs,
reporting *"not in environment — run secrets_refresh"*. `secrets_refresh` exists
nowhere in the repo; it was retired with the loader. The check is therefore
dead **and** its remediation is unfollowable. Resolution moves to the sanctioned
read path — a `System` seam over `Loader.EnvFor`, matching the existing
`AgeRoundTrip` / `BWBackedSecrets` seams — so the check works on the machine the
architecture actually produces.

**4. The backend tag gets an SSOT.** `"bw"` appears as a bare literal in
`registry.go` (×2), `github.go`, `checks_deploy.go` and `checks_bw_reach.go`,
and the set of valid backends is spelled out independently in two switches in
`registry.go` and in the `resolvers()` map in `resolve.go`. Exported constants
plus one canonical list — feeding the parser's validation, and bound to the
Loader's resolver map by a coverage test — turn "a backend the parser accepts
has no resolver" into a red test rather than a runtime `unknown backend` error
on someone's machine. Go cannot make it a compile error without an exhaustive
switch, which this repo's lint configuration does not provide (see Out of
scope); a test is the honest guard, not a weaker substitute for one.

## Out of scope

- **`type Backend string`.** Considered: it types the field at every boundary and
  is the more idiomatic shape. Rejected for this PR because the safety it is
  usually chosen for — exhaustive switches — is not available here: the repo runs
  `golangci-lint` with default linters and has no `.golangci.yml`, so
  `exhaustive` is not enabled and would have to be adopted first. Constants plus
  the resolver-coverage test deliver the guard this class actually needs; the
  typed field is a separate, opt-in change.
- **Re-tracking the DR floor of migrated secrets.** `reportUnreferencedBlobs`
  degrades to one WARN because `migrate` drops the `age:` pointer. That is #971
  and is not touched here.
- **Splitting `GITHUB_PERSONAL_ACCESS_TOKEN` and `RELEASE_TOKEN` into distinct
  tokens.** They share one age source pending `migrate --split` (C9, #941). This
  change probes the shared token once, as today.

## Risks / open questions

- **Doctor gains the ability to resolve secrets.** It could not before; it can
  after. Bounded deliberately: only entries carrying `validate: github-token`
  are resolved (2 of 28 secrets today), only in the full sweep — never under
  `--quick`, the SessionStart hot path — and the value is used for one
  `Authorization` header and never reported. `dotf secrets verify` already
  resolves the entire registry, so the capability is not new to the CLI.
- **A locked Bitwarden must fail fast, not hang.** BUG-080 was a 45-second
  startup stall caused by resolving all 28 bw secrets per launch. A nil
  `BWReader` already errors immediately (`resolve.go`), but the shellout reader
  with no session does not, so the wiring must select the reader from the serve
  probe rather than optimistically shelling out. **This is a claim to verify by
  running it with the daemon down, not by reading the code** — recorded as a
  task, not as an assumption.
- **The severity taxonomy is a fleet-wide contract.** Today 401 is the only FAIL
  in this section and therefore the only branch that drives a non-zero exit.
  Resolution introduces new failure modes (absent secret, locked vault, backend
  error) and misclassifying any of them changes `dotf doctor`'s exit code on
  every mid-migration machine. The mapping is specified in AC4 rather than left
  to the implementation.
- **Adding `validate: github-token` widens `dotf secrets sync ci`.** The same
  marker gates the uploader's liveness probe (`secrets_sync.go:108`), so
  `GITHUB_PERSONAL_ACCESS_TOKEN` becomes probed before upload too. Reviewed and
  wanted: it is a GitHub token and the probe is the same `gh api user` call.

## Acceptance criteria

- [ ] **AC1** `Entry` exposes a backend-qualified source identity, and both
      `render.go`'s duplicate-var guard and `checks_pat.go`'s dedupe use it.
      Guarded by a fixture with two distinct bw secrets exposing one var,
      asserting the render fail-fast fires (it does not today).
- [ ] **AC2** `githubPATSecrets` selects on `validate: github-token`, not on an
      age-blob name prefix, and `Entries()` carries `Validate`. Guarded by a
      registry fixture containing a bw-backed GitHub PAT, asserting it is
      returned (it is dropped today).
- [ ] **AC3** The PAT token is resolved through the secrets Loader via a `System`
      seam, and no code path in `checks_pat.go` reads the token from the ambient
      environment. No message in the check names a command that does not exist.
- [ ] **AC4** The severity contract holds: HTTP 401 remains the only FAIL;
      a secret that is absent or whose backend is unavailable is a SKIP naming a
      command that exists; any other resolution error is a WARN. Table-tested per
      branch.
- [ ] **AC5** Exported backend constants exist, one canonical list feeds the
      registry's validation, and a test binds the Loader's resolver map to that
      list — every valid backend has a resolver, plus `""` → age for back-compat.
      Adding a backend to the list without a resolver fails the suite.
- [ ] **AC6** Verified live, not by inspection: `dotf doctor` reports the PAT
      section without the ambient-env SKIP on this machine, and with the bw
      daemon down the check returns promptly rather than stalling.

## References

- Bitácora board: mlorentedev/dotfiles#972
- #969 / #973 — BUG-077, the first instance of this class, and its fix
- #606 — where `Entries()` widened into a tagged union
- #961 — the migration that made `BITACORA_PAT` bw-backed and thus unmonitored
- #971 — the DR-floor pointer gap, deliberately not addressed here
- #941 — `migrate --split` (C9), why the two GitHub vars still share one source
- `docs/adr/adr-028-secrets-two-tier-bitwarden-age` — why the ambient environment is empty by design
- BUG-080 (#977) — the bw-resolution latency this must not reintroduce
