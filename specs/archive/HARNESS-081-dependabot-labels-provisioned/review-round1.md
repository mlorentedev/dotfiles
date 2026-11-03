---
spec: "HARNESS-081-dependabot-labels-provisioned"
verdict: "FAIL"
reviewed_sha: "31ccb79dfd2577a3f52fb4baa0e0e978750b0c5c"
reviewer: "nan/glm5.3-flash"
date: "2026-11-02"
---

## Adversarial review

**Scope**: HARNESS-081-dependabot-labels-provisioned (dotfiles#1222 — provision the labels a repo's dependabot.yml declares)
**Sources**: `specs/HARNESS-081-dependabot-labels-provisioned/{proposal,tasks,verification,features.json}`; `git diff 1a6a8b076952ab2c4796a17b28c7d1ff561e5baf...HEAD` (single commit `31ccb79d`); live runs of the Go and bats suites plus binary-level mutation tests, this session.

### Spec and task alignment

- Diff maps 1:1 onto the proposal's "What": `ApplyDependabotLabels` (`cli/internal/initrepo/labels.go`), wiring into `dotf init github` (`cli/internal/cmd/init.go`) and `dotf init` (`cli/internal/initrepo/orchestrator.go`), one rollout step (`scripts/bitacora-rollout.sh`), tests for both layers. No scope creep; no test deletions.
- Contract digests in `review-request.json` match `dotf spec digest HARNESS-081-dependabot-labels-provisioned` exactly (all three files). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec files.
- `features.json` entries are all `pending` with empty `evidence` — consistent with the tasks.md rule that only the harness sets `passing`.
- Both `features.json` verification commands I ran (`f1` go test, `f5` bats filter) are executable and pass; the bats filter matches all five AC5 tests.
- tasks.md correctly leaves "PR opened" unticked; every other box has diff evidence.

### Verification performed (this session, fresh)

- `go build ./... && go vet ./... && GOOS=windows go vet ./...` — pass. `go test ./internal/initrepo/ -count=1` — pass. `golangci-lint run` — 0 issues. `shellcheck scripts/bitacora-rollout.sh` — clean. `bats tests/bitacora-rollout.bats tests/stub-real-pairing.bats` — 22/22, including the five new AC5 cases.
- `dotf spec digest` matches `review-request.json` `contract_digests`.
- Built the real binary (`go build ./cmd/dotf`) and ran `dotf init github` against a stubbed `gh`: `--dry-run` lists `ci` and the call log contains **no** `label create` (AC2, exit 0); a rejected create exits **1** with `create label "ci" on owner/repo …` (AC4); `--color 0366d6 --description "Dependency updates opened by Dependabot"` sent as claimed (AC1 details).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | test-coverage | AC1's "present ones (case-insensitive) left alone" is only half-proven. The suite covers existing-side normalization (existing `Dependencies` vs declared `dependencies`), but the **declared-side lookup** (`have[strings.ToLower(name)]`) is unguarded: mutating it to `have[name]` **survives the whole suite**. A dependabot.yml declaring `"Dependencies"` against an existing lowercase `dependencies` label is the uncovered direction. | Mutation run this session: `perl` edit to `labels.go` line 83, `go test ./internal/initrepo/ -run TestApplyDependabotLabels -count=1` → `ok` with the mutation in place. Control mutation (`case dryRun:` → `case false:`) WAS caught, so the suite is not uniformly weak. | UNTESTED | tests (add a fixture with a mixed-case *declared* label matching a lowercase existing one, e.g. declared `["Dependencies"]`, existing `dependencies`) |
| Major | REAL | test-coverage / resilience | The proposal's stated risk mitigation — "Any other failure to read is a `[WARN]` skip … and the rollout turns that into an `err`" — has no test for either degradation path: a non-404 contents read failure (`could not read .github/…`) or a failed `gh label list` (`could not list the labels`). Mutating the 404 filter to accept every read error (unreadable repo treated as "no labels declared" → `[OK] none declared` → rollout reports "current") **survives the suite**. A regression there is exactly the silent failure #1222 is about. | Mutation run this session: `if errors.As(err, &exit) && strings.Contains(…, "404")` → `&& true`; full `TestApplyDependabotLabels*` run → `ok`. `grep "could not read\|could not list" cli/internal/initrepo/*_test.go` → no hits. | UNTESTED | tests (stub `gh api` failing with a non-404 message and `label list` failing; assert `Action == "skipped"` and the rollout `err`) |
| Minor | THEORETICAL | brittleness | The missing-file test matches the literal substring `404` in gh's stderr. A gh release changing the message ("Not Found (HTTP 404)" → anything without "404") turns every repo without dependabot.yml into a `[WARN]` skip — fail-loud direction, so rollout-visible, but noisy. The converse ambiguity (GitHub returns 404 for *forbidden* as well as *missing*) is inherent to the API and mostly mitigated: an unauthorized repo then fails `label list` → skip. | Code read of `labels.go:114-117`; no repro of a changed gh message. | UNTESTED | code (match `HTTP 404`) or accept; pair with the finding-2 tests |
| Minor | SPECULATIVE | scale | `gh label list --limit 1000` silently misses labels beyond 1000; a repo with >1000 labels would get duplicates attempted (which fail loudly). No observed repo near the limit. | Code read of `labels.go:76`. | UNTESTED | — (surface only; do not gate) |

No Blockers. No spec-vs-code mismatch found: every "What" bullet and AC is implemented as written; AC1–AC4 are proven at the unit and binary level, AC5 at the bats level.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All five ACs met and verified at unit + binary level; two negative paths (declared-side case match, unreadable-repo skip) lack regression coverage. |
| Verification       | B | Named tests map to every AC and all ran fresh this session; the two gaps above are precisely where the mapping stops. |
| Scope              | A | Diff matches the proposal exactly — no creep, no test deletions, single focused commit. |
| Reliability        | B | Read-before-write idempotence is proven (second-run test); degradation paths are implemented correctly but unguarded by tests. |
| Maintainability    | A | Functions small, golangci-lint 0 issues, comments explain why (Dependabot never creates custom labels), helpers reused from `github_test.go`. |
| Handoff-readiness  | B | Spec triad complete and promotion candidates reasoned; PR box still open (correctly unticked pre-review). |

### Verdict

**FAIL** — two **REAL** Majors (both test-coverage gaps, demonstrated by surviving mutations this session, both UNTESTED). Per the skill, an UNTESTED Major is not resolved by the implementer's claim; each needs a named regression test before PASS.

Note on proportionality: the **production code is correct as far as every check I ran can tell** — the failures here are missing tests, not observed misbehavior. Both fixes land in `cli/internal/initrepo/labels_test.go` (and optionally one stub tweak), which is **outside the contract set** (`proposal.md`, `tasks.md`, `features.json`), so the contract digests stay valid; only a re-review round is needed.

### Recommended next steps

For the implementer to disposition in `verification.md` (applied / ticketed / declined) — none require a contract-set edit:

1. **tests** — Extend the AC1 fixture: declare a mixed-case label (e.g. `["Dependencies"]`) against existing lowercase `dependencies`, assert no `label create` is issued. Name it (e.g. `TestApplyDependabotLabelsMatchesDeclaredCaseInsensitively`) and reference it from `verification.md` AC1.
2. **tests** — Add cases where the stubbed `gh api` fails with a non-404 message and where `label list` fails; assert `Action == "skipped"`, and (bats) that the rollout reports the repo as failed. Reference from `verification.md` AC3/proposal risk section.
3. **code (optional, with 2)** — Tighten the 404 match to `HTTP 404` while touching that line.
4. After 1–2 land, run `dotf spec review HARNESS-081-dependabot-labels-provisioned` for the confirming round; `dotf spec archive` is **not advisable** until that round passes.

---

*(Second-opinion note per repo interaction discipline: this verdict is the reviewer pool's output for this round; no stronger-model cross-check was run inside this session — the round mechanism itself is the check.)*
