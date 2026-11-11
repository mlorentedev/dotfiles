---
spec: "CLI-136-ci-secrets-from-workflows"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "c42cb3dba5d642ea9d6f1495ba7726e8691276f3"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-11-10"
---

## Adversarial review

**Scope**: `CLI-136-ci-secrets-from-workflows` (issue mlorentedev/dotfiles#3290)
**Sources**: `specs/CLI-136-ci-secrets-from-workflows/{proposal,tasks,verification,features}.md`;
`git diff fd6bd4d6ba121005ade422ca26b5f3453981c0ca...HEAD` (14 files, +1275);
`git log --first-parent --no-merges fd6bd4d6ba121005ade422ca26b5f3453981c0ca..HEAD` (5 commits on branch).

### Verified by running (this session)

| Check | Command | Result |
|---|---|---|
| Build / vet | `cd cli && go vet ./internal/secrets/... ./internal/cmd/...` | clean, exit 0 |
| Format | `cd cli && gofmt -l internal/secrets/ internal/cmd/` | clean, no output |
| Unit tests (secrets) | `cd cli && go test ./internal/secrets -run 'TestWorkflow\|TestInherited\|TestCallee\|TestCheckRepo\|TestDiffReads' -count=1` | all 10 tests passed (0.162s), exit 0 |
| Unit tests (cmd) | `cd cli && go test ./internal/cmd -run 'TestRunCICheck' -count=1` | both tests passed (0.179s), exit 0 |
| CLI command help | `cd cli && go run ./cmd/dotf secrets ci-check --help` | usage and flags printed, exit 0 |
| Linter | `cd cli && golangci-lint run ./internal/secrets/... ./internal/cmd/...` | `0 issues.`, exit 0 |
| Contract digests | `dotf spec digest CLI-136-ci-secrets-from-workflows` | proposal `0480ad…`, tasks `a17d9a…`, features `8d2643…`; identical to `review-request.json` |
| Round 1 F1 verification | `cd cli && go test ./internal/secrets -run 'TestWorkflowReads$' -count=1` | verified YAML comments (`# F: ${{ secrets.IN_A_COMMENT }}` and `# ${{ secrets.TRAILING_COMMENT }}`) are ignored |
| Round 1 F8 verification | check `verification.md` and disk | `docs/lessons/lesson-577-an-optional-secret-on-a-reusable-workflow-is-the-one-that-goes-missing-silently.md` exists and other candidates dispositioned |
| Round 1 F2 verification | `cd cli && go test ./internal/cmd -run 'TestRunCICheckNamesTheRepositoryOnce'` | repo error output names repo once (`o/broken: ERROR list workflows: boom`), exit 0 |
| Round 1 F3 verification | `cd cli && go test ./internal/secrets -run 'TestCheckRepoNamesAreCaseInsensitive'` | case-insensitivity in both directions passes, exit 0 |
| Round 1 F5 verification | `cd cli && go test ./internal/secrets -run 'TestWorkflowFilesReadEveryDocument'` | all YAML documents read for secrets and inherited calls, exit 0 |
| Mutation: `GITHUB_TOKEN` exclusion | mutate line 94 in `ciwf.go`, run `TestWorkflowReads` | **killed** (`names = [… GITHUB_TOKEN …]`); reverted |
| Mutation: case insensitivity | mutate line 94 in `ciwf.go` (omit `ToUpper`), run `TestCheckRepoNamesAreCaseInsensitive` | **killed** (`same name in two cases must match`); reverted |
| Probe: bare expression in `if:` | regex probe on `exprRe` with `if: secrets.FOO != ''` | `exprRe` fails to match bare expressions without `${{ }}` — see F9 |
| Probe: string literal with `'secrets'` | regex probe on `exprReads` with `contains(..., 'secrets')` | matches `'secrets'` and treats as `unknown` read — see F10 |

### Spec and task alignment

- **AC1–AC5 alignment**:
  - AC1 (report names, no values, non-zero on failure): Covered by `TestRunCICheckReportsNamesAndFails` and `TestCheckRepo`. Verified.
  - AC2 (inherited workflow missing undeclared secrets; explicit mappings report caller name): Covered by `TestInheritedCalls`, `TestWorkflowReads`, `TestCheckRepo`. Verified.
  - AC3 (declared but unread is excess): Covered by `TestCheckRepo` (`OLD_TOKEN`). Verified.
  - AC4 (whole-context read is unknown, never pass): Covered by `TestWorkflowReadsUnknown`, `TestCheckRepoCleanAndUnknownOnly`. Verified.
  - AC5 (absent `required: false` callee secret is missing and marked optional): Covered by `TestCalleeSecrets`, `TestCheckRepo`. Verified.
- **Round 1 findings status**:
  - F1 (Major REAL, commented YAML expressions read as secrets): RESOLVED in commit `c42cb3db` by parsing AST scalars with `decodeDocs` + `eachScalar`.
  - F8 (Major REAL, unanswered promotion candidates in `verification.md` blocking archive pre-flight): RESOLVED. All three items answered, lesson-577 authored and committed.
  - F2 (Minor REAL, error double-prefix): RESOLVED in commit `c42cb3db` and tested by `TestRunCICheckNamesTheRepositoryOnce`.
  - F3 (Minor THEORETICAL, case-sensitive secret names): RESOLVED in commit `c42cb3db` (names upper-cased) and tested by `TestCheckRepoNamesAreCaseInsensitive`.
  - F4 (Minor SPECULATIVE, `required: yes` YAML 1.1): Declined in `verification.md` with rationale (YAML 1.2 vs 1.1; fleet standards; marking only).
  - F5 (Minor REAL, multi-document workflows): RESOLVED in commit `c42cb3db` with `decodeDocs` and tested by `TestWorkflowFilesReadEveryDocument`.
  - F6 (Minor REAL, `tasks.md` `gh repo list` drift): Acknowledged and dispositioned in `verification.md` without modifying contract-set files.
  - F7 (Question REAL, unknown-only exit 0): Confirmed and dispositioned in `verification.md`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | correctness (static scan) | F9 — Secret references in bare `if:` conditionals (omitting `${{ }}`) are not detected. In GitHub Actions, expressions inside `if:` do not require `${{ }}` syntax (and `actionlint` warns against wrapping them). `workflowReads` strictly matches `exprRe` (`\$\{\{(.*?)\}\}`). A workflow writing `if: secrets.DEPLOY_KEY != ''` or `if: !secrets.MY_TOKEN` without `${{ }}` is not detected as a read; if declared, it is reported as `excess` and fails; if undeclared, it is silently missed | probe on `exprRe.FindAllStringSubmatch("secrets.UNWRAPPED != ''", -1)` yields 0 matches. Idiomatic Actions syntax in `.github/workflows/` across this fleet heavily uses bare `if:` conditions | UNTESTED (all tests currently wrap `if:` expressions in `${{ }}`) | code + tests (inspect `if:` mapping node scalars directly through `exprReads` or support unwrapped expressions) |
| Minor | THEORETICAL | correctness (false unknown) | F10 — String literals containing `'secrets'` inside an expression (e.g. `${{ contains(github.event.comment.body, 'secrets') }}`) are falsely reported as `unknown` reads. `exprReads` finds `\bsecrets\b` across the expression body without distinguishing string literals (`'...'`) from context access. When `literalRe` fails to match member access after `'secrets'`, it treats the expression as a computed read and appends to `rep.Unknown` | probe on `exprReads("contains(github.event.comment.body, 'secrets')", nil, nil)` yields `unknown=["contains(github.event.comment.body, 'secrets')"]` | UNTESTED | code + tests (`exprReads` should ignore matches inside quoted string literals) |
| Minor | SPECULATIVE | correctness (YAML) | F11 — Reusable workflow `on.workflow_call.secrets.*.required` accepts YAML 1.2 boolean scalars `True` / `TRUE`, but `calleeSecrets` compares `req.Value == "true"`. `yaml.v3` decodes `required: True` with `Tag: !!bool` and `Value: "True"`, so a callee using TitleCase `True` will be treated as `required: false` | probe on `yaml.v3`: `doc := "required: True"` -> `Tag: !!bool, Value: "True"` | UNTESTED | code + tests (`strings.EqualFold(req.Value, "true")`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All acceptance criteria verified and round 1 bugs (comments, case-sensitivity) resolved; bare `if:` syntax gap (F9) remains theoretical for the fleet |
| Verification       | B | Reproducible commands and tests for all criteria; independent mutations killed; fleet live run documented in verification.md |
| Scope              | A | Diff matches proposal exactly; single package and command with reader seam, zero unrelated changes |
| Reliability        | B | Unreadable callees fail with clear error messages, callees are cached per ref, duplicate error prefixes fixed |
| Maintainability    | B | Clean AST traversal, small functions under 40 lines, cyclomatic complexity under 10 across all functions |
| Handoff-readiness  | B | Lesson-577 committed and linked, all promotion candidates answered, decisions and prior round findings dispositioned |

### Verdict
PASS-WITH-GAPS

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is closed and must NOT be edited (doing so would invalidate this passing verdict). Recommended dispositions for the implementer:

1. **Follow-up ticket / PR (code + tests)**:
   - Handle bare expressions in `if:` conditionals (F9) by checking `if` nodes in YAML workflows or scanning for unwrapped `secrets.` references.
   - Ignore string literals in `exprReads` (F10) to avoid false-positive `unknown` reports on expressions containing `'secrets'`.
   - Use `strings.EqualFold(req.Value, "true")` for `required` flags in `calleeSecrets` (F11).
2. **Archive gate**:
   - `dotf spec archive CLI-136-ci-secrets-from-workflows` is **ADVISABLE** in this state. The two round-1 blocking findings (F1 and F8) are resolved, and the new findings are theoretical/speculative gaps suitable for follow-up maintenance.
