---
tags: [spec, verification, templates]
created: "2026-09-26"
---

# Verification - AI-047-nan-quota-alarm

## Evidence

- [x] AC1 -> `TestEvaluateClassifiesByShareOfQuota`, `TestEvaluateFailsAtTheQuota`, `TestEvaluateTreatsAnAbsentModelAsUnused`, `TestEvaluateWarnsOnAnUndeclaredModel`.
- [x] AC2 -> live run below.
- [x] AC3 -> `TestCheckNaNQuota_FailsOnAnUnservedBinding` (a copy of the real model map with `services.rerank` set to `qwen3-rerank`), `TestEvaluateFailsOnAnIDNaNDoesNotServe`.
- [x] AC4 -> `TestCheckNaNQuota_SkipsWhenNaNIsUnreachable`, `TestCheckNaNQuota_SkipsWithoutAKey`, `TestCheckNaNQuota_DoesNotResolveABWKeyWithoutTheDaemon`.
- [x] AC5 -> `TestCheckNaNQuota_NeverPrintsTheKey`, on the rejected, unparseable and transport branches, each with the key planted where a careless message would echo it.
- [x] AC6 -> vault `3bad279a` (knowledge): `00_meta/agents/scripts/budget-report.sh` lost its quota knobs, per-model percentages, single-pool total and warning, and the false "no usage endpoint" header. It now reports the agent's footprint and points at `dotf doctor`. Run against a one-row `state.db` fixture, it prints the per-model tokens and the metered cost, and nothing about quota. hermes-nan is retired (2026-09-30), so the digest has no running instance.
- [x] AC7 -> `TestEvaluateWatchesEveryMeteredModelNotOnlyBindings`, `TestEvaluateWatchesAnUndeclaredModelWithUsage`, `TestEvaluateIgnoresAMeteredModelTheKeyCannotSee`, `TestEvaluateReportsUsageOnAMeteredModelTheKeyCannotSee` (from pr-agent's review of #1956), `TestEvaluateIgnoresARetiredModelNothingCanReach`, `TestCheckNaNQuota_WatchesMeteredModelsNothingBinds`. Mutation: dropping the metered and used sets from the watched union fails the first two and the doctor test.

Mutations applied one at a time, each caught by the named test: transport error as FAIL; the usage body echoed into the parse warning; unserved id as WARN; the warn threshold moved to 90%; the key not sent; the bw daemon gate removed. The gate mutation survived the first version of the tests, because the default resolver already answers "absent". `TestCheckNaNQuota_DoesNotResolveABWKeyWithoutTheDaemon` was added for it.

## Measurements (2026-09-26, through `dotf secrets run`; only status codes and metadata printed)

- The default `/v1/usage` window is a rolling 30 days (`2026-08-28` to `2026-09-27`). Dates are UTC: the run was at about 01:00 UTC on the 27th.
- `start_date` and `end_date` are accepted.
- `totals.by_model` covers the whole window when `data` is paged: under `limit=3` it equals the unpaged totals.
- NaN answers 403 to the `Python-urllib` User-Agent on `/v1/usage` and `/v1/models`. It answers 200 to Go's default UA (1.1 and 2.0) and to curl. The check sends `User-Agent: dotf-doctor`.
- Per key vs per member: moot as deployed, resolved by inspection on 2026-10-01. One NaN key exists (`secrets/registry.yaml`), and every consumer and kubelab's pr-agent copy use it.
- Open: which period NaN meters against. The 83% figure in AI-044 was computed from the calendar month, so it cannot settle this.

## Live run (AC2)

`DOTFILES_DIR=<this worktree> go run ./cmd/dotf doctor --verbose`, NaN section:

```
[NaN quota]
  [ OK ] deepseek-v4-flash: 682.8M / 3000.0M tokens this month (23%)
  [ OK ] mimo-v2.5: 83.0M / 1000.0M tokens this month (8%)
  [INFO] qwen3-embedding: unmetered, 0.0M tokens this period
  [FAIL] qwen3-rerank is bound in model-map.json but NaN does not serve it (absent from /v1/models)
  [INFO] qwen3.6: unmetered, 0.0M tokens this period
  [WARN] qwen3.8-flash: 415.4M / 500.0M tokens this month (83%) — move routed traffic off it before it runs out
```

That run was on the pre-#1772 main, and it caught both defects #1772 fixes.

After rebasing on main with #1772 merged (`qwen3.8-flash` unbound, rerank named `rerank`):

```
[NaN quota]
  [ OK ] deepseek-v4-flash: 682.8M / 3000.0M tokens this month (23%)
  [ OK ] glm5.3-flash: 117.3M / 2000.0M tokens this month (6%)
  [ OK ] mimo-v2.5: 83.3M / 1000.0M tokens this month (8%)
  [INFO] qwen3-embedding: unmetered, 0.0M tokens this period
  [INFO] qwen3.6: unmetered, 0.0M tokens this period
  [INFO] rerank: unmetered, 0.0M tokens this period
```

The rebase also broke `TestCheckNaNQuota_ReportsEachBoundModel`. It had read the `qwen3.8-flash` binding from the repo's map, and #1772 removed that binding. The fixture now pins `tiers.low.nan` itself.

## Live run (AC2, AC7), 2026-10-01

`DOTFILES_REPO_DIR=<this worktree> dotf doctor` from a scratch build, default verbosity:

```
[NaN quota]
  [INFO] deepseek-v4-flash: 61.6M / 3000.0M tokens this month (2%)
  [INFO] glm5.3-flash: 14.4M / 2000.0M tokens this month (1%)
  [INFO] mimo-v2.6-flash: 19.3M / 1000.0M tokens this month (2%)
  [INFO] qwen3.8-flash: 12.6M / 500.0M tokens this month (3%)
```

`qwen3.8-flash` is bound nowhere in `model-map.json`; the bound-only check never showed it. A first cut warned that `glm5.3` was "no longer served". That was wrong: `/v1/models` filters premium models by the key's tier (`ai/pi/README.md`), so absence there is not retirement, and the warning was dropped.

## Test status

- `go build ./... && go vet ./...`: clean.
- `go test ./...`: `doctor` and `nanquota` pass. `initrepo` and `spec` fail on vault drift from the spec-id grammar thread (#1479). They read the live vault, not this diff, and fail the same on main.

## Decisions made during implementation

- The table is closed-world: a bound NaN model in neither list is a WARN, so a model NaN adds is never silently treated as unmetered.
- Severity: a spent quota and an unserved binding FAIL. No key, an outage or an unexpected answer is a SKIP or a WARN, so a CI runner or an outage never reads as a breach.
- The key is resolved through the Loader seam, as the PAT section does. It is never read from the environment (ADR-028).
- The table and the model map are read from `DOTFILES_DIR`, like `checkModelMap`. `dotf harness mirror` copies the whole `harness/` tree, so the new file reaches the deploy dir on the next setup.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-331-a-shared-budget-is-watched-at-the-account-not-the-binding.md (#1956).
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — a doctor check inside an existing contract; the declared table carries its own rationale.
- [x] New pattern candidate for `00_meta/patterns/`? no — one project so far.

## Review dispositions (round 1, agy/gemini-3.1-pro-high, PASS WITH GAPS)

- **Major, `/v1/models` pagination (`nanquota.go:85`): declined on measurement.** On 2026-10-01, through `dotf secrets run`, only metadata was printed. `GET /v1/models` answers 200 with exactly the keys `data` and `object`: 13 models, no `has_more`, no cursor. `?limit=2` is ignored, and the same 13 come back. The endpoint is the unpaginated OpenAI list shape. If NaN ever adds paging, the first symptom is a bound model reported as not served, which is a loud FAIL rather than a silent pass.
- **Minor, first `NAN_API_KEY` entry (`checks_nan_quota.go:115`): declined.** `secrets/registry.yaml` declares one entry whose var is `NAN_API_KEY` (id `NAN_API_KEY`, also exposing `HIVE_WORKER_API_KEY`). There is no second, scoped key to pick wrongly. The same single-key fact settles AC6's per-key question above.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/AI-047-nan-quota-alarm/` -> `specs/archive/AI-047-nan-quota-alarm/`
- [x] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018): #1766 closes through #1961
- [x] Promotions above executed (if any)
