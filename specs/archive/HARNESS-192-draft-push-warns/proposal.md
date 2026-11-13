---
id: "HARNESS-192-draft-push-warns"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-11"
issue: "mlorentedev/dotfiles#3364"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# HARNESS-192-draft-push-warns

> **Naming**: file lives at `<repo>/specs/HARNESS-192-draft-push-warns/proposal.md`. `HARNESS-192-draft-push-warns` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3364: spec-gate: the pre-push archive-on-merge refusal holds review fixes whose proof is CI -->

A PR that closes an issue whose spec is still active cannot receive its review fixes: the pre-push SDD gate refuses the push (`archive-on-merge violation`), the spec archives only on a passing adversarial review, and the review fails a REAL finding whose only proof is a CI leg the refused push would produce. Measured on #3355: round 1 found a test red on `test (windows-latest)` only, the fix was committed and the push refused, round 2 failed on "the Windows proof does not exist". The way through was to switch the PR body to `Refs #3263` for one push and restore it with the archive, which works and misdescribes the PR for hours. The archive-on-merge rule is a merge-time invariant; checked at push time on a draft under review it blocks the fix it needs.

## What

On a draft PR, the pre-push gate reports an archive-on-merge violation as a warning that names the archive command, and the push goes through. Nothing else moves: CI's `spec-gate` keeps refusing the merge until the spec is archived in the PR, a ready PR is refused at push time as before, and the `skip-archive` escape keeps its rationale requirement on drafts too.

- `scripts/spec-gate-prepush.sh` resolves the open PR's `draft` field with its labels, body and author, and hands the gate `SDD_PR_DRAFT=true|false`.
- `scripts/check-spec-gate.sh` and `dotf gate spec` (the parity twin, CI-018) print `[WARN] SDD archive-on-merge: this draft PR closes an issue whose spec is still active:` with the spec ids and the `dotf spec archive <id>` lines, and exit 0, only when `SDD_PR_DRAFT` is `true`.
- `scripts/spec-gate-pr.sh` (CI's adapter) never sets the variable, and `dotf gate spec` does not read the event's own `draft` field: a draft PR in CI is refused exactly as before.

## Out of scope

- Option B of #3364 (exempting a head that only adds commits to a PR with a `review.md`): reads the forge at push time and is harder to explain than the draft flag.
- Any change to the LOC threshold, the `skip-archive` escape or the `Refs #N` path; this adds one exit for one state.
- Honouring the event's `draft` field in CI: a gate that went green on a draft would stay green when the PR is marked ready without a push (`ready_for_review` runs no push-triggered check).

## Risks / open questions

- Resolved: the Go twin must say the same thing in the same words, since `tests/lib/spec-gate-parity.bash` compares the outputs. Both print the warning from one text; the parity suite runs the draft case on both legs.
- Resolved: a draft can be marked ready without a push, which is why the waiver rides on a variable only the pre-push adapter sets, never on the event; AC4 pins both.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1: on a draft PR (`SDD_PR_DRAFT=true`) a closing reference to an issue whose spec is active exits 0 with a `[WARN] SDD archive-on-merge` that names the spec and `dotf spec archive <id>`, from the script under both shells and from `dotf gate spec`.
- [x] AC2: with `SDD_PR_DRAFT=false` or unset the same PR is refused as before (exit 1, `[FAIL] SDD archive-on-merge violation`), and a draft with a `skip-archive` label and no rationale is still refused.
- [x] AC3: the pre-push adapter hands the gate `SDD_PR_DRAFT=true` for a draft PR, `false` for a ready one or an answer without the field, and leaves it unset when there is no PR to resolve; its `--jq` expression carries `draft`.
- [x] AC4: CI keeps the refusal: `scripts/spec-gate-pr.sh` does not mention `SDD_PR_DRAFT`, and `dotf gate spec` on an event with `draft: true` and no variable still fails.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: none; BUG-061 (`scripts/spec-gate-prepush.sh`) and CI-018 (`dotf gate spec` parity) are the two decisions this sits between
- Related lessons: `docs/lessons/lesson-590-a-merge-time-invariant-checked-at-push-time-blocks-the-fix-it-needs.md`, lesson 193 (why "weaker locally, CI catches it" needs the two tiers to differ on an input CI never has)
