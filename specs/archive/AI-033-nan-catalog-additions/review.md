---
spec: "AI-033-nan-catalog-additions"
verdict: "PASS"
reviewed_sha: "a922401e1fe67f8db8ac7c8cf3243221067e9679"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-26"
---

## Adversarial review

**Scope**: AI-033-nan-catalog-additions (commit `1188c9f` + spec folder commit `a922401`, HEAD `a922401`)
**Sources**:
- `specs/AI-033-nan-catalog-additions/{proposal,tasks,verification}.md`
- `specs/AI-033-nan-catalog-additions/features.json` (features f1–f5)
- Committed diff: `git show 1188c9f` (5 files: `ai/pi/models.json`, `ai/pi/settings.json`, `ai/pi/README.md`, `ai/opencode/opencode.jsonc`, `tests/opencode.bats`)
- Gate implementation: `cli/internal/spec/{review,archive,reviewer_pool}.go`
- `harness/reviewer-pool.json` (reviewer allow-list)
- Independent corroboration: public vendor documentation for both models (HuggingFace `Qwen/Qwen3.8-Flash-Next`, the-decoder.com, NVIDIA developer blog, docs.z.ai GLM-5.3-Flash, emergent.sh, z.ai blog)

No PR exists yet (`tasks.md` "PR opened referencing this spec folder" is the one unchecked task); the review is against the two commits on `main`-tracked HEAD. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any spec file. Contract files (`proposal.md`, `tasks.md`, `features.json`) are unchanged since `reviewed_sha` and clean in the working tree — the review cannot go stale the moment it is written.

### Spec and task alignment

- The spec is explicitly retroactive (config diff crossed the 50-LOC spec gate after implementation; `tasks.md` records actual order, `verification.md` the same-day decision). That is a documented process choice against issue #1254, not hidden rework — the unit of review is the actual committed diff, which I read in full.
- All five acceptance criteria map to named coverage (below). Every claim in `verification.md` was **reproduced in this session**, not read off the page.
- Out-of-scope gates hold: `harness/model-map.json`, `.pr_agent.toml`, `harness/reviewer-pool.json` contain no reference to either model (grep-verified); `defaultModel` (pi) and `model`/`small_model` (opencode) are untouched (`nan/qwen3.6`).
- `features.json` f1–f5 each carry an executable verification command; f1 was run in full, f2/f3 via bats `-f` filters, f4 via the exact grep, f5 by running `scripts/nan-debug.sh` against the live API — all exit 0.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | SPECULATIVE | opencode display name | The picker label "Qwen 3.8 Flash Next (Alibaba - 1M)" names the architecture-preview model, while the served NaN id `qwen3.8-flash` is the production-ized "Qwen3.8-Flash" (HuggingFace: "Qwen3.8-Flash is the official version based on Qwen3.8-Flash-Next with more production features, e.g., 1M context length by default"). User-facing label may misname the served artifact by a generation marker | `ai/opencode/opencode.jsonc` line read; HF model card (Qwen/Qwen3.8-Flash-Next); live NaN response identifies only by id | UNTESTED — no test asserts display-name content (tests 26/41 assert keys/counts, pi test 12 asserts display-name uniqueness only) | code (opencode.jsonc `name`) |
| Minor | SPECULATIVE | cross-file context cap | pi `ai/pi/models.json` declares `contextWindow: 1048576` while opencode declares `limit.context: 1000000` for the **same** two models — the effective picker cap differs by 4.7% (48,576 tokens) between pi and opencode. Matches the sibling convention (`deepseek-v4-flash`, `mimo-v2.5` carry the same split), so it is inherited, not newly introduced; a user who checks one picker's spec against the other's sees a discrepancy | Direct read of both files (all six chat models show the split) | UNTESTED — no cross-file context-window assertion exists in `tests/opencode.bats` or `tests/pi-config.bats` | code (align or document in one place) |
| Minor | REAL | spec artifact hygiene | `proposal.md` acceptance-criteria checkboxes are all `[ ]` and frontmatter `status: draft`, while `tasks.md` and `verification.md` are fully ticked and all five ACs are demonstrably met — a contract-file drift that will confuse the next reader and any `/spec check`-style tooling | Direct read of `proposal.md` vs `tasks.md`/`verification.md`; all five ACs verified below | UNTESTED (spec artifact concern; gate does not validate it) | spec (tick ACs and set `status: verifying`/`archived` at archive time — do not edit now, it would invalidate this review) |
| Question | THEORETICAL | deploy ordering (#1247 interplay) | This change alone does not put the two models in the picker of any machine whose `~/.pi/agent/settings.json` already exists (seed-if-missing deployment contract), and the delivery vehicle for that — AI-032/#1247's field-level `enabledModels` sync — is currently sitting as **uncommitted** edits in this same worktree (`setup-linux.sh`, `setup-windows.ps1`, `tests/pi-config.bats`), with test fixtures already referencing `nan/glm5.3-flash`. If AI-033 ships before #1247, existing machines silently get no picker entry until the follow-up lands. This is the proposal's declared scope carve-out (tracked in #1247), so it is a sequencing note, not a defect | Proposal "Out of scope" third bullet; `git status --porcelain` for the three files; fixture read in `tests/pi-config.bats` (uncommitted) | UNTESTED — no integration test asserts post-deploy picker presence on a pre-seeded machine (documented in verification.md as uncoverable here per lesson-150) | none for AI-033 (tracked); ordering note for the release |

### Evidence reproduced this session (not read off verification.md)

- **Full bats suite**: `bats tests/*.bats` → 1521 results, 1519 ok + 2 skips (`pwsh not available`), exit 0 — matches verification.md's "1519/1519 ok, exit 0" once the two in-flight AI-032 tests in this worktree are accounted for. `tests/pi-config.bats` 18/18, `tests/opencode.bats` 44/44, `tests/pr-agent-config.bats` 28/28.
- **AC1** — `@test "ai/pi/settings.json nan/* models all resolve to an id in models.json"` (pi-config test 9): passes with all six `enabledModels` entries resolving; `models.json` parses, ids and display names unique (tests 11/12), `defaultModel` resolves (test 10).
- **AC2** — `@test "ai/pi/README.md model list matches settings.json enabledModels"` (pi-config test 13): passes, comparing `provider/leaf` so tier is kept honest.
- **AC3** — `@test "opencode.jsonc exposes 6 chat NaN models ..."` and `@test "opencode.jsonc maps NaN reasoning_content via interleaved on all 6 chat models (DX-004 AC1)"` (opencode tests 26/41): 6/6 model keys present, 6/6 interleaved blocks. JSONC parses under a comment-aware parser.
- **AC4** — features.json f4 command (and manual read): neither id appears as `defaultModel`/`model`/`small_model` value; pi `defaultModel` and opencode `model`/`small_model` remain `nan/qwen3.6`.
- **AC5** — **live, REPRODUCED**: `dotf secrets run --only NAN_API_KEY -- scripts/nan-debug.sh -m qwen3.8-flash 'responde solo: pong'` → exit 0, `reasoning_content` present, answer `pong`; same for `glm5.3-flash`. Both models also **correctly diagnosed this repo's planted `((count++))`/`set -e` bug** in a live Spanish probe (qwen: identifies post-increment returning 0 as the `set -e` trigger; glm: same root cause plus `|| true` remediation) — reproducing verification.md's strongest claim.
- **cli**: `go build ./... && go vet ./...` clean; `go test ./...` all packages ok (incl. `internal/spec`, `internal/doctor`).
- **Vendor metadata cross-check** (proposal's "accepted risk" now corroborated): Qwen3.8-Flash-Next natively 262,144 tokens, extendable to 1M via YaRN, released 2026-08-26 (HuggingFace model card, the-decoder.com, NVIDIA developer blog); GLM-5.3-Flash released 2026-08-26, native multimodal (text+vision), 1M-context evals (docs.z.ai, emergent.sh, z.ai). Both `"input": ["text","image"]` modality declarations match public capability descriptions.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All five ACs verified with named tests or reproduced live evidence; integrity/uniqueness/JSON/tier-honesty negative paths covered by the suite; no defect observed across 1521 tests and two live probes |
| Verification       | A | verification.md's claims are fully reproducible — suite counts (1519 ok), go build/vet clean, live API probes incl. planted-bug diagnosis all re-run this session with identical outcomes |
| Scope              | A | Commit is exactly the five files the proposal names; features f1–f5 map 1:1 to ACs; every declared out-of-scope gate verified untouched |
| Reliability        | A | Config-only, no runtime code; YaRN-vs-native and promotional-allocation caveats documented inline so metadata drift is a conscious decision; stale-entry removal path accepted in the proposal |
| Maintainability    | A | Both entries mirror the `mimo-v2.5` template field-for-field; comments explain WHY (deployment contract, allocation expiry, context provenance); no code to judge for CC |
| Handoff-readiness  | B | Full spec triad + features.json with runnable verification; promotions decided with reasons; minor drift: `proposal.md` ACs unticked/`status: draft` while tasks+verification ticked, and the PR task is admittedly still open |

### Verdict
PASS

### Recommended next steps (before archive)
1. Open the PR referencing the spec folder (the one unchecked task in `tasks.md`); this review is against the commits on HEAD, which a PR must not change.
2. Sequence the release so #1247's uncommitted field-level `enabledModels` sync lands with or right after AI-033, or existing machines will not see the new models until it does (Question row, tracked by design).
3. At archive time only: tick the five `proposal.md` acceptance criteria and flip `status:` to `archived` — do not edit the contract files now, since any edit invalidates this review's staleness check.
4. Optional, non-gating: one-word label correction in `ai/opencode/opencode.jsonc` (`"Qwen 3.8 Flash Next"` → served production naming per HF), and/or a cross-file context-window assertion aligning `1048576` vs `1000000`.

**Archive advisability**: `dotf spec archive` is **advisable** in the current state. All gate conditions verified against `cli/internal/spec`: frontmatter `spec` equals the folder id, `verdict: PASS` is a recognized passing verdict, `reviewed_sha` equals HEAD and the contract files are clean vs `git` history and the working tree, `reviewer: nan/deepseek-v4-flash` is pool entry 0, provenance matches `review-request.json` (`reviewed_sha` and `reviewer` identical, `review_digest_before` empty), and no unresolved `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags exist. Not FAIL, so no flip-set is required; the Minor/Question rows above are surfaced and tracked, not gating.