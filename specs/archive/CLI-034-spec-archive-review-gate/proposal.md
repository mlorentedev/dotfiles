---
id: "CLI-034-spec-archive-review-gate"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-09"
issue: "mlorentedev/dotfiles#875"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
review: waived
review_waived_reason: "Work shipped and merged under #875; the issue was then closed by hand, so the archive-on-merge gate (keyed on a PR closing keyword) never saw it and the spec was left active. A retroactive adversarial review cannot gate code already on main, so the waiver is recorded instead of manufacturing one. Backlog reconciliation 2026-08-19."
---

# CLI-034: Spec archive review gate

> **Naming**: file lives at `dotfiles/specs/CLI-034-spec-archive-review-gate/proposal.md`.

## Why

<!-- from issue #875: bind the adversarial-review gate to spec archive -->

The `adversarial-review` skill has existed since 2026-05-14 and has **never run as its designed
gate**: a search for its output block (`## Adversarial review`, `Evaluator rubric`,
`PASS (adversarial)`) across `specs/` and `docs/` in every personal repo returns zero hits, against
~218 archived specs. The cause is not quality — the skill carries a severity × reality ×
test-traceability model and a six-dimension rubric. The cause is that **nothing requires it**:
`dotf spec archive`'s only pre-flight is the `[AGENT-DRAFT]` tag check, and the skill's own
"Pairs with `/spec archive` lock" line describes an integration that was never implemented.
Issue #255's context line ("gated to high-stakes SDD changes") records the same belief; the
evidence disproves it.

This is the coverage-not-age doctrine in the negative: a rule with no enforcing layer does not
fire. This spec supplies the layer.

## What

After this PR:

1. `dotf spec archive <id>` runs a **second pre-flight**: the spec folder must contain `review.md`
   whose frontmatter carries a `verdict:` field. Missing artifact or `verdict: FAIL` refuses the
   archive; `PASS` and `PASS-WITH-GAPS` proceed (gaps are already tracked by the review itself).
2. The refusal is escapable two ways, both **declared, never judged at the moment**:
   - a waiver in `proposal.md` frontmatter — `review: waived` **plus a non-empty
     `review_waived_reason:`**; a waiver without a reason still refuses;
   - `--force-without-review`, mirroring the existing `--force-with-drafts`.
3. `review.md` carries a **staleness floor**: a `reviewed_sha:` field. If any *contract* file in the
   spec folder changed after that SHA, the review no longer describes the archived change and the
   archive refuses. This is the analogue of `check-spec-gate.sh`'s `SPEC_FLOOR=10` — the
   anti-alibi that stops an empty or outdated artifact from satisfying a presence check.
4. The `adversarial-review` skill gains a **persistence contract**: it writes its verdict to
   `specs/<id>/review.md` with the frontmatter schema below. Today its output format is a chat
   response block with no frontmatter and no destination — the gate would have no compliant
   producer without this change.

### `review.md` frontmatter schema (the contract both sides reference)

```yaml
---
spec: "<feature-id>"                  # must equal the containing folder name
verdict: "PASS"                       # PASS | PASS-WITH-GAPS | FAIL
reviewed_sha: "<40-hex commit sha>"   # the commit the review actually examined
reviewer: "<agent or model id>"       # e.g. claude-opus-5, deepseek-v4-flash
date: "YYYY-MM-DD"
---
```

The body remains the skill's existing markdown output (findings table + rubric + verdict +
next steps) — unchanged, only persisted.

### Staleness scope

Staleness compares `reviewed_sha` against changes to the **contract files only**:
`proposal.md`, `tasks.md`, `features.json`.

Deliberately excluded:

- **`review.md` itself** — its own commit always postdates `reviewed_sha`, so including it would
  make every review stale by construction.
- **`verification.md`** — it is an implementation log whose archive checklist is ticked *at archive
  time*, which would false-positive on every archive.

## Out of scope

- The NaN judge lane (CI-hosted producer of `review.md`) — separate PR, gated on **AI-001**
  (`knowledge#150`) for model-name reconciliation.
- The aggregate rubric pass over archived reviews (hermes batch job) — separate PR, only worth
  building once enough `review.md` files exist to aggregate.
- Reflecting the gate in CI (`check-spec-gate.sh` / `spec-gate.yml`). Adjacent to #854 but a
  distinct surface; this PR gates the CLI only.
- Retro-filling `review.md` for the ~218 already-archived specs.
- Any change to the review's *content* model (severity/reality/rubric) — shipped in #255.

## Risks / open questions

- **The waiver becomes the default path.** Mitigated by requiring a non-empty reason string, which
  makes routine waiving visible in the diff and auditable after the fact. Not fully preventable in
  code; accepted.
- **Staleness brittleness.** A rebase rewrites SHAs and can make `reviewed_sha` unresolvable in the
  current history. Decision: an *unresolvable* SHA is treated as stale (refuse), not as pass — the
  safe direction — and `--force-without-review` remains the documented escape.
- **Vault SSOT, not the render.** Both SKILL edits must land in `$VAULT_PATH/00_meta/skills/…`, not
  `harness/skills/…`; per the CLI-005 lesson, editing the committed render alone is reverted by the
  next `compile-harness.sh --refresh`.
- Archived specs predate the gate; `dotf spec archive` must not retro-validate them.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1 — `Archive()` refuses when `review.md` is absent, naming the missing artifact and the
      two declared escapes in the error.
- [ ] AC2 — `Archive()` refuses when `review.md` carries `verdict: FAIL`.
- [ ] AC3 — `Archive()` proceeds when the verdict is `PASS` or `PASS-WITH-GAPS`.
- [ ] AC4 — `review: waived` **with** a non-empty `review_waived_reason:` skips the gate; `review:
      waived` **without** a reason still refuses.
- [ ] AC5 — `--force-without-review` archives despite a missing or `FAIL` review, mirroring
      `--force-with-drafts`.
- [ ] AC6 — a review whose `reviewed_sha` predates a change to `proposal.md`, `tasks.md` or
      `features.json` is rejected as stale.
- [ ] AC7 — the commit that adds `review.md` does not make it stale, and a later change to
      `verification.md` does not make it stale.
- [ ] AC8 — the vault `adversarial-review` SKILL documents the `review.md` destination and
      frontmatter schema, and the vault `spec` SKILL's `archive` section documents the new
      pre-flight; the "Pairs with `/spec archive` lock" claim is now true.

## References

- Bitácora board: see the `issue:` frontmatter field
- Related: `#255` (HARNESS-004, review content model — close-as-done, its ACs are met by vault
  commit `3dd696af`), `#786` (TOOL-013, the pre-merge PR-review axis — different axis, shared
  inference provider), `#854` (BUG-061, the CI-side spec gate)
- Prior art in-repo: `cli/internal/spec/archive.go` (`FindUnresolvedTags` pre-flight),
  `scripts/check-spec-gate.sh` (`SPEC_FLOOR` anti-alibi, declared escapes)
- Vault: `00_meta/skills/adversarial-review/SKILL.md`, `00_meta/skills/spec/SKILL.md`,
  `00_meta/patterns/pattern-change-lifecycle.md` (stage 2 = this gate)
