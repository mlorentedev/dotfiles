---
spec: "CLI-136-ci-secrets-from-workflows"
verdict: "FAIL"
reviewed_sha: "6d0a0a6aaa3edbe09e4554df6fa1ea2dc2542dd3"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-10"
---

## Adversarial review

**Scope**: `CLI-136-ci-secrets-from-workflows` (issue mlorentedev/dotfiles#3290)
**Sources**: `specs/CLI-136-ci-secrets-from-workflows/{proposal,tasks,verification,features}.md`;
`git diff fd6bd4d6ba121005ade422ca26b5f3453981c0ca...HEAD` (12 files, +1018);
`git log --first-parent --no-merges fd6bd4d6ba121005ade422ca26b5f3453981c0ca..HEAD` → the three spec
commits and no merge, so the range is exactly this change.

### Verified by running (this session)

| Check | Command | Result |
|---|---|---|
| Build/vet | `cd cli && go vet ./...` | exit 0 |
| Format | `gofmt -l internal/ cmd/` | no output |
| Full suite | `cd cli && go test ./... -count=1 -timeout 8m` | all packages `ok`, exit 0 (incl. `internal/secrets` 1.31s, `internal/cmd` 44.1s) |
| Command wired | `go run ./cmd/dotf secrets ci-check --help` | usage + `--repo strings` printed, exit 0 |
| Lint | `golangci-lint run ./internal/secrets/... ./internal/cmd/...` (v2.12.2, repo `.golangci.yml`) | `0 issues.`, exit 0 |
| Comment idiom in real workflows | `grep -rE '^\s*#.*\$\{\{' .github/workflows/` | two hits (`.github/workflows/cli.yml:136`, `.github/workflows/from-zero.yml:201`): real workflows in this fleet already carry `${{ }}` inside comments |
| Contract digests | `dotf spec digest CLI-136-ci-secrets-from-workflows` | identical to `review-request.json` `contract_digests` (proposal `0480ad…`, tasks `a17d9a…`, features `8d2643…`); contract untouched by me |
| Mutation: `GITHUB_TOKEN` exclusion removed | mutate `workflowReads`, `go test -run 'TestWorkflowReads$'` | **killed** (`names = [… GITHUB_TOKEN …]` vs want) — the verification.md mutation claim is credible; file restored, tree clean |
| Probe: comment containing an expression | temp `_test.go` in `internal/secrets`, then deleted | `names=[IN_A_COMMENT REAL]` — see F1 |
| Probe: lowercase reference | same | `missing=[{nan_api_key }] excess=[NAN_API_KEY]` — see F3 |
| Probe: `required: yes` | same | `map[A:false B:true]` — see F4 |
| Probe: multi-document workflow | same | only the first document's `secrets: inherit` call seen — F5 |
| Probe: error path + unknown-only exit | temp `_test.go` in `internal/cmd`, then deleted | `o/broken: ERROR o/broken: list workflows: boom`; unknown-only repo prints `UNKNOWN`, overall exit 0 — F2, F7 |

Not run (and therefore UNVERIFIED by this reviewer): the live fleet run over the operator's
repositories (recorded in `verification.md`; re-running it spends the operator's REST budget and prints
private repositories' secret *names*, which the implementer deliberately kept out of the record).

### Spec and task alignment

- AC1–AC5 each map to a named test, and each named test exists and passes: `TestRunCICheckReportsNamesAndFails`,
  `TestWorkflowReads`, `TestWorkflowReadsUnknown`, `TestInheritedCalls`, `TestCalleeSecrets`, `TestCheckRepo`,
  `TestCheckRepoCleanAndUnknownOnly`, `TestCheckRepoCalleeUnreadable`, `TestDiffReadsPrecedence`.
- `features.json`: five entries, one per AC, states `pending` (not `passing` with empty evidence) — the gating rule is
  respected. Verification commands resolve to real test names.
- Out-of-scope boundaries hold: no CI wiring, no `registry.yaml` edit, nothing about synced-vs-declared (#1596 is
  referenced, not implemented).
- The proposal's YAML 1.1 risk is pinned by `TestCalleeSecrets` ("bare `on:`"): confirmed present.
- The `secrets: inherit` + `required: false` decision (AC5) is implemented as described and captured as lesson-577.
- Drift: `tasks.md` says "the fleet from `gh repo list` when no `--repo` is given"; `proposal.md` says
  `gh repo list` would spend the GraphQL budget and mandates REST `user/repos?affiliation=owner`;
  `ciwf_gh.go` `Fleet()` uses REST. The task text contradicts both the proposal and the code (F6).
- `verification.md` "Promotion candidates" carries its three placeholder lines unanswered, although lesson-577 was
  in fact written; `dotf spec archive`'s third pre-flight refuses exactly that and no flag skips it (F8).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | correctness (static scan) | F1 — a `${{ secrets.X }}` inside a **YAML comment** is counted as a read, so a repository that comments out or documents an expression is reported `missing X` and the check exits non-zero. `workflowReads` matches raw file bytes, never the YAML structure. This also falsifies `verification.md`'s stated decision ("Only `${{ }}` expressions are scanned, so a comment … is never read as a secret reference") | probe on `ciwf.go` at reviewed sha: `# remember to add ${{ secrets.IN_A_COMMENT }}` → `names=[IN_A_COMMENT REAL]`. The trigger idiom is not hypothetical: this fleet's own workflows already put `${{ }}` in comments (`cli.yml:136`, `from-zero.yml:201`), so only the expression's contents have to be a secret for the false report; a commented-out `#  X: ${{ secrets.Y }}` line is the common case | UNTESTED (no test feeds a commented expression) | code + tests (strip comments or walk the YAML AST before extracting expressions; add a `TestWorkflowReads` row) and spec `verification.md` (the claim is wrong) |
| Major | REAL | handoff | F8 — `verification.md` "Promotion candidates" is unanswered (`<yes: path / no: reason>`), which `dotf spec archive` refuses with no override flag, so the spec cannot archive as it stands even though the promotion exists | `dotf spec archive --help`: "A third pre-flight reads verification.md's 'Promotion candidates' … No flag skips it"; the three lines are placeholders; lesson-577 exists and is the answer | UNTESTED (gate is outside the repo; behaviour observed from the CLI's own contract text) | spec — `verification.md` (excluded from the staleness check) |
| Minor | REAL | resilience | F2 — the per-repository error line double-prefixes the repo (`o/broken: ERROR o/broken: list workflows: boom`): `runCICheck` prefixes, and `collectReads` already did | probe output above | UNTESTED (`TestRunCICheckReportsNamesAndFails` covers ok/FAIL, not the error path) | code + tests |
| Minor | THEORETICAL | correctness | F3 — matching is case-sensitive, so `${{ secrets.nan_api_key }}` against a declared `NAN_API_KEY` reports a false `missing` **and** a false `excess` for the same credential. GitHub Actions secret names are case-insensitive (premise not verifiable offline in this run) | probe: `missing=[{nan_api_key }] excess=[NAN_API_KEY]` | UNTESTED | code + tests (`strings.EqualFold` normalisation, or a documented decision that names are case-sensitive) |
| Minor | SPECULATIVE | correctness | F4 — a callee declaring `required: yes` (YAML 1.1 boolean) is read as *not* required, so the name is reported `missing … (optional in …)` instead of plainly missing. Marking only; the name is still reported | probe: `map[A:false B:true]` | UNTESTED | code + tests |
| Minor | REAL | correctness | F5 — a multi-document workflow: only the first document's `secrets: inherit` calls are read (`yaml.Unmarshal` decodes one document), so later documents' callees are silently dropped rather than reported | probe: two documents → `got=[w.yml@v1]` only | UNTESTED | code + tests (or reject multi-document files loudly) |
| Minor | REAL | spec drift | F6 — `tasks.md` assigns the fleet to `gh repo list`; the proposal forbids it (GraphQL budget, #3228) and `Fleet()` uses REST. Contract-set file, so it is dispositioned in `verification.md`, not edited under this review | `tasks.md` Implementation bullet vs `proposal.md` "Fleet" vs `ciwf_gh.go:Fleet` | N/A (documentation) | spec — disposition in `verification.md` |
| Question | REAL | exit semantics | F7 — a repository whose only signal is an unknown read prints `UNKNOWN` and exits 0. That matches `features.json` f4 ("neither passed nor failed") and the AC4 label, but a CI gate consuming the exit status goes green knowing nothing about that repository. Confirm exit 0 is the intended contract | probe: `o/unknown: UNKNOWN` + `unknown …: toJSON(secrets)`, overall exit 0 | `TestCheckRepoCleanAndUnknownOnly` (asserts `Failing()` is false for unknown-only) | spec — confirm in `verification.md`; code only if the owner wants unknown to gate |

Notes on the code-level checklist: no injection surface found — `gh` is invoked through `exec.Command`
with discrete args and no shell, so `--repo`/path/ref cannot break out; only the operator's own
repositories are read. No secret values anywhere in the path (`SelectCI` returns names; `gh` reads
workflow files, never `actions/secrets`). No unbounded loops; one callee fetch per distinct
`repo/path@ref`, cached across repositories in one run. `gosec` on `exec.Command` is suppressed with
a reason, and the reason is accurate. No dead code observed.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Happy path and all five ACs verified, but two reproduced misreports (comment-sourced read F1; case-sensitive name comparison F3) mean the report can state a name is read when it is not, and can flag the same credential missing and excess at once |
| Verification       | B | Evidence names the tests, the commands and a mutation log (one mutation independently re-run here and killed; `golangci-lint` claim re-run clean), but one documented claim is false (comments) and the mutation entries omit their commands |
| Scope              | A | Diff is the proposal: one package, one command, a lesson and the runbook section; no CI wiring, no registry edit, nothing from the non-goals |
| Reliability        | B | Unreadable callee fails loudly naming it, callee cached, no-workflows repo handled as "none"; error text duplicated and a parse failure takes the whole repository down without isolating the offending file |
| Maintainability    | B | Small pure functions, documented regexes, node-walk for `on:` — readable; the byte-level regex scan is what F1 and F3 fall out of |
| Handoff-readiness  | C | Lesson-577 and the runbook section land, but `verification.md` cannot pass the archive's promotion pre-flight and `tasks.md` contradicts the proposal on fleet listing |

### Verdict

FAIL

One **Major + REAL** (F1: a commented expression is reported as a secret the repository must hold, and
the same claim is written into `verification.md` as a guarantee) and one **handoff Major + REAL**
(F8: the archive's promotion pre-flight refuses the spec as it stands). Rubric has two C's and no D.

### Recommended next steps

Minimum set to flip to PASS:

1. **Code + tests** — stop matching expressions inside YAML comments (walk the decoded YAML, or strip
   comment lines/positions before `exprRe`), and add the `TestWorkflowReads` row with a commented
   `${{ secrets.X }}`. This is the only finding whose fix changes behaviour.
2. **Spec (`verification.md`, outside the contract set)** — answer the three "Promotion candidates"
   lines (`yes: docs/lessons/lesson-577-…` for the lesson, with reasons for ADR and pattern), and
   correct the "Only `${{ }}` expressions are scanned" decision so it stops asserting a guarantee the
   code does not give.
3. **Spec (`verification.md`)** — disposition F6 (tasks.md's `gh repo list` wording) and F7 (unknown-only
   exits 0) in the decisions log; do **not** edit `tasks.md` — that would stale this review's verdict,
   and the wording difference is already resolved in the code's favour.

Then, and only then, a re-review round for the new state (the contract-set digests still match, so a
re-review is cheap; F1's fix alone leaves the contract intact).

Tracked, not blocking: F2 (error-line duplication), F3 (case-insensitive names), F4 (`required: yes`),
F5 (multi-document). Each is a small code+tests change and none changes what the check concludes for a
well-formed single-document workflow.

