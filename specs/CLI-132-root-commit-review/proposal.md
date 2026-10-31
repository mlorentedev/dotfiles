---
id: "CLI-132-root-commit-review"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-10-30"
issue: "mlorentedev/dotfiles#2566"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, review, adversarial-review]
template_version: "1.0"
wip_override: "P1: unblocks an active review of a spec born in its repository's root commit; no other active spec provides the capability (10 active, limit 10, 2026-10-30)"
---

# CLI-132: Root-commit review

## Why

<!-- from issue #2566: CLI-132: Support explicit initial reviews for root-commit specs -->

`dotf spec review` refuses a spec that was first added, under any name, in its
repository's root commit: no commit precedes the work, so `ResolveReviewBase`
returns no base (BUG-108, #1829). That refusal is correct by default, but there
is no supported way to review such a spec at all. A project whose
implementation and spec were committed together as its first commit cannot get
an independent review, cannot archive, and has no path forward that does not
rewrite history, guess a base, or hand-author provenance.

## What

- `dotf spec review <id> --initial` reviews a root-commit spec from git's empty
  tree: the scope is `git diff <empty-tree> HEAD`, which holds the root
  commit's initial implementation and every later correction.
- `--initial` is accepted only when the resolver's refusal is specifically
  "first added in the root commit", including after renames. An ordinary spec,
  an uncommitted spec, and an untraceable or ambiguous move are refused before
  any reviewer starts. Without `--initial`, the default refusal is unchanged
  and names the flag when it would apply.
- The launcher records the scope in `review-request.json` as
  `scope: "initial"` with `root_sha` and `base_tree` (no `base_sha`), states it
  in the reviewer prompt, and prints it on `--dry-run`. The prompt uses a
  two-dot diff because the baseline is a tree, not a commit.
- `dotf spec archive` refuses a request with an unknown scope, or an initial
  scope missing its root commit or baseline tree. Pool pinning, signing slots,
  contract digests, and freshness checks are unchanged.

## Out of scope

- An explicit `--base <commit>` flag or any guessed base.
- Pathspec limits on the initial scope.
- History rewriting, automatic whole-repository review without the flag, or
  weakening the rename guards.
- Changes to any consuming project.

## Risks / open questions

- **Resolved:** the initial scope is the whole repository at HEAD. For a
  root-commit spec that is the initial implementation plus its corrections;
  a project whose root commit also holds unrelated code gets a larger review,
  which the prompt states rather than hides.
- **Resolved:** the empty tree is computed with `git hash-object -t tree`, not
  hard-coded, so SHA-256 repositories get their own empty tree.
- **Resolved:** archive freshness is decided by contract digests (SDD-042),
  which never read the base, so the initial scope cannot weaken it.
- A repository with several root commits records the root that added the spec;
  the empty-tree diff still covers everything reachable from HEAD.

## Acceptance criteria

- [ ] **AC1:** `ResolveReviewBaseDetail` classifies a spec as resolved, not
  committed, first added in the root commit (with that commit), or
  untraceable. A direct root-commit spec and a spec renamed out of the root
  commit both report the root commit; ordinary specs keep their current base.
- [ ] **AC2:** `dotf spec review <id> --initial` launches only for a
  root-commit spec. It refuses an ordinary, uncommitted, or untraceable spec
  before writing a request or starting a reviewer. Without `--initial`, a
  root-commit spec is still refused and the message names `--initial`.
- [ ] **AC3:** the initial scope covers the root commit's files and every later
  correction. The prompt states the empty tree, the root commit, and the
  two-dot diff, and never a triple-dot diff from a tree.
- [ ] **AC4:** `review-request.json` records `scope`, `root_sha`, and
  `base_tree` for an initial launch, with `base_sha` absent. `--dry-run`
  prints the scope and writes nothing.
- [ ] **AC5:** archive refuses a request with an unknown `scope` or an initial
  scope missing `root_sha` or `base_tree`; an ordinary request and a pre-scope
  request still pass the provenance check.

## References

- Bitácora: `mlorentedev/dotfiles#2566`
- Prior work: BUG-108 (`mlorentedev/dotfiles#1829`), SDD-042, HARNESS-112
- Code: `cli/internal/spec/review_request.go`,
  `cli/internal/spec/review_launch.go`, `cli/internal/spec/review.go`,
  `cli/internal/cmd/spec.go`
- Pattern: `00_meta/patterns/pattern-verification-fails-toward-unproven.md`
