---
id: "BUG-050-spec-gate-satisfiable"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-07"
issue: "mlorentedev/dotfiles#800"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-050-spec-gate-satisfiable

## Why

<!-- from issue #800: BUG-050: spec-gate's two halves are mutually unsatisfiable for a large PR that closes its issue -->

`scripts/check-spec-gate.sh` runs two independent checks, and a single PR that is
**over the LOC threshold** *and* **closes its issue** cannot satisfy both.

| Check | Requires |
|---|---|
| Discipline Gate | production diff ≥ 50 LOC must touch a file under an **active** `specs/<id>/` |
| archive-on-merge (#767) | a PR closing an issue must **archive** that issue's spec |

Archiving satisfies the second and breaks the first; not archiving does the
reverse. Each half is deliberate in isolation — `_is_active_spec_path()` rejects
`specs/archive/*`, and `_normalize_rename_path()` collapses a rename to its
destination *specifically* so an archive-move cannot pose as an active-spec touch
(#397). #397 hardened the Discipline Gate against archive-moves **before** #767
made archive-moves mandatory; the two were composed without noticing they now
contradict each other.

Observed on PR #799 (111 production LOC, `Closes #794`), which needed a
`skip-archive` label to land. That is the real cost: the only ways through are
the escape hatches, so the *normal* path has become the escaped path, and both
gates erode.

## What

Count a spec **archived in fulfilment of archive-on-merge** as the Discipline
Gate's spec touch.

The issue proposed counting `specs/archive/<id>/` as a touch. Taken literally
that does not work: the Discipline Gate does not ask "was a spec path in the
diff", it accumulates `SPEC_LOC` and requires it to clear `SPEC_FLOOR=10` (the
#686/C25 "one-line alibi" defence). A real archive move barely registers — PR
#787's archive of `HARNESS-051` is **4 LOC** across four files (three pure
renames at 0/0, plus the `status:` rewrite). So a mandated archive must set the
touch **outright**, not feed `SPEC_LOC`.

Scope:

1. `_archived_spec_issue_map()` — the mirror of the existing
   `_active_spec_issue_map()`, over `specs/archive/<id>/proposal.md`. Needed
   because an archived spec has left `specs/<id>/` entirely, so the active map
   cannot see what a PR archived — including a spec created *and* archived in one
   PR, which is invisible to the active map at base and at head alike.
2. `_mandated_archive_ids()` — the spec IDs archived at head whose `issue:`
   frontmatter names an issue this PR closes.
3. `_is_mandated_archive_path()` — consulted in the diff walk; sets
   `SPEC_TOUCHED=1` directly.
4. `--explain` names the mandated archives, so the decision is inspectable.

## Out of scope

- Widening the escape hatches, or relaxing `SPEC_FLOOR`.
- The closing-keyword scanner's markdown-blindness (#773) — same file, different
  defect, its own PR.

## Acceptance criteria

- [ ] **AC1** A PR over the LOC threshold that archives the spec of the issue it
      closes passes both halves (today: impossible without a label).
- [ ] **AC2** #397 intact: archiving a spec **not** linked to a closed issue earns
      no spec touch, and a large PR doing only that still fails.
- [ ] **AC3** A spec created and archived in the same PR counts — the case
      neither active map can observe.
- [ ] **AC4** No closing keyword ⇒ no mandate ⇒ an archive move remains worthless.
- [ ] **AC5** Every new assertion verified red against the unfixed gate, and the
      two "protection preserved" assertions verified green both before and after.

## Risks / open questions

- **Does this reopen the #686/C25 alibi?** No. The alibi is a trivial edit to an
  *unrelated* active spec. The new credit is reachable only through the `issue:`
  frontmatter of a spec whose issue this very PR closes, so there is no unrelated
  spec to hide behind. AC2 pins it.
- **Self-application.** This PR is itself over the threshold and closes its own
  issue, so it is the first consumer of its own fix: CI runs the PR's copy of the
  gate (checkout of the merge ref), and the PR passes only if the fix works.
