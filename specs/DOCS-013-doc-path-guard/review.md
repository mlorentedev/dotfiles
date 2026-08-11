---
spec: "DOCS-013-doc-path-guard"
verdict: "FAIL"
reviewed_sha: "f91a08d7c9936773911a1ede1e306e4e359f074e"
reviewer: "claude-sonnet-5"
date: "2026-08-11"
---

## Adversarial review

**Scope**: DOCS-013-doc-path-guard, re-review at a new commit. No PR is open; diffed directly
against `main` (`6a40da3`) with `git -C /home/manu/Projects/dotfiles-wt-r2 diff main...HEAD`.
`main` already carries the first half of this feature (merged as #922); this branch
(`fix/docs-013-review-findings`, head `f91a08d7c`) is the fix pass responding to the prior
`review.md` (verdict FAIL at `d6681a6`, one Major + six Minors) plus CodeRabbit findings on #922.

**Sources**: full diff (`scripts/check-doc-paths.sh`, `.claude/CLAUDE.md`, `README.md`,
`tests/check-doc-paths.bats`, `specs/DOCS-013-doc-path-guard/{verification.md,review.md}`),
the spec triad, a from-source `cli/cmd/dotf` build (`go build ./cmd/dotf`), the prior `review.md`,
and direct reproduction/mutation of the script and its tests in the worktree (all mutations
reverted; worktree confirmed clean before writing this file).

### Spec and task alignment

All 7 acceptance criteria in `proposal.md` remain independently reproducible at this sha
(`check-doc-paths.sh` OK on all claimed files, zero false positives on `AGENTS.md`, no
`Projects/knowledge` literal, `bats tests/check-doc-paths.bats` → 11/11). The round-1 Major
(zsh-broken `. versions.conf` in the golangci-lint pin) is genuinely fixed: `zsh -c 'echo
"v$(. ./versions.conf; echo "$GOLANGCI_LINT_VERSION")"'` → `v2.12.2`, reproduced directly. All
six round-1 Minors were checked and are applied as claimed: the "live claim" callout is scoped to
slash-containing paths, `versionMatches`/`matchPin` wording matches the actual Go source
(`cli/internal/doctor/checks_tools.go:96`, `checks_deploy.go:416`, called from
`checks_golangci.go:63`), the shebang is `#!/usr/bin/env bash`, both `setup-macos.sh` mentions in
`README.md` are de-backticked, and case 6's defense-in-depth status is now documented in both the
script and the test comment. CodeRabbit's three findings (README's stale "auto-loaded at login" /
"21 skills" / "316 tests", the stale basename-resolution script header, the path-traversal false
negative) are also fixed and independently verified: `harness/skills/` has exactly 37 entries,
`grep -c '^@test' tests/*.bats` sums to 1209 (README's "1200+" holds), and `dotf secrets --help`
from a from-source build lists exactly the nine subcommands the rewritten Secrets System section
names. A full `bats tests/*.bats` run reproduces **1208 passed, 1 failed** of 1209 — the one
failure is `not ok 405 converges over a running dotf: a live binary in dest is replaced, not
refused` (BUG-054/#807), the same pre-existing, disclosed, unrelated failure the prior review
already traced to `main`; no new regression.

Where this stops short: the fixes for round 1 introduced their own defects, and the file-set this
PR claims to have made complete (six/eight "instruction files") is not actually complete. None of
these are in the prior review's six Minors or one Major — this pass found them independently, by
reproducing the round-2 verification claims rather than reading them.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | `check-doc-paths.sh` traversal fix (lines 134-143) | The `..`-escape check runs *before* `is_repo_rooted` and fires on **any** backticked token containing a `/../` component, even one whose first segment is not a real top-level repo entry. This reintroduces the exact false-positive failure mode the guard's own design section warns against ("A guard with false positives is worse than none — it gets bypassed"): a token that should be silently ignored by construction (non-rooted, e.g. describing an unrelated/hypothetical path) instead fails the guard with "path escapes the repo root". | Reproduced: `printf 'See \`not-a-real-toplevel-dir/../other/thing.md\` for context.\n' > doc.md; ./scripts/check-doc-paths.sh doc.md` → exit 1, `path escapes the repo root: not-a-real-toplevel-dir/../other/thing.md`. Also reproduced for a token that nets back inside the repo (`scripts/../scripts/utils.sh` → also rejected, which is at least consistent, but the non-rooted case is the real defect). | UNTESTED — the only escape test (`check-doc-paths: rejects a token that escapes the repo root [#916]`) uses `scripts/../../dotfiles/README.md`, where `scripts` **is** a real top-level entry, so it never exercises the non-rooted branch. | code — move the `..`-rejection after (or fold it into) the `is_repo_rooted` gate so a token with a fictitious first segment is ignored, not flagged. While there: the script header lists "contains no `..` component" among the conditions for a token being *checked*, which reads as "silently ignored" like the other bullets in that list — the implementation actually rejects loudly, so the header should say so |
| Major | REAL | `tests/check-doc-paths.bats` — zsh-sourcing regression test (line 132) | The regex `^[^#]*[^./a-zA-Z0-9_-]\. [a-zA-Z_][a-zA-Z0-9_.-]*\.(conf\|sh\|zsh)` requires a delimiter character immediately before `. file`. A source line at true start-of-line — flush-left inside a ` ```bash ` fence, exactly the style used throughout this file's own existing code blocks, and exactly the form the original round-1 bug took — has nothing before the `.`, so the regex never matches it. The test protects an inline-backticked mention (where the leading backtick supplies the delimiter) but not the live, copy-pasteable command form it was written to guard. | Reproduced twice by mutation, both reverted cleanly: (1) inserted a flush-left `. versions.conf` line into `.claude/CLAUDE.md`'s own Verification-Commands fence — `bats tests/check-doc-paths.bats` stayed 11/11 green. (2) Same insertion into `README.md`'s install snippet (`cd ~/dotfiles-repo` / `. versions.conf` / `./setup-linux.sh`) — also stayed green. This directly contradicts `verification.md`'s Round-2 claim: *"Mutation: a bare `. versions.conf` added to a live README code block turns case 11 red; removing it turns it green again."* That claim does not reproduce for the realistic flush-left form. | Named test exists (`check-doc-paths: no instruction file sources a config without ./ [#916]`) but does not discriminate the case it names — effectively UNTESTED for the shape that matters | tests — broaden the regex's leading alternative to also match true start-of-line, e.g. `(^\|[^./a-zA-Z0-9_-])\. `; then re-verify the verification.md claim before trusting it |
| Major | REAL | Governed file-set completeness (`instruction_files()` in `tests/check-doc-paths.bats`, and `verification.md`'s "six instruction files" claim) | `cli/AGENTS.md` and `ai/hermes/AGENTS.md` are genuine, pre-existing instruction files — each opens with agent-directed meta-instructions ("Nearest-file instructions for the Go CLI…", "Target agent: Hermes Agent…") and each fits the test file's own stated governance criterion ("files that tell an agent what to do") verbatim. Neither is in `instruction_files()` (pre-existing six-file list, inherited unchanged from #922 — not new in this diff) nor in the *new* zsh-sourcing regression test's three-file list (`.claude/CLAUDE.md`, `AGENTS.md`, `README.md` only — half of the set the suite already governs). `check-doc-paths.sh` is not wired into any CI workflow, tracked git hook (`git-hooks/{pre-commit,pre-push,commit-msg,prepare-commit-msg,post-checkout}`), or `.github/hooks/` script — checked all three locations, no hits beyond the bats file and the script's own usage text. So these two files currently have **zero** protection against the exact stale-path regression class #916 exists to catch, despite `verification.md` and the "Editing this file" callout implying the convention now covers instruction files uniformly. | `git show main:cli/AGENTS.md`, `git show main:ai/hermes/AGENTS.md` — both predate this branch. `./scripts/check-doc-paths.sh cli/AGENTS.md ai/hermes/AGENTS.md` (also run under real `zsh`, not just `zsh -n`) → both currently `OK` (dormant, not a live failure today), but nothing re-checks them going forward. | UNTESTED | tests — add both files to `instruction_files()` and to the zsh-sourcing regex's file list |
| Minor | REAL | `.claude/CLAUDE.md` self-consistency (prohibited-patterns table, lines ~9-24) | This PR appended a new table row (`. file` bare-sourcing) as the table's new last row, whose own "Why" column says "**Fails silently.**" — but the summary blockquote immediately below still says "The last two rows fail **silently**", now off by one: three consecutive rows are silent-failure cases, not two. | Read directly; `sed -n '9,24p' .claude/CLAUDE.md`. | UNTESTED | docs — "last two rows" → "last three rows" |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All 7 ACs and the round-1 Major reproduce as fixed, but the traversal fix and the file-set claim both have substantial, reproducible negative-path gaps of their own. |
| Verification       | C | Most claims (version pin, `versionMatches`/`matchPin`, skill/test counts, `dotf secrets` interface, full-suite regression) independently reproduced exactly — but one specific, load-bearing Round-2 mutation claim does not reproduce, which is worse than an unverifiable claim because it was asserted as tested. |
| Scope              | B | Diff matches the fix-pass proposal (round-1 findings + CodeRabbit findings); no unrelated changes. |
| Reliability        | B | The guard's core matching logic is sound and re-verified by actually *executing* (not just parsing) under both bash and zsh — including the traversal repros — with identical output; the new traversal branch's false-positive class is the one soft spot. |
| Maintainability    | B | Well-commented, each defensive branch explains its own rationale; the off-by-one blockquote is a small, isolated smell. |
| Handoff-readiness  | C | Spec triad present and mostly honest, but `verification.md`'s Round-2 evidence contains a claim that does not hold up under reproduction, which is exactly the failure mode a handoff reader can't detect without re-running it themselves. |

### Verdict
FAIL

Three Major findings, each independently reproduced in this session (not inherited from the prior
review, which did not find any of them): a new false-positive class in the traversal fix, a
regression test that misses the realistic form of the bug it exists to catch — contradicting
`verification.md`'s own mutation-test claim — and two known instruction files silently excluded
from the only enforcement path this guard has. Per the skill's aggregation rule, any one Major
forces FAIL regardless of the otherwise-solid rubric; three do so more firmly. The round-1 Major
and all six round-1 Minors, plus all three CodeRabbit findings, are genuinely fixed and were
independently reproduced rather than taken on faith.

### Recommended next steps (before archive)

1. **Required to flip to PASS**: fix the traversal-check ordering in `scripts/check-doc-paths.sh`
   so `..`-rejection only fires on tokens that are already `is_repo_rooted`. Re-verify: the
   `not-a-real-toplevel-dir/../other/thing.md` reproduction above must exit 0 (silently ignored),
   while the existing `scripts/../../dotfiles/README.md` case must still exit 1.
2. **Required to flip to PASS**: broaden the zsh-sourcing regression regex in
   `tests/check-doc-paths.bats` to catch a flush-left `. file` line, and re-run the exact
   reproduction in this review (insert `. versions.conf` as a standalone line in a `README.md`
   bash fence, confirm the test goes red, remove it, confirm green) before trusting
   `verification.md`'s claim again.
3. **Required to flip to PASS**: add `cli/AGENTS.md` and `ai/hermes/AGENTS.md` to
   `instruction_files()` (and to the zsh-sourcing test's file list), and confirm both still pass
   `check-doc-paths.sh` cleanly (they do today).
4. Optional, cheap: fix `.claude/CLAUDE.md`'s "last two rows" → "last three rows" in the
   prohibited-patterns blockquote.
5. After 1-3 land, get a **fresh** review at the new sha — a Major cannot be waived by re-reading
   the same commit, and this file's own findings were only found by re-running the round-2
   verification claims rather than reading them.

`dotf spec archive` is **not advisable** in the current state — verdict is FAIL, and
`cli/internal/spec/review.go`'s `Verdict.Blocks()` refuses to archive on anything other than PASS
or PASS-WITH-GAPS.
