---
spec: "BUG-110-spec-home-repo-resolver"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "966225010b4ad4ae2d88f83532039d6877652bdc"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-01"
---

## Adversarial review

**Scope**: BUG-110-spec-home-repo-resolver (spec triad + `git diff 75d857364d6bdfbede17bdcce3989707d97b982d...HEAD`, single branch commit 96622501, PR #3047 merged)
**Sources**: `specs/BUG-110-spec-home-repo-resolver/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`; diff touching `cli/internal/spechome/`, `cli/internal/cmd/spec.go`, `cli/internal/cmd/spec_audit.go`, `cli/internal/doctor/checks_spec_issue_state.go`, `cli/internal/initrepo/forge.go`, `cli/internal/spec/issuestate.go`, `docs/lessons/lesson-535-…`

### Spec and task alignment

- Contract digests: `dotf spec digest BUG-110-spec-home-repo-resolver` matches `review-request.json` exactly for all three files (features 6146eb7f…, proposal 5eab579b…, tasks 5f981eda…) → review will not be stale. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec files.
- All six `features.json` verification commands were run fresh in this session: **all pass** (`ok` for spechome, cmd, doctor on each).
- AC1–AC6 mapped to named tests (below) and probed with three mutations, each reverted; working tree clean apart from `review.md`/`review-request.json`:
  - **M1** (resolver precedence: env before file) → `TestResolvePrecedence/file_beats_env` and `TestSpecInitAndAuditAgreeOnTheIssueHome` red. Precedence and init/audit agreement are genuinely enforced.
  - **M2** (`h.Archive = h.Issues` in Resolve) → `TestSpecAuditArchiveUsesOrigin` + `TestResolvePrecedence` red. AC6 is enforced.
  - **M3** (delete the new `GHArchiveLookup` empty-home guard in `issuestate.go`) → **every test stays green** — see finding 1.
- AC1 "existing tests stay green": build, `go vet` (linux + windows), `golangci-lint run` (pinned 2.12.2) → 0 issues, `dotf lessons fmt --check` → OK. Full `go test ./...` has 3 environment-gated failures — see finding 2.
- `spechome` is a leaf package (no `spec`→`initrepo` cycle); `workGateRepo` fully removed (grep: no references left). `initrepo.OriginURL/OriginHost/ClassifyURL` export renames are mechanical; forge_test updated accordingly.
- Security walk of the diff: no injection surface (all gh paths built from `ValidRepoSlug`-validated slugs, `^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`); no secrets committed (the Gitea test URLs embed a fake token only as a canary — tests assert it never leaks: `TestResolveFailsClosedOffGitHub`, `TestCheckSpecIssueStateGiteaOrigin`); no URL with credentials reaches any error message (OriginHost strips userinfo, proven by `TestOriginHostNeverEchoesCredentials` + the new tests). Functions ≤40 lines, lint clean.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | THEORETICAL (coverage gap itself REAL, mutation-proven) | tests / AC5 | The new `if homeRepo == ""` guard in `spec.GHArchiveLookup` (the code that keeps `gh` off github.com when the archive home is empty) is **UNTESTED**: with the guard deleted, the whole suite stays green — `TestSpecAuditGiteaOriginWithOverride` still passes because the resulting `gh api repos//contents/…` call misses the stub, falls through to a real `dial tcp: lookup api.github.com` (observed), and the resulting error still satisfies the `could not be checked` string assertion. The proposal promise "It never queries github.com for a repository that does not live there" is currently held only by the resolver unit assertion (`TestResolvePrecedence/gitea_origin,_file` → archive `""`) plus message luck. No wrong verdict can result (a `repos//contents/` request 404s, fail-closed stands), hence THEORETICAL consequence. | mutation M3 (clean run): `--- PASS: TestSpecAuditGiteaOriginWithOverride` with guard removed | **UNTESTED** (no named test fails without the guard) | tests — e.g. a direct `GHArchiveLookup(_, "")` unit test, or assert zero recorded archive calls in `TestSpecAuditGiteaOriginWithOverride` |
| Minor | REAL (reproduced) | verification evidence | `verification.md` claims `go test ./...` → "exit 0, 52 packages ok"; it does not reproduce here: `internal/initrepo`, `internal/spec`, `internal/vault` FAIL `TestEmbeddedTemplatesMatchVault` (embedded templates vs the live vault at `~/Projects/knowledge`). The diff touches no file under `cli/internal/*/templates/`, so this is pre-existing vault drift, not this change; the test is vault-conditional (with `VAULT_PATH=/tmp/no-such-vault-xyz` all three go `ok`), so CI without the vault never sees it. | this session's `go test ./...`; template dirs unchanged vs base; VAULT_PATH-absent run `ok` | `TestEmbeddedTemplatesMatchVault` | code (re-vendor templates — separate change/ticket; disposition note in `verification.md`) |
| Minor | THEORETICAL | behavior vs proposal | A malformed `DOTF_BITACORA_REPO` now fails `spec audit` and WARNs the doctor check **even where a valid github.com origin exists** (init behaved this way before; audit newly reads the env — the proposal's Risks section documents the new read but not fail-closed-on-malformed-env, only the malformed *file* case is specced). Nothing sets the var today (proposal's `git grep` evidence), so no current machine hits it. | code read of `spechome.Resolve`; proposal Risks text | `TestResolveRejectsAMalformedFlagOrEnv` | spec wording (disposition in `verification.md` — contract set is closed under this verdict) or code |
| Minor | REAL (observed) | tests / hermeticity | The audit test harness falls through to the **real `gh` binary** for unstubbed paths — mutation runs issued `dial tcp: lookup api.github.com: no such host`. On a networked, `gh`-authenticated machine, unit tests can hit the live API (flaky or false-signal runs). Pre-existing helper, but the new `recordSpecAuditGH` builds on it. | dial errors in mutation-run output | no test asserts hermeticity — UNTESTED | tests |
| Minor | SPECULATIVE | AC2 coverage | "exit non-zero **before any forge call**" is asserted with call-recording only for `spec audit` (`recordSpecAuditGH`). For `spec init`: `TestSpecInitGiteaOriginFailsClosed` covers the HTTPS form only (no SSH at command level, though the resolver covers SSH) and records no calls. Structurally safe today — `specRepoRoot`/`checkWIPLimit` are local and `checkWorkGate` resolves before `spec.Gate` — but the property is one refactor away from unguarded. | code read of `spec.go` RunE order | `TestSpecInitGiteaOriginFailsClosed` (partial) | tests |
| Question | SPECULATIVE | AC2 wording | AC2 as literally written says `spec init` exits non-zero on a Gitea origin with no override, but `dotf spec init <id> --force-no-gate` exits 0 there (the gate is deliberately skipped; `issue:` frontmatter stays empty, exactly as on a GitHub origin). Tests cover only the gated path. Confirm this reading is intended — no change proposed. | code read: `if !forceNoGate { checkWorkGate… }` | none (bypass path by design) | spec (author confirmation; disposition in `verification.md`) |

Contract-file note (not a finding against the diff): `tasks.md` still has `[ ] PR opened referencing this spec folder` although PR #3047 is MERGED (2026-11-02T05:17Z). Checkbox ticks are digest-exempt, so ticking it will not stale this review. Bitácora issue mlorentedev/dotfiles#1906 is still OPEN — correct until the change closes it with the archive.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All six ACs verified by named, mutation-sensitive tests; minor negative-path/wording gaps (malformed env, force-no-gate reading). |
| Verification       | B | Per-AC evidence reproduced in full this session; one claim (`go test ./...` exit 0) fails to reproduce for environmental reasons and one new guard is UNTESTED. |
| Scope              | A | Diff maps to proposal exactly; the one audit fixture change is documented; lesson-535 added as planned; no unrelated changes. |
| Reliability        | B | Fail-closed on every unresolvable path; leak-safe errors proven by tests; only cosmetic gaps (raw `os.ReadFile` error surfaced on unreadable file). |
| Maintainability    | A | Small leaf package, functions well under 40 lines, golangci-lint 2.12.2 → 0 issues, comments explain WHY. |
| Handoff-readiness  | B | proposal/tasks/verification/features + lesson updated and current; stale `tasks.md` PR checkbox and open #1906 pending the implementer's close-out. |

### Verdict

**PASS WITH GAPS** — no Blockers; one **Major, THEORETICAL** (untested archive guard, coverage gap itself mutation-proven) whose consequence cannot produce a wrong verdict; rubric all B or above. Per the severity × reality rule, the Major does not block: it is tracked with a disposition below.

### Recommended next steps

Route each item by set — `proposal.md`/`tasks.md`/`features.json` are **closed** under this verdict (an edit would stale the review); dispositions go in `verification.md` or a follow-up ticket.

1. **tests**: add a named regression test that fails without the `GHArchiveLookup` empty-home guard (direct unit test of `GHArchiveLookup(run, "")`, or zero-recorded-archive-calls in `TestSpecAuditGiteaOriginWithOverride`) — this flips the one UNTESTED Major to covered; tests are outside the contract set, safe to add without re-review.
2. **code (separate ticket)**: re-vendor the drifted embedded templates (`TestEmbeddedTemplatesMatchVault` red on this machine) — unrelated to this spec, file or disposition it, do not bundle here.
3. **spec (disposition in `verification.md`)**: record the malformed-env fail-closed behaviour and confirm the `--force-no-gate` reading of AC2 — no contract edit requested.
4. **tests (optional)**: extend `TestSpecInitGiteaOriginFailsClosed` to the SSH form and/or record forge calls, closing the AC2 call-recording asymmetry.
5. Tick the `tasks.md` PR checkbox (digest-exempt) and close #1906 with the archive per ADR-018.

**Definition of Done (closing pass, as reviewer):** Debt — all six findings recorded above with routes (this deliverable); no issue filed by me: my instructions limit me to writing this review (stated skip). Knowledge — lesson-535 shipped with the diff; nothing further to promote. Board — #1906 OPEN, correct at this stage. Review — n/a: PR #3047 merged before this round; this review is its gate. Evidence — every pass/fail claim above carries the command output from this session (six feature commands, build/vet×2, lint, lessons fmt, three mutations all reverted).

**(a) Verdict: PASS WITH GAPS.** **(b) `dotf spec archive` is advisable once the implementer records dispositions for items 1–4 in `verification.md`** — the verdict itself satisfies the review pre-flight; item 1 (guard regression test) is the one worth doing first. **(c)** n/a — not a FAIL; to reach plain PASS, add the named guard test (items 1) so no Major remains open.
