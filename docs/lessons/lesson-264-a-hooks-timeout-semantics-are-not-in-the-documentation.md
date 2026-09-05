---
id: lesson-264
type: lesson
status: active
created: "2026-09-03"
owner: manu
tags: [lesson, harness, hooks, verification, claude-code]
---

# When the documentation does not name a field, read the executable — twice it decided a design

## What happened

Two questions came up on HARNESS-109 (#1434) that the vendor documentation could not answer, and
both were answered in twenty minutes by extracting strings from
`~/.local/share/claude/versions/2.1.260` — the Claude Code binary already installed on the machine.

**Question 1: does the hook payload carry the true subagent type anywhere?** The whole ticket turned
on it. `agent_type` carries the caller-supplied NAME when a dispatch is named, so the gate's persona
lookup misses and enforcement silently turns off; #1434's first proposed fix was to "fall back to a
second payload field if one carries the true type", explicitly flagged as *requiring measurement
because nothing here has captured a full subagent payload verbatim*. The payload builder settles it:

```js
function pa(e,n,r,o){ let d = o?.agentType ?? Sy(); /* … */
  return { session_id:e.id, transcript_path:Hf(e.id), cwd:n, scratchpad_dir:…,
           prompt_id:…, permission_mode:r, agent_id:o?.agentId, agent_type:d, effort:E } }
```

That is the complete base field set. There is no second field, so the proposed direction could only
ever have produced a fallback that never fires — working code, passing tests, and enforcement still
off. The fix had to come from somewhere else entirely (the parent's own dispatch call), and *that*
design was only reachable once the first one was known to be impossible.

**Question 2: is bounding a hook's timeout safe?** The gate's `PreToolUse` hook shipped with no
`timeout` while the suggester on the same interactive path had `5`. Adding one is obviously good for
latency and obviously risky if a timeout blocks the tool call. The binary:

```js
if (p && (n==="PreToolUse"||n==="UserPromptSubmit"||n==="UserPromptExpansion"))
  return { answer: p6(n,F), exitClass:"timeout", blocked:true };
return { answer:{}, exitClass:"timeout", localWarning:`… timed out after ${N}s — output discarded …`,
         blocked:false };
```

`p` is `timeoutFailsClosed`, defaulting to `false` and set only for a call served to a cloud session
(the message it guards reads *"timed out … on the attached machine"*). So **locally a hook timeout
discards the hook's output and allows** — bounding converts a long stall into a fast fail-open,
which is the gate's own documented contract, and cannot introduce a block. Without that fact the
honest options were to leave the hook unbounded or to add a bound nobody could justify.

## Why it matters

**A field the documentation does not name is not a field that does not exist, and it is not a field
that does exist either.** This repository has now been bitten in both directions: #1434 happened
because `agent_type` was *assumed* to carry the type, and the `UserPromptSubmit` prompt-text field
was *assumed* to have a particular spelling (recorded in the gate-decision-record thread). The
correction in both cases is the same and it is cheap: the artefact that will actually run the hook
is on disk, it contains its own schema, and `strings` plus a regex answers in a minute what a day of
inference gets wrong.

**And it changes designs, not just details.** Neither of these was a lookup. The first killed a
direction the ticket had proposed and forced a different mechanism; the second turned a change
nobody could justify into one with a stated reason. Measurement before design, not after.

## What to do

- **Before building on a payload field, event name, or timeout semantic that the reference does not
  state, read it out of the installed artefact.** `strings -n 6 <binary> > /tmp/x` once, then grep
  the file — grepping the 215 MB binary repeatedly with backtracking regexes times out; extracting
  first takes 1.3 s.
- **Write the finding where the decision lives**, not only in the session. The timeout reasoning is
  in `TestEveryInteractiveHookIsBounded`'s comment so the next reader trusts the bound instead of
  re-litigating it, and here so the next person bounding *any* hook does not have to re-derive it.
- **Prefer a mechanism that needs no undocumented field at all.** HARNESS-109's fix reads the
  parent's dispatch — a call the gate already receives — rather than a payload field that might be
  renamed. `PreToolUse` is synchronous, so the parent's hook completes before the child exists by
  construction: an ordering guarantee beats a field name every time.

## The same discipline applies to the verification commands

Closing this spec produced a second instance of the same mistake, one level up. Every
`features.json` command was **run** rather than written and trusted, and one of them was wrong:
`grep -c -- '--- PASS: Test'` also counts *indented subtest* lines, so an entry expecting 4
top-level passes measured 18. Anchoring to `^` fixed it.

**A `features.json` verification command is itself untested code, and the archive reads it as
evidence.** It belongs to the family this repository keeps finding — `bats | tail` reporting
`tail`'s status, `go test -run '<no-match>'` exiting 0, `${PIPESTATUS[0]}` expanding to nothing
under zsh — and what every member shares is that **the verification command succeeds**. None of
them crashes. A green result that measured the wrong thing is indistinguishable from a green result
that measured the right one, which is the whole reason running them is not optional. This one was
loud only by luck: 18 against an expected 4 is impossible to miss, and it would have been silently
correct forever had the suite happened to hold four subtests.

## Related

- `specs/HARNESS-109/proposal.md` — the refutation and the design it forced.
- [[lesson-271-hooks-reload-personas-do-not-and-the-difference-was-assumed-away]] — the same class:
  a harness behaviour assumed rather than measured.
- [[lesson-260-the-path-with-no-seam-is-the-path-with-no-test]] — why the timeout assertion had to
  run through `MergeHooks` rather than marshal the struct.
