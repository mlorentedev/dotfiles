---
id: "BUG-030-doctor-env-shared-resolver"
type: spec
status: implementing
created: "2026-07-10"
tags: [spec, proposal, doctor, env, path-resolution, adr-025, fresh-machine]
template_version: "1.0"
---

# BUG-030-doctor-env-shared-resolver

Give `dotf doctor` and `dotf env` one shared repo-first resolver for
`env-contract.json` / `versions.conf`, and print a provenance line, so they can
never hand out contradictory stale/fresh verdicts.

## Why

`dotf doctor` located `env-contract.json` / `versions.conf` **deployed-copy-first**
(`config.go`: `firstExisting(DOTFILES_DIR/…, repo/…)`), while `dotf env generate`
resolved **repo-first** (`env.ResolveContractPath`). On a machine whose deploy dir
lags the repo, the same binary in the same shell said both:

- `[FAIL] paths.ps1 is stale — run dotf env generate` (doctor, reading the stale
  deployed contract), and
- `ok: … up to date` (env generate --check, reading the fresh repo contract).

Doctor's fix advice could not converge, and version-drift directions came out
nonsensical (`installed=0.28.0 pinned=0.23.0` off a **stale** deployed pin — the
"fix" would act on the wrong number). It also silently re-introduced the #518
class: a generate run that resolved the stale deployed contract renders paths
WITHOUT the age keys #663 added.

Root cause: two independent resolvers with **opposite precedence**. The two
drifting apart is exactly the failure — so the fix is one shared resolver, not
two that happen to agree today.

Source: issue #697 (audit process-audit-2026-07-07 §4 P4, CONFIRMED; §3 marks
#663 AT-RISK on this vector). Sibling of BUG-029/#696 (same fragmented-resolution
theme).

## What

- **Shared resolver.** New `env.ResolveRepoFirst(name, repoDir, dotfilesDir,
  startDir)`: checkout copy (the version-controlled SSOT, fresher when the deploy
  dir lags) wins over the deployed copy under `DOTFILES_DIR`, then a walk-up from
  `startDir`. `env.ResolveContractPath` is refactored to call it (behavior
  unchanged); `doctor.loadConfig` calls it for **both** the contract and
  `versions.conf`. Precedence now lives in one function both commands share, so
  they cannot drift again.
- **Consistent repo discovery in doctor.** `loadConfig` resolves the checkout the
  way env does — `DOTFILES_REPO_DIR` when it points at a real dir (now reliably
  seeded by BUG-029), else the `.git` walk-up from `startDir` — instead of only
  the walk-up.
- **Provenance.** `dotf doctor` prints an always-visible `[INFO] contract: <path>`
  and `[INFO] versions.conf: <path>` (via `rep.Info`, which is not suppressed in
  non-verbose mode as `Pass` is), so a stale-copy read is self-diagnosing rather
  than a silent contradiction.
- **Anti-drift guard.** A doctor test asserts that with **both** the checkout and
  deployed copies present, `loadConfig` resolves the checkout copy and reads the
  repo pin (not the stale deployed one), and cross-checks that
  `env.ResolveContractPath` agrees. Plus `env` unit tests for `ResolveRepoFirst`
  precedence (repo → deployed → walk-up → "").

## Out of scope

- **Unifying repo *discovery* itself.** `env.ResolveContractPath` reads
  `DOTFILES_REPO_DIR` via `os.Getenv` (it must not call `ResolvePath`, which would
  recurse into it); doctor reads it via its `System` seam. The *precedence* is now
  shared; the discovery seam legitimately differs (globals vs injectable seam) and
  matches in production. Not merged further.
- **The other fresh-machine bugs** (#691, #690, #689) — their own issues.
- **Re-deploying a stale contract.** This makes doctor *read* repo-first and say
  so; it does not auto-refresh `~/.dotfiles` (that is `dotf update` / setup).

## Risks / open questions

- A machine that legitimately runs the deployed copy with NO checkout present is
  unaffected: `repoDir` is empty, so resolution falls straight through to the
  deployed copy (tier 2) exactly as before.
- Changing the contract "loaded" line from `Pass(… from <path>)` to
  `Pass(loaded)` + `Info(contract: <path>)` keeps the `env-contract.json loaded`
  substring the existing test asserts, and makes the path visible in non-verbose.

## Acceptance criteria

- [x] `env.ResolveRepoFirst` resolves repo → deployed → walk-up → "" in that order.
- [x] `doctor.loadConfig` resolves contract AND versions.conf via the shared
      resolver, repo-first, honoring `DOTFILES_REPO_DIR`.
- [x] `dotf doctor` prints `contract:` and `versions.conf:` provenance lines
      (visible in non-verbose).
- [x] Anti-drift guard: with both copies present doctor picks the repo copy and
      reads the repo pin; env agrees.
- [x] `go build`/`vet`/`test ./...` clean; golangci-lint clean; provenance
      observed end-to-end on this machine (resolves the repo copy).

## References

- GH issue: [#697](https://github.com/mlorentedev/dotfiles/issues/697)
- ADR-025 (cross-machine paths / the cascade)
- Sibling: BUG-029/#696 (seed machine.json; same fragmented-resolution theme)
- At-risk prior fix: #663 (AGE_KEY_PATH/SOPS_AGE_KEY_FILE), #518 (age-key
  discovery regression this stale read re-introduced)
