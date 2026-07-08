---
tags: [spec, tasks, secrets, registry]
created: "2026-06-25"
---

# Tasks - CLI-024-secrets-registry

> TDD order (red → green per AGENTS.md). One task ≈ one focused commit. Frozen at `implementing`.

## Setup

- [x] Branch created from main: `feat/secrets-registry`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

### 1. Registry model + parser (`cli/internal/secrets/registry.go`)

- [x] **T1.1** `registry_test.go`: `ParseRegistry` over a sample with each `expose` shape
      (single env, env list, env map `VAR→{age}`, file `{var,path,mode}`) + the 3 backends. (RED)
- [x] **T1.2** `registry.go`: `Registry`/`Secret`/`Expose` types + `ParseRegistry` (`go.yaml.in/yaml/v3`,
      add to `go.mod`). Validate `version==1`, unique ids, known backend, exactly one of env/file,
      age source present for age/age-offline. (GREEN)
- [x] **T1.3** test: malformed / duplicate id / unknown backend / missing source → clear error. (AC6)

### 2. Resolver → entries (round-trip with env-mapping)

- [x] **T2.1** test: `(*Registry).Entries()` == `ParseMapping(env-mapping fixture)` set, incl.
      `github-token`'s two names + every `@VAR` file secret. (AC1, RED)
- [x] **T2.2** `Entries()`: expand single/list/map env → `Entry`; file → IsFile w/ Dest=path, Var=`file.var`.
      Reuse `Entry` + `Loader.EnvFor`. (GREEN)
- [x] **T2.3** test + impl: `Lookup(idOrVar)` resolves id first, then env-var name. (R3)

### 3. Seed data (`secrets/registry.yaml`)

- [x] **T3.1** author `secrets/registry.yaml` from `docs/secrets-inventory.md` + `env-mapping.conf`
      (25 secrets; plane/consumers/rotate from inventory; `backend: age`, floor=`age-offline`).
- [x] **T3.2** test loads the **real** `secrets/registry.yaml`; asserts entry set ==
      `ParseMapping(real env-mapping.conf)` (drift guard until env-mapping dies in PR-C).

### 4. `dotf secrets ls`

- [x] **T4.1** test: `ls` prints id + plane + exposed vars, no values. (AC4, RED)
- [x] **T4.2** `newSecretsLsCmd`; wire into `newSecretsCmd`. (GREEN)

### 5. `dotf secrets show <id>`

- [x] **T5.1** test: `show <single-env-id>` → value, no trailing newline (fake decryptor). (AC2)
- [x] **T5.2** test: `show <multi/file-id>` → non-zero + message pointing at `run`. (AC3, AC5)
- [x] **T5.3** `newSecretsShowCmd`; wire in. (GREEN)

### 6. Rewire `run` to the registry

- [x] **T6.1** test: `buildChildEnv` reads `secrets/registry.yaml`; `--only` accepts id or env name. (AC5, RED)
- [x] **T6.2** repoint `buildChildEnv` to `Registry.Entries()`+`Lookup`; keep file/exit-code behaviour.
      `env-mapping.conf` no longer read by Go. (GREEN)

## Closing

- [x] Every acceptance criterion covered by ≥1 test; `features.json` written (evidence set by harness)
- [x] `go test ./internal/secrets ./internal/cmd`, `gofmt`, `go vet` clean (3 pre-existing template-drift failures in spec/vault/initrepo are unrelated — see verification.md)
- [x] No unrelated changes (no scope creep); `env-mapping.conf` untouched
- [x] `verification.md` filled in; PR opened referencing this spec

## Machine-readable features

Emits sibling `features.json` ([[pattern-feature-list-as-primitive]]). Agent CANNOT set `"state":"passing"` — only the harness, after running `verification` (exit 0), sets it. Each AC → ≥1 feature with executable `verification`.
