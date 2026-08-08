---
id: "HARNESS-056"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#820"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, doctrine, cross-agent]
template_version: "1.0"
---

# HARNESS-056

## Why

<!-- from issue #820: HARNESS-056: bind the standing orders to the moment a change is declared done -->

The standing orders already require fixing or ticketing every defect, writing knowledge down in-session, keeping the board honest, and never claiming completion without evidence. All four are written, all four are agent-agnostic, and all four keep being skipped — which is why the same corrections return to the conversation session after session. Nothing binds a rule to the instant a change is declared finished, so the rule is available exactly when nobody is looking for it. Three defects in a single day came from that gap: a fence declared from memory, an auto-archive rule that emitted its trigger for 79 days without firing, and a hardcoded count that quietly stopped being true.

## What

A five-item **Definition of Done** authored in `pattern-change-lifecycle.md`, injected verbatim into every agent surface through the existing `enforced[]` mechanism, and *executed* by the `verification-before-completion` skill — which already triggers on precisely the right moment and, until now, enforced only the fifth item. The skill gains a closing pass that produces a verdict per item and states that a skip is a decision to be named, not a silence.

## Out of scope

- Hooks of any kind. Action-level determinism is #561 and #803; a per-agent hook would break the agnosticism the rest of this depends on.
- The `AGENTS.md` diet (#673), though this adds twelve lines of pressure to it.
- Any new document. The checklist lives in the pattern that already owns the change lifecycle.

## Risks / open questions

- **Theatre risk.** A checklist an agent recites without acting on is worse than none, because it manufactures the appearance of rigour. Mitigated by requiring a verdict per item ("filed as #123", "no debt found") and by naming the intention-not-verdict failure explicitly in the skill.
- **Second-source-of-truth risk.** A checklist that paraphrases the standing orders becomes a competing copy that drifts. Mitigated by binding rather than restating, and by an assertion in the skill that the standing order wins any disagreement.
- **Size.** Every enforced id is injected into files under line caps, and into the compact payload under a character cap. Measured: `ai/claude/CLAUDE.md` 66 → 78 lines against a 100-line cap.

## Acceptance criteria

- [x] A compact Definition of Done exists in `pattern-change-lifecycle.md` under a stable heading anchor.
- [x] It is registered as an enforced id and injected into every declared agent surface, the compact doctrine payload included.
- [x] `verification-before-completion` executes the checklist and names what an unmet item requires.
- [x] Deployed files stay inside their line caps and their character caps, and `--check` reports no drift.
- [x] The checklist binds the standing orders rather than paraphrasing them into a second source of truth, and a test asserts it.

## References

- Bitácora board: mlorentedev/dotfiles#820 (see the `issue:` frontmatter field)
- `00_meta/patterns/pattern-change-lifecycle.md` (host), `pattern-track-or-fix.md`, `pattern-fix-small-debt.md`, `pattern-decision-persistence.md`
- `00_meta/skills/verification-before-completion/SKILL.md` (executor)
- Standing Orders #3, #4, #6, #7, #8 in `AGENTS.md`
- Prior art in this chain: #817 / #819 (the compact doctrine payload this rides on)
