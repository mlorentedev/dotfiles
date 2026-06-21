---
id: genre-picker-skill
type: skill
status: active
created: "2026-06-20"
owner: manu
name: genre-picker
description: Classify a piece of knowledge into its 00_meta meta-type (skill / pattern / runbook / template / spec) BEFORE creating it, so it lands in the right folder and never drifts. Triggers on /genre-picker, "is this a pattern or a runbook?", "should this be a skill or a pattern?", "where does this go?", "classify this knowledge", "what meta-type is this", "qué es esto, patrón o runbook", and BEFORE authoring any new 00_meta/ artifact. Applies the boundary tests from pattern-knowledge-placement at write-time.
allowed-tools: [Read, Grep, Glob]
---

# Genre Picker — classify before you create

> The taxonomy SSOT is [[pattern-knowledge-placement]] § "Within the store: the meta-type taxonomy". This skill is the **write-time application** of it: run it BEFORE creating any `00_meta/` artifact so it lands in the right type/folder from the start — preventing exactly the drift the sweeps keep cleaning up. It is itself a skill (a capability invoked on a trigger) that *defers* the boundary definitions to a pattern — the taxonomy practising what it preaches.

## When to use

- Before creating any new `00_meta/` artifact (skill / pattern / runbook / template).
- When unsure: "is this a pattern or a runbook?", "skill or pattern?", "where does this go?".
- When an existing artifact feels mis-filed — re-classify and move it (never duplicate).

## When to skip

- The knowledge is **project-bound** (an ADR / lesson / runbook / troubleshooting note for ONE repo) → it does NOT belong in `00_meta/` at all; it goes to that repo's `docs/` (the directionality invariant). Use the [[pattern-knowledge-placement]] placement matrix, not this skill.
- It is a **tactical task or discussion** → forge (GitHub issues), not the store.

## Step 0 — is it even a store artifact?

First apply the discriminator from [[pattern-knowledge-placement]]:

> **Decide/position → store · Build/operate → repo · Collaborate → forge.**

Only **cross-project** knowledge ("what an agent does / a principle / a procedure / a skeleton", spanning repos) belongs in `00_meta/`. A procedure for ONE repo is a repo `docs/runbooks/` file, not a vault runbook. If Step 0 sends it to a repo or the forge, stop — this skill is done.

## Step 1 — the boundary tests (apply IN ORDER, first match wins)

1. **Executed by an agent when a trigger phrase fires?** → **Skill** → `00_meta/skills/<name>/SKILL.md` (author per [[creating-skills]]).
2. **An ordered sequence of steps run start→finish, AND cross-project?** → **Runbook** → `00_meta/runbooks/<name>.md` (skeleton: [[runbook]]). *A one-repo procedure → repo `docs/runbooks/`; stop here.*
3. **A principle applied with judgment (no fixed step order; mostly the *what & why*)?** → **Pattern** → `00_meta/patterns/pattern-<topic>.md`.
4. **A blank skeleton to instantiate a new file of some type?** → **Template** → `00_meta/templates/<name>.md` (register it in [[templates/_index]]).
5. **Scoped to ONE change with acceptance criteria?** → **Spec** → repo `specs/<id>/` (skeletons: [[spec-proposal]] / [[spec-tasks]] / [[spec-verification]]).

## The line that blurs — Pattern ⇄ Runbook

- **Pattern** = *"prefer X over Y because Z"* — you decide where it applies.
- **Runbook** = *"do step 1, then 2, then 3"* — you execute in order.
- A "pattern" that is really a checklist run top-to-bottom → it is a **runbook**. A "runbook" that spends most of its text on *why* rather than *what next* → it is a **pattern**. **Reclassify, never duplicate.**

## Output (state explicitly, before creating the file)

- **Genre:** skill | pattern | runbook | template | spec
- **Destination:** exact folder + filename
- **Authoring guide / skeleton:** which template or skill to follow
- **Justification:** the one boundary test that matched (and why not the adjacent one)

## Worked examples

| Knowledge | Test matched | Genre → home |
|---|---|---|
| "What an agent does on `/handoff`" | 1 (trigger) | Skill → `00_meta/skills/handoff/` |
| "Always write strategic decisions to the vault the same session" | 3 (principle) | Pattern → `pattern-decision-persistence.md` |
| "Onboard ANY repo to the placement model, step by step" | 2 (cross-project procedure) | Runbook → `00_meta/runbooks/` |
| "Deploy THIS service to staging, step by step" | 0 (one repo) | Repo `docs/runbooks/` — NOT the vault |
| "Skeleton for a new ADR file" | 4 (skeleton) | Template → `00_meta/templates/adr.md` |
| "Proposal/tasks/verification for feature FOO-012" | 5 (one change) | Spec → repo `specs/FOO-012/` |

## References

- [[pattern-knowledge-placement]] — taxonomy SSOT (the boundary tests live here) + the store/repo/forge placement matrix
- [[creating-skills]] — how to author a skill once classified as one
- [[pattern-project-structure]] — store-side project layout
