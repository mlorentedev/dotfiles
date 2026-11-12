---
spec: "CI-018-dotf-gate-and-fleet-lint"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "dd4009de330c921bd58a528e04f3265fbf1cb190"
reviewer: "nan/qwen3.8-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: CI-018-dotf-gate-and-fleet-lint (mlorentedev/dotfiles#3147) — `dotf gate knowledge`, `dotf gate spec` (+ adjacency), `dotf lint fleet` with eight `fleet/*` rules, exceptions, `--fleet`, runbook.
**Sources**: `specs/CI-018-dotf-gate-and-fleet-lint/{proposal,tasks,verification}.md`, `features.json`; `git diff 0c6892f7cb520caa0443eb33619109794f9c6be2...HEAD` restricted to the spec's 8 commits (d245b25c, aa3f59b7, f392566e, 7bccd3f2, d783a689, 7434e253, 6a0a0906, dd4009de); implementation in `cli/internal/gate/`, `cli/internal/cmd/gate*.go`, `cli/internal/fleetlint/`, `cli/internal/cmd/lint_fleet.go`; `tests/check-knowledge-gate.bats`, `tests/lib/spec-gate-parity.bash`, `tests/{check-spec-gate,spec-gate-archive,spec-gate-adjacency}.bats`; `docs/runbooks/guide-fleet-lint.md`.

Contract freshness: `dotf spec digest CI-018-dotf-gate-and-fleet-lint` reproduced the three `contract_digests` in `review-request.json` exactly (features.json `4ad5423c…`, proposal.md `a3e20a7a…`, tasks.md `993a2d63…`), so this review judges the current contract. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tag remains in the spec folder.

### Spec and task alignment

- **Every acceptance criterion re-verified, not accepted.** Fresh runs at `dd4009de`: `go test ./... -timeout 5m` → exit 0, no failures (incl. `internal/fleetlint` 18 tests, `internal/cmd` LintFleet tests); `go vet ./...` → exit 0; `bats tests/check-knowledge-gate.bats` → 45 cases, exit 0; `bats tests/check-spec-gate.bats tests/spec-gate-archive.bats tests/spec-gate-adjacency.bats` → exit 0 (AC1, AC2, AC5's command half).
- **AC3 (no network) is proven by construction, not asserted.** Both shims run dotf under `env -u GH_TOKEN -u GITHUB_TOKEN PATH="$BATS_TEST_TMPDIR/git-only"` with a symlinked real `git` (`_dotf_gate`, `spec_gate_both`). `DOTF_BIN` is built from `cli/` by `dotf_bin_resolve`, and a build failure is a failure rather than a skip — so the parity really compares this tree's code. The comparison requires identical status and identical output except on exit 2 (declared decision in `verification.md`).
- **Exit-code doctrine held under probe.** Event `{}` → `[ERROR] no PR context: the event carries no pull_request`, rc 2; `body: null` → judged as an empty body, rc 1; a valid body against an unresolvable `--base nosuchref` → `[ERROR] … fails closed …`, rc 2 (never a pass).
- **Two planted mutations were caught by named tests, then reverted** (tree verified clean): `filterRegexp`'s `*` → `.*` turned `TestMatchesBranch` red (`matchesBranch(["*"], "release/v1") = true, want false`); dropping the path from the exception key turned `TestExceptionSilencesOnlyItsRuleAndPath`-class expectations red. This corroborates the mutation lines in `tasks.md` rather than taking them on faith.
- **Scope discipline is real.** `scripts/check-knowledge-gate.sh`/`check-spec-gate.sh` are kept (deletion is CI-008's adoption PR, per `proposal.md` "Delivery"); `docs/lessons/lesson-590-…md` and the runbook exist as claimed; the eight rule ids in `rules.go`/`workflow_rules.go` match `proposal.md`'s list exactly; dotfiles itself still has no `.github/lint-exceptions.yml` and no CI wiring for `lint fleet`, consistent with "lands with the adoption, by the owner (#3336)".
- Running `dotf lint fleet --dir <this repo> --default-branch main` gave 26 findings (22 `fleet/persist-credentials`, 2 `fleet/skip-cache`, 1 `fleet/os-on-pull-request`, 1 `fleet/cancel-in-progress`) — matching `tasks.md`'s "26 once #3305 gated two Windows legs" rather than the earlier 29.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | fleetlint YAML | YAML aliases and merge keys are not resolved: a job written `real: *base` keeps only the "no `timeout-minutes`" branch, so it gets a **spurious** `fleet/timeout` while its `uses:` (sha-pin, persist-credentials) and its `runs-on` (os-on-pull-request) go **unchecked**; `<<: *base` behaves the same. Under-report direction is the silent one. | Probes at `dd4009de` on synthetic `anchor.yml`/`alias.yml` in a scratch dir: aliased job reported for `fleet/timeout` only; `grep -rl '<<:\|&base' cli/internal/fleetlint/testdata` → no match | UNTESTED (no fixture names an anchor) | code + tests (`testdata/…`), or one runbook line saying aliases are not read |
| Minor | THEORETICAL | adjacency output trust | Advisory issue text is written to stdout, a `::warning` annotation and `$GITHUB_STEP_SUMMARY` escaping only `\|` (cells) and `%` (annotation). A `\r` embedded in an issue title ends the annotation early and emits a rogue workflow-command line; markdown/HTML in a title is injected into the job summary unneutralised. Issue titles are writable by anyone who can open an issue. | Code read of `reportAdjacency`/`summarizeAdjacency`; feed is line-based so `\n` cannot pass, `\r` can | `tests/spec-gate-adjacency.bats` covers the `%` escape, the `\|` escape and the 88-byte clip; **not** `\r`/HTML → UNTESTED for this input | code + tests (strip `\r`, or escape `<`) |
| Minor | THEORETICAL | spec-gate repo slug | `repoSlug()` takes `origin` verbatim after the `github.com` substring: `https://github.com/o/r/` (trailing slash, a plausible typo) yields slug `o/r/`, so a qualified `Closes o/r#N` never matches and archive-on-merge **silently skips** — the fail-open direction. `strings.LastIndex(url,"github.com")` also matches that substring inside a mirror path (`…/github.com/o/r.git`) and inventories a wrong slug. | Code read; `grep -rn repoSlug cli/internal/cmd/*_test.go` → none. Parity fixtures have no `origin` remote, so both sides always take the empty-slug branch and the branch is never exercised | UNTESTED (`TestRepoSlug…` absent; `spec-gate` fixtures unset `GITHUB_REPOSITORY`) | code + tests (table test over ssh/https/trailing-slash/mirror URLs) |
| Minor | THEORETICAL | exceptions semantics | Exceptions are keyed `(rule, path)` with exact string equality: a glob path (`…workflows/**`, `*`) silences nothing, and two entries with the same rule+path collapse to one line, so the duplicate's staleness is lost. It fails closed (a `fleet/exception` "matches no finding" appears), so the cost is noise, not silence. | Code read of `exceptions.filter`/`readExceptions`; `exc[exception{…}] = n.Line` overwrites | `TestExceptionSilencesOnlyItsRuleAndPath`, `TestIncompleteExceptionIsAFindingAndSilencesNothing` cover scoping and required fields; globs/duplicates UNTESTED | tests + runbook ("path is exact, no globs") |
| Minor | THEORETICAL | knowledge-gate parity | `changedPaths` runs `git diff -z` (NUL, never quoted) while `scripts/check-knowledge-gate.sh` reads quoted output; a lesson path with a space or a non-ASCII name is judged "not changed" by the script and accepted by dotf. The divergence favours the port, but it is a parity case neither side holds. | Code read of `gate.go:changedPaths` vs `scripts/check-knowledge-gate.sh`; no case in `tests/check-knowledge-gate.bats` uses such a path | UNTESTED | tests (a parity case, noting the script side would then fail — retire-on-adoption) |
| Minor | SPECULATIVE | branch-filter scope | `branchFindings` reads `push` and `pull_request` only, while `triggersPR` counts `pull_request_target` for the cancel/os rules: a workflow that triggers solely on `pull_request_target` with a `branches:` filter excluding the default branch is never checked for `fleet/branch-filter`. | Code read; `proposal.md`/`tasks.md` wording also says "push and pull_request", so this is spec-shaped, not a code slip | UNTESTED | spec wording in the runbook's rule table (not the contract set) + tests if widened |
| Question | UNVERIFIED | AC5 external half | AC5 requires "the rules are documented in one runbook section **linked from ci-workflows ADR-001**", and `verification.md`/`tasks.md` additionally claim a 221-finding run over 13 local checkouts and 14 per-repo adoption issues. None of that is reachable from this checkout (out of repo, and home-directory walks are prohibited here). | Local half verified: `docs/runbooks/guide-fleet-lint.md:16: ## The rules`, indexed in `docs/runbooks/_index.md`, `go test ./internal/cmd -run LintFleet` green. The ci-workflows side (ADR-001 decision 5, PR #99 at `06fa706b`) — **not checked** | n/a (cross-repo) | author confirmation / CI record in `verification.md` |
| Minor | REAL | verification evidence locus | `verification.md`'s closing run is recorded "2026-11-12 on main at 5854b97c", a different commit from the `reviewed_sha` this verdict pins. The contract digests match, so it is the same change; the datum is nevertheless the weakest line in the evidence table and I re-ran the suites at `dd4009de` rather than rely on it. | `grep` of `verification.md` ("on main at 5854b97c"); my fresh runs above at `dd4009de` returned the same counts (45 / 30+35+15 / go tests ok) | n/a (evidence artifact) | verification.md — free to edit, outside the staleness set |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All five ACs' verifiable halves reproduce here; negative-path gaps are alias handling, slug parsing and control chars in advisory text, none demonstrated against a real repository. |
| Verification | B | Commands re-run fresh at `reviewed_sha` agree with `verification.md`; two mutations confirmed red then reverted; AC5's cross-repo half is UNVERIFIED and the closing evidence names another commit. |
| Scope | B | Diff maps to the five declared delivery PRs one-for-one; shell scripts deliberately retained; no unrelated production changes in the spec's 8 commits. |
| Reliability | B | Fail-closed verified (unresolvable base → 2, no `pull_request` → 2, unreadable checkout → 2 with the rest still checked, unknown default branch → 2); alias/merge-key under-report is the one silent path found. |
| Maintainability | B | Files ≤ 264 lines, functions small and single-purpose (`filterRegexp`, `matrixValues`, `reportFleet`); no dead code, no `interface{}`-style escape hatches; measured CC not re-run with the `cyclomatic-complexity` skill this round. |
| Handoff-readiness | B | Lesson 590 captured with a real path, runbook added and indexed, ADR promoted to "no: ci-workflows ADR-001's decision"; the archive checklist stays unticked pending this verdict, which is correct pre-archive state. |

Aggregation: no D, no C → all-B path gives PASS; minors carry open gaps → the more honest of the two permitted outcomes is PASS WITH GAPS. No Blocker and no REAL Major exists, so nothing forces FAIL.

### Verdict

**PASS WITH GAPS**

Gaps to track (each with a disposition in `verification.md`, which is outside the staleness set): alias/merge-key resolution or an explicit runbook limitation; `\r`/HTML neutralisation in the adjacency annotation and summary cell; a table test for `repoSlug`'s origin branch (trailing slash included); glob/duplicate exception semantics; a spaced-path parity case; and confirmation of AC5's ci-workflows ADR-001 link.

`dotf spec archive` **is advisable** in the current state: the review is fresh against the contract digests, the verdict passes, no `[AGENT-DRAFT]` tag remains, and `reviewer: nan/qwen3.8-flash` is a `harness/reviewer-pool.json` member. The one thing to do first is disposition the gaps above in `verification.md` — not edit `proposal.md`, `tasks.md` or `features.json`, which would invalidate this verdict.

### Recommended next steps

For the implementer to disposition (apply / ticket / decline with a reason) in `verification.md` or a follow-up ticket. **None of these are contract-set edits** — on this verdict the contract set is closed:

1. **Code + tests (fleetlint):** resolve YAML aliases and `<<` merge keys, or refuse them with a `fleet/exception`-style finding, and add a `testdata` workflow that uses an anchor. Today an aliased job is reported only for `fleet/timeout` while its `uses:`/`runs-on` checks vanish.
2. **Code + tests (gate spec adjacency):** strip `\r` and neutralise `<` in the summary cell before writing issue text to `$GITHUB_STEP_SUMMARY` and the `::warning` annotation; extend `tests/spec-gate-adjacency.bats`.
3. **Tests (cmd):** a table test over `repoSlug` — `git@github.com:o/r.git`, `https://github.com/o/r`, `https://github.com/o/r/`, a mirror path containing `github.com`, and no origin — pinning the `GITHUB_REPOSITORY`-first order the task already declares.
4. **Runbook:** state that an exception `path` is exact (no globs) and that one rule+path may be declared once; both are current behaviour and both read as surprises otherwise.
5. **Cross-repo:** record the ci-workflows ADR-001 → `guide-fleet-lint.md` link (PR #99, `06fa706b`) in `verification.md` so AC5's second half stops being reviewer-trust work; if it did not merge, AC5 is not met and the spec should not archive.
6. **Carry to CI-008's adoption PR** (declared out of scope here): deleting `scripts/check-knowledge-gate.sh`/`check-spec-gate.sh` retires the parity harnesses — keep them until the scripts go, and land dotfiles' own `.github/lint-exceptions.yml` (#3336) or the fleet lint stays red at 26 findings.
