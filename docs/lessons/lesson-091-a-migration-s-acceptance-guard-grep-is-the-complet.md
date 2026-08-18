---
id: lesson-091-a-migration-s-acceptance-guard-grep-is-the-complet
type: lesson
status: active
created: "2026-06-13"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 091: A migration's acceptance guard-grep is the completeness oracle, not the spec's hand-listed targets

**Context**: CLI-005 retired the `init-spec`/`archive-spec` shell twins and repointed every reference to `dotf spec`. The proposal enumerated five repoint targets by hand (AGENTS.md, `agents-md.bats`, `check-spec-gate.sh`, the spec `SKILL.md`, the architecture-map).

**Problem**: The hand-list was incomplete. The acceptance criterion's own guard — `grep -rE 'init-spec|archive-spec'` must return only historical artifacts — surfaced **two more live references the list missed**: a comment in `scripts/check-md-escapes.sh` and two lines in `harness/skills/adversarial-review/SKILL.md` that named `archive-spec.sh` as a command to run. Shipping the spec's list verbatim would have left broken references that no test named.

**Solution**: Treat the acceptance guard-grep as the authority and run it *before* claiming done, then repoint whatever it returns until only provenance (CHANGELOG, ADRs, lessons, `specs/`) remains. The hand-list is a starting hypothesis; the grep is the proof.

**Rule**: When a change's acceptance criterion is "no live reference to X remains except historical," the grep that expresses it is the completeness oracle — not the enumerated edit list in the proposal. Run it as a gate, not an afterthought; it finds the surfaces a human inventory forgets.
