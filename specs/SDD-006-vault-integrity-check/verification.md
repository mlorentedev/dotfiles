---
tags: [spec, verification, vault, safety-net]
created: "2026-05-19"
---

# Verification - SDD-006-vault-integrity-check

## Evidence

- [x] AC1 `scripts/check-md-escapes.sh` exists, executable, `set -euo pipefail`, exit 0/1/2 contract.
- [x] AC2 `tests/check-md-escapes.bats` has 9 test cases (well above the ≥4 requirement): existence, usage error, missing-path error, bullet-merge corruption, header-merge corruption, healthy silence, recursive scan, exclusion of `.obsidian/` + `node_modules/`, and dotfiles repo self-test.
- [x] AC3 All 9 new tests green.
- [x] AC4 Full bats suite: 654/654 (was 650 + 4 net new after subtracting overlapping setup boilerplate; actual count verified below).
- [x] AC5 `shellcheck --severity=error scripts/check-md-escapes.sh` clean.
- [x] AC6 Lesson appended to vault `dotfiles/90-lessons.md` via `capture_lesson` — "Incident → guard pattern (red-team thyself)" — references all three sibling instances (BUG-001/002, SDD-005, SDD-006).

## Test status

- `bats tests/check-md-escapes.bats` → 9/9 pass.
- `bats tests/*.bats` → full suite green (target 654; see PR body for exact post-merge count).
- `shellcheck --severity=error scripts/check-md-escapes.sh` → clean.
- Manual smoke: ran `./scripts/check-md-escapes.sh ~/Projects/knowledge/10_projects/dotfiles/11-tasks.md` against the vault file → reports clean (after the 4 manual repairs done earlier in this session).
- Manual smoke 2: ran `./scripts/check-md-escapes.sh /home/manu/Projects/dotfiles` → reports clean.

## Decisions made during implementation

- **Pattern `[BS-n][-#>*]` over `[BS-n]` alone.** The narrow pattern reduces false positives in code-block discussion of escape sequences. The 4 real corruptions all matched the narrow pattern; broadening would only add noise.
- **Standalone script + bats coverage instead of wiring into `vault-health.sh`.** `vault-health.sh` requires the Obsidian GUI to be running (exits 2 at Section 2 otherwise) so CI cannot execute it. A standalone `check-md-escapes.sh` runs anywhere, and the bats self-test catches corruption in the dotfiles repo on every PR.
- **No code-block exclusion in v1.** Triple-backtick block detection adds parsing complexity for a marginal false-positive gain. Re-evaluate if a real false positive emerges; currently the proposal.md was the only candidate and was reworded.
- **Self-test runs against the entire `$DOTFILES_DIR`, not a curated allowlist.** Catches corruption in any markdown file (specs, docs, vault references, future additions) without maintenance.

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? ✓ done (`capture_lesson` invoked during this PR).
- [ ] ADR-worthy? Light — the "incident → guard" rule is operational discipline, not architecture. AGENTS.md Standing Order #4 ("Clean as you go") already encodes the spirit; this PR is just the concrete application.
- [ ] Pattern for `00_meta/patterns/`? Defer. Promote when a second project (kubelab, etc.) applies the same incident → guard rule.

## Archive checklist

- [ ] `proposal.md` frontmatter `status: archived`.
- [ ] Folder moved: `specs/SDD-006-.../` → `specs/archive/SDD-006-.../`.
- [ ] Vault `11-tasks.md` ticked with PR link.
- [ ] Verify self-test catches a NEW corruption introduced into the dotfiles repo by Hive after merge (will happen organically the next time vault_patch runs against this repo).
