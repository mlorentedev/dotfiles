## Summary

<!-- 1-3 bullets. What and why. -->

-

## SDD checklist

<!-- Enforced by .github/workflows/spec-gate.yml. See AGENTS.md "Discipline Gate". -->

- [ ] Vault backlog entry exists in `$VAULT_PATH/10_projects/dotfiles/11-tasks.md` (default `~/Projects/knowledge`)
- [ ] Spec folder `specs/<feature-id>/` is included in this PR (or `skip-sdd` label below)
- [ ] `proposal.md` has filled Why / What / Acceptance criteria
- [ ] `tasks.md` is in TDD order
- [ ] `verification.md` will be filled before merge (evidence + commit hashes)

## SDD skip rationale

<!--
Only fill this section if you are adding the "skip-sdd" label to this PR.
Provide a real, specific reason. Examples:
  - "Mechanical rename across 12 files; no logic change."
  - "Pure formatting pass after rustfmt config update."
  - "Documentation-only refactor of 6 README files into pointer-style."

Empty rationale = the spec-gate fails even with the "skip-sdd" label.
-->

## Test plan

- [ ] `~/.local/bin/bats tests/*.bats` passes
- [ ] `~/.local/bin/shellcheck` clean on changed `.sh`
- [ ] Manual verification (describe):
