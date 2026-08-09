---
id: "CLI-021-dotf-vault-build-knowledge"
type: spec
status: draft
created: "2026-08-08"
issue: "mlorentedev/dotfiles#490"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — CLI-021-dotf-vault-build-knowledge

> Not started. Implementation begins after the open question in `tasks.md` §0 is resolved and the
> golden corpus in §1 is captured. Filled per increment as they land.

## Evidence

**Increment 1 — `dotf vault crystallize`:** not started
**Increment 2 — `dotf vault health`:** blocked on the health-noun question
**Increment 3 — `dotf vault maintain`:** not started

## Test status

- Golden characterization (#672 / CLI-031): corpus not yet captured.
- Table-driven units: not written.
- `test-windows` CI: unchanged by this spec so far.

## The proof that matters most here

Because this is a **build-beside** PR, the load-bearing check is negative: `git diff --stat` must
touch only `cli/` and `specs/`. Any hit in `scripts/`, `setup-*.{sh,ps1}`, or the vault means the
cutover leaked into the build PR, which is precisely the risk AUDIT-007 split PR5 from PR7 to
avoid.

## Promotion candidates

- [ ] Lesson? Candidate from BUG-060, likely promoted at *that* spec's archive rather than here:
      *a maintenance script that has never run is not "safe", it is untested — its first execution
      is a deployment.*
- [ ] ADR-worthy? No. `docs/adr/audit-007-cli-convergence-state.md` already owns the decision; this
      is its PR5.
- [ ] Pattern candidate? Possibly — "characterization-test the oracle before you replace it" as a
      twin-port discipline, if CLI-022..028 repeat the shape. Decide after the third port, not the
      first.

## Archive checklist

- [ ] All three increments landed, or the remainder explicitly descoped with a reason
- [ ] `proposal.md` frontmatter -> `status: archived`
- [ ] Folder moved to `specs/archive/CLI-021-dotf-vault-build-knowledge/`
- [ ] Issue #490 closed with PR links
- [ ] Flip checklist handed to CLI-023 (PR7) — not silently dropped
