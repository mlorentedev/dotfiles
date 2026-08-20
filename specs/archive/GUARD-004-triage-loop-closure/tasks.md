---
tags: [spec, tasks, templates]
created: "2026-08-20"
---

# Tasks — GUARD-004-triage-loop-closure

- [x] Issue #1099 filed with the measurements (marker absent in three places;
      `triage-queue` wired to nothing)
- [x] Vault SSOT: `00_meta/skills/pr-review-triage/SKILL.md` gains step 7,
      recording the table under the registry's marker
- [x] Step 3's early exit routed into step 7 so the empty case is recorded rather
      than ending the skill (the two would otherwise contradict each other)
- [x] Record regenerated with `scripts/compile-harness.sh --refresh`
- [x] `prtriage.Fetch` / `prtriage.Marker` extracted so the command and the brief
      share one query and one domain conversion
- [x] `dotf pr triage-queue` re-pointed at the extraction; output confirmed
      unchanged
- [x] `triageQueue` section added to the mem package with its three states
- [x] Wired into `BriefOptions` (agnostic) and `ClaudeContextInput` (Claude)
- [x] `memTriageQueue` resolves the registry, returns nil when absent, bounds the
      call at 5s
- [x] Tests written for all three states, including the failure-is-not-empty rule
- [x] Full verification run and recorded in `verification.md`
- [ ] Independent adversarial review before archive (must not be the implementer)
