---
id: "HARNESS-063-spec-gate-adjacency"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-09"
issue: "mlorentedev/dotfiles#858"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-063-spec-gate-adjacency

## Why

<!-- from issue #858: HARNESS-063: a fix can ship green while being a no-op on the shape that motivated it -->

A fix can merge with a correct, passing, honestly-written test and still be a no-op
on the case that motivated it. On 2026-08-08 that happened end to end in eleven
minutes: #850 filed at 17:36Z, #851 merged at 17:47:56Z with a test asserting the
right invariant, #850 closed one second later — and the defect was still live on
every `content: |`-wrapped `MEMORY.md` (#857). The second shape was not unknown:
**#849 described it on the board at 17:35:06Z, thirteen minutes before #851
merged.** Nothing connected the two, because every gate we have — review,
`spec-gate`, the test suite — has the same field of view: *the issue this PR
closes*. An open issue about the same file, filed by someone else, is invisible.

## What

`spec-gate` gains an **advisory adjacency report**. On a PR that closes an issue,
CI lists every *other* open issue whose title or body names a file in the diff, as
a `::warning::` annotation plus a step-summary table. It never fails the gate.

The network and the logic are split deliberately, because
`scripts/check-spec-gate.sh` is offline by design — it also runs in pre-push
(#854), where no token exists:

| Half | Lives in | Why there |
|---|---|---|
| Fetch the open-issue list | `.github/workflows/spec-gate.yml` | the token lives there; one `gh issue list` call, no per-file search-API queries |
| Match issues against changed files | `check-spec-gate.sh --adjacency-issues <file>` | pure, offline, fixture-testable; absent flag or file ⇒ silent no-op |

That split is what makes acceptance criterion 2 reproducible: the red-test is a
fixture file holding #849's real title and body, not a mocked API.

## Out of scope

- **Fixing #857.** This spec is about why it shipped, not what it is.
- **The fixture-shape inventory** (#858's other adopted direction). Its first
  populated inventory is `MEMORY.md`'s two shapes, which only exist inside #857's
  fix; building the mechanism here would merge uncalled code. It travels with
  #857, and that is the PR that closes #858. This one carries `Refs #858`.
- **Direction 1 of #858** (`verification.md` must name the reported artifact) —
  withdrawn, see the criterion-3 measurement below.
- **Direction 3 of #858** (manual reproduce-from-the-issue) — does not scale past
  `bug`-labelled PRs and nothing can enforce it.
- Any change to `spec-gate`'s LOC threshold or Discipline Gate semantics.
- Posting the report as a PR comment (needs `pull-requests: write`); an annotation
  needs no extra permission. Escalation path noted under Risks.

## Risks / open questions

- **Noise.** 159 open issues matched by basename will produce false adjacencies.
  Mitigated by being advisory-only: a cry of wolf costs a glance, whereas a hard
  gate would cost merges. Matching is on the full path *or* the bare basename, and
  the report names which issue matched which file so the reader can dismiss it in
  one look.
- **`_strip_markdown_code()` has the wrong polarity for this job.** It exists so
  `_closing_issue_numbers` does not fire on references inside code blocks — there,
  a false positive is the expensive error. For adjacency the expensive error is a
  false *negative*, and #849's body carries `scripts/knowledge-crystallize.sh`
  inside an inline code span that the stripper would delete. The worked example
  survives only because #849's *title* carries the bare basename unformatted.
  **Adjacency matches over unstripped title + body**, and AC2 pins that so a
  future refactor reusing the helper goes red.
- **Token absence must be indistinguishable from today's behaviour**, or the
  pre-push path (#854) breaks. AC4 covers this.
- **Visibility.** If an annotation proves too easy to miss in practice, escalating
  to a PR comment is a follow-up, not a redesign — the report content is unchanged.
- Rate limits are not a concern at one list call per PR event.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] On a PR that closes an issue, CI emits an advisory report naming every other
      open issue whose title or body references a **production** file in the diff
      (production as `_excluded()` already defines it, and excluding active spec
      folders), and which file matched. Narrowed from "a file in the diff" after
      measuring against the live backlog: template and doc filenames matched 34 of
      37 rows and buried the signal — see `verification.md`.
- [ ] **Red-tested against #851.** Replaying #851's changed-file list
      (`scripts/knowledge-crystallize.sh`, `.ps1`) against a fixture holding #849
      as open flags it. The fixture keeps #849's real formatting, so the test fails
      if matching is ever run over stripped text.
- [ ] The check never fails the gate: a PR with adjacent issues exits with the
      same status it would have exited with before this change.
- [ ] With no `--adjacency-issues` file and no token, `check-spec-gate.sh` produces
      byte-identical output to the current version, keeping the offline pre-push
      path (#854) intact.

## References

- Bitácora board: `mlorentedev/dotfiles#858` (see the `issue:` frontmatter field)
- Direction decision + criterion-3 evidence table: #858 comment of 2026-08-09
- Worked example: #850 (reported), #851 (merged fix), #849 (the unseen open issue), #857 (the surviving defect)
- Adjacent, deliberately not merged into this: #852 (HARNESS-061, stub/real pairing — the missing axis there is test *fidelity*, here it is *input shape*)
- Related patterns: `00_meta/patterns/pattern-verify-state-before-acting.md`
