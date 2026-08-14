---
id: "OPS-028-bw-folder-taxonomy"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-14"
issue: "mlorentedev/dotfiles#951"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-028-bw-folder-taxonomy

> **Naming**: file lives at `<repo>/specs/OPS-028-bw-folder-taxonomy/proposal.md`. `OPS-028-bw-folder-taxonomy` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #951: OPS-028: ADR-028 ratified a Bitwarden folder taxonomy the registry schema cannot express -->

ADR-028 ratified a Bitwarden folder taxonomy (`Dotfiles/apps`, `Dotfiles/infra`, `Dotfiles/floor`) and a worked example showing `bw.folder` in the registry, but `BWSource` (`cli/internal/secrets/registry.go`) never got the field — zero of 33 entries declare a folder, and every item `migrate`/`set` creates lands flat. Issue #585 is about to bulk-migrate ~21 secrets; doing that against the current schema manufactures 21 more flat items with no way for the registry to later say where they belong — unowned, unchecked, manual-reorg debt of exactly the kind #951 was filed to prevent. The already-migrated canary (`OPENAI_API_KEY` → `openai-api-key`) already demonstrates the gap live: it sits in "No Folder" today, contradicting the ADR's own worked example for that same item.

## What

`BWSource` gains a `Folder` field (`yaml:"folder"`). The Bitwarden write path (`CreateItem`, used by `set`/`migrate`) resolves the declared folder name to a Bitwarden folder id — creating the folder via `bw create folder` when it doesn't exist yet — and places the new item there instead of leaving it unfoldered. Every app/infra-plane registry entry that carries a `bw:` target gets its `folder:` populated (`Dotfiles/apps` or `Dotfiles/infra`, matching its `plane:`). The already-created `openai-api-key` item is moved into `Dotfiles/apps` as a one-time manual Bitwarden move (not a re-migration — the value doesn't change).

## Out of scope

- The 21-secret bulk migration itself (#585) — this spec only makes the schema/writer able to place items correctly; the batch runs on top of this, after merge.
- Personal-plane folder taxonomy (`ZOHO_*`, `GMAIL_BACKUP_CODE`, `CHATGPT_*`, `STRIPE_BACKUP_CODE`) — ADR-028's ratified taxonomy has no `Dotfiles/personal` folder; deferred to #586. None of these six are in #585's first batch anyway (they're file secrets or multi-line), so this gap doesn't block anything today.
- The GitHub token per-purpose split (#321) and the `migrate --split` flag (C9) — an unrelated schema gap (shared age source, not folder placement).
- Reorganizing the ~125 unrelated personal Bitwarden items (Finance/Travel/Work/…) — explicitly out of `dotf secrets`' bounds per ADR-028.

## Risks / open questions

- **Casing collision, must resolve before writing any registry value.** Issue #951's own worked example quotes `folder: "dotfiles/apps"` (lowercase); ADR-028 §"Bitwarden folder taxonomy" canonically writes `Dotfiles/apps` (capital D). Treating the ADR as canonical and #951's quote as a transcription slip — use `Dotfiles/apps` / `Dotfiles/infra`.
- **`bw create item` takes a `folderId`, not a name.** Needs a `bw list folders` name→id lookup plus `bw create folder <name>` for the folders that don't exist yet in the vault (confirmed neither `Dotfiles/apps` nor `Dotfiles/infra` exists today — ADR: "Vault = 152 items, still all in No Folder"). Resolve-or-create must be idempotent (a second run must not create a duplicate folder).
- **Moving the existing `openai-api-key` item.** Needs confirming the `bw` CLI mechanism (`bw edit item` with `folderId` vs. an equivalent) that changes an item's folder without touching its value — must verify the field value is unchanged after the move.
- **Blast radius.** This touches the shared write path (`CreateItem`) used by both `set` and `migrate`. Must not regress the existing single-item, no-folder-declared flows still in use elsewhere (e.g., an entry with no `folder:` should keep working exactly as today, not error).

## Acceptance criteria

- [ ] `BWSource` carries `Folder` (registry.yaml `bw.folder`); `ParseRegistry` validates it against the ratified set (`Dotfiles/apps`, `Dotfiles/infra` — floor and personal excluded per Out of scope).
- [ ] The writer places a newly created item in its declared folder, creating the folder via `bw create folder` if it doesn't exist yet; a test asserts the folder the registry declares is the folder the writer's request body carries (mirrors #951's own acceptance criterion).
- [ ] Every app/infra-plane registry entry with a `bw:` target declares `folder:` matching its plane.
- [ ] The `openai-api-key` Bitwarden item is moved into `Dotfiles/apps` (one-time manual move, tracked as a task, not a code change) and its value is verified unchanged after the move.
- [ ] An entry with no `folder:` declared (if any remain, e.g. future personal-plane entries) still creates/resolves exactly as it does today — no regression.

## References

- Related ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md` (§"Bitwarden folder taxonomy")
- Bitácora: #951 (this spec), #585 (bw backend + migration, downstream consumer of this work), #586 (broader curation — personal folder taxonomy, token split, DR escrow)
