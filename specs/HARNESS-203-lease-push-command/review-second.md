---
spec: "HARNESS-203-lease-push-command"
verdict: "FAIL"
reviewed_sha: "2734d2edc6e304ea2f6d64a946069bfcb78b524e"
reviewer: "anthropic-review/claude-sonnet-5-5"
date: "2026-11-23"
---
## Adversarial review

**Scope**: HARNESS-203-lease-push-command (#3612), `git diff 31360ae3...HEAD`, the spec's 21 commits.
**Sources**: `proposal.md`, `tasks.md`, `features.json`, `verification.md`, `cli/internal/pushlease/pushlease.go`, `cli/internal/cmd/pr_push_lease.go`, the tests beside them, `AGENTS.md`, ADR-066. I did not read `review.md` or its transcript. Contract digests from `dotf spec digest` equal those in `review-second-request.json`. No `[AGENT-DRAFT]` tags in the contract files.

### Spec and task alignment
- Every task box is ticked except the PR/closing box, which the spec says waits for the signatures. Diff matches the proposal; no scope creep outside the spec folder, AGENTS.md line, ADR, lesson and tests.
- Re-ran fresh: `go test ./internal/pushlease ./internal/cmd ./internal/claude -run 'Push|Prepare|Lease|GitEnv' -count=1` PASS (pushlease 79 s, cmd 88 s, claude 2 s); `go vet` clean.
- Re-ran one mutant: inverting the split-remote guard (`err != nil || push != fetch` to `&&`) fails `TestPrepare_RefusesASplitRemote` (push target's `feat/x` force-updated). The claim in `verification.md` holds. Tree restored.
- Probes against real bare repositories (in a `mktemp -d`, global git config isolated):
  - `remote.origin.mirror=true`: git itself dies (`--mirror can't be combined with refspecs`), nothing pushed. Held.
  - `remote.origin.push = refs/heads/other:refs/heads/main`: ignored, only `feat/x` moves. Held.
  - `remote.pushDefault` set to a path: refused as unknown remote. Held.
  - Remote's HEAD pointing at the branch (`other`): refused `default-branch`. Held.
  - Remote with **two `remote.origin.url` values: bypass (F1 below).**

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | REAL | split-remote guard (same class as the earlier F1) | A remote with several fetch URLs (`git config --add remote.origin.url <second>`) passes `resolveRemote`: `get-url --all` and `get-url --push --all` return the same two lines, so no `split-remote` refusal. `ls-remote` (default-branch and lease check) reads only the first URL, but `git push` pushes to every URL of the remote. Reproduced: R.git default `main`, R2.git default `feat/x` holding the `--expect` sha; `dotf pr push-lease --expect <sha>` exit 0, git printed `+ ... -> feat/x (forced update)` for both URLs, and R2's `feat/x`, its default branch, was force-updated with no default-branch check ever run against it. This is the exact failure `proposal.md` says the split-remote refusal exists to prevent ("every check reads the fetch URL and the push would write elsewhere"); the F1 fix compares the two lists but not that the push list is a single target. | Probe output: `R2: c33f335a... == HEAD`; `git remote get-url --all origin` printed two URLs, `--push --all` the same two. | UNTESTED. `TestPrepare_RefusesASplitRemote` has only the `pushurl` and `pushInsteadOf` rows; nothing sets `remote.<r>.url` twice. | code (refuse when `get-url --all` returns more than one URL, or check every URL), tests (a third row of `TestPrepare_RefusesASplitRemote`); spec (the "What" refusal list and AC2 name the case, ADR-066 §2) |
| Minor | REAL | command output | When the remote already equals HEAD and `--expect` equals HEAD, git says "Everything up-to-date", exit 0, and the command still prints `feat/x <sha>..<sha>`, offered as evidence of a push that did not happen. Observed in probe 1. | observed | UNTESTED | code (print a no-op marker) or accept; low risk |
| Minor | THEORETICAL | TOCTOU | The branch is read first and `HEAD` is rev-parsed later; a checkout in between pushes another branch's commit to the first branch's name. The lease still guards the remote. | code read, `Prepare` | UNTESTED | accept |
| Question | n/a | multi-URL scope | Is a multi-URL remote meant to be refused or checked per URL? Refusing mirrors the split-remote decision and is the cheaper fix. | | | owner |

Reviewed and not found wanting: the stale-lease pre-check, the union default-branch rule (reported and recorded), case-insensitive `main`/`master`, argv construction (no shell, remote cannot be option-like, branch name cannot start with `-`), `GIT_TERMINAL_PROMPT=0`, hooks not skipped, the settings-pinning test. Long-lived non-default branches and the missing timeout are owner-accepted or ticketed (#3617), and I do not re-litigate them.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Default-branch guard bypassed on a multi-URL remote; everything else held under probing. |
| Verification       | B | Real-git tests and recorded mutants, reproduced here; the multi-URL case is not in the table. |
| Scope              | A | Diff matches the proposal. |
| Reliability        | B | Refusals precede any push, errors surface; the no-op prints a misleading line. |
| Maintainability    | A | Small functions, stable codes, clear comments. |
| Handoff-readiness  | A | Spec, ADR, lesson, AGENTS.md and dispositions all present. |

### Verdict
FAIL

One REAL Major (reproduced) in the safety guard that is the point of the change. Per the severity x reality rule a REAL Major requires a fix and a re-review.

### Recommended next steps
- Code and tests (outside the contract set): refuse a remote with more than one URL in `resolveRemote` (code `split-remote` or a new one), add a test row that sets `remote.origin.url` twice, with a second bare repo whose HEAD is the branch, and assert its refs unchanged. Mutation: remove the check, the row must go red.
- Contract set (`proposal.md` What and AC2, `tasks.md`): name the multi-URL refusal beside the split-remote one; this changes the digests, so the re-review is the mechanism, as the skill states for a FAIL.
- ADR-066 and the `--help` text: add the case.
- Optional: print a distinct line when the push is a no-op.
- `dotf spec archive` is not advisable at this head.
