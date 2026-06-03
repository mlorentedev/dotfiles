---
id: "WORKMODE-001-knowledge-placement-mode"
type: tasks
status: draft
created: "2026-06-02"
template_version: "1.0"
---

# WORKMODE-001 — Tasks

> Single fused PR (closes #159 + #197). Larger-than-usual by explicit user decision ("un spec único que cierra el tema"). Kept coherent: all changes serve the one invariant — *decide-vs-operate is the primary knowledge-routing axis everywhere agents read.*

## T1 — Reconcile AGENTS.md Standing Orders (AC1, AC3)
- [ ] SO#2 (SSOT): lead with decide-vs-operate; demote work/personal to "where the cross-project brain lives".
- [ ] SO#3 (hygiene): route bug→repo `docs/troubleshooting/`, ADR→repo `docs/adr/`, trick/lesson→repo `docs/lessons.md`; keep the *in-session timing* rule; cross-project (pattern-worthy) lessons + AI memory → vault.
- [ ] SO#7 (noted=recorded): re-point "Placement follows #2+#3" to the reconciled wording.
- [ ] Re-key tooling guard: "for work projects" → "for any repo on the knowledge-placement model".

## T2 — Reconcile AGENTS.md runtime sections (AC2)
- [ ] "Document Dynamic" (L248-252): ADR → repo `docs/adr/`; trick → repo `docs/lessons.md`; cross-project synthesis → vault.
- [ ] "Lessons" (L259) + `capture_lesson` note (L318): project lessons → repo; only cross-project/methodology → vault `90-lessons`.
- [ ] "vault hygiene" labels (L389/L393): keep the timing discipline; neutralize the vault-as-default presumption.

## T3 — Add the #159 residual declaration (AC4)
- [ ] One explicit `## Knowledge Placement` declaration block in AGENTS.md: `brain:` (default `vault:00_meta/`), `tasks:` (default `repo issues`). Documented as overridable for a future team repo; absent ⇒ defaults ⇒ no current-repo change.

## T4 — Incident→guard (AC5)
- [ ] `tests/agents-placement.bats`: assert AGENTS.md does NOT route any build/operate class (ADR|runbook|project-lesson|spec|troubleshooting) to the vault as the personal default. Red on the pre-change file (proves it bites), green on the reconciled file.
- [ ] Wire into the existing bats suite + (if cheap) a `compile-harness --check` / healthcheck line.

## T5 — Engine integrity (AC6)
- [ ] `scripts/compile-harness.sh --check` passes post-edit (managed regions + `AGENTS.md` not over line caps; markers intact).
- [ ] `shellcheck` clean on any touched `.sh`; `bash -n` + `zsh -n` on touched scripts.

## T6 — Vault pattern reconcile (AC7, separate vault master commit)
- [ ] Edit `pattern-platform-governance.md`: personal column defers to `pattern-knowledge-placement`; keep org-Project/team docs-as-code. Commit direct to vault `master` (isolated detached worktree if a parallel session holds staged work).

## T7 — Close-out (AC8)
- [ ] Tick HARNESS-001 epic (#162) consumer-4 box.
- [ ] PR body: `Closes #159`, `Closes #197`. No AI-attribution markers. English only. No phase refs.
- [ ] Update spec status draft → verifying → archived on merge.
