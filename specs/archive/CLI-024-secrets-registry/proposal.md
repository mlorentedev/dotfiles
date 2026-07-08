---
id: "CLI-024-secrets-registry"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-25"
issue: "mlorentedev/dotfiles#583"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, registry, age, dotf]
template_version: "1.0"
---

# CLI-024-secrets-registry

> ADR-028 §2 — the `dotf secrets` facade over a **registry** (`secrets/registry.yaml`) as the mapping SSOT. Adds `ls` + `show <id>` and rewires `run` to read the registry, **over the age store as-is** (no `bw` yet). Reconciles #378 (registry direction) and #493 (converged Go command).

## Why

ADR-028 makes `secrets/registry.yaml` the **mapping SSOT** that ties *store source ⇄ env/file ⇄ consumer*. Phase 1 (#579/#581) shipped `dotf secrets run` over `sensitive/env-mapping.conf`. Keeping `env-mapping.conf` as the live map while the ADR declares the registry the SSOT is a two-SSOT violation (Standing Order #2) — exactly the "uncoordinated surfaces" the ADR exists to collapse.

Two concrete needs force the registry now: (1) the #493 convergence (migrate `nan-*` + the setup eager-load off `load-secrets`, delete the twins) needs a **single-value read primitive** — `dotf secrets show <id>` — that does not exist; (2) the registry adds the `plane`/`consumers`/`rotate` metadata the rotation runbook and a future `dotf doctor` staleness check depend on, which `env-mapping.conf`'s flat `VAR=file` cannot carry.

## What

`secrets/registry.yaml` becomes the mapping SSOT; the Go facade reads it (age backend):

1. **`secrets/registry.yaml`** — `version: 1` + a `secrets:` list, seeded from `docs/secrets-inventory.md`, round-tripping every current `env-mapping.conf` entry. Per secret: `id` (stable kebab handle), `plane` (app|infra|personal|floor), `backend` (age|age-offline|bw — the *target*; PR reads age/age-offline), the age source, an `expose` contract (env one/many/per-var, or file), `consumers`, `rotate`.
2. **`dotf secrets ls`** — lists ids with plane + exposed vars (no values).
3. **`dotf secrets show <id>`** — resolves the id, decrypts (age), writes the value to stdout with **no trailing newline** (capture-friendly: `KEY=$(dotf secrets show nan-api-key)`). Single-env secrets only; multi-var/file secrets error with guidance to use `run`.
4. **`dotf secrets run`** rewired to resolve from the registry instead of `env-mapping.conf`. `--only` accepts an **id or an env-var name** (back-compat with the wrappers + the run-jit examples). Behaviour (env + file secrets, child-only injection, exit-code propagation) is unchanged.

### Registry schema (the core design)

```yaml
version: 1
secrets:
  - id: nan-api-key                 # stable handle; `show nan-api-key`, `run --only nan-api-key`
    plane: app
    backend: age                    # age | age-offline | bw (bw resolved in Phase 3)
    age: nan.api-key                # base under sensitive/, no .secret.age
    expose: { env: NAN_API_KEY }    # single env var ← the age value
    consumers: ["agent:opencode", "agent:pi"]
    rotate: 90d

  - id: github-token                # one age file, two env names (current reality; #321 splits it in Phase 4)
    plane: app
    backend: age
    age: github.token
    expose: { env: [GITHUB_PERSONAL_ACCESS_TOKEN, RELEASE_TOKEN] }   # list ← same value
    consumers: ["ci:release", local]

  - id: x-twitter                   # per-var sources (7 age files today; collapses to 1 bw item w/ fields in Phase 4)
    plane: app
    backend: age
    expose:
      env:
        X_API_KEY:             { age: x.api-key }
        X_BEARER_TOKEN:        { age: x.bearer-token }
        # …5 more
    consumers: ["ci:social"]
    rotate: 180d

  - id: kubelab-kubeconfig          # file secret: materialize + point an env var at the path
    plane: infra
    backend: age
    age: kubelab.kubeconfig
    expose: { file: { var: KUBECONFIG, path: "~/.kube/kubelab.config", mode: "0600" } }
    consumers: [local]

  - id: ssh-id-ed25519              # floor: offline-rooted, boot dependency
    plane: floor
    backend: age-offline
    age: id_ed25519
    expose: { file: { var: SSH_KEY, path: "~/.ssh/id_ed25519", mode: "0600" } }
    consumers: [local]
```

Resolution: `expose.env` may be a string (one var), a list (many vars, same value), or a map `VAR → {age: file}` (per-var sources). `expose.file` materializes the age source to `path` (`mode`, parent dirs created) and sets `var` to the path. `age:`/`age-offline:` read the same way (the floor distinction is governance, not a different reader); `backend: bw` resolution is Phase 3.

## Out of scope

- **`bw` backend (`bw serve` live read) + item migration** — ADR-028 Phase 3. The schema declares `backend: bw` targets but the reader implements age only.
- **Deleting `env-mapping.conf` and the `load-secrets.{sh,ps1}` twins** — the #493 convergence (PR-B migrates setup + `nan-*`; PR-C deletes the twins **and** `env-mapping.conf`, their only remaining consumers). This PR leaves `env-mapping.conf` in place as twins-only legacy; **nothing in the Go path reads it after the rewire**, so there is no live second SSOT.
- **`sync` (materialize for headless CI/containers)** — Phase 2 follow-up / Phase 3.
- **Curation** (folders, token split #321, naming-drift fixes openai/openrouter) — ADR-028 Phase 4.

## Risks / open questions

- **R1 — registry must round-trip `env-mapping.conf` exactly**, or `run` regresses the wrappers. *Mitigation:* a table-driven test asserts the registry resolves to the *same* `{var → age-file}` / file-secret set as `ParseMapping(env-mapping.conf)`. The two-name `github.token` and all `@VAR` file secrets are explicit cases.
- **R2 — `show` value must not leak beyond intended stdout.** `show` prints the value *by design* (its purpose is capture), but only on explicit `show <id>`; it must not log the id's value elsewhere, and `ls`/`run` never print values. *Mitigation:* `show` writes only the raw value to stdout; tests assert `ls` output contains no value and `run`'s own output is empty.
- **R3 — `--only` ambiguity (id vs env name).** A token could match both. *Mitigation:* resolve id first, then env-var name; document the precedence; the sets are disjoint in practice (ids kebab, env vars UPPER_SNAKE).
- **R4 — new direct dep `go.yaml.in/yaml/v3`.** Already in the module graph (go.sum, transitive); promoting to a direct require, the canonical YAML lib. *Mitigation:* noted here per the Discipline Gate; no new supply-chain surface.
- **R5 — `show` on a multi-var or file secret is ambiguous.** *Mitigation:* error with a clear message ("`x-twitter` exposes 7 vars; use `dotf secrets run --only x-twitter -- <cmd>`").

## Acceptance criteria

- [ ] **AC1** — `secrets/registry.yaml` parses and resolves to the **same** var→source / file-secret set as the current `env-mapping.conf` (round-trip test, incl. `github-token`'s two names and every `@VAR` file secret).
- [ ] **AC2** — `dotf secrets show <id>` prints the decrypted age value with **no trailing newline**; `KEY=$(dotf secrets show nan-api-key)` captures it. *Verify:* `go test` with an injected decrypt seam.
- [ ] **AC3** — `dotf secrets show <multi-or-file-id>` exits non-zero with a message pointing to `run`. *Verify:* `go test`.
- [ ] **AC4** — `dotf secrets ls` lists every id with its plane + exposed vars and **no values**. *Verify:* `go test`.
- [ ] **AC5** — `dotf secrets run` resolves from the registry; `run --only <id>` and `run --only <ENV_VAR>` both work; existing opencode/pi/agy wrappers are unaffected (full mapped set by default). *Verify:* `go test` + smoke.
- [ ] **AC6** — registry parser rejects a malformed/duplicate-id/unknown-backend registry with a clear error (fail-fast, not silent). *Verify:* `go test`.
- [ ] **AC7** — `go test ./...` green; `env-mapping.conf` untouched and still parseable by the (still-present) twins.

## References

- Issue: mlorentedev/dotfiles#583 (work gate); reconciles #378, #493
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` (§2, §"Registry schema", Phase 2)
- Inventory: `docs/secrets-inventory.md` (the seed)
- Code: `cli/internal/secrets/{secrets,resolve}.go` (Phase 1a), `cli/internal/cmd/secrets.go` (run wiring), `sensitive/env-mapping.conf` (the map being superseded)
