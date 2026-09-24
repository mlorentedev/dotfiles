---
id: "GUARD-017-protection-as-code"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-23"
issue: "mlorentedev/dotfiles#1451"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, forge, branch-protection, iac, doctor]
template_version: "1.0"
---

# GUARD-017-protection-as-code

> Epic #1625 (HARNESS-138), item **W1.5**. Acceptance reference: AC-1.5.
> Decisions D-1 (approvals = 0, rationale committed) and D-6 (built in `dotf`,
> declaration keyed by repo, kubelab's tool and web#374 retired on contact)
> are recorded on #1625.

## Why

Branch protection is the last piece of per-repo GitHub configuration still
applied by hand. It leaves no trace in git: a required context that is
renamed, dropped or never added is invisible until a merge that should have
been impossible happens (web#275). Measured on 2026-09-23:

- `spec-gate` is required **nowhere**.
- `iris` is protected with **zero required contexts**, so every one of its
  checks is advisory.
- Nothing records what the 9 protected repos are supposed to require, so
  nothing can report drift.

The Zero-Manual-Operations order forbids fixing this with a hand-run
`gh api -X PUT`.

## What

Two PRs, split at the read/write boundary so that the change which can lock a
repository is reviewed on its own.

### PR-A — declare and detect (read-only)

1. **`forge/branch-protection.json`**, validated by
   `forge/branch-protection.schema.json`. It declares, per repo:
   - the protected branch;
   - the **complete** protection object, with every field the API returns.
     `PUT` replaces the whole object, so a field left out of the declaration
     is silently cleared on apply (#1451 note 1). Required checks are declared
     with their **source** (`{context, app_id}`), because a context of the same
     name posted by another app would otherwise satisfy it.
   - or one explicit state instead of an object: `unavailable` (private repo on
     the free plan; the API returns 403), or `unprotected`. Each state carries
     a mandatory `reason`.

   The first commit of the file declares **today's live state** for every repo
   that has protection. A declaration that matches reality from day one turns
   drift detection on immediately, and leaves nothing for PR-B to apply except
   the deliberate change.
2. **Policy block, D-1.** It holds `required_approving_review_count: 0` and the
   rationale, as prose committed next to it: single maintainer; GitHub forbids
   self-approval and `enforce_admins` is true, so a value ≥ 1 locks the owner
   out; the human gate is the owner's deliberate merge plus the required
   `review-attestation`. Every repo's declared count must equal the policy
   value unless the repo declares its own override and reason.
   `TestProtectionApprovalsRationalePinned` pins both the value and a digest of
   the rationale, so neither can change without an edit to the test that the
   diff shows.
3. **`cli/internal/forge`**: load and validate the declaration; normalise the
   GET shape (`{"enabled": bool}` wrappers, `checks[]`) into the declared
   shape; diff declared against live **field by field**.
4. **`dotf forge protection check [--repo owner/name]`** prints one line per
   drifted field. Its exit contract is the same as `dotf pr triage-queue`:
   non-zero on drift *or* on an unanswerable lookup.
5. **A `branch-protection` section in `dotf doctor`** (full sweep only). Drift
   is a FAIL that names the repo and field. `unavailable` and `unprotected` are
   a Skip that shows the declared reason. `gh` absent is a Skip, and an API
   error is a WARN. None of these is ever a PASS.

### PR-B — apply (writes the forge)

6. **`dotf forge protection apply [--repo] [--dry-run]`**. It GETs the live
   object, builds the complete PUT body from the declaration, PUTs it, and GETs
   again to assert that every declared field took effect. A 200 only means the
   request was accepted; it does not mean the fields were applied (kubelab's
   lesson). A second run reports `changed=0`.
7. **A reported-context preflight.** `apply` refuses to make a context
   *newly* required unless it has reported on at least one of the repo's last
   5 merged PRs (#1451 note 2, risk R-1). With `enforce_admins: true`, a
   required context that never reports makes the branch unmergeable, the
   owner included.
8. **The deliberate change:** add `spec-gate` to `mlorentedev/dotfiles` `main`.
   It is applied by the owner running `dotf forge protection apply`, never by
   the implementing agent unprompted.

## Out of scope

- Rolling `spec-gate` out to other repos. That is W1.6 (#1627), which reads its
  target list from this declaration.
- Rulesets and merge queues (web#375). Rulesets need the paid plan for orgs,
  and classic protection is what every repo uses today.
- Scheduling the drift check in CI. That needs an admin-scoped machine
  identity, which is decision (d) of the CI-identity note. Until then, drift is
  checked by `dotf doctor` on the owner's machine.
- Retiring kubelab's `toolkit/features/branch_protection.py` and closing
  web#374. Both happen when that work reaches them (D-6). kubelab's session
  owns its repo.

## Risks / open questions

- **Owner lockout (R-2).** This is prevented by D-1's pinned test and by the
  PR-B preflight. The rollback: protection changes through the admin API, not
  through a merge, so re-applying the previous declaration from git history
  recovers a deadlocked branch.
- **A context that never reports (R-1).** The PR-B preflight covers it, and the
  first live apply is the owner's decision.
- **Open question: the declaration's path.** I propose `forge/`, as the
  collaborate layer of the knowledge-placement model: GitHub-side state,
  declared in git. The alternatives are `harness/` (agent harness state, which
  is not the right layer) or a private infra repo (decision (b) of the
  CI-identity note, still open). `forge/` is the default unless the owner picks
  otherwise.
- **API cost.** About 10 GETs per `dotf doctor` run, on the REST budget
  (5,000/h), bounded and only in the full sweep.

## Acceptance criteria

- [ ] **AC1 (PR-A).** `dotf forge protection check` exits 0 against today's
  live state for every declared repo. A test fixture that disagrees on one
  field (a dropped context, or `enforce_admins` flipped) exits non-zero and
  names the repo and the field.
- [ ] **AC2 (PR-A).** The `dotf doctor` section `branch-protection` reports
  PASS on today's tree. `unavailable` and `unprotected` repos appear as Skips
  with their reasons.
- [ ] **AC3 (PR-A).** `TestProtectionApprovalsRationalePinned` passes. It fails
  when the policy count changes, when the rationale changes without the test
  changing, and when a repo's count diverges from the policy without an
  override.
- [ ] **AC4 (PR-A).** Schema validation rejects a repo that declares neither a
  protection object nor a state, and a state that has no reason.
- [ ] **AC5 (PR-B).** A test against a fake forge shows `apply` sends a
  complete PUT body (no declared field missing), re-reads, and reports
  `changed=0` on the second run. The preflight refuses a never-reported
  context.
- [ ] **AC6 (PR-B, live, owner-run).**
  `gh api repos/mlorentedev/dotfiles/branches/main/protection -q '.required_status_checks.contexts'`
  includes `spec-gate`. Then `dotf forge protection apply --dry-run` reports
  `changed=0`.

## References

- #1451 (the design notes: whole-object PUT, never-reporting contexts, the
  web#275 incident); web#374 and kubelab#1678 (the prior Python reconcile loop,
  whose GET→overlay→PUT→GET-assert shape this adopts); epic #1625 §3 W1.5,
  §5.2 AC-1.5, D-1, D-6, D-7, R-1, R-2.
- Measurement, 2026-09-23: 24 active repos, 13 with a `specs/` tree, 9
  protected; see the #1627 comment.
