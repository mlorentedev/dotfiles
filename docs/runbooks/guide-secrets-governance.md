---
id: "dotfiles-runbook-secrets-governance"
type: runbook
status: active
tags: [runbook, dotfiles, secrets, bitwarden, age, governance]
created: "2026-06-25"
owner: manu
---

# Secrets Governance — Add / Rotate / Backup / Recover

> Operational protocols for the two-tier model in [ADR-028](../adr/adr-028-secrets-two-tier-bitwarden-age.md): **Bitwarden = live SSOT**, **age = DR escrow + bootstrap floor**, behind a `dotf secrets` facade + `secrets/registry.yaml`. As the model rolls out, the `dotf secrets` steps activate (Phase 1-2); until then the **manual `bw`/`age` equivalents** apply (marked _manual today_). Supersedes the age-only flow in `secrets-management.md`.

## Conventions (from ADR-028)

- Managed secrets live under **`Dotfiles/{apps,infra,floor}`** in Bitwarden; the ~125 personal items are a separate tree, out of `dotf secrets`' bounds.
- The **registry** `secrets/registry.yaml` is the SSOT: `id → bw item/field → env|file → consumers → rotate`.
- **Values never render** to a human, log, or chat; **never `bw export` to plaintext on disk** (always pipe `--raw` into `age`).

## Protocol — ADD a secret

When a new API key/token/credential enters the system:

1. Decide the **plane**: `app` (service key) / `infra` (access) / `personal` / `floor` (needed before bw — rare).
2. Create the Bitwarden item under `Dotfiles/<plane>`, named `<service>-<purpose>` (kebab):
   - single value → item password; multi-value → custom fields (kebab names).
   - _manual today:_ `bw get template item | jq '.name="…" | .folderId="…" | …' | bw encode | bw create item`.
3. Add a **registry** entry: `{id, plane, backend: bw, bw:{folder,item,field}, expose:{env|file}, consumers, rotate}`.
4. Wire consumers (env-var contract / file target).
5. **Verify** it resolves without leaking: `dotf secrets run --only <id> -- printenv <VAR>` (never pipe the value anywhere shared).
6. **Do NOT** add it to `sensitive/*.age` — that path is retiring; the DR escrow already covers it.

## Protocol — ROTATE a secret

Scheduled cadence (registry `rotate`), suspected exposure, or offboarding:

1. **Map consumers first** (registry `consumers`) — rotation breaks them all at once. Per-purpose items keep the blast radius small (#321). **Never big-bang.**
2. Generate the new value (`bw generate -uln --length N` or the provider console).
3. Update the Bitwarden item (_manual today:_ `bw edit item <id> …`), then `bw sync`.
4. Roll consumers: `dotf secrets sync <target>` re-materializes CI/containers/agents; local picks it up on the next `run --`.
5. **Verify each consumer works, THEN revoke the old value** at the provider.
6. Refresh the DR escrow (below) so the new value is recoverable.

## Protocol — BACKUP / DR escrow

Scheduled (e.g. weekly) + before any big change:

1. `bw sync`
2. `bw export --format json --raw | age -r <pubkey> -o sensitive/dr/bitwarden-export.age`
   — plaintext **never** touches disk (`--raw` → pipe → `age`).
3. **Overwrite** the previous export (don't accumulate full-vault snapshots in git history); commit.
4. **Mirror off-box** (#454) and ensure the **age key has an OFFLINE authoritative copy** (`backup-secrets-to-usb.sh` + paper/OS keychain) — it must NOT live only in Bitwarden (circular dependency).
   - _Target:_ `dotf secrets backup` wraps steps 1-3.
   - The escrow covers the **entire** vault (API keys, tokens, logins, TOTP seeds) → losing Bitwarden is fully recoverable.

## Protocol — RECOVER (disaster)

Lost Bitwarden access / new machine / account compromise (the OPS-001 #257 chain):

1. Restore the **age key** from its offline backup → `~/.config/age/key.txt`.
2. Clone the dotfiles repo.
3. `age -d -i ~/.config/age/key.txt sensitive/dr/bitwarden-export.age > $TMPDIR/vault.json` (ephemeral / tmpfs).
4. Stand up a fresh Bitwarden (or any manager) and **import** `vault.json`; or read individual secrets for immediate needs.
5. Re-establish: `bw login` + unlock; `dotf secrets sync` to re-materialize consumers.
6. **Rotate** anything that may have been exposed during the incident.
7. Securely delete `$TMPDIR/vault.json`.
   - Account-independent: **age key (offline) + repo clone = full recovery**, even if the Bitwarden account is gone.

## Maintainability (what keeps it from drifting)

- `dotf doctor` checks (target): `bw`/`age` present (#577); DR-export freshness; **registry ↔ vault consistency** (flag `Dotfiles/**` items that break the naming/registry convention).
- All adds/rotations go through the **registry** — the single map. No ad-hoc env edits, no second authoritative copy.

## References

- [ADR-028](../adr/adr-028-secrets-two-tier-bitwarden-age.md) (decision + structure), `docs/secrets-inventory.md` (the map), [ADR-002](../adr/adr-002-age-over-gpg.md) (age).
- Tickets: #378, #493, #321, #518, #257, #454, #577.
