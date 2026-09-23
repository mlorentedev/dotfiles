---
spec: "HARNESS-136-model-limit-drift"
verdict: "PASS"
reviewed_sha: "65edefe72eb5bd40a6e2a21776c4a6e2380e16df"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-22"
---

## Adversarial review

**Scope**: HARNESS-136-model-limit-drift, round 2 (issue #1594, PR #1595 merged as
`c52e637`; round-1 fixes in `65edefe`)
**Sources**: `specs/HARNESS-136-model-limit-drift/{proposal,tasks,verification,features}.md`;
diff `3f98705863f02e5ae601de91e46090eb2294ac97...HEAD`, the base the launcher resolved
and stated. That range also contains three commits from other work (`799ca66` = CLI-078,
which carries its own spec folder; `5780215` and `c7a8adc`, dependabot bumps) — an
artifact of the base being resolved before those merged, not scope creep of this change.
This round reviewed `c52e637`'s files as the spec's diff (`ai/pi/models.json`,
`cli/internal/doctor/checks_model_limits.go` + `_test.go`, the `doctor.go`
registration, lesson-283, the spec triad) plus `65edefe`'s fixes to the check, its tests
and the spec artifacts. Round 1's `review.md` is preserved at `b956ead`; this file
replaces it, and its dispositions are recorded in `verification.md`.

**Reviewer identity** is the one the launcher drew and stated: `nan/deepseek-v4-flash`,
recorded here exactly as `harness/reviewer-pool.json` spells it. I did not write this
change.

Everything below was verified by running, not by reading claims. `go build ./...`,
`go vet ./...` and `GOOS=windows go vet ./...` clean; `go test -count=1 ./...` green
across the module; `golangci-lint run` (pinned v2.12.2) 0 issues; all 8 `features.json`
commands re-run verbatim → 8/8 exit 0; `bats tests/{pi-config,guard-pi-models-schema,reviewer-pool}.bats`
→ 18/18, 3/3, 4/4. The consequence proof reproduces in both directions: the built binary
prints `[ OK ] 7 models match the provider catalog` against this tree and **9 findings
(2 FAIL, 7 WARN)** against the pre-fix `ai/pi/models.json` from `3f98705` (staged into a
temp `DOTFILES_REPO_DIR`, so the real tree was never touched). AC1's seven values were
re-derived independently from `~/.cache/opencode/models.json` (python, not the check's own
parser — 7/7 match, zero mismatches) and cross-checked against
<https://nan.builders/docs/models>, fetched live this session, which states 1M context for
glm5.3/glm5.3-flash/deepseek-v4-flash/mimo-v2.5, 262K for qwen3.8-flash (with "Max answer
131K"), 262K for gemma4 and 262K for qwen3.6 — consistent with the corrected declaration.
The mutation battery was re-run: 9 of 12 mutations killed, and the 3 survivors are
findings F2–F3 below.

### Spec and task alignment

- AC1 [x] — met; restored independently this session (cache parse + provider docs).
- AC2/AC3/AC4 [x] — met; both directions and both providers pinned by named tests.
- AC5 [x] — met.
- AC6 [x] — met; the round-1 vacuous-PASS corner (F2) is now fixed **and** its fix is
  genuinely pinned (mutating the new `fields > 0` guard is killed by
  `TestModelLimitsDoesNotCountAModelWithNoPublishedLimits`).
- AC7 [x] — **now fully met.** Round 1's REAL Major (F1: the unreadable half WARNed, so
  doctor exited 0 on a declaration nobody read) is applied: `readDeclaration` FAILs on any
  read error but absence, and `TestModelLimitsFailsOnAnUnreadableDeclaration` is red with
  the FAIL mutated back to WARN — reproduced, and the test is *not* skipped here
  (`os.Getuid() == 0` is false on this machine, and the mode-000 read genuinely fails).
  Note the happy accident in f7's command — `grep -c '^--- PASS:' | grep -qx 2` — which
  turns an environment where the test *skips* (root, or a filesystem ignoring mode 000)
  into a red verification rather than a silent one.
- AC8 [x] — met (registration passes `sys, cfg, rep` only; the file's only `os.` calls are
  two `ReadFile`/`IsNotExist` pairs), though its verification command is weak (F5).
- `tasks.md` closing boxes are now all ticked, the PR box included, and `features.json`
  carries `state: "passing"` with per-feature evidence — round 1's F7 applied.
- Round-1 dispositions in `verification.md` are honest: F1–F5 and F7 applied, F6 declined
  with a reason. The one overstated line among them is F5's (F4 below).
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in any contract file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | docs accuracy (AC4 narrative) | Round 1's F3 correction is **partial**: `proposal.md` and the code doc-comment now name the real collision partners, but `checks_model_limits_test.go:114-115` still asserts "`qwen3.8-flash` is published by BOTH nan (262144 context) and openrouter (1000000)" — openrouter keys the row `qwen/qwen3.8-flash` and publishes no bare entry. The corrected wording is also numerically loose: it says "a dozen other providers" where the cache shows **9** others (alibaba, alibaba-cn, alibaba-token-plan, alibaba-token-plan-cn, hyper, llmgateway, opencode-go, requesty, vancine). The mechanism AC4 protects is unaffected; only the prose is wrong. | Cache scan this session, independent of the check: bare `qwen3.8-flash` under 10 providers total (nan 262144; the other 9 at 1000000/1048576); `openrouter` has only `qwen/qwen3.8-flash` (1000000) | n/a (documentation) — UNTESTED | tests (free to edit, outside the staleness set); `proposal.md`'s "a dozen" is contract-set → see next steps |
| Minor | REAL | test coverage — PASS/finding exclusivity | Nothing pins "a PASS is never printed on a run that reported a finding". Rewriting `if findings == 0 { rep.Pass(...) }` to an unconditional `rep.Pass(...)` leaves all 10 tests green, and the report would then print "1 models match the provider catalog" *and* the drift line it contradicts — the same shape of contradiction round 1's F2 was about, one branch over. The counting semantics themselves **are** pinned (the fix's own mutation is killed). | Mutation M7 this session: `*** SURVIVED ***` (all `TestModelLimits*` green) | UNTESTED | tests |
| Minor | REAL | test coverage — the remaining "cannot compare" branches | Round 1's F4 pinned the no-checkout SKIP but not its two siblings: (a) declaration absent → SKIP (deleting the `os.IsNotExist` case and letting it FAIL survives), (b) catalog present but unreadable → WARN (swapping it to SKIP survives). Both are the check's own "a broken path must not answer 'found nothing'" doctrine. | Mutations M10, M11 this session: both `*** SURVIVED ***` | UNTESTED | tests |
| Minor | REAL | report accuracy — PASS message overstates coverage | When the catalog publishes only one of the two limits, the model still counts and the check prints `1 models match the provider catalog`, though `maxTokens` was never compared. AC6 is not violated (something *was* compared, so this is not the vacuous PASS round 1 found), but the sentence reads as agreement about both fields. | Probe this session: catalog `{"nan":{"models":{"qwen3.6":{"limit":{"context":262144}}}}}` → `PASS` printed, `maxTokens` unexamined | UNTESTED | code + tests — but f1's verification greps this exact sentence, so a change lands with a `features.json` edit; ticket it rather than edit the contract under a passing verdict |
| Minor | REAL | AC8's verification command is vacuous when the artifact is absent | `features.json` f8 is `! grep -nE 'os\.(WriteFile\|Create\|Remove\|Rename)\|rep\.Fix\|fix bool' internal/doctor/checks_model_limits.go`. Against a **missing** path `grep` exits 2 and `!` turns that into 0 — the command passes with the file gone, which is precisely the "an absent check read as agreement" shape `verification.md` says it fixed for AC1 and lesson-283 documents. Mitigations: f2–f7 compile the same package, so a deleted file cannot pass the suite as a whole, and the command does discriminate the positive case (`exit=1` on a file containing `os.WriteFile`). The alternation also cannot see `os.OpenFile(…, os.O_WRONLY)`, an enumeration weakness rather than a live defect. | Reproduced: absent path → `grep: … No such file` + `exit=0`; file with `os.WriteFile` → `exit=1` | UNTESTED (nothing guards f8 itself) | features.json (contract set) → follow-up ticket / `verification.md` disposition |
| Question / assumption | REAL | coverage boundary (spec) | The spec never names the repo's **second** declaration of the same two fields: `ai/opencode/opencode.jsonc` carries `provider.nan.models[*].limit.{context,output}` for the same six models, and nothing reads it — `checkModelPins` resolves ids only, and no bats/Go test asserts those numbers (grep across `cli/`, `tests/`, `scripts/`). Its values disagree with the catalog (qwen3.8-flash context 1000000 vs 262144 — the check's own FAIL direction; qwen3.6/gemma4 256000 vs 262144; `output` 8192 throughout). They are **deliberate, not drift**: `ai/pi/README.md` documents that pi declares qwen3.8-flash's *native* 262,144 rather than the *served* 1M (YaRN), and AI-033's risks record the asymmetry. So this is a boundary question, not a blocker — but two consequences are worth a decision: (i) the spec's Out-of-scope list explains *which surface* the check reads ("a limit finding is only actionable where it can be committed, so this reads the checkout") without naming this file, and a reader may over-read an archived HARNESS-136 as covering the repo's declarations; (ii) the check treats the catalog as the ceiling, so a future declaration of the *served* window — the policy opencode.jsonc follows today — would FAIL as an over-declaration (THEORETICAL: not the state of `ai/pi/models.json` today). | `opencode.jsonc:66-67, 95-96, 124-125, 153-154, 182-183, 211-212`; `ai/pi/README.md` ("capability nuance for `qwen3.8-flash`"); `specs/archive/AI-033…/proposal.md` risks; cache scan | UNTESTED | spec (contract set — cannot be edited under a passing verdict) → ticket + `verification.md` disposition |
| Minor | REAL | verification claim accuracy / repo rule | `verification.md`'s F5 disposition says "the orchestrator is now under the AGENTS.md bar". It is not, literally: `gocyclo` reports `checkModelLimits` CC=11 (AGENTS.md wants <10; the rubric's own B band is ≤15), and the function is 52 lines raw / 41 non-comment (AGENTS.md wants <40). The *fix* is real and large (round 1 measured CC≈17 / 84 lines) and the rubric lands on **B**, not C — but the claim overshoots the measurement. | `gocyclo ./internal/doctor/` → `11 doctor checkModelLimits checks_model_limits.go:93`; `checkModelPins` still CC=20, 101 lines, untouched | n/a (claim, not behaviour) | verification.md (excluded from the staleness check — free to correct) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | AC1–AC8 all verified this session, the round-1 REAL Major (AC7's unreadable half) and Major (AC6's vacuous PASS) applied and mutation-proof; deductions for the report-accuracy residue (PASS message on a partially-compared model) and the untested declaration-shape corners (`{"provider":…}`, `null`, per-field typo → SKIP blaming the catalog / WARN claiming "0"). |
| Verification       | B | Every claim I could re-run I re-ran and it held: 8/8 feature commands, build/vet/tests/lint/bats, both-direction consequence proof, 9 killed mutations, AC1 re-derived from two sources; deductions for f8's absent-file vacuity, two unpinned branches, and one overstated claim (CC 11 ≠ "under the bar"). |
| Scope              | B | The spec's own diff matches the proposal and the declared out-of-scope list exactly (no `--fix`, no committed snapshot, two fields only, checkout not deployed copy) and the nine-value data correction is the whole of the non-mechanism change; the three other-PR commits in the range are the base-resolution artifact; one coverage boundary left unstated (the Question above). |
| Reliability        | B | Loud failure on an unusable declaration, documented asymmetry (repo content FAILs, machine cache WARNs), deterministic provider ordering, absent-cache and no-checkout both SKIP-not-PASS, read-only by inspection; costs are the two unpinned branches (F3) and a SKIP whose message blames the catalog when the declaration is what declared nothing. |
| Maintainability    | B | Split into `readDeclaration`/`readCatalog`/`compareModel` (CC 11, 5 and 4 and 4) with WHY comments that carry the round-1 reviews' reasoning; still 1 over the repo's <10 CC bar, and the sibling `checkModelPins` (CC 20) is untouched by design. |
| Handoff-readiness  | A | `features.json` records state + evidence (round 1's F7 applied), the PR box is ticked, the round-1 dispositions are tabulated on the record, lesson-283 is captured and indexed with the right generalisation, and the next owner can re-run every claim from the feature commands alone. |

### Verdict

**PASS** — `severity × reality`. No Blocker and no REAL Major remains: round 1's two
Majeurs are applied and I confirmed both by mutation rather than by reading the
disposition table (`readDeclaration` FAIL→WARN reddens
`TestModelLimitsFailsOnAnUnreadableDeclaration`; restoring the old counting shape reddens
`TestModelLimitsDoesNotCountAModelWithNoPublishedLimits`). What is left is six tracked
gaps — five Minor and one Question — and the rubric has no C and no D (all B or above),
so the mechanical aggregation agrees with the severity axis rather than escalating it.
The Question row is deliberately *not* scored as a Major: the `opencode.jsonc` values it
concerns are deliberate and documented, so there is no live defect to block on; what is
missing is a written boundary, and `dotf spec archive` is the wrong instrument for
writing one. Round 1's F6 (empty `HOME` → cwd-relative catalog path) stays declined on
the same ground the round-1 reviewer gave.

### Recommended next steps

This is a **PASS**, so the contract set (`proposal.md`, `tasks.md`, `features.json`) is
closed: the gaps below are tracked, not fixed, and any edit to those three files
invalidates this verdict and forces another round. Route each one through
`verification.md` (excluded from the staleness check) as *applied / ticketed / declined
with a reason*, or into a follow-up ticket — never as a contract edit.

1. **tests (free to change today, no re-review) — F1+F2+F3.** Correct the test comment
   that still names openrouter as a bare-id collision partner; add the three missing
   pinned behaviours: an unconditional `Pass` alongside a finding, an absent declaration,
   and an unreadable catalog. None of them touches behaviour the ACs already promise, so
   they are cheap regression armor rather than scope.
2. **ticket — F4 (PASS message) and F5 (f8's negated grep).** F4 changes the sentence
   f1's verification greps, so it lands with a `features.json` edit and therefore outside
   this round; F5 is the same file. A follow-up can make AC8's command structural
   (`[ -f <path> ] && ! grep …`, or an assertion that the function takes no fix flag)
   rather than enumerative.
3. **ticket — the Question row.** Ask the owner to decide whether
   `ai/opencode/opencode.jsonc`'s `limit.{context,output}` are in scope for limit drift
   (extend the check, reading a second config shape) or named in a spec's Out-of-scope
   list with the served-vs-native reason `ai/pi/README.md` already gives. Its
   qwen3.8-flash context (1000000 vs the catalog's 262144) is the check's FAIL direction
   and worth a conscious decision rather than an inherited one.
4. **verification.md.** Restate the F5 disposition honestly (CC 11 / 41 code lines:
   inside the rubric's B band, still over the repo's <10 bar) and record this round's
   dispositions with a one-line reason each.
5. **Archive.** `dotf spec archive HARNESS-136-model-limit-drift` is **advisable** in this
   state: the verdict is PASS, it is fresh against `65edefe`, and I changed no contract
   file. Before it runs, `proposal.md`'s frontmatter still says `status: implementing`
   and the folder is still under `specs/` — the archive checklist in `verification.md`
   lists both, and flipping that status is a contract edit, so it is the archive step's
   own job and not a reason for another review round (the gate re-reads the contract set
   *before* the move, per CLI-034).
