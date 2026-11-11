---
id: "CLI-136-ci-secrets-from-workflows"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-11-10"
issue: "mlorentedev/dotfiles#3290"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# CLI-136-ci-secrets-from-workflows

> **Naming**: file lives at `<repo>/specs/CLI-136-ci-secrets-from-workflows/proposal.md`. `CLI-136-ci-secrets-from-workflows` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

A repository receives the CI secrets the registry lists it under as a consumer (`ci:<owner>/<repo>`), and that list is edited by hand. Nothing ties it to what the repository's workflows actually read.

- **Wrong both ways.** A repository that adopts the shared pr-agent workflow (CI-007), which reads `PR_AGENT_ANTHROPIC_API_KEY`, fails no check if nobody declares the key. It just reviews on NaN alone and loses reviews under load (#3188: 3 of 8). A secret declared for a repository that no workflow reads sits there unnoticed.
- **Cost.** Declaring is a step someone has to remember on every adoption PR, and today forgetting it is silent.

## What

`dotf secrets ci-check [--repo <owner>/<repo>]...` reads each repository's workflows and compares the secrets they read with the ones the registry declares for it. It prints, per repository, the **missing**, **excess** and **unknown** names, never values. It exits non-zero when any repository has a missing or excess name.

- **Read set.** Every secret reference in the repository's own `.github/workflows/*.yml|yaml`, in any of the forms `secrets.X`, `secrets['X']` and `secrets["X"]`, excluding `GITHUB_TOKEN`. This already covers the explicit mappings a caller passes to a reusable workflow (`with`/`secrets: { NAME: ${{ secrets.MINE }} }`), where the repository needs `MINE`, not the callee's name.
- **Inherited secrets.** For a job that calls a reusable workflow with `secrets: inherit`, the callee's `on.workflow_call.secrets` keys are added, read at the ref the caller pins. A callee secret marked `required: false` is still read, so absent is still **missing**, marked `(optional in <callee>)`. pr-agent declares `PR_AGENT_ANTHROPIC_API_KEY` optional so that a review degrades to NaN instead of failing, and that silent degradation is the case this issue exists to report. ci-workflows#65 keeps that declaration exact (every secret read is declared, and every declared secret is read), so the declaration stands for what the callee reads.
- **Unknown.** A whole-context read (`toJSON(secrets)`, or `secrets[<expression>]` whose key is not a literal) cannot be listed. The repository is reported **unknown** with the file and the expression, not passed.
- **Declared set.** `Registry.SelectCI(<repo>)`'s upload list: the same selection `dotf secrets sync ci` uploads, so the check and the sync cannot disagree.
- **Fleet.** With no `--repo`, every non-archived, non-fork repository the operator owns (`gh repo list <owner> --no-archived --source`). The fleet has to include repositories the registry does not list at all, because a repository that reads a secret and is declared for none is the "missing" case this issue opened on.
- **Seam.** Workflow files arrive through an injected reader (production: `gh api repos/<r>/contents/<path>?ref=<ref>`, REST), so tests run on fixtures with no `gh` and no network.

## Out of scope

- **Whether a declared secret was actually synced** to the repository (`actions/secrets`): that is #1596 (CLI-078), and it needs admin scope this check does not.
- **Running the check on adoption PRs.** That is `.github/**` wiring in each repository and stays with the owner as a follow-up.
- **Repairing the registry.** The check reports and never edits `registry.yaml`.
- **Organisation or environment secrets.** Every repository here is a personal repository with repository secrets only.

## Risks / open questions

- **YAML 1.1 reads a bare `on:` key as the boolean `true`.** The repository's YAML library is go.yaml.in/yaml/v3. A test pins that `on.workflow_call.secrets` is reachable from a fixture, so a library change cannot hide it.
- **API budget.** About two REST calls per repository, plus one per distinct reusable workflow and ref, which the check caches. This is far under the core budget, and GraphQL is not used (#3228).
- **A secret read only behind an `if:` that never fires** is still a read. The check is static by design: a secret that a workflow can read has to exist.
- **Vendored pr-agent.yml.** Fleet repositories that still vendor their own `pr-agent.yml` (reading `NAN_API_KEY` directly) are covered by the direct-reference half alone, which is correct for them.

## Acceptance criteria

- [ ] AC1: Run over the fleet, `dotf secrets ci-check` lists, per repository, the missing, excess and unknown CI secret names, with no values, and exits non-zero when any repository has a missing or excess name.
- [ ] AC2: A repository calling the shared pr-agent workflow with `secrets: inherit`, without `PR_AGENT_ANTHROPIC_API_KEY` declared, is reported missing that name. A caller that maps the secret under another name is reported missing the name it maps from.
- [ ] AC3: A repository declared for a secret that no workflow reads is reported excess.
- [ ] AC4: A whole-context read is reported as unknown with its file and expression. It is never treated as a pass.
- [ ] AC5: An absent callee secret declared `required: false` is reported missing and marked optional, naming the callee; it is not dropped.

## References

- Issue: mlorentedev/dotfiles#3290. Related: #1596 (CLI-078, synced vs declared), mlorentedev/ci-workflows#65 (the callee contract), #3228 (REST over GraphQL)
- ADR-029 (CI secrets are routed per repository through the registry), ADR-028 (secrets never in output)
