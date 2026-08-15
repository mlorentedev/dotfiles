---
id: HARNESS-071-reviewer-pool
verdict: FAIL
reviewed_sha: b24e105e78429b23d82a49f5dca5c0c08c2b4119
reviewer: agy/gemini-3.1-pro-high
date: 2026-08-14
---

# Adversarial Review: HARNESS-071-reviewer-pool

## Rubric Evaluation

1. **Scope & Spec Alignment:** High. The implementation faithfully executes the spec's intent: checking the reviewer pool on archive and launching the specified reviewer.
2. **Correctness:** Low. A critical logical flaw exists in how the reviewer pool's tracked history is queried.
3. **Failure Modes & Edge Cases:** Low. The system fails to correctly handle the "formal retirement" of the gate, leading to a permanent dead end, and it continues to trust a staleness check that ignores the working tree.
4. **Test Coverage & Traceability:** Medium. The test coverage is excellent for the happy path and explicit error states, but misses verifying the claim that the gate can be safely retired via commit, and misses the uncommitted changes edge case for staleness.
5. **Documentation & Readability:** High. The code is readable, self-documenting, and error messages are highly descriptive.
6. **Maintainability & Future-Proofing:** Medium. The explicit dependency injection and seams allow easy testing, but the reliance on history-crawling for the gate creates a rigid, unbreakable lock.

## Findings

### 1. Blocker — The reviewer pool gate cannot be formally retired, permanently bricking archives
**Reality/Test-Traceability**: REAL — UNTESTED
**Description**: The error message in `loadReviewerPoolEntries` instructs the user: *"if the gate is genuinely being retired, remove it in a commit that says so"*. However, doing exactly this will *still* cause `dotf spec archive` to fail permanently. The function `poolWasTracked` uses `git log -1 --format=%H -- harness/reviewer-pool.json` which searches the entire history. Even if the file was removed in the latest commit, `git log` successfully returns the hash of the commit that removed it. Thus, `poolWasTracked` returns `true`, `os.IsNotExist` is true, and the user is presented with the exact same error blocking the archive, no matter how many commits are made.
**Impact**: If a repository decides to opt-out of the adversarial review gate by deleting `harness/reviewer-pool.json` and committing the change, `dotf spec archive` will fail permanently for all future specs. The only workaround is to use `--force-without-review` every single time.
**Remediation**: `poolWasTracked` must differentiate between a file that is "missing from the working tree but exists in HEAD" (meaning it was accidentally deleted) versus "not existing in HEAD" (meaning it was formally removed). Use `git ls-tree HEAD -- harness/reviewer-pool.json` instead of searching the whole git history. Add a test in `reviewer_pool_test.go` covering the "formally retired" (committed removal) scenario.

### 2. Major — Staleness check ignores dirty working tree changes to contract files
**Reality/Test-Traceability**: REAL — UNTESTED
**Description**: The `gitStaleness.Stale` implementation checks for staleness using `git log --format=%H SHA..HEAD -- contractFiles`. This only inspects *committed* history. If a user modifies `proposal.md` after the review passes but *does not commit* the changes, `gitStaleness` returns false (not stale), allowing the archive to proceed with unreviewed contract modifications.
**Impact**: This defeats the entire purpose of the staleness check. A developer can get a passing review, edit the acceptance criteria in `proposal.md`, and immediately run `dotf spec archive` which will accept the change as fresh, violating the SDD contract. While this existed before HARNESS-071, this PR modifies the `checkReviewGate` and relies heavily on its integrity.
**Remediation**: `gitStaleness.Stale` must be updated to also check for uncommitted modifications to the contract files in the working directory (e.g., using `git diff --name-only HEAD -- contractFiles` or `git status --porcelain`). Add a test covering the dirty working tree scenario.

### 3. Minor — `agy` runner requires positional arguments to follow flags correctly
**Reality/Test-Traceability**: REAL — TESTED
**Description**: In `ReviewerCommand`, the arguments for `agy` are constructed using `--print prompt` at the very end. While currently correct (as `agy` seems to handle `--print` consuming the next argument as its value), it leaves the invocation vulnerable to cli parser ambiguity if `--print` was ever intended as a boolean flag followed by a positional argument. The test `TestReviewerCommandGivesAgyThePromptAsThePrintValue` explicitly verifies this shape, so it is safe for now.
**Impact**: Minimal. The test acts as a safeguard.
**Remediation**: None required immediately, but could be refactored to `--print=` if `agy`'s CLI parser supports it to remove ambiguity.

## Conclusion
The system successfully implements the runner logic and correctly enforces the model pool, but the logic surrounding the gate's lifecycle is flawed. The inability to formally retire the gate is a blocking issue that traps the repository in a broken state. Additionally, the pre-existing staleness check gap undermines the integrity of the gate entirely. Address the `poolWasTracked` logic to allow true retirement and fix the dirty tree staleness check to safely deploy this gate.

*(Note: AC7 in the verification doc is now met, as this actual review run proves the `agy` Gemini arm works.)*
