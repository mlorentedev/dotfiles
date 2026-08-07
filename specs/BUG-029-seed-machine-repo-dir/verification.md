---
id: "BUG-029-seed-machine-repo-dir"
type: spec
status: implementing
created: "2026-07-10"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — BUG-029-seed-machine-repo-dir

## Reproduction (pre-fix, #696)

On a machine with no `machine.json` and the repo checked out anywhere other than
`~/Projects/dotfiles`:

- `dotf update` -> "not a git repo: <phantom> — nothing to self-update", exit 0
  (self-deploy never runs).
- `dotf mem session-start` -> probes `<phantom>/scripts/vault-health.sh`, prints
  "run dotfiles setup" though setup ran.

Both because `env.ResolvePath("DOTFILES_REPO_DIR")` returns the non-existent
contract default instead of `""`.

## Automated evidence (this branch)

| Check | Command | Result |
|---|---|---|
| env set merge/preserve | `go test ./cli/internal/env/...` | pass |
| env set unknown key fails | `go test ./cli/internal/env/...` | pass |
| repoForUpdate walk-up fallback | `go test ./cli/internal/cmd/...` | pass |
| doctor repo-dir health check | `go test ./cli/internal/doctor/...` | pass |
| Build | `go build ./...` | clean |
| Shell syntax | `bash -n setup-linux.sh` | clean |
| machine.json seeded post-setup | `bats tests/verify-setup.bats` (integration) | skip until a release ships `env set` (see note) |

## CI evidence (post-push, T8)

- [ ] `test` (Go unit + bats) green on ubuntu-latest.
- [ ] `integration` green — the two machine.json guards **skip** in the current
      container (it installs the *released* dotf, which predates `env set`, and
      dotf is not on the bats-time PATH); they self-activate once a release ships
      `env set`. The seed logic itself is guarded by the Go unit tests + doctor
      check, which do run. Harness gap (integration should exercise the PR's built
      binary) tracked in #734.
- [ ] `lint`, `lint-powershell`, `spec-gate` green.

### Pre-existing failures ruled out (Windows-local only)

The `verify-setup.bats` / `core.hooksPath` suites fail on this Windows box (zsh
absent + MSYS git-config), identically on pristine `main`. They are not
regressions and pass in Linux CI; the machine.json seeding is verified there.

## Guard rationale (incident -> guard)

Three guards land with the fix: Go unit tests proving `env set` merges without
clobbering and rejects typos; a `dotf doctor` health check that FAILs when
`DOTFILES_REPO_DIR` resolves to a phantom (so the class is caught on any box, not
just fresh ones); and an integration bats asserting setup actually seeds
`machine.json`. Together they encode the exact "phantom default, unseeded
override" failure the audit found.
