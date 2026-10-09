---
tags: [spec, verification, templates]
created: "2026-09-29"
---

# Verification - AI-045-nan-catalog-alignment

## Evidence

PR 1 (AC1-AC3). AC4 and AC5 land in PR 2.

- [x] AC1 -> `tests/reviewer-pool.bats`, `tests/opencode.bats`, `tests/pi-nan-package.bats` (5/5 against a real pi and the pinned package). Live `dotf doctor --verbose` from this tree, 2026-09-30: `[ OK ] mimo-v2.6-flash: 106.8M / 1000.0M tokens this month (11%)`, no NaN quota FAIL. The same check on `main` reports `[FAIL] mimo-v2.5 is bound in model-map.json but NaN does not serve it`.
- [x] AC2 -> `tests/pr-agent-config.bats`: "the chain the preflight probes equals the toml's, model and fallbacks". Mutation: a workflow `DECLARED_MODEL` different from the toml's fails it, naming both.
- [x] AC3 -> `tests/pr-agent-model-preflight.bats` (10/10, stub `curl`: routing, key on stdin, remedy per status class), `tests/pr-agent-model-preflight-real.bats` (real `curl` against a local server: the key read from `-K -`, the status, a real hang ending at the timeout), and "the preflight runs wherever PR-Agent runs, and a failed one skips the guard". Mutations: the key moved to curl's argv fails the stdin test; a misspelt `-K` directive passes every stub test and fails the real one.
- [x] AC4 -> `TestModelMapBudgetIsDeclarationOnly`, `TestModelMapSchemaAcceptsPerModelConcurrency`, `TestModelMapSchemaRejectsMalformedPerModelConcurrency`. The loader exposes the base-plan key cap (7) and all six per-model caps; the shipped schema accepts integer maps and rejects a string value.
- [x] AC5 -> `tests/opencode.bats` pins all six windows/output caps, rejects `enable_thinking`, and permits variants only where NaN applies `reasoning_effort`; `tests/pi-config.bats` pins the same effective limits as `modelOverrides`; `tests/pi-nan-package.bats` verifies the pinned package plus those overrides against a real pi. `qwen3.8-flash` is 1,048,576 in both consumers.

## Test status

- `bats tests/*.bats` -> 1714/1716 locally before the real sibling existed: `stub-real-pairing` (fixed by adding the sibling) and `shell-alias-collision`'s oh-my-zsh snapshot freshness, which reads this machine's oh-my-zsh install (new `glolm` alias upstream) and is unrelated to this change.
- `cd cli && go build ./... && go vet ./... && go test ./... && GOOS=windows go vet ./...` -> all green. `golangci-lint run` -> 0 issues.
- `shellcheck scripts/pr-agent-model-preflight.sh` and `actionlint .github/workflows/pr-agent.yml` -> clean.
- Live smoke, 2026-09-30, `dotf secrets run --only NAN_API_KEY -- ./scripts/pr-agent-model-preflight.sh` over the whole catalog, with the retired model as primary:

  | Model | Result |
  |---|---|
  | `mimo-v2.5` | HTTP 401 (retired) |
  | `mimo-v2.6-flash`, `deepseek-v4-flash`, `glm5.3-flash`, `qwen3.8-flash`, `qwen3.6`, `gemma4` | HTTP 200 |
  | `glm5.3` | HTTP 401 (premium tier) |
  | `minimax-h3` | HTTP 401 (listed in `/v1/models`, not callable) |

  Output: `model=openai/mimo-v2.6-flash`, the five other live models as fallbacks in declared order, and a `::warning::` for each 401 and for the primary being skipped.

PR 2 (AC4-AC5), 2026-09-30:

- TDD red: the three focused Go tests failed on the old key value (`5`), absent `ModelConcurrency`, and a schema that rejected `model_concurrency`; the focused BATS tests listed all six stale `8192` output caps, `qwen3.8-flash` at `262144`, remaining `enable_thinking`, missing pi overrides, and missing README facts.
- After fast-forwarding to `origin/main` at `a4ef47ee`, `cd cli && go test ./internal/harness ./internal/cmd ./internal/doctor -count=1` -> PASS (`harness` 28.450s, `cmd` 80.758s, `doctor` 76.068s).
- Git Bash: `PI_BIN=/c/Users/mlorente/scoop/apps/nodejs-lts/current/bin/pi PI_NAN_PACKAGE_REQUIRED=1 bats tests/opencode.bats tests/pi-config.bats tests/guard-pi-models-schema.bats tests/pi-nan-package.bats tests/reviewer-pool.bats` -> PASS, 79/79. The real-pi guard used pi 0.87.1; the package test installed the pinned `@gtrabanco/pi-nan-provider@0.7.0` in an isolated agent directory.
- CI initially caught `tests/dotf-agent-run.bats` still saturating the old
  dispatchable budget (`5 - 2 = 3`). It now holds 5 slots (`7 - 2`) before
  asserting the chain advances. Git Bash and WSL discover the focused case but
  skip it locally because their Python lacks `fcntl` / WSL lacks Go; CI Linux is
  the non-skipped proof.
- `git diff --check` -> PASS. Executable production delta is 16 added Go lines in `cli/internal/harness/model_map.go`; the remaining production changes are declarative schema/catalog/config records and documentation.
- Draft PR: #1916.

Prevention PRs (AC6-AC10), 2026-10-01:

- AC8 -> `tests/model-canary.bats` (9/9, stub `gh` and stub canary under `bash -e`: open, rewrite in place, close, untouched on a canary that could not run, an annotation on every failed issue write) and the `cli/internal/nanprobe` tests (per-API routing, a completed body as the only answer, a timeout told apart from an unreachable endpoint). Real sibling: `workflow_dispatch` run 36887489906 on `main` at `f8c1d76e`, green in 46 s, 6/6 bound models answered, recorded on #1860. Six, not the local run's nine, by design: `harness/model-pins.json` declares one site under `$HOME` (`$HOME/.pi/agent/settings.json`), which a runner does not have and the canary skips. The deployed `enabledModels` added `qwen3.8-flash`, `mimo-v2.5` and `gemma4` locally. CI covers every repository-bound model; deployed drift is visible only to a local run. That run proves the key reaches every API and `gh issue list` works with the workflow token. The issue writes are proven only by the stub suite until the first red run.
- AC10 -> on `main` after #1940, three review runs from two PRs were queued at once: 36881054024 (`/review` on #1944) ran 15:05:42-15:09:14, 36881389377 (#1944) ran 15:09:56-15:10:41, and 36881441749 (#1943) ran 15:10:45-15:18:08. All three succeeded and none was cancelled. Without `queue: max`, the older pending run would have been dropped. One earlier run (36880519025) was cancelled by the per-PR workflow group when a newer push to the same PR arrived. That is the intended supersede, not the job queue. Recorded on #1923.

AC9, 2026-10-08:

- AC9 -> `tests/pr-agent-route.bats` (21, stub curl) drives the draw by index for each member, a 300-draw run that reaches
  all three with at least 60 each, members out of the draw (401/404/429/529/timeout, no key: no probe at all), the four
  override values and an unknown one (fails before any probe), and the key on stdin only. `tests/pr-agent-route-real.bats`
  (4) runs the real curl against a local server: the `x-api-key` from `-K -`, `/v1/messages` with `anthropic-version`,
  a refusal, a hang and a stalled body. `tests/pr-agent-config.bats` pins the step order and conditions, one credential
  per step, the model for `model` and `model_weak`, an empty chain, a prompt cap equal to the NaN attempt's,
  `DEFAULT_ANTHROPIC_CHAT_MAX_TOKENS=32000`, a worst review under $0.25 at the over-100K prices, adaptive thinking at
  `high` with the override naming the model, `ai_timeout` inside the step, no temperature, the two Anthropic copies identical,
  the NaN attempt's `low` effort listed for glm5.3-flash only, the preflight's exit code reaching the draw, and the
  action pin and ten review settings shared by all three attempts. `tests/pr-agent-publish-guard.bats` (23) runs the
  guard: the probe's answers and the final guard judging the last attempt in `ATTEMPTS`; its empty-outcome case found a
  real defect (lesson 369). Mutations, each turned red: an unanswered member admitted, the key check dropped, `nan-only`
  probing, an unknown override accepted, the drawn NaN model not first, a constant draw, the non-numeric count check
  dropped, `CONFIG__MODEL_WEAK` dropped, the setup-error stop disabled, the Anthropic branch of the no-review message
  removed, the output cap doubled, glm's effort list dropped or its level raised, the job timeout back to 30. One mutant survived and was equivalent (a redundant `nan-only` clause, removed).
  Live evidence on #2188: run 37879933613 attempt 1 (forced Anthropic: Haiku published, NaN skipped) and attempt 2 (the
  draw picked mimo, which published; Anthropic skipped). glm5.3-flash, streamed from NaN on review prompts with 16K
  output tokens: default effort 289 s and `medium` 338 s, both `finish_reason: length` with no content (the
  60,000-reasoning-character ceiling); `low` 285 s on 37,251 prompt tokens (45,651 reasoning characters, a review) and
  215 s on 96,707 (33,541, a review). Run 37885509177: the draw picked Haiku, which logged adaptive thinking at `high` and published, with all
  three NaN members answering the probe. Run 37886959259: the draw picked glm5.3-flash,
  which received `low` and published a complete review in 74 s. Pending: a real second attempt.

AC9 amendment B, 2026-10-09:

- `tests/pr-agent-route.bats` (32, stub curl) drives every point of a fixture pool's total weight and asserts each member
  is drawn exactly its weight, the same over the shipped pool, a non-answering member's points shared out, a weightless
  NaN member kept in the chain but never drawn first, the risk route at the threshold and one below it, by the exact
  label only, Sonnet probed only for a risky PR, its failure falling back to the draw with a note, an unread size, the
  overrides never taking the risk route, and the allowlist refusing another model before any probe.
  `tests/pr-agent-route-real.bats` (4) runs the real curl with the allowlisted id; the local server answers by a mode
  file, so no test-only model id has to pass the allowlist. `tests/pr-agent-config.bats` (70) pins the routed model and
  effort, the allowlist read from the script, a price row per allowed model with its worst review under a ceiling (Haiku
  $0.21 under $0.25, Sonnet $0.84 under $1.00), the pool's NaN models equal to the preflight's chain, and run details
  among the settings all three attempts share. The Go gate: `cli/internal/spec/review_slot_test.go` and
  `cli/internal/cmd/spec_review_test.go`. Mutations, each turned red: a third model allowlisted, a pool NaN model
  drifting from the chain, an off-by-one in the cumulative draw, the risk comparison made strict, Sonnet given a weight;
  six more on the Go gate (commit 77c89fd5).

Budget, at about 2,500 PR reviews a month (60% of a 30-35 hour week, 10-15 PRs in parallel) plus 4-5 adversarial
reviews an hour of spec work, against $60-65 of Anthropic credit:

| Spend | Volume | Unit | Month |
|---|---|---|---|
| Sonnet, PR-Agent risk route (1,500 lines) | ~3% of PRs, ~75 | ~$0.37 measured | ~$28 |
| Haiku, PR-Agent draw (35% of the weight) | ~875 | ~$0.008 | ~$7 |
| Sonnet, second signature on `risk: high` specs | ~8 | ~$1.60 | ~$13 |
| Haiku, fallback first signature | rare | | ~$2 |
| Total | | | ~$50 |

The margin covers the Sonnet share running above 3% while #2215 and #2216 measure it.

**Measured against the estimate, 2026-10-09.** The first live Sonnet review (run 37907390670, #2188 itself, 3,682 changed
lines) read 176,372 prompt tokens and wrote 1,642: about $0.37, against the $0.13 assumed above. A PR on the risk
route is large by definition, so its prompt is near the 200K cap, and $0.13 is a mid-size PR's figure. At $0.37 the
Sonnet PR-Agent line becomes about $74, and the total about $96, over the $60-65 budget. The owner chose the
correction the same day: `min_changed_lines` goes from 900 to 1,500. Over the last 395 merged PRs, 9.1% reach 900
lines, 4.8% reach 1,200 and 3.0% reach 1,500. At 1,500 the line is about $28 and the total about $50. The table above
already uses the measured unit and the new share. Capping Sonnet's prompt was rejected: it truncates the large diffs
the route exists for. #2215 and #2216 hold the measurement. Sonnet is the only `signs:
second` member, so a high-risk spec's second signature depends on the Anthropic key: deliberate, and named in the
pool's `$comment`.

## Decisions made during implementation

- The chain is declared twice (workflow and toml) and a test holds them equal, rather than the preflight reading the toml. PR-Agent reads the toml from the default branch, so a PR changing the model would otherwise probe the old one.
- The "exactly one secret" test now counts distinct secret names instead of references. The preflight reuses `NAN_API_KEY` in its own step: a second use of one credential, not a second credential.
- The push gate's sparse checkout became one checkout of both scripts, since the preflight must run on every event and the gate only on `synchronize`.
- `doctor`'s NaN quota test served a fixture `/v1/models` that the real model map must match; it now lists `mimo-v2.6-flash`.
- PR 2 keeps `pools.nan.concurrency` as the key-wide cap so the existing pool semaphore remains conservative; `model_concurrency` is a separate declared map because AC4 does not add per-model enforcement.
- pi keeps package ownership of NaN model registrations. `ai/pi/models.json` uses supported `modelOverrides` for the published limits, including NaN's served 1,048,576-token qwen3.8-flash window, rather than restoring duplicate model definitions.
- OpenCode exposes `reasoning_effort` variants only for glm5.3-flash, qwen3.6 and gemma4. deepseek-v4-flash, qwen3.8-flash and mimo-v2.6-flash manage their own depth, so presenting fast/thinking variants for them would be a no-op.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-320-a-retired-model-hangs-before-it-refuses.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the preflight applies ADR-032's "never degrade silently" to one more failure shape; it decides nothing new
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <decided at archive: six more repos bind PR-Agent to NaN (#1763), so it may recur>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-045-nan-catalog-alignment/` -> `specs/archive/AI-045-nan-catalog-alignment/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
