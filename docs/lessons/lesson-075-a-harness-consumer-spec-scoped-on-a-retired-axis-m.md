---
id: lesson-075-a-harness-consumer-spec-scoped-on-a-retired-axis-m
type: lesson
status: active
created: "2026-06-02"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 075: A harness consumer spec scoped on a retired axis must be reconciled, not implemented (WORKMODE-001)

**Context:** Picking up the HARNESS-001 backlog, the next planned consumer was WORKMODE-001 (#159) — "the harness adapts its knowledge-SSOT target to repo type (personal → vault, work → repo + Project)". Before implementing, a vault+repo sweep checked whether the work/personal model had already been decided.

**Problem:** It had. `pattern-knowledge-placement` (KPM-001, 2026-05-28) replaced the **work/personal** axis with a **decide-vs-operate-by-layer** axis: build/operate docs → repo `docs/` for *every* placement-model repo; cross-project brain → vault; collaborate → forge. WORKMODE-001's premise ("personal → vault") was the retired axis. Implementing it as written would have hard-coded the obsolete split into the deploy engine — the opposite of the fix. Worse, `AGENTS.md` (the cross-agent SSOT) still carried the old axis in Standing Orders #2/#3/#7 + the Document Dynamic / Lessons sections, contradicting its own Neural Hive section and mis-routing artifacts (the kubelab regression: a personal+placement repo's lesson sent back to the vault). The open reconciliation ticket #197 already named this.

**Solution:** Fuse #197 + #159 into one spec that retires the axis instead of cementing it: make decide-vs-operate primary across all of AGENTS.md, demote work/personal to its one residual (where the cross-project *brain* lives + whether tasks sit on a shared board) as a defaulted `## Knowledge Placement` declaration, re-key tooling guards from "for work projects" to "for any placement-model repo", and ship an incident→guard (`tests/agents-md.bats`) that fails if any build/operate class is routed to the vault again. Verify a held spec's *premise* against later patterns/ADRs before writing code — a spec is a hypothesis with a shelf life, not a work order.

**Tags:** `#sdd` `#knowledge-placement` `#decide-vs-operate` `#agents-md` `#ssot` `#incident-to-guard` `#reconcile-dont-implement`
