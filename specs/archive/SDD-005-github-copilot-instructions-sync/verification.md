---
tags: [spec, verification, copilot, docs-drift]
created: "2026-05-19"
---

# Verification - SDD-005-github-copilot-instructions-sync

## Evidence

- [x] AC1 `.github/copilot-instructions.md` has `## Model Tier (per AGENTS.md "Model Selection")` section with TBD wording matching `ai/copilot/copilot-instructions.md`.
- [x] AC2 Pointer-banner fallback line aligned to include "from the current repo" wording. Only the AGENTS.md reference form differs (linked vs plain) — by design per file deploy context.
- [x] AC3 `tests/docs-drift.bats` exists with parity test (`copilot-instructions: ai/copilot/ and .github/ match outside pointer banner`).
- [x] AC4 Parity test passes on the synced state.
- [x] AC5 Full bats suite green: **650/650 pass** (was 645 pre-PR; 5 new docs-drift cases added).
- [x] AC6 `ai/copilot/copilot-instructions.md` untouched (canonical source for this PR).

## Test status

- `bats tests/docs-drift.bats` → 5/5 pass.
- `bats tests/*.bats` → 650/650 pass.
- Manual `diff`: only the pointer-banner reference form differs (markdown link `[`AGENTS.md`](../AGENTS.md)` vs plain `` `AGENTS.md` ``). All other content byte-identical.

## Decisions made during implementation

- **Parity rule by blockquote stripping, not section-by-section.** The bats test strips lines beginning with `> ` and `cmp -s` the rest. Simpler and tied to the actual difference (banner). Section-by-section would require maintaining a list of expected headers — more code, same coverage.
- **Cosmetic fallback alignment included** even though the bats test ignores blockquote lines. The 15-character diff was trivial to fix; leaving it would have left a small invisible inconsistency for the next reader.
- **No `--no-verify` skip; pre-commit hooks ran clean.** gitleaks + dotfiles-test (full bats) passed at commit time.

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? **Yes** — "Files that must move together silently diverge unless CI asserts parity. When AI-013-style pointer refactors split content across mirror files, ship a bats parity test in the SAME PR." Recurring pattern (BUG-001, BUG-002, AI-019). Captures generic prevention.
- [ ] ADR-worthy? No — this is operational tooling for an existing decision (cross-agent docs structure already decided in ADR-009).
- [ ] Pattern for `00_meta/patterns/`? **Maybe** — "Pattern: bats parity assertions for mirror files." Defer until a third project hits it.

## Archive checklist

- [ ] `proposal.md` frontmatter `status: archived`.
- [ ] Folder moved: `specs/SDD-005-.../` → `specs/archive/SDD-005-.../`.
- [ ] Vault `11-tasks.md` ticked with PR link.
- [ ] Lesson promoted to `dotfiles/90-lessons.md`.
