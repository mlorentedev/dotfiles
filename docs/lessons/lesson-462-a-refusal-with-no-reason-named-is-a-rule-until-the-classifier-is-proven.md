---
id: "lesson-462-a-refusal-with-no-reason-named-is-a-rule-until-the-classifier-is-proven"
type: lesson
status: active
title: "A refusal with no reason named is a rule until the classifier is proven"
created: "2026-10-16"
owner: manu
tags: [permissions, claude, classifier, deny, forge]
---

# A refusal with no reason named is a rule until the classifier is proven

**Context:** During an unattended run, `gh issue create` was refused four times ("Permission to use Bash with command ... has been denied"), also through the `new-ticket` skill. The standing order to file defects found while working could not be followed, and the cause was read as the auto-mode classifier. PR #2629 added `Bash(gh issue create:*)` to the allow list on that reading.

**Problem:** The classifier was not the cause of these four refusals. All four refused commands carried the literal text `dotf secrets show` in a commit or issue body, and the deny rule `Bash(*dotf secrets show*)` matches that text anywhere in a command line. A deny resolves before an allow, so the new rule could not have helped; `gh issue create` without the phrase had already passed twice, and with the body moved to `--body-file` it passed again. The allow rule shipped, and the claim that it fixed the problem was false until the session's own history was read for the full refused commands (#2634).

**Solution:** Read the refused command, not the symptom, before attributing a refusal. A rule refusal says "Permission to use Bash with command ... has been denied"; a classifier refusal names a rule such as "External System Writes" and says it is the auto-mode classifier. When prose is the trigger, pass bodies through `--body-file` or `-F`, which keeps them off the command line. The durable fix is `dotf forge`, which reads a body only from a file (ADR-052). The classifier's behaviour toward a `dotf forge` write was measured once, with a development binary, and is recorded on #2635 as `classifier-result: allowed`; the released binary has not been measured, so the binding stays open until it is.

**Why:** A permission fix is an authority widening, so it deserves the same evidence as any other change: a measurement that distinguishes the hypotheses. Here one grep of the refused commands for the deny phrase separated "deny glob" from "classifier" in seconds. A leading `*` in a deny glob makes every body, commit message and heredoc a place the rule can fire.
