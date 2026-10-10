---
spec: "PLAT-001d-macos-ci-leg"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "b6fd39e9ebcb46c3e76f982f94c062435865f3f9"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-09"
---

## Adversarial review

**Scope**: PLAT-001d-macos-ci-leg — `git diff 5a59b68e894beadc0ec7dcddad7e98441690edaf...HEAD` (416 files, 13531+/2416-), reviewed on a macOS host (Darwin 27, arm64, `/bin/bash` 3.2.57) — the OS this spec exists for.

**Sources**: `specs/PLAT-001d-macos-ci-leg/{proposal,tasks,verification,features}.md|json`; spec-owned commits `aa2f0f5d` (#2145), `d31ca10a` (#2146), `f6e9507d` (#2147), `b6fd39e9` (docs close); `scripts/run-bats.sh`, `tests/guard-bats-tags.bats`, `tests/run-bats{,-real}.bats`, `.github/workflows/ci.yml`.

**How the range decomposes.** The launcher-resolved base `5a59b68` (2026-10-07 20:17) is an ancestor of HEAD; `main..HEAD` is exactly one commit (`b6fd39e9`), so the range = the spec's four commits **plus 44 commits from other PRs that merged to main concurrently** (#2145–#2222, other specs' work, each carrying its own review). I reviewed the spec's own commits in depth and triaged the rest for interactions with this spec's claims (e.g. #2150 vault-health-golden retirement, #2070, #2160 mock-environment hardening, #2170 `[[ ]]` guard, #2211 lessons-linter retirement — all correctly reflected in `verification.md`).

### Spec and task alignment

All six `features.json` verification commands were executed fresh, by me, on this Mac — all pass:

- **f1 / AC1**: `cd cli && go test ./internal/doctor/ -run 'TestContractOS|TestCheckContractPath_Dialects|TestCheckContractEnvVars_WindowsDialect' -count=1` → `ok github.com/mlorentedev/dotfiles/cli/internal/doctor`. Also full `go build ./... && go vet ./... && go test ./...` → exit 0.
- **f2 / AC2**: `PATH="/bin:$PATH" ./scripts/run-bats.sh --expect-bash 3 --filter-tags os-sensitive` → `bash 3.2.57 at /bin/bash`, `268 test(s) tagged os-sensitive`, **268 ok, 0 not ok**. Re-run a second time with `/sbin` stripped from PATH (the proposal's stricter phrasing, `sha256sum` therefore absent) → 268/268 ok again.
- **f3 / AC3**: `bats tests/guard-bats-tags.bats tests/run-bats-real.bats tests/run-bats.bats` → 15/15 ok. Mutation-verified, red-green, all reverted:
  - `# bats file_tags=` → `# bats file_tag=` in `tests/utils.bats` → guard test *"every bats tag comment is spelled exactly, with only known tags"* **red**; tree restored.
  - `# bats file_tags=os-sensitive wrongtag` (unknown tag) in `tests/shell-profile.bats` → guard red on *both* the spelling test and *"the os-sensitive tier selects tests, and bats agrees with the comments"*; restored.
  - Empty selection in the real environment: `./scripts/run-bats.sh --filter-tags no-such-tag-xyz` → `::error::run-bats: no test carries the tag(s)…`, exit 1.
- **f4 / AC4**: `./scripts/run-bats.sh` → **1896 tests, 0 failed** on this Mac (full suite green; count has grown past the 1804 recorded in `verification.md` because later merged PRs added tests — see finding F2).
- **f5 / AC5**: python/yaml assertion on `jobs['test-macos']` + `! grep -q test-macos forge/branch-protection.json` → passes. Job runs `macos-latest`, `if` on `needs.changes.outputs.code`, asserts zsh and `/bin/bash`, `--expect-bash 3`, native `go build/vet/test`, tier on PR, full suite on push. The `code` filter includes `tests/**`, `.github/**`, `scripts/**`, so test-only PRs trigger the leg.
- **f6 / AC6**: `bats tests/run-bats.bats tests/run-bats-real.bats` → ok. `grep -n nproc .github/workflows/ci.yml` → only a comment. Mutation: restoring `run: ./scripts/run-bats.sh --jobs "$(nproc)" …` turns *"ci.yml counts CPUs through run-bats.sh, not nproc, which macOS does not have"* **red** (printed the restored line); restored, `git status` clean.

`tasks.md`: every `[x]` I sampled has diff or run evidence (branch creation, guard-first TDD, three-PR split, first green runs cited by CI run id). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec folder. Contract digests in `review-request.json` correspond to the files at `reviewed_sha` (working tree clean; only launcher-created untracked `review-request.json`/`review-transcript.jsonl` present).

Code checklist over the spec's own diff: no secrets, no injection surface (fixed strings, pinned release URLs, `versions.conf` pins asserted in-workflow), no auth surface. `shellcheck -S warning` clean on `run-bats.sh`, `setup-mock-env.sh`, `check-bats-names.sh`, `compile-harness.sh`, `install-dotf.sh`, both git hooks, `setup-linux.sh`. The BSD fixes are sound on read: `grep -E` + `exit 2` handling in `check-bats-names.sh` (the swallowed-`grep -P`-error silent pass is closed and tested), `pwd -P` canonicalisation with safe fallback in both hooks, `_dotf_sha256` falling back to `shasum` and failing closed (empty result → mismatch → abort).

One candidate finding I refuted before filing: the step named *"bats, full suite (bash 3.2, main only)"* carries only `if: github.event_name == 'push'` with no `github.ref` check, which looked like a spec-vs-code mismatch — but the workflow's `on: push: branches: [main]` (ci.yml:4-5) makes push events main-only in fact, so the label is accurate. No finding.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | REAL | spec artifact | `proposal.md` "Risks" (line 43) and the AC1 line of `verification.md` claim `checks_catalog.go` and `checks_repodir.go` "read `runtime.GOOS` directly… not fixed here". False at HEAD and false at the base: `checks_catalog.go:94` reads the `s.GOOS` seam, `checks_repodir.go` contains no GOOS reference at all, and the gap was closed by #2095 (`7f648fd6`, merged 2026-10-07 07:42 — *before* base `5a59b68` at 20:17), which also added `goos_seam_test.go`. | `grep runtime.GOOS` over `cli/internal/doctor/` → only the sanctioned production wiring (`system.go:261`, nolint) and host-only `fs.go:41`; `git log 5a59b68..HEAD -- checks_catalog.go` → empty (file unchanged in range) | `TestCheckShadowedCatalogTools_FindsWindowsShapesThroughTheSeam`, `TestCheckPathFiles_ChecksTheTargetOSsPathFile` (both pass) — these *disprove* the claim | Spec: correct `verification.md` freely; `proposal.md` is contract set — disposition (correct with reason or decline) in `verification.md`, do not edit after this review |
| Minor | REAL | verification | Recorded counts no longer match HEAD, and are internally inconsistent: tier stated as 254 (AC2, Test status) and 257 (tier section) while it is **268** at HEAD; full suite stated as 1804 while it is **1896**. The per-file tier justification list (AC2 "justified") does not cover files tagged after this spec (`pr-agent-route-real`, `setup-linux-only-downloads{,-real}`, `guard-zsh-tied-names`, `guard-bats-dbracket` — added by later PRs). Both numbers are timestamped as historical measurements, so this is evidence drift, not a false claim. | Fresh runs this session: `bats --count --filter-tags os-sensitive` → 268; full suite → 1896 ok | `@test "the os-sensitive tier selects tests, and bats agrees with the comments"` (floor ≥100, deliberate per the recorded design decision) — the exact counts are UNTESTED by design | `verification.md` (outside contract set): re-measure at archive time; per-file justifications for post-spec tags belong to the PRs that added them |
| Minor | SPECULATIVE | tests | The guard's tier floor (`>= 100`) tolerates losing up to ~168 tagged tests (268 → 101) without failing — wholesale loss is caught, partial silent loss is not. | Guard code read; no reproduction of partial loss | `@test "the os-sensitive tier selects tests…"` covers only the floor | Tests (optional follow-up: a per-file tag manifest would be the stronger guard, rejected by a recorded design decision in `verification.md` — surface only, do not gate) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All six ACs verified fresh on macOS with `/bin/bash` 3.2 and `/sbin` off PATH; negative paths (near-miss tag, unknown tag, empty selection, missing/non-GNU parallel, nproc regression) proven red-green by mutation. |
| Verification       | B | Every `features.json` command reproduced here; but one AC1 evidence line states a gap that no longer exists and recorded counts have drifted from HEAD (F1, F2). |
| Scope              | B | The spec's four commits match the proposal exactly (scripts, guard, runner, workflow, spec, lesson 342); the launcher-resolved range additionally carries 44 concurrent main commits reviewed under their own specs — documented, not attributable to this change. |
| Reliability        | A | `run-bats.sh` fails loudly with `::error::` at every broken-environment boundary (missing tools, bad bash, empty selection, empty option values), each with a named test; CI job asserts shells, pins, timeouts. |
| Maintainability    | A | Functions ≤40 lines, shellcheck clean on all touched `.sh`, comments explain WHY at every seam; guard tests are small and single-purpose. |
| Handoff-readiness  | B | Proposal/tasks/verification/features all present, lesson 342 written, follow-ups ticketed (#2059, #2061, #2062); minus one stale risk note sitting in the contract set (F1). |

Aggregation: no C, no D → PASS territory on the rubric; severity axis has minors only, all REAL except one SPECULATIVE.

### Verdict

**PASS WITH GAPS** — no blockers, no majors; three minors, two REAL, each with a disposition line above.

### Recommended next steps

Route by set — the contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** by this verdict; do not edit it, or the archive gate re-litigates this review:

1. **`verification.md` (free to edit):** correct the AC1 "Gap noted, not fixed" sentence — the seam gap was closed by #2095 before this spec's base; re-state if #2061 has other remaining scope, else note it closed. Re-measure the tier/full-suite counts at archive time (268 / 1896 as of `b6fd39e9`).
2. **Disposition record (in `verification.md`):** F1's `proposal.md` line-43 edit is declined as a contract edit made post-review, with the reason above — or ticketed for a contract-touching follow-up if the risk note must be corrected in place.
3. **Follow-up ticket (optional):** F3 — partial tier loss under the ≥100 floor — only if the recorded "no second copy of the tag list" decision is ever revisited.
4. Deck checked and clean: `git status --porcelain` shows only the launcher's untracked `review-request.json`/`review-transcript.jsonl`; all mutations reverted.

**(a) Verdict:** PASS WITH GAPS. **(b)** `dotf spec archive` is **advisable** in the current state: the frontmatter above is well-formed, `reviewer` matches the pool spelling, `reviewed_sha` is the commit examined, no draft tags remain, and all six feature commands pass fresh on main's content. **(c)** n/a (not FAIL) — minimum actions to reach PASS proper would be none; the gaps are tracked, not blocking.
