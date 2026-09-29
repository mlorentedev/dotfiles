---
spec: "AI-046-pi-nan-provider"
verdict: "PASS"
reviewed_sha: "53e2101a6fbb6958a8f920dbf38f92e0b2223c69"
reviewer: "nan/qwen3.8-flash"
date: "2026-09-28"
---
## Adversarial review

**Scope**: AI-046-pi-nan-provider (round 2)
**Sources**: specs/AI-046-pi-nan-provider/{proposal,tasks,verification,review-request}.md/json,
features.json, measure-ac6.sh, `git diff 030ade1e1ca5539b1adf7d38eae20ea3524fd957...HEAD` (the base the
launcher stated), tests/{pi-config,pi-packages,pi-nan-package,reviewer-pool}.bats,
ai/{pi/{models,packages,nan-provider,settings}.json,deploy.json}, cli/internal/{deploy,doctor},
installed package at `~/.pi/agent/npm/node_modules/@gtrabanco/pi-nan-provider` (0.7.0),
`.github/workflows/ci.yml`.

### Spec and task alignment

- All six acceptance criteria are ticked and each maps to a named test or a recorded measurement.
- AC1-AC4 gates were **re-verified in this session, not read**: `bats tests/reviewer-pool.bats
  tests/pi-config.bats tests/pi-packages.bats` → 37/37 ok; `PI_BIN=~/.local/bin/pi
  PI_NAN_PACKAGE_REQUIRED=1 bats tests/pi-nan-package.bats` → 5/5 ok with the real pi and the
  installed package, in an isolated agent dir with no `models.json`.
- AC5 and AC6 rest on recorded measurements (a network cut and one live qwen3.6 call) and cannot be
  re-run here inside this budget; they are judged on the quality of the recorded artifact, which is
  high: `measure-ac6.sh` reads `model=`/`stopReason=` from the session, not just the exit code, and
  fails when the control arm does **not** overflow, so a vacuous pass is refused by construction.
- `cd cli && go build ./... && go vet ./... && go test ./...` → exit 0.
- The test-deletion ledger's claims were spot-checked by mutation (below), not accepted on prose.

### Round-1 dispositions — independently verified

| Round-1 finding | Disposition | My verification |
|---|---|---|
| Blocker: reviewer-pool reasoning check narrowed to provider `nan` | applied | **Confirmed.** `tests/reviewer-pool.bats` "every pi member the package does not own is a reasoning-class model in ai/pi/models.json" is the complement of `pi-nan-package.bats` test 4; the two partition on `.provider`. Mutation (add an `openrouter/no-such-model` pi member) turned the test **red** with the message naming the member; file restored, tree clean. Wiring also holds: the `test` job runs `tests/*.bats` and its `code` filter covers `harness/**` and `tests/**`, so a pool-only PR cannot escape it. |
| Major: media bridge defaults ON; `packages apply` does not deploy `nan-provider.json` | declined | **Decline is sound, and two of its premises were wrong.** Setup order is as claimed: `dotf deploy` at `setup-linux.sh:764` before `dotf pi packages apply` at `:854`; `setup-windows.ps1:1199` before `:1281`. The bridge is **not** unpinned: `DEFAULT_NAN_MEDIA_MCP_VERSION = "1.1.2"` (`src/mcp/nan-media.ts:31`). A missing deployed file is surfaced: `deploy.go:274` sets `p.Changed = readErr != nil \|\| …`, and `checks_deploy_manifest.go:55/60` emit the `(run: dotf deploy <name>)` warning — with a named covering test (`checks_deploy_manifest_test.go`, the "absent g is drift" case). Residual tracked below. |
| Minor: `f5` verification ends `exit 1`, harness can never mark it passing | declined | **Decline is sound.** No consumer gates on `features.json` state: the only non-test reference in `cli/internal/spec` is `contractFiles` (`review.go:24`), used for staleness digests. `dotf spec archive` therefore does not read `state`, so `exit 1` blocks nothing while an `echo && exit 0` would be a check that passes without checking (lesson 309). |

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | media bridge / deploy | The off-state lives in a deployed file, so its absence is silent-on-launch and loud-only-in `dotf doctor`, where it is a **WARN**, not a FAIL. A machine that deletes `~/.pi/agent/nan-provider.json`, or runs `pi` before setup reaches `:764`, spawns `npx -y nan-mcp-server@1.1.2` per tool call. Round 1 called this a REAL Major; the evidence above (setup order, pinned version, doctor reporting absence) is why it is tracked as a residual Minor, not a blocker. | `src/mcp/nan-media.ts:79` (`resolveBridgeEnabled("mediaMcp", …, true)`), `checks_deploy_manifest.go:55`, setup line order | Covered for the repo-side contract by `tests/pi-config.bats` "pi-nan-provider's media MCP bridge is deployed OFF, merged into the package's state file" — **mutation-proven this session**: `mediaMcp: true` → red, `strategy: merge` removed → red. The machine-local absence path is UNTESTED end-to-end | tests (`dotf doctor` fixture asserting the WARN for this named entry) — or decline: the WARN is the intended gate |
| Minor | SPECULATIVE | spec artifacts | AC1 names `tests/pi-config.bats` as "the pinned-entry contract"; that contract lives in `tests/pi-packages.bats` (25-37). The criterion is met — I ran both files green — but a reader hunting the pin check in `pi-config.bats` finds none. | `proposal.md` AC1 vs `verification.md` AC1 line | `tests/pi-packages.bats` "every declared source is pinned to a version" | spec (contract file — do **not** edit under this PASS; record the disposition in `verification.md`) |
| Minor | THEORETICAL | verification durability | AC5 (offline start) is proven once, by a Node preload that throws on `fetch`/`net.connect`/`tls.connect`, and is not re-runnable in CI (`f5` says so). A package bump could regress offline start with only AC5's 2026-09-26 transcript as evidence. Partial mitigation already in place and worth naming: the CI job lists with `NAN_API_KEY=dummy`, so NaN answers 401 and the snapshot path is exercised on every `pi_nan` change — a live-catalog failure, not a network failure. | `verification.md` "Package alone, offline"; `tests/pi-nan-package.bats` header | UNTESTED (no named offline test) | tests (a CI case that points `HTTPS_PROXY` at a dead port, the approach `verification.md` says already produced the same result) |
| Question | — | package ownership | `modelOverrides` under `providers.nan` are the documented way to keep a per-model change, and AC3's test deliberately ignores them, checking only `.providers.nan.models`. If one is ever added, nothing verifies it against the package's live limits. Nothing is added today (`ai/pi/models.json` now carries only `openrouter`), so this is a question about the next change, not a defect in this one. | `tests/pi-config.bats:46-51`, comment at `:42-45` | UNTESTED | tests, later |

Strengths that directly mitigate documented risks (not praise): the `pi-nan-package.bats` design — asserting against the **package alone**, with no `models.json` in the agent dir — is exactly what stops AC2 from passing for the wrong reason (pi lists the union when both are present, `verification.md`); and the `PI_NAN_PACKAGE_REQUIRED` flag converts a silent skip into a job failure, so the gate cannot go green on a setup that never ran. Both were confirmed by reading the CI job (`ci.yml:300-329`) and running the file with the flag set.

### Mutation register (applied and reverted this session; `git status` clean apart from the launcher's own `review-request.json`)

| Mutation | Expected | Result |
|---|---|---|
| Add `openrouter/no-such-model` as a pi pool member | non-NaN pi members checked nowhere → red | **red**, names the member (round-1 Blocker fix proven, not asserted) |
| Re-add `providers.nan.models` with `qwen3.6` to `ai/pi/models.json` | AC3 → red | **red**, "NaN models defined in ai/pi/models.json: qwen3.6" |
| `ai/pi/nan-provider.json` → `{"mediaMcp": true}` | AC4 → red | **red** (`pi-config.bats:87`) |
| Drop `"strategy": "merge"` from the `pi-nan-provider` deploy entry | AC4 → red | **red** (`pi-config.bats:85`) |
| Delete `pi-nan-package.bats` setup's `nan-provider.json` copy, or run without `PI_BIN` | skip must not pass silently | `PI_NAN_PACKAGE_REQUIRED=1` without `PI_BIN` fails per file header; with `PI_BIN` set, all 5 ran (no skips) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | AC1-AC4 re-verified green and mutation-proven in this session; AC5-AC6 rest on measurements with a built-in non-vacuity control; no observed defect. |
| Verification       | A | Every claim in `verification.md` I sampled reproduced: 37/37 + 5/5, four mutations red, ledger names M1-M9 with the command per row. |
| Scope              | B | AI-046's hunks are confined to pi config, its tests, and the pool comment; the stated base also carries AI-047/SEC-006/CLI-036/WIN-014/CI-004 merges, which is the base, not creep. |
| Reliability        | B | Offline fallback and missing-deploy-file warning both handled and reported; the residual is that absence is a WARN and per-machine state can drift from the repo. |
| Maintainability    | B | Checks partitioned by provider with the reason written beside them; the deleted-test ledger is the mechanism that keeps the partition honest; no oversized functions in the touched code. |
| Handoff-readiness  | A | Round-1 dispositions table, ledger, lesson 311, Decisions section, promotion candidates answered; the tick that remains (`tasks.md` final box) is checkbox-only and the digest ignores ticks, so it will not invalidate this verdict. |

### Verdict

**PASS**

No Blocker at any reality; no open Major. Round 1's Blocker is fixed and mutation-proven, and round 1's
REAL-classified Major is downgraded on evidence (setup order verified, the npx target is version-pinned
at 1.1.2, absence is reported by `dotf doctor`), leaving tracked Minors. Rubric: no C, no D.
`dotf spec archive` is **advisable** as it stands.

### Recommended next steps

Under a PASS the contract set (`proposal.md`, `tasks.md`, `features.json`) is closed: an edit to any of
them invalidates this verdict. So route accordingly.

- **Outside the contract set — apply or ticket** (all optional; none blocks the archive):
  - tests: a `dotf doctor` fixture asserting the named WARN when `{HOME}/.pi/agent/nan-provider.json`
    is absent, closing the only UNTESTED edge of the round-1 Major (Minor 1).
  - tests: an offline-listing case (`HTTPS_PROXY` to a dead port — `verification.md` says it produced
    the same result as the Node preload) so AC5 survives a package bump (Minor 3).
- **Disposition in `verification.md`** (not a contract edit):
  - AC1's file-name slip, `pi-config.bats` → `pi-packages.bats` (Minor 2), recorded as applied-next-round
    or declined-with-reason.
  - The `modelOverrides` question: name it as a follow-up condition for whoever first adds one.
- Process for closing: the `tasks.md` final box ("Independent adversarial review before archive") may be
  ticked — `dotf spec review`'s contract digests ignore checkbox state
  (`cli/internal/spec/contract_digest.go:47-60`).
