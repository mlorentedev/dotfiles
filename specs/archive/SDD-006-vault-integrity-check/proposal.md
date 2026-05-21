---
id: "SDD-006-vault-integrity-check"
type: spec
status: draft
created: "2026-05-19"
tags: [spec, proposal, vault, safety-net]
template_version: "1.0"
---

# SDD-006-vault-integrity-check

## Why

In a single session today, Hive's `vault_patch` MCP wrote the literal 2-character sequence `\n` (backslash-n) into the vault's `dotfiles/11-tasks.md` **four times** instead of interpreting it as a newline. Each occurrence corrupts a markdown bullet list by merging two items into one physical line, invisible in rendered markdown but breaking downstream parsers:

- `scripts/init-spec.sh` greps for `^- \[[ x-]\] \*\*<id>\*\*` at line start; the corrupted bullets never match → vault-gate fails on real entries.
- `scripts/vault-health.sh` frontmatter / orphan scans assume one bullet per line.
- Future audits and obs-cli backlinks operate on parsed line structure.

Each occurrence was fixed manually with `Edit`. The 4th time made the meta-issue explicit: **every bug class encountered must emit a CI assertion or health check in the same PR that fixes the symptom**. Otherwise the class recurs. This is the `incident → guard` pattern; this PR formalises it for the Hive `\n` corruption class and adds a reusable script + bats coverage.

## What

After this PR merges:

1. New script `scripts/check-md-escapes.sh` scans markdown files for the literal `\n` sequence followed by a recognised line-start glyph (`-`, `#`, `>`, `*`). Accepts file or directory args. Exits 1 on any match.
2. New bats file `tests/check-md-escapes.bats` (≥4 cases): fixture detection, healthy-file silence, recursive directory scan, **self-test** against the dotfiles repo's own markdown — catches corruption in *this* repo on every CI run.
3. New lesson in `90-lessons.md` (vault) documenting the incident → guard pattern as a general rule, referencing sibling instances (BUG-001/002 verify-string drift, AI-019 missed `.github/copilot-instructions.md`, this Hive `\n` corruption).

## Out of scope

- Fixing Hive's MCP tool. Out of this repo's control.
- Wiring the check into `scripts/vault-health.sh` Section 7. Defer: vault-health requires Obsidian GUI which CI can't satisfy. Standalone script is more useful and bats-testable directly.
- Pre-commit hook in the vault repo to block corruption at commit time. Lives in the vault repo, not dotfiles.

## Risks / open questions

- **Risk: false positives in code blocks.** A markdown code block legitimately showing the corruption pattern (backslash-n then a bullet glyph) would trip the check. **Mitigation**: the canonical corruption is specifically `[BS-n]` followed by one of the line-start glyphs in *content* lines; code blocks normally have these chars in different contexts. If false positives appear, refine to exclude triple-backtick blocks (deferred until evidence). Note: this proposal itself avoids embedding the literal byte sequence in prose so the self-test stays green.
- **Risk: the self-test trips on this very PR** because the proposal.md mentions the literal `\n` corruption pattern. **Mitigation**: the regex matches `\n` followed by `-#>*`; in this proposal we always quote `\n` inside backticks or surround with non-trigger chars. Verify by running the check against this spec folder before commit.
- **Open**: bats parity in CI lint or regular test job? **Decision**: regular bats job. Same lifecycle as the rest.

## Acceptance criteria

- [ ] `scripts/check-md-escapes.sh` exists, has `set -euo pipefail`, accepts file/dir args, exits 0 on clean / 1 on corruption / 2 on usage error.
- [ ] `tests/check-md-escapes.bats` has ≥4 test cases (corruption fixture, healthy fixture, recursive directory scan, dotfiles-repo self-test).
- [ ] All new tests green.
- [ ] Full existing bats suite green (no regression). Target: 650 + 4 new = 654.
- [ ] `shellcheck --severity=error scripts/check-md-escapes.sh` clean.
- [ ] Lesson appended to vault `dotfiles/90-lessons.md` documenting incident → guard pattern.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` SDD-006 entry.
- Sibling instances: BUG-001 PR [#40](https://github.com/mlorentedev/dotfiles/pull/40), BUG-002 PR [#47](https://github.com/mlorentedev/dotfiles/pull/47), SDD-005 PR [#62](https://github.com/mlorentedev/dotfiles/pull/62).
- Trigger: 4 corruption hits in the 11-tasks.md edits earlier this session.
- Standing Order #4 in `AGENTS.md`: "Clean as you go".
