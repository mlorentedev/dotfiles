---
id: "BUG-029-seed-machine-repo-dir"
type: spec
status: implementing
created: "2026-07-10"
tags: [spec, proposal, env, machine-json, path-resolution, adr-025, fresh-machine]
template_version: "1.0"
---

# BUG-029-seed-machine-repo-dir

Seed `~/.config/dotfiles/machine.json` at setup so the `DOTFILES_REPO_DIR`
cascade resolves to the real checkout, and make `update`/`mem` fall back to the
git walk-up instead of a phantom default.

## Why

Two "where is the repo" seams coexist in one binary (ADR-025):

- `env.ResolvePath("DOTFILES_REPO_DIR")` — the cascade `env -> machine.json ->
  contract default[GOOS]`. It returns a value **even when that path does not
  exist**, because the contract default `$HOME/Projects/dotfiles` (Windows:
  `%USERPROFILE%\Projects\dotfiles`) is always non-empty.
- `env.RepoDir()` — a `.git` walk-up from the working directory that returns
  `""` when no checkout is found.

No onboarding step ever writes `machine.json` (the README presents it only as a
relocation tool), so on a docs-faithful fresh machine the phantom contract
default propagates through the cascade — and through the generated
`paths.sh`/`paths.ps1` — into every consumer that uses `ResolvePath`:

- `dotf update` -> `repoForUpdate()` resolves the phantom -> `update.Run` reports
  "not a git repo: ~/Projects/dotfiles — nothing to self-update" and exits 0.
  **Self-deploy is a silent no-op on every fresh machine.**
- `dotf mem session-start` -> `memScriptsDir()` probes
  `~/Projects/dotfiles/scripts/vault-health.sh`, misses, and prints "run
  dotfiles setup" — even though setup ran. The `repo == ""` guard never fires
  because the cascade never returns `""`.
- `dotf doctor` path checks resolve the same phantom and hint at a nonexistent
  path.

Meanwhile `secrets`/`spec`/doctor's repo-drift check use `RepoDir()` (walk-up)
and work correctly — so two contradictory "where is the repo" answers ship in
one binary.

Root cause: nothing seeds the per-machine override `machine.json` that ADR-025
already designed as the layer that wins over the contract default. Setup is the
natural seeder — it runs *from* the checkout, so it already knows the answer.

Source: issue #696 (audit process-audit-2026-07-07 §4 P3, CONFIRMED).

## What

- **A — root cause. New `dotf env set <KEY> <VALUE>`** (write-side counterpart of
  the existing read-side `dotf env path <KEY>`): loads `machine.json`, validates
  `KEY` is a declared contract var (unknown key -> fail loud), sets
  `paths[KEY]=VALUE` **preserving every other key**, and writes atomically
  (temp + rename), creating `~/.config/dotfiles/` if absent. Idempotent (same
  value -> "unchanged"). One Go JSON-merge instead of fragile hand-rolled JSON in
  both `setup-linux.sh` and `setup-windows.ps1` (ADR-020).
  - **setup-linux.sh**: `dotf env set DOTFILES_REPO_DIR "$CURRENT_DIR"` right
    before `dotf env generate` (so the generated path file carries the real
    value). **setup-windows.ps1**: same with `$DotfilesDir`. Both inside the
    existing `command -v dotf` / `Get-Command dotf` guard.
- **B — defense in depth.** `repoForUpdate()` and the `mem` resolvers prefer a
  cascade value only when it is an existing directory, else fall back to
  `env.RepoDir()` (walk-up), else (update only) the last-resort literal for a
  bare scheduler env. Fixes interactive `dotf update`/`mem` run from inside a
  checkout even before/without `machine.json`.
- **Doctor check.** A new `dotf doctor` assertion: `DOTFILES_REPO_DIR` (via the
  cascade) resolves to a real checkout (a dir containing `.git`). PASS when it
  does; FAIL with an actionable hint (`run setup`, or `dotf env set
  DOTFILES_REPO_DIR <path>`) when it resolves to a phantom / missing path — the
  SRE health-check that would have caught this class on a fresh box.
- **Guards (incident -> guard).**
  - Go unit tests for `dotf env set`: merge preserves other keys, unknown key
    fails, idempotent re-set, machine.json created when absent.
  - Integration bats (`verify-setup.bats`): after setup, `machine.json` exists,
    its `DOTFILES_REPO_DIR` equals the checkout, and `dotf env path
    DOTFILES_REPO_DIR` resolves to a real dir.

## Out of scope

- **Changing the contract default.** `$HOME/Projects/dotfiles` stays as the
  documented last-resort for a bare scheduler env that has neither `machine.json`
  nor a discoverable checkout. Seeding `machine.json` makes it irrelevant on a
  real machine; removing it would only trade a phantom for an empty string.
- **The other fresh-machine bugs.** #697 (doctor deployed-first vs repo-first),
  #691 (GUARD-001 memory-sink on Windows), #690 (dead `injectors.enabled`), #689
  (Windows project-key) are their own issues. This PR only unifies the
  `DOTFILES_REPO_DIR` resolution and seeds `machine.json`.
- **Migrating `VAULT_PATH` / other overrides.** `dotf env set` is general (any
  contract key), but setup only seeds `DOTFILES_REPO_DIR`; seeding other keys is
  left to the user / future work.

## Risks / open questions

- **Setup re-writes `DOTFILES_REPO_DIR` on every run** to `$CURRENT_DIR`. This is
  intentional and self-healing (setup runs from the authoritative checkout
  location); the merge preserves all other keys, and the cascade's env-var tier
  still wins for a user who overrides at runtime.
- **Windows `dotf` not yet installed** in some flows: the seed sits inside the
  existing `Get-Command dotf` guard, so it is skipped exactly when `env generate`
  is, and `profile.ps1` keeps its inline fallback. No regression.
- **`env set` writing under a read-only `$XDG_CONFIG_HOME`**: fails loud with the
  write error rather than silently continuing; setup treats it as a warning like
  the adjacent `env generate` step.

## Acceptance criteria

- [x] `dotf env set <KEY> <VALUE>` writes/merges `machine.json`, preserves other
      keys, rejects an unknown contract key, and is idempotent.
- [x] setup-linux.sh and setup-windows.ps1 seed `DOTFILES_REPO_DIR` before
      `dotf env generate`.
- [x] `repoForUpdate()` and `mem` resolvers fall back to `RepoDir()` walk-up when
      the cascade value is not an existing directory.
- [x] `dotf doctor` FAILs (with an actionable hint) when `DOTFILES_REPO_DIR`
      resolves to a non-checkout path, PASSes when it resolves to a real one.
- [x] Go unit tests (env set) + integration bats (machine.json seeded) green.
- [x] `go build ./...` and `go test ./...` clean; `bash -n setup-linux.sh`.

## References

- GH issue: [#696](https://github.com/mlorentedev/dotfiles/issues/696)
- ADR-025 (cross-machine paths; the cascade this unifies)
- Related fresh-machine cluster: #697, #691, #690, #689
- Prior art: BUG-026/#694 (setup must not write into the checkout — same
  "deploy != checkout" invariant), the existing read-side `dotf env path`
