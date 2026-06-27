---
id: "CLI-024-secrets-sync-verify"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#635"
tags: [spec, proposal, secrets, sync, cli, go]
template_version: "1.0"
---

# CLI-024-secrets-sync-verify

## Why

`dotf secrets sync ci` uploaded a **dead** `BITACORA_PAT` to Actions and reported success
(the redeploy refreshed `updated_at` on a token that returns HTTP 401). Nothing checked
that the value still authenticates, so a broken credential reached CI and the board
automation 401'd for days. The upload step verifies the *write*, never the *liveness* of
the payload.

Industry practice does not validate arbitrary payloads at sync time — liveness does not
generalize across providers (each has a bespoke probe, or none). But for the credentials
we **can** cheaply probe — GitHub tokens via `gh api user` — an opt-in pre-upload check is
high value at near-zero cost, and catches exactly this incident class.

## What

- Registry gains an optional `validate:` key on a secret (`validate: github-token`).
  Empty = today's behavior (no check). Opt-in, per entry.
- `secrets.GitHubTokenValidator` seam (`GHTokenValidate` in production: `gh api user`
  with the resolved token as `GH_TOKEN`, authenticating AS the token under test).
- `dotf secrets sync ci` runs a **pre-upload** liveness pass: every selected entry marked
  `validate: github-token` must authenticate **before any upload**. A dead token aborts
  the whole sync (fail loud) — a broken credential is never pushed. `--skip-verify`
  bypasses (operator escape hatch).
- `RELEASE_TOKEN` and `BITACORA_PAT` are marked `validate: github-token`.

Scope is deliberately narrow: only GitHub tokens, only opt-in entries. It does NOT try to
validate every secret — that is a per-provider tar pit the industry avoids. Token expiry
*monitoring* stays a separate concern (Tier 0, `pat-expiry.yml`); *eliminating* the PAT is
Tier 2 (GitHub App).

## Acceptance criteria

- **AC1** — A selected entry marked `validate: github-token` whose value fails the probe
  aborts the sync before any upload (nothing is pushed).
- **AC2** — A live marked token validates, then uploads as before (with a `verified` line).
- **AC3** — `--skip-verify` bypasses the check (validator never called); unmarked entries
  never trigger the validator (opt-in only).
- **AC4** — Registry parses the new `validate` field; suite green, vet + fmt clean, builds.

## Out of scope

- Validating non-GitHub secrets (no generic liveness probe exists).
- "Skip identical" — Actions secrets are write-only (no readback to compare); a re-upload
  is idempotent and harmless.
- Tier 0 (pat-expiry fail-loud) and Tier 2 (GitHub App) — separate PRs.
