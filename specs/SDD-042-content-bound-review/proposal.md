---
id: "SDD-042-content-bound-review"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-09-23"
issue: "mlorentedev/dotfiles#1566"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, sdd, archive-gate, review]
template_version: "1.0"
---

# SDD-042-content-bound-review

> Epic #1625 (HARNESS-138), item **W1.2**. Also covers #970 (BUG-079), part 2
> of #998, and the bypass-record precondition of W1.4 (#1626). Acceptance
> reference: AC-1.2, plus AC-1.4's `review_bypass:` clause.

## Why

`dotf spec archive` decides whether a passing review still describes the
spec's contract by comparing `reviewed_sha` against `HEAD`, and it first asks
`git cat-file -e`, which checks whether the commit **object exists
locally**. This repository squash-merges. That orphans the reviewed commit on
every spec, so the answer depends on garbage-collection state, not on the
review. On the machine that did the work the check passes, and on a fresh
clone or in CI it refuses and blames a rebase that never happened (#1566).
The same model breaks the documented squash-rebuild pattern (#970), and it
treats a reviewer-demanded checkbox tick as a contract change, so the review
invalidates itself (#998 part 2).

The escape from all three is `--force-without-review`. That flag leaves
**no durable trace** (`archive.go:272`), and its name is false whenever what
it overrides is freshness rather than the review's existence. The W1.4 sweep
bans the bypass flags, and a ban that leaves no trace can only be checked by
trusting commit messages.

## What

1. **The launcher records what the reviewer read, by content.**
   `review-request.json` gains `contract_digests`, a SHA-256 per contract
   file (`proposal.md`, `tasks.md`, `features.json`) taken from disk at launch.
   The launcher writes it; the reviewed party never does.
   - **The digests are taken over normalised content, so bookkeeping does not
     count as contract drift.** Line endings are folded to LF (a Windows
     checkout). Checkbox markers are folded to unchecked, so ticking
     `- [ ]` → `- [x]` is progress, not a new criterion (#998 part 2).
     `features.json` is compared with each feature's `state` and `evidence`
     blanked, because those are written by the harness after the review
     (#1087 AC template, GUARD-010).
2. **Freshness is decided by content when digests exist.** If the request
   carries `contract_digests`, archive recomputes them from disk and refuses
   only when one differs, naming the file. No git history is consulted, so a
   squash, a rebase, a merge commit, a branch rebuild or a fresh clone makes no
   difference (#1566, #970). An uncommitted edit that changes a criterion is
   still caught, because the comparison is against disk.
3. **Legacy reviews without digests keep the SHA check, and it tells the
   truth.** "The reviewed commit is absent from this clone" and "a contract
   file changed" are reported separately. The absent case still refuses
   (fail closed), and it names the cause: a squash, rebase or fresh clone
   discards the object, and the review predates content digests.
4. **A bypass is recorded.** `--force-without-review` and
   `--force-with-drafts` now require `--reason "<text>"`. The skipped check
   still runs, and what it *would* have refused is captured. The archived
   `proposal.md` then gains one frontmatter line:

   ```yaml
   review_bypass: "<flags>; overrode: <what the gate would have refused, or nothing>; reason: <reason>; date: <YYYY-MM-DD>"
   ```

   The line records what was overridden, not the flag's name, which answers
   #998's complaint that `--force-without-review` asserts something false.
5. **Refusal messages name the recovery path, never a bypass flag** ([QW]
   R-2, the half the epic had not yet carried over). The flags stay documented
   in `dotf spec archive --help`, together with the fact that using them is
   recorded.
6. **The skill's documented signature** (`/spec archive … [--force-*]`, in
   `00_meta/skills/spec/SKILL.md` and its `harness/skills/` render) states
   that `--reason` is required and that the bypass is recorded.

## Out of scope

- **Binding the review to code bytes** (`reviewed_tree`, #1153). That is W3.6.
  The digest mechanism here is built so W3.6 can widen its file set, but this
  spec digests the contract files only. AC-1.2's "post-review code change" is
  read here as a post-review **contract** change; a code change is W3.6's
  acceptance.
- A reviewer-declared list of files its findings may touch. Checkbox
  normalisation covers the measured case (#998: "tick the boxes"), and a
  declared list is a `review.md` contract change with its own design.
- Retroactively adding digests to the reviews already on disk. They stay
  under the legacy path.
- The W1.4 sweep itself (#1626).

## Risks / open questions

- **Normalisation too wide could hide a real change.** Only checkbox markers,
  line endings and the two harness-owned `features.json` fields are folded.
  A test asserts that editing an acceptance criterion's *text* still makes the
  review stale.
- **The digests are written by the launcher, and a hand-edited request could
  forge them.** This is the same trust boundary as `reviewed_sha` in
  `review-request.json` today (GUARD-005): drift protection between review and
  archive, not tamper evidence against the operator.
- **Refusal texts are asserted by existing tests.** They are updated in the
  same change, and every refusal is re-asserted *not* to name a bypass flag.
- **Contract change for operators.** A scripted `--force-*` without `--reason`
  now fails. It fails loudly and names the missing flag, and the flags were
  banned for the sweep anyway.

## Acceptance criteria

- [x] **AC1** — `gorun ./internal/spec/... 'Stale|Squash|Rebase'` passes with
  at least one test run. Real-git fixtures, each built from a review launched
  on a branch commit:
  - a squash landing on `main`, the branch deleted and its objects pruned, is **accepted**;
  - a rebase landing is **accepted**;
  - a merge-commit landing is **accepted**;
  - a post-review edit to an acceptance criterion's text is **refused**, naming `proposal.md`.
- [x] **AC2** — Ticking checkboxes in `proposal.md`/`tasks.md` and the harness
  setting `state`/`evidence` in `features.json` after the review leave it
  **fresh**. A CRLF-only difference leaves it fresh.
- [x] **AC3** — A legacy review (no `contract_digests`) whose `reviewed_sha` is
  absent from the object store is refused with a message naming the absent
  object and the missing digests, distinct from the "file changed" message.
- [x] **AC4** — `--force-without-review` or `--force-with-drafts` without
  `--reason` is refused. With `--reason`, the archived `proposal.md` carries
  exactly one `review_bypass:` line naming the flags, what the gate would have
  refused (or "nothing"), the reason and the date, and
  `grep -l '^review_bypass:'` finds it.
- [x] **AC5** — No refusal from the tag pre-flight or the review gate names
  `--force-without-review` or `--force-with-drafts`. A test asserts this over
  every refusal path.
- [ ] **AC6** — `dotf spec review` writes `contract_digests` into
  `review-request.json`, and this spec's own archive is decided by them.

## References

- Epic: mlorentedev/dotfiles#1625 §3 W1.2, the W1.4 precondition, §5.2
  AC-1.2/AC-1.4, and the addenda comment (refusal hygiene).
- #1566 (option 1 chosen), #970, #998 (part 2), GUARD-005 (review-request
  provenance), GUARD-010 (`features.json` state).
- Siblings in Wave 1: SDD-041 (#1630, `dotf spec audit`) and W1.3 (#1631,
  the exported `ReviewStateFiles` list that W3.6 will reuse).
