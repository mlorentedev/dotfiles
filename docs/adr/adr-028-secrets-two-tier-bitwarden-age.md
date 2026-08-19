---
id: "dotfiles-adr-028-secrets-two-tier-bitwarden-age"
type: adr
adr: "028"
title: "Two-Tier Secrets Governance — Bitwarden live SSOT + age DR floor, via a dotf secrets facade"
tags: [adr, dotfiles, secrets, bitwarden, age, governance]
status: accepted
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

Bitwarden personal folders nest by `/` in the name. `dotf secrets`-managed items live in a flat set of purpose folders, kept distinct from the ~125 personal items (which stay where they are) by name rather than by a shared parent:

```text
apps              # service API keys/tokens the projects consume
infra             # infrastructure access (servers, k8s, VPN, registries, DNS, CI runners)
floor             # DR/bootstrap roots — age-key convenience copy, master-pw backup (offline-authoritative)

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

`dotf secrets` only ever reads/writes the folders named above. The personal tree is curated as a separate, optional, incremental pass and is out of `dotf secrets`' bounds. The whole vault gets a domain taxonomy because a 125-item flat vault is neither auditable nor maintainable — but organizing the personal tree never blocks the dotfiles security work. **Decided:** keep the `Dotfiles/` namespace; **split** the GitHub 8-in-1 item per purpose (#321). Future scale ceiling: move shared secrets to a Bitwarden **Organization + Collections** (ACL + Secrets Manager) if/when agents/CI/team consume them.

> **Amendment (2026-08-15): the `Dotfiles/` prefix is dropped — folders are `apps`, `infra`, `floor`.**
>
> The namespace half of the decision above is reversed; the GitHub-item split (#321) stands. The prefix bought isolation from the personal items, but that isolation was never load-bearing: `bw.folder` is placement metadata only — item lookup is by unique name and ignores folders entirely — so nothing resolves differently with or without it. What it cost was paid on every read of the registry and every folder name typed by hand, for a distinction the folder names already make on their own.
>
> The planned personal tree (`Finance`, `Travel`, …) becomes a sibling of `apps`/`infra` rather than a peer of the `Dotfiles/` root. That is the real trade accepted here: one flat namespace where names must not collide, instead of two trees that cannot.
>
> Migration is not automatic. Renaming the folders in Bitwarden moves the items with them; until that is done, a vault still holding `Dotfiles/apps` keeps resolving every secret correctly and only new items land in the new folders.
>
> **Two different sets, easy to conflate — and easier now that the prefix is gone.** The folders that *exist in the vault* are `apps`, `infra`, `floor` (above). The folders a *registry entry may declare* in `bw.folder` are only `apps` and `infra` — `validBWFolders` in `cli/internal/secrets/registry.go` rejects anything else. `floor` is absent because floor secrets are age-only and carry no `bw:` block at all; a personal-plane folder is absent because none is ratified yet (deferred to #586). So "`<plane>` names a vault folder" holds for organizing the vault by hand, and does **not** mean every plane is a legal `bw.folder` value. Before the flattening, `Dotfiles/<plane>` read unmistakably as a vault path; `<plane>` alone reads like a registry value, which is why the distinction is now written down rather than left to a code comment.

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

  - id: AGE_KEY_PERSONAL               # the ROOT: authority is this disk, bw copy = convenience
    plane: floor
    backend: file-authority            # not age-offline — see below
    bw: { item: AGE-SECRET-KEY-PERSONAL }
    expose: { file: { var: AGE_KEY_PERSONAL, path: "~/.config/age/key.txt", mode: "0600" } }
    consumers: [local]
```

**Amended 2026-08-19 (#937, OPS-026).** This example previously declared
`backend: age-offline`, an `offline: required` field, and a file expose without
`var`. None of the three was implementable, and the entry was never added for that
reason rather than by oversight:

- `age-offline` with a file expose **requires** an `age:` source — the basename of a
  ciphertext under `sensitive/`. The age identity has none and can have none: every
  such ciphertext is decrypted *with* this file, so it would have to be encrypted
  under itself.
- `expose.file` requires `var` as well as `path`.
- `offline:` is not a field on the schema. It would have been silently ignored,
  which is worse than being rejected.

The root is therefore its own backend, `file-authority`: a secret whose authority
**is** the local plaintext file. Nothing resolves it, so `verify` asks the questions
that actually mean something for a root — present, `0600`, and (once #1000 lands)
still matching the copy held off this machine — and `dotf secrets run` refuses to
materialize it, because handing the key that decrypts every other secret to a child
process through the same facade as those secrets widens the blast radius instead of
narrowing it.

The `bw:` block carries no folder: the ratified taxonomy has none for `floor`
(`validBWFolders` is `apps`, `infra`), and adding one is a separate decision. It
names where the convenience copy lives, which is where the drift comparison of
#1000 will look. **Authority stays on disk.**

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
`{apps,infra,personal,floor}` separately (matching `plane`) — and the taxonomy
gains **`personal`** for the personal-plane recovery codes / app-passwords the
registry manages (§"folder taxonomy" listed only apps/infra/floor). The implemented
`expose.env` accepts scalar / list / `{age|field}`-map forms, matching §6's intent.

### Floor stays age-only

Floor secrets (`ssh-id-ed25519`, the age keys) carry **no `bw:` block** and are never
migrated — they are needed before Bitwarden is reachable (circular dependency, §4).

Implementation: `specs/archive/CLI-024-secrets-set` (write primitive), `specs/archive/CLI-024-secrets-migrate`
(cutover). A draft A1 registry rewrite seeds the per-entry `bw:` targets.

## Ratification (2026-06-28): accepted — execution begins

The `dotf secrets` lifecycle (`set`/`migrate`/`sync`/`verify`) is shipped and hardened
(#612 Phase B, v0.29.0), and `bw`/`age` are provisioned (#577). The design above is
**accepted**; the migration off age begins. Three rollout decisions are ratified here so
they live in the record, not only in conversation:

1. **Repo-side foundations precede vault-side curation.** The DR escrow (`dotf secrets
   backup`, §5 — not yet built) and the `recover` runbook (§7) must exist *before* the
   least-reversible vault mutations (moving the `AGE-SECRET-KEY-*` offline per §4,
   restructuring live items). Repo-side work needs no `bw unlock`, so it proceeds in
   parallel with the operational unlock. A one-off manual escrow snapshot is the interim
   safety net until the command lands.

   **Addendum (built, #661):** `dotf secrets backup` shipped (`cli/internal/cmd/secrets_backup.go`,
   `cli/internal/secrets/escrow.go`). The "not yet built" framing above is historical —
   see §5 for the current behavior.

2. **The §"Phased plan" step-3 "~20" resolves into three tranches** under the shipped
   `migrate` guard (which refuses file secrets and shared-source entries):
   - **A — ~23 env secrets, unique age source:** migratable now, batch-by-batch, `verify`
     each (canary first).
   - **B — 6 `file:` secrets** (kubeconfig, the recovery/backup codes): need a `migrate`
     file-target extension before they can cut over.
   - **C — the 2 shared-source GitHub PATs** (`GITHUB_PERSONAL_ACCESS_TOKEN` +
     `RELEASE_TOKEN`, both on `github.token`): need `migrate --split` (#321), which issues
     *distinct* tokens — sequenced last.

3. **Retiring the age files (C6) is gated on the DR escrow existing.** The escrow keeps the
   age DR export covering everything, so a migrated secret's `.secret.age` is deleted only
   after the escrow (and its verified round-trip) is in place — never as a side effect of
   `migrate` (which keeps the age file by design).
