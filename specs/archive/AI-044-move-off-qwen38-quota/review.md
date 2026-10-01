---
spec: "AI-044-move-off-qwen38-quota"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "3112d09f070964683bea4fbacd0bfd5b75a29c63"
reviewer: "nan/glm5.3-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: AI-044-move-off-qwen38-quota (round 2 — re-review after round 1's FAIL, #1762 / PR #1772)
**Sources**: `specs/AI-044-move-off-qwen38-quota/{proposal,tasks,verification,features}.md/json`, `review-round-1.md`; `git diff 2e73eab34510fdfe0573f207e783bd352ebc02f7...HEAD` (launcher-resolved base; the spec's own commit is `75eb855`, 9 files). Commands run fresh at `3112d09`: the f2 contract command with a real pi, the same command under a reverted `999999` mutation, the same command with a nonexistent `PI_BIN`, `bats -f 'default model' tests/opencode.bats`, `bats tests/opencode.bats` (clean), the f3 `jq` check, a snapshot-vs-opencode window comparison, and greps of `forge/branch-protection.json`, `.github/workflows/ci.yml` and the spec folder.

### Spec and task alignment

- **Round 1 Major F1 (contract set) — FIXED and verified red-green.** `features.json` f2 now records `PI_BIN=… PI_NAN_PACKAGE_REQUIRED=1 bats -f 'same context window' tests/pi-nan-package.bats`. All three outcomes its evidence claims were reproduced at HEAD: **exit 0** with a real pi; **exit 1** after setting the three opencode `262144` windows to `999999` (mutation applied, failure names all three drifted models — gemma4, qwen3.6, qwen3.8-flash — then reverted, tree clean); **exit 1** with `PI_BIN=/nonexistent/pi` (REQUIRED turns the skip into a failure). The recorded command is falsifiable again.
- **Round 1 Minor (`ai/pi/models.json` pointer) — FIXED.** `verification.md` AC2 now names `tests/pi-nan-package.bats` and the pinned `@gtrabanco/pi-nan-provider@0.7.0` in `ai/pi/packages.json`.
- **AC1** — `bats -f 'default model' tests/opencode.bats` → ok 1, exit 0; `opencode.jsonc` routes `model`/`agent.plan` to `nan/glm5.3-flash` and `small_model` to `nan/qwen3.6`. **Verified.**
- **AC3** — the f3 `jq -e` returns `true`, exit 0; `tiers.low.nan = qwen3.6`, `chains.low = [nan:qwen3.6, nan:glm5.3-flash, claude:haiku]`, `services.rerank.model = "rerank"`. `qwen3.8-flash` survives in `harness/model-map.json` only inside the `$comment` recording the history. **Verified.**
- **AC2 substance** — the parity test is non-vacuous by construction (it exits 1 if the opencode/package intersection is empty) and I enumerated the intersection from the package's `models.generated.ts`: six models, all agreeing (glm5.3-flash 1M, deepseek-v4-flash 1M, qwen3.8-flash 262144, qwen3.6 262144, mimo-v2.6-flash 1048576, gemma4 262144); opencode carries no NaN model the package does not. **Verified.**
- **Round 1 Major F2 (no offline guard) — still true, carried as a dispositioned ticket.** Under the mutation, `bats tests/opencode.bats` is green 43/43 (reproduced at HEAD; round 1 measured the same). Deferred to #1866 with root cause and snapshot design recorded. This round adds new evidence about the compensating control, below.
- **Round 1 Question (base range) — closed.** `75eb855` is the spec's commit and matches the proposal file-for-file; the 258-file resolved range is mainline work (AI-045/046/047, secrets, ADRs 041/042) the base drags in, as round 1 attributed.
- `tasks.md`: all implementation boxes `[x]`, each maps to diff lines. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec folder (grep hits are round-1 review prose only). `harness/reviewer-pool.json` contains `nan/glm5.3-flash` (HARNESS-093 admission) — this review is pool-built.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | CI gating (new this round) | The compensating control for the deferred offline guard is weaker than the disposition states: the `pi-nan-package` job is **not a required status check** and no aggregate job needs it, so a future window drift can merge with the parity job red. It is *observed*, not *enforced*. Mitigations: the `pi_nan` path filter covers `ai/opencode/opencode.jsonc`, so any drift PR runs the job (visible red in PR triage); it runs on every push to main; merges are human-supervised with auto-merge forbidden and mandatory check triage by doctrine. | `forge/branch-protection.json` `required_status_checks.checks` = cli-gate, lint, lint-powershell, review-attestation, spec-gate, test, test-windows — no `pi-nan-package`; `.github/workflows/ci.yml` `pi-nan-package` job has `needs: [changes]` only; `pi_nan` filter includes `ai/opencode/opencode.jsonc`. | `tests/pi-nan-package.bats` parity test exists and is red-green (verified here), but nothing makes its red block a merge — as a *gate property*, UNTESTED | **tests + CI config** (fold into #1866: either make `pi-nan-package` required in `forge/branch-protection.json` + live protection, or land the offline guard — one of the two closes it) |
| Minor | REAL | Spec artifact wording (non-contract) | `verification.md` round-1 disposition says "The property is enforced in CI" — overstated per the finding above; it runs in CI but does not block. | Same evidence as above. | Same as above. | **spec** (`verification.md` — outside the contract set, editable without invalidating this review) |
| Minor | REAL (carried from round 1 Major F2) | Guard coverage | No offline guard: the mutant passes the default local suite. Round 1 graded this Major; **re-graded Minor this round** with reasoning stated: the property is red-green verified at HEAD by a named test; the guard runs on the exact PR that could introduce drift (path filter) and on every push to main; merges are human-supervised; root cause and fix design are ticketed (#1866). This is enforcement-strength debt with a valid ticket, not a defect in the change under review. | Mutation reproduced at HEAD: `bats tests/opencode.bats` → 43/43 ok, exit 0 with windows at 999999; f2 exit 1 on the same mutant. | The parity test itself (`tests/pi-nan-package.bats`, `@test "…same context window…"`); the offline guard is UNTESTED by design until #1866 | **tests** (#1866) |
| Minor | SPECULATIVE | Reliability | The low chain falls back `nan:qwen3.6 → nan:glm5.3-flash → claude:haiku` with no quota alarm; if `glm5.3-flash`'s 2B/month is spent mid-month the chain degrades silently. Declared out of scope in the proposal and tracked in AI-047 (#1766). | Read of `harness/model-map.json` `chains.low`; no reproduction. | UNTESTED | none — surfaced only; do not gate |

Severity × reality: no Blocker, no open Major. The one REAL Major from round 1 inside the contract set (f2's unfalsifiable evidence) is fixed and verified; the other was dispositioned to #1866 and is re-graded above with the reasoning in the open. Three REAL Minors and one SPECULATIVE Minor remain, all tracked.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All three ACs verified by fresh commands, including red-green mutation evidence and the no-pi negative path; no observed defects. |
| Verification       | B | Every f2 evidence claim reproduced exactly (exit 0 / exit 1 mutant / exit 1 no-pi); full-suite claim (1655/1656, test 1257 environmental) not re-run here; one wording overstatement ("enforced in CI"). |
| Scope              | B | The spec's own commit is 9 files matching the proposal; the wide resolved range is a base artifact, not creep. |
| Reliability        | B | Config/routing change with sane fallback chain; quota alarm deliberately deferred to AI-047. |
| Maintainability    | A | Comments record why and dates; the moved test left a tombstone; the parity test refuses to be vacuous. |
| Handoff-readiness  | B | Round-1 dispositions recorded in `verification.md`, promotions answered; one disposition wording needs the correction below. |

### Verdict
PASS WITH GAPS

### Recommended next steps

1. **`verification.md` (non-contract set, free to edit)** — correct the round-1 F2 disposition: "the property **runs** in CI (the `pi-nan-package` job, on every push to main and on PRs touching `ai/opencode/opencode.jsonc`) but is **not a required status check**, so a red job does not block a merge." Record round-2 dispositions beside round 1's.
2. **#1866 (ticket, outside the contract set)** — add the second exit to its scope: either land the offline local guard **or** make `pi-nan-package` a required status check (`forge/branch-protection.json` + live protection, verified with `dotf forge protection check`). Today neither holds; one of the two closes the enforcement gap for good.
3. **Contract set (`proposal.md` / `tasks.md` / `features.json`) — closed.** This review requests no edits there; any edit invalidates this verdict and forces another round.
4. **`dotf spec archive` IS advisable** in this state: the verdict is a recognized passing one (`cli/internal/spec/review.go` — only FAIL blocks), the review is pool-built and fresh at `3112d09`, no draft tags remain, and the promotion candidates are answered. The three Minors above are tracked, not blocking.
5. **UNVERIFIED here, not asserted either way:** the full-suite claim (`bats tests/*.bats` → 1655/1656, test 1257 red on main too) and the 2026-09-26 live-NaN latency samples were not re-run/measured inside this round's budget; the suites covering the spec's surface (`tests/opencode.bats` 43/43 clean, the f2 command, the f3 jq check) were.
