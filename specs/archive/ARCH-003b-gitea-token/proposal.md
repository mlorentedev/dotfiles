---
id: "ARCH-003b-gitea-token"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-29"
issue: "mlorentedev/dotfiles#1912"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# ARCH-003b: Teledyne Gitea automation token

> **Naming**: file lives at `<repo>/specs/ARCH-003b-gitea-token/proposal.md`. `ARCH-003b-gitea-token` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- work gate: https://github.com/mlorentedev/dotfiles/issues/1912 — ARCH-003b: Register the Gitea automation token -->

The private `teledyne/projects-toolkit` PoC needs API automation without
embedding a Gitea token in shell history, repository configuration, or scripts.
The token must be governed by the existing `dotf secrets` registry so callers
receive it only inside an explicitly scoped child process.

## What

- `dotf secrets set GITEA_TELEDYNE_TOKEN` stores the token in the mapped
  Bitwarden item through a hidden prompt.
- `dotf secrets run --only GITEA_TELEDYNE_TOKEN -- <command>` injects it only
  into the Gitea API child process.
- The registry declares the Bitwarden item, environment variable, consumer,
  rotation interval, and infrastructure plane.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- Creating or rotating the Gitea token itself.
- Storing a token value in Git, a URL, process arguments, or documentation.
- Selecting or merging changes between `fae-brain`, `openkm-brain`, and
  `projects-toolkit`.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- Gitea PAT scopes are global to the bot account's accessible resources, not
  restricted per repository; the dedicated bot's organization membership is
  therefore the resource boundary.
- The token uses `write:organization` and `write:repository`; no admin scope is
  required.
- No open design question remains.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] **AC1:** The registry declares `GITEA_TELEDYNE_TOKEN` as an infrastructure
  Bitwarden secret mapped to `gitea-teledyne-bot/api-token` and exposed only as
  the same environment variable.
- [x] **AC2:** The normal `dotf secrets` parser and command tests accept the
  registry, and `secrets ls` lists the ID without printing its value.
- [x] **AC3:** An operator can run `dotf secrets set` and `verify --require-all`
  successfully without placing the token on the command line or in repository
  content.

## References

- Bitácora: `mlorentedev/dotfiles#1912`
- Controller decision: `mlorentedev/knowledge#177`
- Controller ADR: `projects-toolkit/docs/adr/adr-004-private-controller-and-work-context-routing.md`
- Secret doctrine: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`
