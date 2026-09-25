---
id: "HARNESS-145-doctrine-triggers-ledger-agy"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#1682"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-145-doctrine-triggers-ledger-agy

## Why

<!-- from issue #1682: HARNESS-145: the deployed doctrine sits 23 characters under agy's cap, 8 triggers name patterns that do not exist, and calls with no session id share one gate ledger -->

An audit of the harness reported defects; re-measured on 2026-09-24, four were real and all four fail silently. The compact doctrine sat 23 characters under agy's 12,000-character cap, eight triggers named patterns the vault does not have, every hook call with no session id shared one gate ledger, and the agy gate was bound to a file agy does not read and fed a payload it did not parse. The fixes were built on one branch before this ticket existed, so this spec is retroactive: it records what shipped, gates the split into four PRs, and holds the evidence for each.

## What

- The compact doctrine is derived from the vault sections and fits every capped surface with room to spare (under 8,000 characters against agy's 12,000). It is pure ASCII, so the character count equals the byte count, and a test on the committed records pins the budget.
- Every trigger names a pattern that exists. `--refresh` refuses a dangling one before writing anything, and `dotf doctor` reports one on a machine that has the vault. Skill linking follows the rules that matched, not the patterns they share.
- A hook payload with no session id has no ledger: the call is allowed and journaled as `session-unscoped`, and nothing is stored under a shared key.
- agy's own payload is parsed and answered in its protocol (`ask` or `deny`, never `allow`), and the gate is bound in `~/.gemini/config/hooks.json` with the stale `settings.json` entry retired. The new bind format lives under `agents.bind_named`, which older binaries never read.

## Out of scope

- Archiving zombie specs (#1626 owns them).
- A persona source for agy: agy sends no agent type, so its gate is measurement plumbing until one exists.
- Doctor's handling of linked worktrees, and dedicated terraform, kubernetes and hardware patterns (those triggers keep their nearest match).
- Live capture of an agy payload: workspace-local hooks never loaded headless, so the parser is built from the vendor documentation embedded in the binary and confirmed by AC5.

## Risks / open questions

- **The slim doctrine duplicates rules.** The vault keeps the original wording beside the dense text so nothing is lost, and a later edit to one half can diverge. Trim the original-wording sections to reasoning only in a follow-up.
- **The agy payload is documentation-derived.** Mitigation: an unparseable payload fails open and is journaled as `payload-unrecognised`; AC5 closes it with a real call.
- **Manifest and binary ship separately.** An older binary reads only `agents.bind` and treats every target as Claude's shape, so agy lives under `agents.bind_named` and the loader refuses a format in the wrong key. Tests pin both.
- **Ordering.** S2 shares `scripts/compile-harness.sh` with S1, and S4 shares `cli/internal/cmd/harness_gate.go` with S3, so those two open after their bases merge. Until S1 merges, a deploy from any checkout other than the change's own re-inflates the deployed doctrine.

## Acceptance criteria

- [ ] **AC1** — the compact doctrine is under 8,000 characters and pure ASCII on every capped surface, and the deployed `GEMINI.md` has `wc -m` equal to `wc -c`. Pinned by a bats test that reads only committed records.
- [ ] **AC2** — `compile-harness.sh --refresh` fails, naming the trigger and the pattern, when a trigger's pattern is absent, and writes nothing. `dotf doctor` reports the same, and skill linking follows matched rules.
- [ ] **AC3** — two payloads with no session id share no ledger, proven by an end-to-end test that fails on the old code.
- [ ] **AC4** — an agy payload is parsed and answered in agy's protocol, the gate is bound in `hooks.json` with the stale entry retired, and an older binary given the manifest emits nothing wrong.
- [ ] **AC5** — after all four merge: mirror, install a binary that carries the agy parser, `dotf harness bind`, and one real agy tool call leaves an `agy` record with a real conversation id under `~/.local/state/dotfiles/gate/`.

## References

- Bitácora: `mlorentedev/dotfiles#1682` (see the `issue:` frontmatter field).
- ADR-027 (cross-harness agent pipeline, amended by S4) and ADR-010 (hooks row); lessons 292 and 293 land with S4.
- Vault patterns whose sections are the SSOT of the slim records: `pattern-change-lifecycle` (PR Stewardship), `pattern-git-workflow` (section 10), `pattern-secrets-security` (section 8).
