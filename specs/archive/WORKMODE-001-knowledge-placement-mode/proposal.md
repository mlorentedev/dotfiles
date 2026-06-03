---
id: "WORKMODE-001-knowledge-placement-mode"
type: spec
status: archived
created: "2026-06-02"
tags: [spec, proposal, harness-001, knowledge-placement, ssot, agents-md, cross-agent]
template_version: "1.0"
---

# WORKMODE-001-knowledge-placement-mode

> **Fuses GH [#159](https://github.com/mlorentedev/dotfiles/issues/159) (WORKMODE-001, HARNESS-001 consumer 4) + GH [#197](https://github.com/mlorentedev/dotfiles/issues/197) (AGENTS.md axis reconciliation).** Re-scoped: the original "work-vs-personal knowledge SSOT switch" was framed on a **retired axis**. This spec retires that axis across the harness and aligns it to the canonical **decide-vs-operate** model.

## Why

The knowledge-placement model is already **decided and validated**: `00_meta/patterns/pattern-knowledge-placement.md` (KPM-001, 2026-05-28, migrated 11 repos). Its discriminator is **decide/position → knowledge store (vault); build/operate → repo; collaborate → forge (GitHub)** — *by layer, not by project type*. ADRs, runbooks, project lessons, specs live in the **repo** for **every** placement-model repo, personal or work. The cross-project brain (patterns, memory, strategic backlog) lives in the **vault**.

But `AGENTS.md` — the cross-agent SSOT every agent reads at session start — still carries the **old work-vs-personal axis** in its Standing Orders and runtime sections. The two axes **coexist and mis-route artifacts**:

- **Concrete regression (GH #197):** `kubelab` is a personal project **on** the placement model (its ADRs/lessons were migrated to repo `docs/`). An agent reading Standing Order #3 literally ("personal projects → … `90-lessons.md`") routes a new lesson **back to the vault**, regenerating exactly the debt KPM-001 cleared. The corrected rule exists, but ~200 lines below the primary one.
- **Tooling angle:** `capture_lesson` (Hive) and `/architecture-session` **default to vault output**; the existing guard only fires "for work projects", so a personal+placement repo never triggers it → vault default wins → mis-placement.

GH #159 (WORKMODE-001) was the planned "harness adapts knowledge-SSOT to repo type" consumer — but it was scoped on the **same stale axis**. Implementing it as written would **cement the retired work/personal split into the deploy engine** — the opposite of the fix. Per the repo's own lesson (2026-06-01): *"A held spec can be obsoleted by a later ADR — reconcile+close, don't implement-as-written."*

This spec fuses both: retire the work/personal axis as the *primary* knowledge-routing discriminator, make decide-vs-operate primary everywhere agents read, and reduce work/personal to its one true residual — *where the cross-project brain lives* (personal vault vs a team store) and *whether tasks live on a shared board*.

## What

Observable behavior after this ships:

1. **AGENTS.md reconciled (closes #197).** Standing Orders #2, #3, #7, plus the runtime "Document Dynamic" / "Lessons" sections and the `capture_lesson` note, all state **decide-vs-operate** as the primary discriminator. The retired work/personal axis is demoted to governing only *cross-project-brain location* + *shared-board presence*. No instruction routes a build/operate artifact (ADR, runbook, project lesson, spec, troubleshooting) to the vault.
2. **Tooling guard re-keyed (closes #197).** Every "for work projects" guard becomes "for any repo on the knowledge-placement model" (which is the default for all repos with a `docs/`).
3. **The #159 residual switch, in minimal honest form.** A single explicit, overridable declaration in `AGENTS.md` — `cross-project brain → <personal vault | team store>` and `tasks → <this repo's issues | shared Project>` — defaulting to **personal vault + this repo's issues**. Defaults mean **zero migration** for every current repo. This is the harness "adapt to repo type" mechanism #159 asked for, sized to what the decide-vs-operate model actually leaves variable.
4. **Incident→guard.** A `bats` assertion (Linux) + healthcheck/`compile-harness --check` hook fails if `AGENTS.md` reintroduces "personal → vault" routing for any build/operate artifact class (ADR / runbook / project-lesson / spec / troubleshooting). The kubelab misrouting is the incident; this is its mechanical guard, shipped in the same PR.
5. **Vault pattern reconciled (referenced, separate vault commit).** `pattern-platform-governance.md`'s stale "personal" column defers to `pattern-knowledge-placement` as the placement SSOT; its org-Project / team docs-as-code framework stays. Committed directly to vault `master` per vault discipline — not part of this repo PR, but tracked here.

## Out of scope

- **Implementing a real team-store brain pointer.** No team/work repo exists in the user's set today; the declaration supports it but the team-store path is documented, not built.
- **compile-harness.{sh,ps1} engine changes** beyond emitting/validating the reconciled AGENTS.md text. AGENTS.md is already a manifest target; no new compiler logic is required for the default path.
- **Migrating any existing repo's artifacts.** KPM-001 already did the migrations; this spec fixes the *instructions*, not the data.
- **Windows-empirical validation.** The `bats` guard covers Linux; the PSScriptAnalyzer/Pester parity check is queued for the batched Windows session (no `.ps1` logic added here beyond ASCII-safe parity).

## Risks / open questions

- **🔴 SSOT blast radius.** `AGENTS.md` is read by every agent; a wording regression propagates everywhere. Mitigation: surface the exact SO#2/#3 rewrite for user approval before applying; keep the diff reviewable; the incident→guard test pins the invariant.
- **🟡 Unused machinery.** The team-store/shared-board declaration is used by no current repo. Mitigation: ship it as a *defaulted, documented* declaration (one explicit value), not a parser/compiler — honest forward hook, not speculative code.
- **🟡 "Vault hygiene" framing.** Standing Order #3's name presumes vault-as-default. Reconcile the routing while keeping the *timing* rule (in-session, not "later") intact — that part is still correct and orthogonal.
- **🟡 ID lineage.** Spec keeps the `WORKMODE-001` id for #159 traceability though the content is now placement-axis reconciliation; the slug (`knowledge-placement-mode`) signals the real scope. PR closes #159 + #197.

## Acceptance criteria

- [ ] **AC1** — `AGENTS.md` SO#2, SO#3, SO#7 state decide-vs-operate as the primary discriminator; no build/operate artifact class is routed to the vault by default.
- [ ] **AC2** — The "Document Dynamic" + "Lessons" runtime sections route ADR/runbook/project-lesson/troubleshooting → repo `docs/`; only cross-project (pattern-worthy) lessons + AI memory → vault.
- [ ] **AC3** — Every "for work projects" tooling guard is re-keyed to "for any placement-model repo".
- [ ] **AC4** — `AGENTS.md` carries one explicit, overridable declaration of cross-project-brain location + task-home, defaulting to personal-vault + this-repo issues; absent/default ⇒ no behavior change for current repos.
- [ ] **AC5** — A `bats` test fails if any build/operate artifact class is routed "personal → vault" in `AGENTS.md` (incident→guard). Green on the reconciled file.
- [ ] **AC6** — `scripts/compile-harness.sh --check` still passes (AGENTS.md managed regions + line caps intact post-edit).
- [ ] **AC7** — Vault `pattern-platform-governance.md` personal column defers to `pattern-knowledge-placement` (separate vault `master` commit; linked from this spec).
- [ ] **AC8** — HARNESS-001 epic (#162) consumer-4 checkbox ticked; #159 + #197 closed with PR reference.

## References

- Canonical model: `00_meta/patterns/pattern-knowledge-placement.md` (KPM-001).
- Stale twin: `00_meta/patterns/pattern-platform-governance.md` (personal column → reconcile).
- Tracker: GH [#159](https://github.com/mlorentedev/dotfiles/issues/159) (WORKMODE-001), [#197](https://github.com/mlorentedev/dotfiles/issues/197) (reconciliation), epic [#162](https://github.com/mlorentedev/dotfiles/issues/162).
- Repo: `AGENTS.md` SO#2 (L28), SO#3 (L29), SO#7 (L33), Document Dynamic (L248-252), Lessons (L259), capture_lesson (L318); Neural Hive CORE PRINCIPLE (L228-233, already decide-vs-operate — the consistency target).
- ADR (repo): builds on `docs/adr/adr-013-agent-artifact-deploy-engine.md`, `adr-016-deploy-canonical-agents-md-to-vault.md`.
- Lesson applied: dotfiles `docs/lessons.md` / vault 2026-06-01 — "held spec obsoleted by later ADR → reconcile, don't implement-as-written".
