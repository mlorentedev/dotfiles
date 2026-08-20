---
id: "BUG-062-crystallize-refuse-yaml"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-09"
issue: "mlorentedev/dotfiles#857"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
review: waived
review_waived_reason: "Work shipped and merged under #857; the issue was then closed by hand, so the archive-on-merge gate (keyed on a PR closing keyword) never saw it and the spec was left active. A retroactive adversarial review cannot gate code already on main, so the waiver is recorded instead of manufacturing one. Backlog reconciliation 2026-08-19."
---

# BUG-062-crystallize-refuse-yaml

## Why

<!-- from issue #857: BUG-062: crystallize corrupts YAML-wrapped MEMORY.md -->

`knowledge-crystallize` anchors every marker it looks for at column 0 (`^## Last
Crystallized:`, `^# currentDate`, `^## Session Handoff`). A `MEMORY.md` whose body
lives inside a YAML `content: |` block scalar carries every line indented, so no
anchor matches, control falls through to the bare append, and the text lands
**outside** the block. Three defects follow: the HARNESS-029 handoff invariant
breaks, a second date stamp appears at column 0 while the real one goes stale, and
the file stops parsing as YAML — which is what the format exists for. The script
prints `[SUCCESS]` twice while doing it.

Measured on two projects independently: `pollex` (#849) and `hive` (#857), both
repaired by hand. #851 fixed the plain-markdown shape and is a **no-op** here, so
this is not a regression of that fix but the shape it never saw.

## What

Both twins detect the wrapper and **refuse**, with a message naming the file and
where the real fix lives, instead of corrupting it. A refused file is left
byte-identical. Plain-markdown files are unaffected.

Refusing rather than fixing is deliberate. The robust fix is parse → mutate →
re-dump, which shell does badly and which the Go port (#490, `dotf vault
crystallize`) owns. Writing a shell YAML mutator now would be work built to be
deleted, and #490's acceptance ("build beside, flip nothing") means the *running*
path stays the twins until that port lands. Between now and then the choice is
corrupt or refuse.

## Out of scope

- Making crystallize **work** on the wrapped shape. That is #490's port, gated by
  #672's golden characterization tests. This spec closes the corruption, not the
  capability, so #857 stays open until the port flips the callers.
- Any change to the plain-markdown path, which #851 already made correct.
- The fixture-shape inventory from #858. It belongs with the port, where a
  YAML-aware implementation gives the second fixture something to exercise.

## Risks / open questions

- **Indent width is not fixed** — `pollex` indents six spaces, `hive` four. Any
  detection keyed to a literal width works on one and fails on the other, so the
  guard keys on the wrapper's *structure* (`---` first line plus a `<key>: |`
  opener), never on indentation. Both widths are tested.
- **`/crystallize` stops working for wrapped projects** until the port lands. That
  is a reduction in capability, but the capability it replaces was destructive:
  the previous behaviour did not half-work, it corrupted while reporting success.
- **The `.ps1` twin cannot be executed here** (no `pwsh` on this machine), so its
  verification is static: structural assertions plus PSScriptAnalyzer in CI.
- A false positive would refuse a legitimate plain file. Requires a `---` first
  line *and* a block-scalar opener, which no plain `MEMORY.md` has.

## Acceptance criteria

- [ ] A `MEMORY.md` wrapped in a YAML block scalar is refused with a non-zero exit,
      at both four- and six-space indentation.
- [ ] The refused file is byte-identical afterwards and still parses as YAML.
- [ ] A plain-markdown `MEMORY.md` keeps working, handoff invariant intact.
- [ ] `--all` counts a refusal as skipped, not processed — no "5 / 5 (0 skipped)"
      while one was declined.
- [ ] Both twins carry the guard; the PowerShell one is asserted structurally.

## References

- Bitácora board: `mlorentedev/dotfiles#857` (see the `issue:` frontmatter field)
- Prior art: #849 (same defect from `pollex`, closed into #857), #850/#851 (the
  plain-markdown fix this is not a regression of)
- Owner of the real fix: #490 (CLI-021, `dotf vault crystallize`), gated by #672
  (CLI-031, golden characterization tests before any twin port)
- Why it shipped green in the first place: #858 (HARNESS-063)
