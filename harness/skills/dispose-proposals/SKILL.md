---
name: dispose-proposals
description: Triage the CURATOR recurrence-proposal inbox — the weekly human gate that turns emitted proposal drafts into bitácora issues. Triggers on "/curator dispose", "dispose the proposal inbox", "triage curator proposals", "process recurrence proposals", "review the proposals queue", and as a step of the weekly curator session. Reads 80_agents/hermes-nan/proposals/*.md (status pending), applies the curation SOP (promote / keep / merge / wontfix), and on accept files a proposal-labelled issue and marks the draft disposed.
---

# /curator dispose — recurrence-proposal disposition (S3 human gate)

The **disposer** arm of the curation loop (`00_meta/agents/runbooks/curation.md`). The producer
(the hermes judge + `emit-proposals.py`) writes recurrence proposal *drafts* into the vault inbox;
this skill is the **weekly human gate** that dispositions them in one ordered batch — so the gate
scales with review cadence, not with the agent's activity.

Read-only producer → human disposes. You file the issue; **you never auto-accept**. The judge's
RECURRENCE verdict is a *candidate*, not a decision.

## Inputs

- Inbox: `80_agents/hermes-nan/proposals/*.md` (skip `_TEMPLATE.md`). Each file's schema is in
  `proposals/_TEMPLATE.md`; act only on `status: pending`.
- SOP: the triage table in `00_meta/agents/runbooks/curation.md` (promote / keep / merge / wontfix).

## Protocol

### Step 1 — Gather the queue
List pending drafts and read each one's frontmatter + body:
```bash
ls 80_agents/hermes-nan/proposals/*.md 2>/dev/null | grep -v _TEMPLATE
```
If none have `status: pending`, report "proposal inbox empty" and stop.

### Step 2 — Present each draft for the gate
For every pending draft, show the human, compactly: `candidate_id`, `root_cause`, `occurrences`
+ `dates`, the evidence excerpts, and the judge's suggested `placement` / `tier_target`. Sanity-
check it yourself first against the anti-bloat gate (`pattern-memory-consolidation`,
`pattern-lesson-promotion`): is this a real recurrence not already canonical in `00_meta/`?

### Step 3 — The human decides (curation SOP)
Per the runbook table:

| Decision | Meaning | Action |
|----------|---------|--------|
| **promote** (accept) | real recurrence, worth shared-brain knowledge | Step 4a |
| **merge** | duplicates / extends an existing artifact | fold into the canonical one; Step 4b with `merge` |
| **keep** | valid but agent/project-local, not cross-cutting | Step 4b with `keep` |
| **wontfix** | false cluster / superseded / bad idea | record the reason; Step 4b with `wontfix` |

### Step 4a — On accept: file the proposal issue
Create a `proposal`-labelled bitácora issue from the draft (title = the root cause, one line;
body = evidence links + suggested placement/tier + the `candidate_id` for traceability):
```bash
gh issue create --repo mlorentedev/knowledge --label proposal \
  --title "<root cause, one line>" \
  --body "$(printf '%s\n' \
    '**Recurrence proposal** (curator, candidate `<candidate_id>`).' '' \
    '**Root cause:** <root_cause>' '' \
    '**Evidence (>=2 distinct incidents):**' '<- source (date) per evidence line>' '' \
    '**Suggested placement:** <placement> -> Tier <tier_target> (per pattern-memory-consolidation).' '' \
    'Human gate: accept on the board -> promote to Tier 4 + compile-harness (S5, #139).')"
```
Then add it to the Bitácora board (Backlog) — resolve the project item id via GraphQL as in the
board runbook — and write back into the draft: `status: disposed`, `issue: <N>`, `disposed_as: promote`.

### Step 4b — On merge / keep / wontfix
Update the draft frontmatter: `status: disposed` (or `rejected` for wontfix), `disposed_as: <decision>`.
For **wontfix**, record the one-line reason in the rejected-mechanisms / WONTFIX ledger (HARNESS-005)
so the same false cluster is not re-litigated next week.

### Step 5 — Close out
The drafts live in the vault → obsidian-git auto-commits the disposition edits (do not manual-commit).
Report: N disposed (promoted/merged/kept/wontfix), with the issue numbers for accepts.

## Cadence & boundaries
- **Weekly**, batched with `crystallize` + `insights` (one ordered review, not per-event interrupts).
- Keep the `proposal` queue near zero so the vault never drifts far from clean.
- You disposition recurrence candidates; promotion across the write-boundary into `00_meta/` (S5)
  is a separate, deliberate step after the issue is accepted on the board.

## References
- `00_meta/agents/runbooks/curation.md` (the loop + SOP), `proposals/_TEMPLATE.md` (draft schema)
- `pattern-memory-consolidation`, `pattern-lesson-promotion` (anti-bloat promotion gates)
- Epic knowledge#134 (CURATOR-001); this skill is S3 (#137). Producer: `emit-proposals.py`.
