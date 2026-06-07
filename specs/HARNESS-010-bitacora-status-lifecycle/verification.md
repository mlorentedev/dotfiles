---
tags: [spec, verification]
created: "2026-06-07"
---

# Verification - HARNESS-010-bitacora-status-lifecycle

## Evidence

- [x] AC1 (doctrine in AGENTS.md) → Standing Order #8 + Neural Hive Loop Phase 1 step 6 (claim/self-assign), Phase 2 "Blocked?", Phase 3 Done cross-ref. `features.json` f1 grep passes.
- [x] AC2 (`bitacora-status.yml`) → `.github/workflows/bitacora-status.yml`: `issues:[assigned]`, `if: github.event.issue.state == 'open'`, idempotent `item-add` + `item-edit` to option `6c133cc8`. f2 grep passes.
- [x] AC3 (valid YAML) → `python3 -c "import yaml; yaml.safe_load(...)"` exit 0 **and** `actionlint` exit 0.
- [x] AC4 (runbook) → §4 step 2 deploys both workflows; §5 marks In Progress automated-on-assign (Blocked stays manual); §7 split into 7a/7b with the new template; §8 troubleshooting row added. f4 grep passes.
- [x] AC5 (`/handoff`) → vault `00_meta/skills/handoff/SKILL.md` step 2b "Bitácora status reconciliation" + description updated. f5 grep passes.

## Test status

- Static: `actionlint .github/workflows/bitacora-status.yml` → exit 0. YAML `safe_load` → OK.
- features.json verifications f1–f5 → all exit 0 locally (run from repo root; f5 targets the vault path).
- **Manual smoke (mechanism, pre-deploy):** the Action's exact command pair was exercised live against the real board —
  - `gh project item-edit ... --single-select-option-id 6c133cc8` moved #270 → In Progress (confirmed on the board).
  - `gh project item-add` idempotency confirmed: re-adding #270 returned the existing item id `PVTI_lAHOAM7xJs4BZ6GYzgu7wCI`, no duplicate, status preserved.
- **Live end-to-end** (`assign → In Progress` via the deployed workflow) is verifiable only after this lands on `main` (workflows run from the default branch). Post-merge check: assign a throwaway/open issue and confirm the run is green and the field flips. Recorded as a closing step, not claimed done here.
- No regressions: dotfiles CI (`lint`, `test`, `integration`, `spec-gate`) expected green — spec folder present satisfies the Discipline Gate regardless of LOC.

## Decisions made during implementation

- **Trigger = `issues:[assigned]`, not PR-linked.** The observed gap (work starts while Backlog) happens *before* any PR exists, so a PR-open trigger is a late signal; self-assign at pickup is the true start. No GitHub Projects built-in covers "assigned → status", hence a custom Action.
- **`Blocked` left manual.** No reliable machine signal for "stuck"; naming the blocker is human judgement. Automating it would add label-management surface for little gain.
- **Rollout deferred to OPS-002 (#258).** This PR ships + validates the workflow on `dotfiles` and registers it in the runbook's per-repo bundle (§4/§7); copying it to all repos + secret is OPS-002's job — keeps HARNESS-010 acotado and avoids a half-deployed automation.
- **No helper script.** A `.sh`/`.ps1` wrapper would incur Windows-parity + test surface; the `gh project item-edit` one-liner in runbook §5 is enough for the only remaining manual transition (Blocked).

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **yes (separate, not this PR)** — "re-running a failed GitHub Actions run replays the *original commit's* workflow file, so a workflow-file fix must be verified by firing a *fresh* event, not a re-run." Surfaced while fixing the `add-to-project@v1` failures this session. File in dotfiles `docs/lessons.md` next session.
- [ ] ADR-worthy? **no** — doctrine fits AGENTS.md Standing Order #8; no architectural fork.
- [ ] New `00_meta/patterns/` candidate? **not yet** — revisit if a second project needs the same assign→status automation (OPS-002 may surface it).

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-010-bitacora-status-lifecycle/` -> `specs/archive/HARNESS-010-bitacora-status-lifecycle/`
- [ ] Issue #270 closed (built-in workflow → Done); PR linked
- [ ] Promotions above executed (the `docs/lessons.md` lesson)
