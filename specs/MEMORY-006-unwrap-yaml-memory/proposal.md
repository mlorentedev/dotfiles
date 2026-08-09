---
id: "MEMORY-006-unwrap-yaml-memory"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-09"
issue: "mlorentedev/dotfiles#864"
tags: [spec, proposal]
template_version: "1.0"
---

# MEMORY-006-unwrap-yaml-memory

## Why

Auto-memory `MEMORY.md` files store their body inside a YAML block scalar
(`content: |`) instead of plain markdown — **17 project keys resolving to 16
distinct files** (two keys alias one vault directory after a rename). The
*historical* corpus in vault commit `1c216229` is larger, **23 pairs**, because
it includes entries since archived or renamed; those three numbers are kept
distinct throughout this spec and tabulated in `tasks.md`.

That shape is **invalid state**, not a supported variant:

| Fact | Source |
|---|---|
| Explicitly forbidden | `00_meta/templates/agent-memory.md`: *"Plain-markdown auto-memory — never a `content: \|` YAML block"* (vault `de0d5773`, 2026-06-20) |
| Nothing we ship emits it | `cli/internal/vault/templates/vault-memory.md` and both crystallize twins write plain markdown |
| One accident, not a policy | all 17 wrapped in a single vault commit, `1c216229` — `2026-05-26 21:17:41` |
| Not recurring | nothing has wrapped a file in the 2.5 months since |

The template rule postdates the damage by three and a half weeks. Someone
diagnosed this, wrote the guard that stops *new* files being wrapped, and never
migrated the ones already broken — so the rule protects the future while the past
stays live. #857 is the bill for that missing half: crystallize met one of these
files and corrupted it, and #862 could only make it refuse.

This is the incident→guard pattern applied to one half only. The stock was never
cleaned.

## What

A `dotf doctor` check that detects a YAML-wrapped auto-memory `MEMORY.md` and,
under `--fix`, migrates it back to the `agent-memory.md` template shape:
frontmatter preserved, `content` key removed, body de-indented to column 0 and
emitted as plain markdown after the closing `---`.

**Why a doctor check rather than a one-shot script.** Running it once *is* the
migration; leaving it installed *is* the guard that catches a recurrence. One
artifact covers both halves of incident→guard — which is precisely what was
missed in May, when the rule shipped without the cleanup. It also follows the
existing `checkAutoMemoryLink` contract exactly: verify always, repair only under
`--fix`, idempotent, and never overwrite real data silently.

## Out of scope

- **Porting crystallize to Go** — #490, which no longer carries any YAML scope.
- **Repointing callers / deleting the twins** — #492.
- **Removing #862's refusal guard.** It stays permanently as defence-in-depth: if
  anything ever wraps a file again, refusing beats corrupting regardless of what
  this check does.
- **Changing what crystallize writes.** This migration only restores the shape
  crystallize already expects.

## Risks / open questions

- **De-indenting by the YAML rule alone is not enough, and that is the whole
  trap.** By YAML's own rule the block indent is set by the first non-empty line
  = 4. But the 2026-05-26 wrapper emitted the first body line at 4 and **every
  subsequent line at 6**, in 16 of the 17 files. So a faithful de-indent leaves
  all but the first line at column 2 — and crystallize anchors every marker at
  column 0 (`^## Session Handoff`, `^# currentDate`, `^## Last Crystallized:`).
  A "correct" YAML de-indent therefore produces a file that parses fine and
  *still* fails to crystallize. The migration must strip the block indent **and**
  the uniform residual indent, both derived, neither assumed.
- **`hive` is the one uniform file** (residual indent 0). It is the natural
  regression case against over-stripping: the same code path must leave it right.
- **Bulk edit of live memory.** 17 files holding real session continuity. Dry-run
  diff reviewed before any write is non-negotiable; `--fix` must be explicit.
- **Where the files physically live varies.** Most are bridged into the vault by
  symlink/junction, which makes the write a vault change (direct to `master`, per
  the vault commit convention) rather than a repo change. Enumerate before
  writing; a file *not* bridged is a plain local write.
- **Open question:** should the check FAIL (not WARN) on a wrapped file when
  `--fix` is absent? `checkAutoMemoryLink` fails for a repairable link, which is
  the closest precedent and argues yes.

## Acceptance criteria

- [ ] `dotf doctor` reports every YAML-wrapped auto-memory `MEMORY.md`; `--fix`
      migrates it; a second run is a clean no-op.
- [ ] De-indent = YAML block indent **+** uniform residual indent, both derived
      from the file. No literal width anywhere in the implementation.
- [ ] **Validated against ground truth:** vault commit `1c216229` holds the
      pre-wrap version of every affected file. Run the algorithm over each file
      as it existed in that commit and compare against `1c216229^` — real
      before/after pairs authored by neither this code nor its author, a stronger
      corpus than invented fixtures and what #672 asks for in spirit.

      **The contract is not whole-file equality**, because the May wrap was lossy
      in two ways (both measured, see `verification.md`): it dropped each file's
      trailing `# currentDate` section — and 169 lines from one work project —
      *and* it truncated mid-line in two files. So the assertion is: every
      **complete** recovered line equals its counterpart byte-for-byte, and only
      the final line may be a prefix of its counterpart. The unrecoverable
      remainder is **#865**, not this ticket.
- [ ] Migrated files crystallize successfully — proven by running crystallize,
      not by inspecting indentation.
- [ ] Frontmatter is preserved byte-for-byte apart from the removed `content` key.
- [ ] Markdown hard breaks (trailing double-spaces) and blank-but-indented lines
      survive, since after de-indentation they are ordinary content.
- [ ] The refused-file path in #862 still refuses anything this check has not
      migrated — the two are complementary, not alternatives.

## References

- Bitácora: mlorentedev/dotfiles#864
- #857 — the defect report this migration answers; #862 (`9caedc1`) — the shipped guard
- #490 / #492 — the Go port and the cutover, both now free of YAML concerns
- `specs/CLI-021-dotf-vault-build-knowledge/evidence-yaml-roundtrip.md` — why no
  YAML-library roundtrip can do this instead (hard breaks do not survive)
- Precedent for the check's contract: `cli/internal/doctor/checks_automemory.go`
