---
id: "ADR-042-shared-ci-is-referenced-not-copied"
type: adr
status: accepted
owner: manu
date: "2026-09-30"
supersedes: []
extends: [adr-012-deploy-strategy-copy-with-drift-assertion, adr-022-dotf-init-flagship, adr-031-board-automation-auth]
issue: mlorentedev/dotfiles#1843
tags: [architecture, decision, ci, github-actions, fleet, reusable-workflows]
created: "2026-09-30"
---

# ADR-042: Shared CI is referenced from one repository, not copied into each

## Context

The CI shared across this owner's repositories travels only as copies. Measured for #1843:

- **pr-agent.** Each of the 12 repositories with a `pr-agent.yml` has a different copy, and none matches this repository's. The file changed here 21 times since July, and none of those changes reached another repository, including the TOOL-023 push gate (ADR-040). kubelab's copy has moved ahead of the canonical one, so there are now two designs for one workflow (#1570).
- **Board workflows.** `add-to-project.yml` exists in 14 versions across 22 repositories. 23 of 25 repositories carry an outdated board workflow, and one missed a fix for about three months (#1787).
- **Guards.** The hygiene guards were pushed to 13 repositories as byte-identical copies by a transformer that lives in a session scratchpad. Nothing says which copy is canonical, and nothing detects drift (#1573).
- **spec-gate.** `spec-gate.yml` hard-codes `branches: [main]`, so a copy in a `master` repository never runs (#1627). `dotf init` defers the portable spec-gate for the same reason (ADR-022, `cli/internal/initrepo/ci.go`).
- **No reuse by reference.** No workflow in this repository uses `workflow_call`, and there is no composite action.

Copy-and-detect was proposed before (#1573, #1787). Detection finds drift but does not deliver the fix: every change still needs one pull request per repository, written by hand. Reuse by reference was proposed for single workflows (#786, #347) but never decided as the rule.

Required status checks are declared in `forge/branch-protection.json` (GUARD-017) and applied with `dotf forge protection apply`. For this repository, `main` requires `lint`, `lint-powershell`, `test`, `test-windows`, `review-attestation`, `cli-gate` and `spec-gate`.

## Decision

1. **Every shared CI file belongs to one of four classes, and the class decides how it travels:**

   | Class | Examples | Travels as |
   |---|---|---|
   | Workflow logic | the pr-agent review, spec-gate, the board sync, knowledge-gate, repo hygiene | a reusable workflow (`on: workflow_call`) in the shared repository, called by a short caller workflow in each repository |
   | Step logic reused inside workflows | the pr-agent push gate (ADR-040), action pin checks | a composite action in the shared repository |
   | Checks that also run locally | `check-lessons.sh`, `check-actions-pinned.sh` | `dotf lint` subcommands: one binary everywhere, no per-repository copy (#1584's principle) |
   | Files a platform requires in each repository | `dependabot.yml`, `.coderabbit.yaml`, `CODEOWNERS` | copies, declared in the fleet canon (#1573) and checked for drift (ADR-012's copy-with-drift-assertion, limited to this class) |

2. **The shared repository is `mlorentedev/ci-workflows`**, public, with its own history and releases (owner decision Q3 on #1843). This repository consumes it like any other and keeps no private copy.
3. **Consumers pin by commit SHA,** with the release tag as a trailing comment. That is the rule this fleet already applies to third-party actions. A floating `@main` would let one bad push break every repository at once. A new version reaches a consumer as a one-line pin-bump pull request from Dependabot's `github-actions` ecosystem. Those pull requests are merged by hand (#1574 decides any exception). The rule is enforced, not only stated: `check-actions-pinned.sh` already matches a job-level `uses:` key, so it rejects a caller that references a reusable workflow by tag or branch. It runs in every consumer as `dotf lint` (class 3), and `dotf fleet apply` writes SHA pins only.
4. **Secrets are passed explicitly, never with `secrets: inherit`.** The called workflow declares each secret it needs, and the caller passes each one by name. A personal account has no organisation secrets, so each consumer holds its own copies, provisioned by `dotf secrets sync ci`. `BITACORA_PAT` keeps the rules of ADR-031.
5. **Callers own triggers, branches, permissions and concurrency.** A reusable workflow hard-codes no branch name, so a `master` repository works unchanged (#1627). The caller sets `permissions:`, which caps what the called workflow can do, and the `concurrency:` group (#786).
6. **A required check changes name when it moves, so migration is two pull requests.** A caller job `review` that calls a workflow whose job is `review` reports the context `review / review`. The first pull request adds the caller alongside the old workflow, so both contexts report. After it merges, the declaration in `forge/branch-protection.json` switches to the new context and `dotf forge protection apply` runs. The second pull request removes the old workflow, and it is opened only after `dotf forge protection apply --dry-run` reports 0 changes and the new context is required on the default branch. In that order no commit waits on a required context that nothing reports, and no commit merges without the new check being required.
7. **Rollout is pilot first.** pr-agent migrates first, in one small repository, after the kubelab design from #1570 is ported into the canonical workflow, so there is one design to share. The board workflows and spec-gate follow. Caller pull requests are opened by `dotf fleet apply` (row D5 of #1843), which always opens a branch and a pull request, never merges, and reports 0 changes on a second run.

## How this reconciles the earlier proposals

- **#786** (pr-agent as a reusable workflow): adopted as decision 1. Its decisions on secret sync, the concurrency group and a pilot repository carry over.
- **#347** (a reusable Claude review): uses the same mechanism if it is ever built, and stays gated on its own evaluation.
- **#1573** (fleet SSOT): narrowed. `harness/fleet/` holds only the fourth class and its drift check. The guard scripts move to `dotf lint`.
- **#1627** (spec-gate everywhere): resolved by decisions 5 and 6. The deferral in ADR-022 ends when spec-gate ships as a reusable workflow.
- **#1570** (kubelab's reviewer design): a precondition of the pr-agent pilot.
- **#1584** (vendoring policy): still governs third-party code. This ADR covers only this owner's own CI.
- **#1787** (board workflow drift): becomes the drift check of row D2, which also reports consumers still carrying a copy of a first-class file.

## Consequences

- One fix to shared CI is one pull request in `ci-workflows`, plus one pin bump per consumer that Dependabot opens and a human merges.
- A reusable workflow runs with the caller's `GITHUB_TOKEN`, capped by the caller's `permissions:`. Runner minutes are billed to the caller, as they are today.
- An old pin is a legitimate state, not drift. The drift check reports how far behind each consumer is, and fails only on a copy of logic that should be referenced.
- The new repository, the per-repository secrets and the protection updates are owner actions (#1843, "Owner actions").

## Alternatives rejected

- **Keep copies and detect drift (#1573 as filed).** It finds drift and leaves the fix manual in every repository.
- **Host the reusable workflows in this repository.** It ties every repository's CI to the history and release cadence of a personal dotfiles repository (owner decision Q3).
- **Template repositories.** A template helps only when a repository is created, and nothing flows afterwards.
- **Git submodules.** GitHub does not resolve submodules when it loads workflows, so a submodule can carry scripts but not reusable workflows, and it adds a checkout step to every job.
- **`secrets: inherit`.** It hands every secret of the caller to the called workflow, so no reader can tell which secrets a workflow uses.

## References

- EPIC #1843 (track D, rows D1 to D7); ADR-012, ADR-022, ADR-031, ADR-040; GUARD-017 (`forge/branch-protection.json`)
- #786, #347, #1573, #1570, #1584, #1627, #1787, #1574
