---
id: lesson-055-classify-extract-boilerplate-audit-findings-as-boo
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 055: Classify "extract boilerplate" audit findings as bootstrap (chicken-and-egg) vs logic before estimating LOC savings

**Context:** AUDIT-005 (REFACTOR-001 scripts/ audit, 2026-05-21) proposed POLISH-001: extract get_script_dir + utils.sh source-fallback boilerplate to utils.sh, estimating -75-85 LOC reduction. During closeout investigation, the actual numbers came out very differently: the boilerplate is bootstrap code that runs BEFORE utils.sh is sourced. A helper inside utils.sh cannot replace bootstrap that's needed to find utils.sh in the first place — chicken-and-egg. The only extraction path is a new _bootstrap.sh file each script sources via a 1-liner; net ~-10 LOC after counting the bootstrap file content, with +1 file overhead and non-trivial script-loading risk. Decision: WONTFIX.
**Problem:** Audit-style analysis tends to flag "repeated code across N files" without distinguishing whether the repetition is structural (must happen before any extraction target exists) or logical (after extraction target is available). The two have very different extractability and ROI characteristics. Conflating them produces misleadingly high LOC-saving estimates that look like quick wins but require either a new file (with its own ongoing maintenance cost) or sophisticated meta-tricks like self-locating libraries. The audit-005 agent treated the 11-script SCRIPT_DIR pattern + 10-script utils.sh source pattern as if they were ordinary code duplication, but both are bootstrap.
**Solution:** When an audit (your own or an agent's) flags "extract boilerplate" or "shared-pattern extraction", classify EACH proposed extraction before estimating value: (1) Is the repeated code BOOTSTRAP — runs before the would-be helper file is loaded (e.g., resolving where to find the helper)? Inextricable without adding new files or shell meta-tricks. The honest LOC saving is near-zero. (2) Is the repeated code LOGIC — runs after the helper is loaded? Extractable with a normal function. The LOC saving is real. For bootstrap-class patterns, the cost of extraction (new file + ongoing maintenance + loading-order risk) usually exceeds the LOC saving. Document the classification in the audit so future readers don't re-litigate. Concrete test: can the audit explain how the helper itself would be loaded inside the same boilerplate it's trying to replace? If not, it's bootstrap.
**Tags:** `#audit-discipline` `#refactoring` `#bash` `#boilerplate` `#bootstrap` `#chicken-and-egg` `#loc-estimation`
