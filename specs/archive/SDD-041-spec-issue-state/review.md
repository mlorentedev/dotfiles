---
spec: "SDD-041-spec-issue-state"
verdict: "PASS"
reviewed_sha: "721195b1671deed8faa7dd15106514141eac99c8"
reviewer: "nan/mimo-v2.5"
date: "2026-09-23"
---

## Adversarial review

**Scope**: SDD-041-spec-issue-state (full change from `8a68a49c`)
**Sources**: `specs/SDD-041-spec-issue-state/{proposal,tasks,verification,features}.md`, `git diff 8a68a49c...HEAD` (20 files, +1482 / -1)

### Spec and task alignment

- **AC1** (issue-link resolver + REST parsing): 10 named tests in `internal/spec` — `TestIssueStateResolveFrontmatter` (8 sub-cases), `TestIssueStateResolveEmptyFrontmatterIsUnlinked`, `TestIssueStateResolveProse` (8 measured shapes), `TestIssueStateResolveProseRejectsNonTrackingRefs` (8 negatives), `TestIssueStateFrontmatterWinsOverProse`, `TestIssueStateGHLookup` (7 sub-cases: open, closed, PR, 404, auth failure, rate limit, unparseable output). All green. **Covered.**
- **AC2** (audit classification + exit contract): `TestIssueStateAuditClassifies` (9 fixture specs, 9 severity assertions, deduplication asserted at 5 lookups for 9 findings), `TestIssueStateAuditUnanswerable` (offline spec produces `SeverityUnanswerable`, `AuditNeedsAttention` fires), `TestSpecAuditExitIssueState` (3 sub-cases: clean exit, zombie non-zero, unanswerable non-zero), `TestSpecAuditRefusesWithoutAHomeRepo`. All green. **Covered.**
- **AC3** (doctor section): `TestCheckSpecIssueState` (5 sub-cases: open→PASS, closed→FAIL, 404→FAIL, offline→WARN never PASS, gh absent→Skip never PASS). Section title asserted. Live run evidence in verification.md. **Covered.**
- **AC4** (GOV-004 normalisation + prose guard): `specs/GOV-004-agents-md-diet/proposal.md` carries `issue: "mlorentedev/dotfiles#673"`. `TestIssueStateNoActiveSpecIsProseLinked` resolves from `os.Getwd()`, checks `checked > 0`, fails on prose-only. **Covered.**
- **AC5** (cross-repo audit in hive): Evidence in verification.md — `dotf spec audit` in `~/Projects/hive` exits 1, FEAT-015 and HIVE-119 resolved through prose links, HIVE-267 zombie, HIVE-118 unlinked. **Covered.**

All implementation tasks ticked in tasks.md. No unchecked tasks. `features.json` has 4 entries with `state: "pending"` (correct — only the harness may set `passing`). No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | issue-link-resolver | `issueRefPattern` accepts dots in owner/repo names (e.g. `o.w.n.e/r.e.p.o#1`), which are not valid GitHub identifiers. The practical impact is nil — the home repo owner filter and the API 404 would catch any real mismatch — but the regex is slightly wider than the domain. | code read: `issuelink.go:35` `[\w.-]+` accepts `.` | UNTESTED | tests (add a test with dotted names to confirm API 404 path, or tighten regex) |
| Minor | THEORETICAL | spec-audit-cmd | `specAuditTimeout` (20s) and `specIssueStateTimeout` (15s) are two distinct constants for the same conceptual bound. A future reader might not notice the difference. The doctor timeout is tighter because doctor's sweep must stay fast; the audit command is standalone. Both are correct for their context. | code read: `spec_audit.go:27`, `checks_spec_issue_state.go:18` | UNTESTED | — (design note; no fix needed) |
| Minor | THEORETICAL | doctor-unanswerable-mapping | `checkSpecIssueState` maps `SeverityUnanswerable` to `rep.Warn`, which is correct per the proposal ("unanswerable → WARN, never PASS"). But the audit exits non-zero for unanswerable findings. A reader comparing the two may momentarily expect doctor to FAIL on unanswerable too. The divergence is intentional (doctor gates no CI context; the red state blocks nothing). | code read: `checks_spec_issue_state.go:59-62` | `TestCheckSpecIssueState/an_unanswerable_lookup_is_a_WARN,_never_a_PASS` | — (documented design decision; no fix needed) |

No findings are in the contract set (`proposal.md`, `tasks.md`, `features.json`). The three Minor findings are all theoretical edge-case observations with no demonstrated failure path.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All 5 acceptance criteria verified; full classification table implemented and tested; negative paths (upstream/related/sister labels, unlabelled refs, owner mismatch, malformed links, 404, PR-as-issue, rate limit, auth failure, unparseable output) all covered |
| Verification | A | Every AC has named tests with executable verification commands; mutation battery (12/12 killed); live audit evidence for cross-repo (hive) and this-repo (42 active specs, 14 FAIL) |
| Scope | A | Diff is 20 files, all directly related to the spec; GOV-004 normalisation, lesson-287, and README update are all within spec scope; no unrelated changes |
| Reliability | A | `lookupAll` deduplicates across specs sharing an issue; bounded worker pool (8 goroutines); timeout on every `gh` call; mutex-protected results map; `AuditNeedsAttention` enforces unanswerable-never-clean |
| Maintainability | A | Functions under 40 lines; clear naming (`Grade`, `ResolveIssueLink`, `lookupAll`); `yamlCommentStart` fix is documented with a comment explaining the YAML comment rule; lesson-287 captures the `RepoRoot(".")` skip bug |
| Handoff-readiness | A | Proposal, tasks, verification all complete; lesson-287 captured in `docs/lessons/`; `specs/README.md` updated; features.json populated |

### Verdict
PASS

### Recommended next steps

- **Archive is advisable.** All acceptance criteria are met, all tasks are ticked, the review is fresh against the current `reviewed_sha`, and no contract-set changes are needed.
- The three Minor findings are tracked observations for follow-up; they do not block archiving.
- The `features.json` entries have `state: "pending"` — a harness run setting them to `passing` with evidence is the expected next mechanical step before `dotf spec archive`.
