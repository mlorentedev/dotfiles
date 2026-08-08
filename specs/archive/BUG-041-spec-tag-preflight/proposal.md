---
id: "BUG-041-spec-tag-preflight"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#769"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-041-spec-tag-preflight

## Why

<!-- from issue #769: BUG-041: dotf spec archive pre-flight false-positives on `[AGENT-DRAFT]` inside code spans and completed tasks -->

The archive lock was inverted in **both** directions, and the reported symptom was
only the milder half.

Reported: the pre-flight refuses to archive a spec that merely *quotes* the
editorial markers — inside a code span, or on a completed `- [x]` line. That
pushes users toward a reflexive `--force-with-drafts`, and once the flag is
reflexive a genuine draft sails through too. The irony is structural: any spec
for tooling that handles draft markers cannot be archived without the override,
and this repo builds exactly that tooling.

Found while fixing it, and worse: the pattern required `]` immediately after the
keyword, but `/spec fill` emits the suffixed form documented in
`harness/skills/spec/SKILL.md` ("Capture rules"). That form has never matched, so
**the archive lock has never locked on a marker emitted the way the tooling
emits it.** Live proof in-tree: `specs/archive/CLI-002-repo-structure/proposal.md`
sits in the archive today still carrying a live suffixed AGENT-SUGGESTION marker.

Stated plainly: the guard matched exactly the shape that is always a false
positive, and was blind to exactly the shape that is always a true positive.

A second call site had drifted independently. The session-start injector used a
cruder literal substring scan, so it and the archiver disagreed about what
"unresolved" means — it flagged `AI-028-hive-install-model-migration` on the
strength of a completed task whose own text says the marker is resolved.

## What

1. Widen the pattern to both emitted shapes — bare and suffixed.
2. Exclude occurrences that are quoted rather than live: fenced code blocks,
   inline code spans, and ticked `- [x]` checklist lines.
3. Give both call sites one predicate (`spec.ScanUnresolvedTags`), so the
   archiver and the session-start injector can no longer disagree.

The exclusions are shape-based rather than a markdown parse, deliberately: a spec
is not arbitrary markdown, and a parser would be a far heavier dependency than
the two shapes that actually produce false positives here.

## Out of scope

- `check-spec-gate.sh`'s closing-keyword scan (#773) — same root pattern
  (markdown-unaware text scanning), different file, different language, its own PR.
- The secondary observation in #769 about the refusal's output shape when
  scripted. The exit code is already correct; wrapper discipline is a docs matter.

## Acceptance criteria

- [x] **AC1** A marker inside a code span, a fenced block, or on a `- [x]` line
      does not block archiving.
- [x] **AC2** The canonical suffixed form emitted by `/spec fill` **does** block
      archiving (today it does not — the false negative).
- [x] **AC3** The bare form in live prose still blocks, as before.
- [x] **AC4** Fence handling is delimiter-aware: content after a closed fence is
      scanned again.
- [x] **AC5** The session-start injector uses the same predicate, and no longer
      flags a spec the archiver would happily archive.

## Risks / open questions

- **Does widening the pattern make the guard noisier?** No: the widening applies
  only to live text, and the exclusions remove the shapes that generated every
  observed false positive. Net effect is fewer refusals and more real catches.
- **Could a real marker hide inside a code span?** In principle. Accepted: the
  tooling never emits into code spans, and the alternative — matching quoted
  documentation — is the status quo that trained people to reach for the override.
