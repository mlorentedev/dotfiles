---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - HARNESS-027

## Evidence

- [x] AC1/AC2 (triggers, waits, reports CI) -> `harness/skills/pr-review-triage/SKILL.md` steps 1–2; record renders under `compile-harness.sh --check` (`features.json` `HARNESS-027-f1`)
- [x] AC3 (disposition per comment) -> steps 5–6, the apply/defer/skip table with a rationale column
- [x] AC4 (no action, no merge without confirmation) -> `HARNESS-027-f2`, green
- [x] AC5 (reviewed-nothing vs never-ran) -> `HARNESS-027-f3`, green
- [x] AC6 (one-line exit when there is nothing to do) -> `HARNESS-027-f4`, green
- [x] AC7 (no vendor named) -> `HARNESS-027-f5`, green — asserts the absence of every review-service name, so the check fails if one is reintroduced later
- [x] Universal deploy (no `targets[]`) -> `HARNESS-027-f6`, green

## Test status

- `bash scripts/compile-harness.sh --check` -> no drift; the new record validates and renders for every agent
- Every `features.json` verification command run individually -> all green
- Manual smoke test: the procedure was executed by hand against this repository's own PRs while writing it — that is where the 9-second status-notice measurement came from.
- No regressions: the change adds one record and touches no engine code.

## Decisions made during implementation

- **Vendor names removed after the first draft.** The initial version named today's review service and described its surfaces. The user's correction was right and matches the fence lesson from earlier in the same chain: identity is not a property you should build behaviour on. Reviewers change, and the user's own automations are expected to post comments — so the triage keys on **surface and content**, and the author earns only a reply address.
- **Status notice versus review.** Measured, not assumed: a reviewer posted 9 seconds after PR creation with a rate-limit warning. It reads exactly like a pass, so the skill separates "reviewed, nothing to say" from "never ran" and forbids claiming the first when the second happened.
- **An explicit early exit.** The common case is green checks and no substantive comments; a procedure that demands a table for that trains people to skip the procedure. One line, and stop.
- **Replies are for threads someone is waiting on.** Requiring a posted reply for every skipped remark manufactures noise on the PR; the reason belongs in the report either way.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? no — the transferable half (a status notice that reads like a pass) is encoded in the skill where it is actually needed.
- [ ] ADR-worthy decision? no.
- [ ] New pattern candidate for `00_meta/patterns/`? no — it implements `pattern-track-or-fix` rather than adding doctrine.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-027/` -> `specs/archive/HARNESS-027/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
