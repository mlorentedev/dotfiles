---
id: "BUG-030-doctor-env-shared-resolver"
type: spec
status: implementing
created: "2026-07-10"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — BUG-030-doctor-env-shared-resolver

## Reproduction (pre-fix, #697)

On a machine whose `~/.dotfiles` lags the repo (deployed contract predates #663;
deployed `DOTF_VERSION` older than the repo pin): `dotf doctor` reads the stale
deployed copy → `[FAIL] paths.* stale`, while `dotf env generate --check` reads
the fresh repo copy → `ok: up to date`. Contradictory verdicts, one binary.

## Automated evidence (this branch)

| Check | Command | Result |
|---|---|---|
| ResolveRepoFirst precedence | `go test ./internal/env/...` | pass |
| loadConfig repo-first + reads repo pin | `go test ./internal/doctor/...` | pass |
| env ↔ doctor agree (cross-check) | `TestLoadConfigResolvesRepoFirst` | pass |
| Build / vet / full suite | `go build ./...`, `go vet ./...`, `go test ./...` | clean |
| Lint | `golangci-lint run ./internal/{env,doctor}/...` | clean |
| Provenance rendered | `dotf doctor` on this machine | `[INFO] contract: …\dotfiles\env-contract.json` + `[INFO] versions.conf: …` — the **repo** copy, not `~/.dotfiles` |

## CI evidence (post-push, T6)

- [ ] `test` (Go unit incl. the anti-drift guard) green on ubuntu-latest.
- [ ] `lint` (golangci v2), `lint-powershell`, `spec-gate`, `integration` green.

## Guard rationale (incident -> guard)

The anti-drift test encodes the exact #697 failure: with BOTH the checkout and
the deployed copy present (and differing), doctor must resolve the checkout copy
and read the repo pin — and `env.ResolveContractPath` must return the same path.
Any future change that reintroduces a deployed-first (or otherwise divergent)
order for either command turns the test red.
