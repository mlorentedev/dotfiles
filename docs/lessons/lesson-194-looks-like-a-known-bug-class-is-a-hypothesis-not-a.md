---
id: lesson-194-looks-like-a-known-bug-class-is-a-hypothesis-not-a
type: lesson
status: active
created: "2026-08-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 194: "Looks like a known bug class" is a hypothesis, not a finding — reproduce before you fix

**Context**: HARNESS-070 (deploy convergence, #843/#869/#828). The session's brief carried live evidence: `dotf doctor` flagging 4 deployed skills (`computer-use`, `find-skills`, `orca-cli`, `orchestration`) as symlinks, labeled as a "BUG-100" regression — the historical, closed issue #100 about this repo's own deploy strategy fighting `agy`'s filesystem layout.

**Problem**: the label was plausible (the doctor message literally says "BUG-100") but wrong. `readlink -f` on each of the four showed they all resolve to `~/.agents/skills/<name>` — Orca's own skill-symlink mechanism, not anything this repo's `harness/skills/` manages. None of the four names has a `harness/skills/<name>` record at all. There is even documented prior art for exactly this shape: `specs/archive/AI-022-pi-harness-slot/proposal.md` records that `~/.pi/agent/skills` was deliberately excluded from the old shell healthcheck's strict symlink sweep for the identical reason (pi's installer symlinks sibling skills from `~/.agents/skills`, agent-owned state) — the Go port of that check (`checkDeployedSkillSymlinks`) never carried the exclusion forward, so it silently regressed to flagging every symlink regardless of ownership. Fixing the "regression" as reported (e.g. by trying to make `--deploy` de-symlink these paths) would have fought Orca's own filesystem layout — the exact class of bug #100 was about — instead of fixing the actual defect, which was a check that didn't know the difference between "we manage this" and "something else legitimately symlinks here."

**Solution**: two minutes of `readlink -f` + `grep`-ing the repo for the four names turned "regression of #100" into "false positive in the doctor check," which is a different, smaller, and correctly-scoped fix: narrow `checkDeployedSkillSymlinks` to only flag a symlink whose name has a `harness/skills/` record, mirroring the policy the deploy side (`warn_unmanaged_output`) already implements. Filed as a fresh ticket (#943/BUG-074) with the reproduction evidence, fixed in the same PR, and the mischaracterization was called out explicitly to the user rather than silently "fixed" under the original framing.

**Rule**: when a task handoff characterizes evidence as an instance of a known bug/class, treat the label as a hypothesis to verify, not a fact to build on — especially when the fix implied by the label would mean fighting another tool's filesystem layout, re-touching code that already has a documented exclusion for this exact shape elsewhere in the repo, or when the "evidence" is a single doctor/log line rather than something traced to its source. `readlink -f` / `git grep` for the actual names costs almost nothing; building a fix on a wrong label costs a wasted PR and, if merged, a reintroduced bug.

**Tags**: `debugging`, `harness`, `symlinks`, `verification`
