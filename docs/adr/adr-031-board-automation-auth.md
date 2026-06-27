---
id: "dotfiles-adr-031-board-automation-auth"
type: adr
adr: "031"
title: "Bitácora board automation auth — keep the fine-grained PAT, mitigate don't eliminate"
tags: [adr, dotfiles, secrets, ci, projects, github-app]
status: accepted
created: "2026-06-26"
owner: manu
relates_to: "adr-024 (pat-expiry preflight), adr-028 (two-tier secrets)"
---

# ADR-031: Bitácora board automation auth

## Context

Two workflows move issues/PRs on the bitácora board (a **user-owned** GitHub
Projects v2: `users/mlorentedev/projects/1`):

- `add-to-project.yml` — add opened issues/PRs to the board.
- `bitacora-status.yml` — flip an assigned issue to *In Progress*.

Both authenticate with `secrets.BITACORA_PAT`, a **fine-grained PAT** (the workflow uses
the github-script GraphQL path precisely because `gh project` rejects a fine-grained PAT
with "unknown owner type"). Fine-grained PATs **must** expire (max 366 days). On
2026-06-26 `BITACORA_PAT` expired: the board automation returned HTTP 401 for days, and
`dotf secrets sync ci` even re-uploaded the dead value (a redeploy refreshed `updated_at`
on a 401 token). The recurring question: how do we stop a long-lived board credential
from silently breaking automation?

The obvious answer — replace the PAT with a **GitHub App installation token** (short-lived,
least-privilege, no expiry to manage) — was investigated and **does not work here**.

## Finding: GitHub Apps cannot write a user-owned Projects v2

GitHub App installation tokens can manage **organization**-owned Projects v2 (the
`organization_projects` permission), but **not user-owned** Projects v2. GitHub support has
confirmed apps cannot access user-level v2 projects; the only ways to mutate a user board
are a PAT or an OAuth App. An installation token is scoped to its installation and cannot
see or mutate a user-account project at all (community discussions #159005, #64849; GitHub
docs on App permissions). So `actions/create-github-app-token` is a dead end for this board
unless the board first moves to an organization.

## Options

- **A — Move the board to an organization, then use a GitHub App.** The only path that
  truly *eliminates* the long-lived credential: org Projects v2 support App installation
  tokens (short-lived, least-privilege). Cost: create an org, transfer/recreate the
  Project, re-capture its node/field IDs in both workflows, re-point every repo's rollout.
  Heavy, and disproportionate for a solo personal board today.
- **B — Classic PAT with no expiration.** Kills the expiry chore, but a classic PAT is
  coarse: `project` scope grants read/write to *all* projects and `repo` to *all* repos,
  with no per-resource scoping — and it never expires, so a leak is total and permanent
  until manually revoked. A long-lived, broad bearer token is the exact opposite of the
  least-privilege/JIT posture of ADR-028. **Rejected** as a security regression.
- **C — Keep the fine-grained PAT, mitigated.** The credential stays bounded in scope
  (only the Projects permission + the specific repos) and in time (≤366 days). The expiry
  is no longer *silent*: it is caught loudly and never propagated.

## Decision

**Option C.** Keep `BITACORA_PAT` as a fine-grained PAT and make its expiry survivable
rather than eliminated, via two guards already shipped plus a calendar habit:

1. **Detect loudly (Tier 0, #637).** `pat-expiry.yml` now *fails the job* (red `::error`)
   on an invalid/expired token and warns within 14 days of expiry — no longer an ignorable
   backlog issue. (It also had a latent `GH_REPO` bug that made it never run; fixed.)
2. **Never propagate a dead token (Tier 1, #639).** `dotf secrets sync ci` validates a
   `validate: github-token` entry with `gh api user` **before upload**; a dead token aborts
   the sync, so a 401 value can't reach Actions.
3. **Rotate on signal.** When the preflight goes red (≤ yearly), rotate the fine-grained
   PAT and `dotf secrets sync ci`.

No workflow changes are needed: the workflows already use the PAT correctly; the fix was
the surrounding detection/propagation guards, not the auth mechanism.

## Consequences

- The board automation keeps working with a least-privilege, time-bounded credential.
- The expiry becomes a loud, ~annual, 5-minute rotation instead of a silent multi-day
  outage. The two guards make "silent dead token" structurally hard to reach.
- The long-lived-credential class is *mitigated, not eliminated*. Eliminating it requires
  option A (org migration), tracked as future-enhancement #640 — revisit if the board ever
  moves to an organization for other reasons.
- A classic no-expiry PAT (B) is explicitly off the table: it trades a small recurring
  chore for a large standing risk.

## References

- Tiers: #637 (pat-expiry fail-loud), #639 (sync ci liveness), this ADR (#635 follow-up).
- Constraint: GitHub community discussions #159005 / #64849; GitHub docs, "Choosing
  permissions for a GitHub App".
- Prior art: ADR-024 (pat-expiry preflight), ADR-028 (two-tier secrets governance).
