---
id: "dotfiles-adr-029-secrets-sync-headless-materialization"
type: adr
adr: "029"
title: "dotf secrets sync — backend-agnostic deploy-time materialization for headless consumers"
tags: [adr, dotfiles, secrets, sync, ci, bitwarden, age]
status: accepted
created: "2026-06-26"
owner: manu
issue: "mlorentedev/dotfiles#612"
depends_on: "adr-028 (two-tier secrets), adr-020 (CLI strangler-fig)"
---

# ADR-029: `dotf secrets sync` — backend-agnostic deploy-time materialization for headless consumers

> **Status: accepted.** `dotf secrets sync` shipped (#612 Phase B, v0.29.0, `cli/internal/cmd/secrets_sync.go`); frontmatter above updated from `proposed` to match. `scripts/github-secrets-manager.sh`, the shell path the Context section below describes as "today['s]" state, has since been deleted — retired on contact once `sync` reached parity, per the strangler-fig pattern (ADR-020). The Context and Consequences sections below are left as originally written (historical record, per the convention in ADR-020's Amendment) — read their present-tense claims ("today," "blocks the epic," "once sync lands") as describing the pre-shipment state this ADR proposed to fix, not current reality.

## Context

ADR-028 split secrets into a Bitwarden live SSOT + an age DR floor, resolved on demand
through the `dotf secrets` facade. Local, interactive consumers resolve at runtime
(`dotf secrets run -- …`). **Headless consumers cannot**: CI runners, containers, and
agents have no interactive `bw unlock`, so ADR-028 specifies a `sync` step that
materializes a *scoped* secret set **ahead of time** — the headless context never talks
to Bitwarden at runtime.

Today only the CI path exists, and it is shell:
`scripts/github-secrets-manager.sh` consumes `dotf secrets ls --pairs` — which emits
`VAR<TAB>age-source` (the age base name, **age only**) — then decrypts each age file
itself and uploads to GitHub Actions via `gh secret set`.

This blocks the epic (#612 C5). The instant a `ci:*` secret flips to `backend: bw`
(ADR-028 migration, C4 shipped in #627), `ls --pairs` excludes it (no age source to
emit) and it **silently disappears** from the Actions upload. Migrating any `ci:*`
secret is therefore gated on making this materialization backend-agnostic.

Two facts shape the decision:

- **Backend-agnostic value resolution already exists.** `secrets.Loader.EnvFor`
  (`cli/internal/secrets/resolve.go`) dispatches per-entry through an `ageResolver` /
  `bwResolver` map. Resolving a value regardless of backend is a solved problem inside
  the CLI; only the CI boundary still leaks the backend (it asks for an *age source*,
  not a *value*).
- **ADR-020 strangler-fig.** The `dotf` Go CLI absorbs `.sh`/`.ps1` twins; new surface
  should land in `dotf`, not perpetuate a shell script that re-implements resolution.

## Constraints

| #  | Constraint | Origin |
|----|------------|--------|
| C1 | Headless consumers never talk to `bw` at runtime; `sync` materializes ahead of time | ADR-028 |
| C2 | Backend-agnostic: resolve the *value* (age\|bw) by reusing `Loader`, never branch on backend at the boundary | ADR-028 + code |
| C3 | Strangler-fig: `dotf` absorbs `github-secrets-manager.sh`; no thin per-OS shim | ADR-020 |
| C4 | Minimize plaintext at rest; prefer env-only delivery; `0600` + gitignore + ephemeral where a file is unavoidable | ADR-028 |
| C5 | Idempotent, TDD, `--dry-run` on every mutation | #612 Phase C discipline |
| C6 | The live smoke (`bw unlock` + `gh`) is Windows-empirical → deferred; the command core + unit tests are Linux-doable now | batch-windows-work lesson |

## Decision

Introduce a Go command **`dotf secrets sync <target>`** that resolves a scoped secret
set backend-agnostically and pushes it to the target's delivery surface.

- **Target vocabulary:** `{ci, container, agent}`, keyed off the registry `consumers:`
  field (`ci:<repo>`, `container:<name>`, `agent:<name>`). The contract names all three;
  **this slice implements `ci` only** (container/agent `.env` materialization is
  designed here but deferred — see Scope boundary).
- **`sync ci [--repo OWNER/REPO] [--dry-run]`:** select every entry whose `consumers:`
  contains a `ci:*` tag (optionally narrowed to one repo), resolve each value through
  `Loader.EnvFor` (age **or** bw, transparently), and upload via `gh secret set`. This
  **absorbs** `scripts/github-secrets-manager.sh` (C3); the script is retired once parity
  is confirmed.
- **`ls --pairs` retired at the CI boundary.** Its `VAR<TAB>age-source` shape is the
  backend leak this ADR removes; `sync` owns CI materialization end to end, so `--pairs`
  is removed rather than patched to be "backend-aware" (which would keep exposing source
  internals and the shell decrypt path, violating C2/C3).
- **Idempotent + `--dry-run` (C5):** `gh secret set` is idempotent by nature; `--dry-run`
  reports the intended `VAR → repo` set (names + byte lengths, never values) and writes
  nothing. File secrets and `floor`/`age-offline` secrets are excluded with a specific
  reason, mirroring `migrate`'s guard vocabulary.
- **Ordering with migration:** `sync` is backend-agnostic by construction, so it works
  *before, during, and after* a `ci:*` secret's age→bw flip. That is what unblocks the
  gate: once `sync` is the upload path, `migrate <ci-var>` no longer drops the secret.

## Options Considered

1. **`dotf secrets sync ci` Go command, absorb the script (CHOSEN).** Resolve via
   `Loader.EnvFor`, push via `gh secret set`, retire the script + `ls --pairs` leak.
   Satisfies C1–C6. Cost is low because resolution already exists; the new surface is
   selection + `gh` calls + `--dry-run` + tests.
2. **Minimal: make `ls --pairs` backend-aware, keep the bash script.** Smallest unblock,
   but the script must then resolve bw values itself (a second resolution path in shell)
   or `--pairs` must emit *values* over a pipe (plaintext through a shell boundary).
   **Rejected:** violates C3 (perpetuates the script) and C2/C4 (re-introduces a
   backend-specific, source-leaking boundary).
3. **Hybrid: ship `sync ci` now but keep `ls --pairs` for debugging.** **Rejected:**
   leaves a dead, backend-incomplete `--pairs` as a footgun and standing tech debt; if a
   debug view is wanted later, add an explicit `sync --dry-run` or `ls --resolved`, not a
   half-migrated legacy flag.

## Consequences

### Positive
- Unblocks `ci:*` migration (#612 C5 → C-phase can finish); the gate in `migrate`'s
  guard can be lifted once `sync` lands and parity is shown.
- One resolution path for every consumer (local `run`, CI `sync`) — no shell re-implementation.
- Retires a `.sh` twin (ADR-020 progress) and removes a backend-leaking interface.

### Negative
- `sync ci` reads bw → requires `bw unlock` + `gh auth` for the live run; that smoke is
  Windows-empirical and deferred (C6). Unit tests must mock the `gh` seam and the resolver.
- A transition window where both the script and `sync` exist until parity is confirmed.

### Mitigations
- Gate the script's retirement on a one-time parity check (`sync --dry-run` set ==
  current script's uploaded set) before deleting `github-secrets-manager.sh` and its bats.
- Mock `gh secret set` behind a small seam (mirror the `bwReader`/`BWWriter` pattern) so
  the command is fully unit-testable on Linux without secrets or network.

## Scope boundary

- **In:** `dotf secrets sync ci`, backend-agnostic resolution at the CI boundary,
  `--dry-run`, retiring `ls --pairs` + `github-secrets-manager.sh` (parity-gated).
- **Deferred (designed, not built):** `sync container` / `sync agent` → `0600` `.env`
  materialization (C4 file-at-rest concerns apply); a `sync` for `local` (none — local
  uses `run`).
- **Out:** the `github.token` 1→2 split (#612 C9 / #321); `retire`/`backup` (C6);
  rotation (C7).

## References

- Issue: `mlorentedev/dotfiles#612` (Phase C, item C5).
- ADR-028 (two-tier secrets governance) — the `sync` step is specified there.
- ADR-020 (CLI Go convergence / strangler-fig) — absorb the shell twin.
- Reuse: `cli/internal/secrets/resolve.go` (`Loader.EnvFor`, resolver map).
- Retires: `scripts/github-secrets-manager.sh`, `tests/github-secrets-manager.bats`,
  `dotf secrets ls --pairs`.
