---
id: "dotfiles-adr-028-secrets-two-tier-bitwarden-age"
type: adr
adr: "028"
title: "Two-Tier Secrets Governance — Bitwarden live SSOT + age DR floor, via a dotf secrets facade"
tags: [adr, dotfiles, secrets, bitwarden, age, governance]
status: proposed
created: "2026-06-25"
owner: manu
supersedes_partial: "adr-002 (the shell-startup ambient load; the single-key model)"
---

# ADR-028: Two-Tier Secrets Governance — Bitwarden live SSOT + age DR floor

## Context

Secrets management has sprawled across three uncoordinated surfaces:

- **age in dotfiles** (ADR-002): ~40 `sensitive/*.secret.age` files + `env-mapping.conf`, decrypted and **exported to the shell environment on every login** by `load-secrets.{sh,ps1}`. Works offline, but: secrets are *always exposed* in the ambient env, rotation is manual, cross-device sync is manual (git + key copy), and one `github.token` feeds many consumers (`#321`).
- **Bitwarden**: ~145 items, **all in "No Folder"** (zero organization), mostly personal logins, but also ~20 dev/infra secrets stored inconsistently (8 GitHub tokens crammed into one item's fields; other keys as standalone notes). Already the de-facto cross-device store the user reaches for.
- **Two conflicting plans**: `#378` (HARNESS-022) proposed Bitwarden as SSOT; `#493` (CLI-024) proposed porting the age twins to Go — **both define `dotf secrets` with incompatible backends, never reconciled.** This is the root of the "out of hand" feeling.

The user's primary objectives: **easily maintainable** and **not always exposed in the system** (on-demand, not ambient). Plus a stated need for **vault curation** and explicit **add / rotate / backup / recover protocols**.

A blocking discovery from the inventory: **the age private keys (`AGE-SECRET-KEY-CI/PERSONAL`) are stored inside Bitwarden** — a circular dependency if age is to be the disaster-recovery escrow of the Bitwarden vault.

This ADR does **not** revisit the encryption tool (ADR-002's age decision stands). It re-scopes age's *role*, reverses the ambient shell-startup load, and reconciles the two `dotf secrets` plans.

## Decision

### 1. Two systems, distinct roles, one SSOT per secret

- **Bitwarden = the live SSOT** for day-to-day use: cross-device sync, central rotation, on-demand read. Holds the authoritative copy of every consumable secret.
- **age (in the repo) = the disaster-recovery escrow + bootstrap floor**, offline-rooted. NOT the live source for everyday secrets.

No secret has two *authoritative* homes. Materialized `.env` files and the age DR export are **derived, regenerable copies**, not duplicate SSOTs.

### 2. `dotf secrets` facade over a registry

A single `dotf secrets` command reads a **registry** (`secrets/registry.yaml`) declaring, per secret: `{ backend (bw|age-floor), bw_item, bw_field, plane (app|infra|personal|floor), env_var(s) | file_target, consumers[], rotate_cadence }`. This reconciles `#378` (Bitwarden direction + personal plane) and `#493` (the converged Go command + app plane) into one tool.

### 3. On-demand injection, never ambient (the security objective)

- `dotf secrets run -- <cmd>` injects secrets into **that child process only** (never the shell env). Replaces the `load-secrets` blanket export — reverses ADR-002's "decrypt at shell startup".
- `dotf secrets sync <target>` materializes a **scoped** secret set **at deploy time** for headless consumers (CI → Actions secrets; containers/agents → `0600` `.env`). The headless context never talks to Bitwarden at runtime — the sync is a controlled, ahead-of-time step, sidestepping `bw`'s interactive-unlock friction.

### 4. age key is offline-rooted (fixes the circular dependency)

The authoritative copy of the age private key lives **OFFLINE** (encrypted USB via `backup-secrets-to-usb.sh` + paper + OS keychain), independent of Bitwarden. A copy *may* remain in Bitwarden for cross-device convenience, but it is **not** the root of trust. The age key is the single root from which all DR recovery flows.

### 5. Disaster-recovery escrow

`dotf secrets backup` periodically runs `bw sync && bw export --format json --raw | age -r <pubkey> > sensitive/dr/bitwarden-export.age` (plaintext **never** touches disk), committed to the repo and mirrored off-box (`#454`). This escrows the **entire** vault — API keys, tokens, access credentials, TOTP seeds — so losing Bitwarden access is fully recoverable with the offline age key + a repo clone, independent of the Bitwarden account.

### 6. Curation & naming (three-layer scheme)

- **Store (Bitwarden):** introduce folders `apps/` `infra/` `personal/`; item `<service>-<purpose>`; collapse multi-value creds into one item with custom fields; native types (SSH Key, Secure Note); **split shared tokens per purpose** (`#321`); merge intra-vault duplicates.
- **Env (consumer contract):** stable `UPPER_SNAKE` names (apps depend on them) — derived, not the naming SSOT.
- **Registry:** the mapping SSOT that ties item ↔ field ↔ env var ↔ consumer.

### 7. Governance protocols (detailed in runbooks)

`docs/runbooks/`: **add** (where a new secret goes, naming, plane), **rotate** (consumer-mapped, incremental, never big-bang), **backup/DR** (the escrow above), **recover** (the `#257` disaster chain: age key → repo → export → re-import).

## Target structure & registry schema (concrete)

### Bitwarden folder taxonomy

Bitwarden personal folders nest by `/` in the name. All `dotf secrets`-managed items live under a single `dotfiles/` tree, isolating them from the ~125 personal items (which stay where they are):

```
Dotfiles/apps     # service API keys/tokens the projects consume
Dotfiles/infra    # infrastructure access (servers, k8s, VPN, registries, DNS, CI runners)
Dotfiles/floor    # DR/bootstrap roots — age-key convenience copy, master-pw backup (offline-authoritative)

# Personal tree (SEPARATE, incremental track — NOT on the dotfiles critical path):
Finance           # banks, brokers, crypto
Travel            # airlines, hotels, car rental
Work              # employer/client accounts
Accounts          # Google, Apple, email, social
Home & Utilities  # energy, telecom, IoT
Shopping
Government & Taxes
Entertainment
```

`dotf secrets` only ever reads/writes under `Dotfiles/**`. The personal tree is curated as a separate, optional, incremental pass and is out of `dotf secrets`' bounds. The whole vault gets a domain taxonomy because a 125-item flat vault is neither auditable nor maintainable — but organizing the personal tree never blocks the dotfiles security work. **Decided:** keep the `Dotfiles/` namespace; **split** the GitHub 8-in-1 item per purpose (#321). Future scale ceiling: move shared secrets to a Bitwarden **Organization + Collections** (ACL + Secrets Manager) if/when agents/CI/team consume them.

### Item & field naming

- **Item:** `<service>-<purpose>`, kebab-case — `openai-api-key`, `github-release-pat`, `hetzner-api-token`, `x-twitter-api`, `cloudflare-api-token`.
- **Custom fields:** kebab-case — `api-key`, `client-secret`, `access-token`.
- **Native types:** SSH keys → SSH Key item (type 5); kubeconfig/notes → Secure Note (type 2).

### Field-vs-item rule (the key structural decision)

- **One credential, multiple parts → ONE item + custom fields.** E.g. the 7 X values collapse into `dotfiles/apps/x-twitter-api` with fields `api-key`, `api-key-secret`, `access-token`, `access-token-secret`, `bearer-token`, `client-id`, `client-secret`.
- **Different credentials (different scope / rotation / consumer) → SEPARATE items.** The current `GitHub` item crams 8 distinct tokens into fields; they have different scopes and rotation cadences, so they **split** (`#321`): `github-cli-pat`, `github-release-pat`, `github-bitacora-pat`, `github-runner-token`, `github-kubelab-dispatch-token`, and the PEM → `github-evalkit-ci-key`.

### Registry schema — `secrets/registry.yaml` (the mapping SSOT)

```yaml
version: 1
secrets:
  - id: openai-api-key                 # logical id (kebab); the only stable handle
    plane: app                         # app | infra | personal | floor
    backend: bw                        # bw | age-offline
    bw: { folder: "dotfiles/apps", item: "openai-api-key", field: null }  # field:null → item password
    expose:
      env: OPENAI_API_KEY              # a single env var…
    consumers: [local, "ci:yt-metrics"]
    rotate: 90d

  - id: x-twitter-api
    plane: app
    backend: bw
    bw: { folder: "dotfiles/apps", item: "x-twitter-api" }
    expose:
      env:                             # …or many, each bound to a named field
        X_API_KEY:       { field: api-key }
        X_BEARER_TOKEN:  { field: bearer-token }
        X_CLIENT_ID:     { field: client-id }
    consumers: ["ci:social"]
    rotate: 180d

  - id: kubelab-kubeconfig
    plane: infra
    backend: bw
    bw: { folder: "dotfiles/infra", item: "kubelab-kubeconfig", field: kubeconfig }
    expose:
      file: { path: "~/.kube/kubelab.config", mode: "0600" }   # materialize as a file, not env
    consumers: [local]

  - id: age-key-personal               # the floor: offline-authoritative, bw copy = convenience
    plane: floor
    backend: age-offline
    bw: { folder: "dotfiles/floor", item: "AGE-SECRET-KEY-PERSONAL" }
    offline: required
    expose: { file: { path: "~/.config/age/key.txt", mode: "0600" } }
```

- **`id`** is the only stable handle; `dotf secrets run --only openai-api-key -- <cmd>` resolves it.
- **`expose`** is the contract: `env` (one or many→field) and/or `file` (path+mode). Env names stay `UPPER_SNAKE` (consumer contract); the store name is canonical-by-service.
- **`consumers`** vocabulary: `local`, `ci:<repo>`, `container:<name>`, `agent:<name>` — drives rotation blast-radius and `sync` targets.
- **`rotate`** cadence informs the rotation runbook + a `dotf doctor` staleness check.

The registry is generated/seeded from `docs/secrets-inventory.md` and is the single map tying **store item ⇄ field ⇄ env/file ⇄ consumer**.

## Alternatives considered (rejected)

- **Status quo (all-in-age):** keeps the cross-device pain, manual rotation, and the ambient-env exposure the user explicitly wants gone.
- **All-in Bitwarden Secrets Manager (`bws`):** the "textbook" app-secret tool, but a new **paid** dependency + a full migration, and still needs an offline floor. Over-engineered for a personal multi-device setup. Kept as a *future swap* the facade enables (see Consequences).
- **Plain `bw` for everything including headless runtime:** `bw` needs an interactive unlock/session; making CI/containers/agents call `bw` at runtime introduces a new fragility. Avoided via `sync`-at-deploy.
- **Company-grade (OIDC federation in CI + dynamic short-lived creds):** the real gold standard, but disproportionate now. The registry/facade keeps it a future migration, not a rewrite.

## Consequences

### Positive
- **Not always exposed:** on-demand `run --` replaces ambient export — the primary objective.
- **Cross-device + maintainable:** one SSOT (Bitwarden) synced everywhere; one `dotf secrets` mental model; one registry.
- **DR-safe & account-independent:** the age escrow recovers everything even if Bitwarden is lost tomorrow; age and Bitwarden cover each other's failure modes (mutual redundancy) — *once the age key is offline-rooted*.
- **No vendor lock-in:** the json export is both the migration vehicle and the exit.
- **Incremental, non-blocking:** value is front-loaded (Phase 1); migration is a background batch.

### Negative
- **Two systems** to reason about (mitigated by the registry as the single map).
- **age key criticality rises:** the escrow makes the age key guard the *entire* vault (incl. 2FA), not just 40 dev secrets → mandatory passphrase + serious offline backup; consider a hardware-backed key.
- **The DR export is a full-vault snapshot under one key** in git history → store the latest (overwrite) + off-box mirror rather than accumulating snapshots.
- **Materialized `.env` is a second copy at rest** → prefer `run --` (env-only) over `sync` (file); `0600` + gitignore + ephemeral where unavoidable.

### Mitigations
- Offline-rooted age key (§4) + `#257` recover runbook + Bitwarden emergency access.
- Rotation is consumer-mapped and incremental (`#321`), never big-bang.
- `dotf doctor` checks: `bw`/`age` present (`#577`), DR export freshness, registry consistency.

## Scope boundary
- **In:** the ~20 dev/infra secrets (the age↔bw cross-section) + the governance model + curation of those items.
- **Out (for now):** the ~125 personal/financial logins — left as-is (optionally foldered later); company-grade OIDC/`bws`/dynamic creds (future swap).

## Ticket reconciliation
- `#378` HARNESS-022 → becomes the facade + personal plane + DR escrow.
- `#493` CLI-024 → the app plane of `dotf secrets` (converge the twins).
- `#321` OPS-007 → per-purpose token split (the GitHub-item field mess).
- `#518` → age-key bootstrap; `#257` OPS-001 → the recover runbook; `#454` OPS-015 → off-box mirror of the escrow.
- `#577` OPS-017 → provision `bw`/`age` via setup (Phase 0).

## Phased plan
0. **Provision** `bw` + `age` via setup (`#577`).
1. **`dotf secrets run` + kill the ambient export** (over age as-is → delivers the security objective).
2. **Facade + registry + `bw serve` read** (reconciles `#378`/`#493`).
3. **Migrate the ~20 dev secrets → bw + rotate**, by batch, non-blocking; retire their age files (escrow still covers them).
4. **Curation** (folders, merges, age-key offline) + hardening (`#257`/`#454`/`#321`).

## References
- ADR-002 (age over GPG — the encryption tool; this ADR re-scopes its role and reverses its shell-startup load).
- `docs/secrets-inventory.md` (the 3-source inventory + critical actions).
- Bitwarden CLI docs (studied 2026-06-25): `bw serve` for the facade; `bw export` is plaintext (→ `age -e`); full headless still needs the master password (the floor).
- Tickets: #378, #493, #321, #518, #257, #454, #577.

## Addendum (2026-06-26): bw mapping convention — ratified + reconciled with the implementation

The `dotf secrets` lifecycle build-out (#612, the `set`/`migrate` work) exercised §6 and
surfaced points to ratify and reconcile against the shipped code.

### Item granularity — one item per *credential* (A1 base, A2 selective)

Ratifies the §6 "Field-vs-item rule" as the operative model:

- **A multi-part single credential → ONE item + named fields.** The X OAuth app (7
  values, rotated together) is `x-twitter-api` with 7 kebab fields; a login pair
  (DockerHub) is one item with typed `username` + `password`.
- **Different-purpose credentials → SEPARATE items**, because they rotate / revoke /
  are consumed independently (least-privilege). The GitHub tokens split per purpose
  (`github-cli-pat`, `github-release-pat`, `github-bitacora-pat`) — #321.

"Group by service" never overrides "separate by credential": same OAuth app groups,
different-scope tokens split.

### Value placement — named fields by default

Refines §6's `field: null → password`. Prefer a **named custom field** (`api-key`,
`api-token`, `auth-key`) even for a single-credential item — self-documenting and
uniform with multi-field items. Reserve typed `username`/`password` for genuine login
pairs (DockerHub); `notes` for multi-line blobs (kubeconfig, backup codes). SSH keys
use `notes`/custom fields for now — the native SSH Key item type (5) needs a
`fieldFromItem` reader extension (tracked follow-up), so it is **not** used yet.

### The registry `bw:` block is the SSOT — declared up-front, read not guessed

`set` and `migrate` resolve the Bitwarden target **only** from the entry's `bw: {item,
field}` block. A secret declares its `bw:` target **at rewrite time, while still
`backend: age`**, so the value can be written to bw and parity-checked before the
backend flip. `migrate` never derives `item`/`field` from the `id`; a missing `bw:`
block is an error. At cutover, `SetBackendBW` flips `backend: age → bw` and keeps the
already-declared target in place (it takes no item/field — the declared block is the
single source for both the parity write and the post-flip resolution).

### Registry identity — the `id` is the env var; the `item` groups and is mutable

One registry entry per **env var**, and the `id` **is** that env var name — the stable
consumer contract apps depend on (`OPENAI_API_KEY`, `X_BEARER_TOKEN`). The Bitwarden
`bw.item` is a **mutable pointer**: it GROUPS related vars (the 7 `X_*` entries share
`item: x-twitter-api`, distinct `field`s) and can be renamed by editing only that line —
consumers never reference it. A consequence: every entry is single-var, so
`SetBackendBW`/`migrate` apply uniformly with **no multi-field special case** (the
former M3/M6 blocker dissolves). Global env-var uniqueness falls out for free, since the
`id` is the var and ids are unique (closes the B1 audit gap at the schema level).

### Schema reconciliation — `folder` is not part of the mapping

The shipped `BWSource` is `{ item, field }` — **no `folder` key** (the §"Registry
schema" example showed one). Bitwarden resolves an item by **name or id**; the folder is
organizational metadata, not a lookup key. Items are organized under
`Dotfiles/{apps,infra,personal,floor}` separately (matching `plane`) — and the taxonomy
gains **`Dotfiles/personal`** for the personal-plane recovery codes / app-passwords the
registry manages (§"folder taxonomy" listed only apps/infra/floor). The implemented
`expose.env` accepts scalar / list / `{age|field}`-map forms, matching §6's intent.

### Floor stays age-only

Floor secrets (`ssh-id-ed25519`, the age keys) carry **no `bw:` block** and are never
migrated — they are needed before Bitwarden is reachable (circular dependency, §4).

Implementation: `specs/CLI-024-secrets-set` (write primitive), `specs/CLI-024-secrets-migrate`
(cutover). A draft A1 registry rewrite seeds the per-entry `bw:` targets.
