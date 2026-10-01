---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - HARNESS-067-model-pin-drift

## Build

- [x] [AC1] Declare `harness/model-pins.json`: every routing pin site, its
      normalization rule, its pool, and a `why` per pin
- [x] [AC1] [AC7] Load and validate it in Go with every failure loud — absent,
      unparseable, no sites, a site with no pins, a pin with no `why`, a
      duplicate id, an unknown kind, a locator with the wrong capture count
- [x] [AC2] Resolve a normalized pin against `harness/model-map.json`, keeping
      the two declared sets separate so a harness-keyed tier never invents a pool
- [x] [AC2] Extract values by `json-path`, `toml-key`, `regex` and `regex-all`,
      erroring rather than returning empty when a locator matches nothing
- [x] [AC2] [AC3] [AC4] Go test over the REAL repository files, plus an injected
      bad pin so the guard cannot pass vacuously
- [x] [AC5] [AC6] [AC8] `dotf doctor` check for deployed sites, distinguishing a
      frozen snapshot from a retired provider, and writing nothing
- [x] [AC4] Narrow the catalog rule after a live run reported `nan/gemma4` — an
      unrouted catalog model is not drift; a dated snapshot of a routed one is

## Round-1 review fixes (#902, 2026-10-01)

- [x] [AC11] Spelling before declaration: a declared `prefix`/`suffix` is
      required (`VerdictMisspelled`), `suffix` covers `opus[1m]`, and
      `spelling: display-name` covers agy's `Gemini 3.7 Flash (High)`, measured
      against `agy models`
- [x] [AC9] Declare `ai/claude/settings.json` (`model`, `advisorModel`),
      `ai/copilot/settings.json` (`model`) and `ai/agy/settings.json` (`model`);
      exclude agy's `modelConfigs.defaultModel`, which agy never reads
- [x] [AC9] [AC11] Nested and list extraction: dotted `json-path` with `*`,
      `toml-key` lists; declare opencode `small_model`,
      `provider.nan.options.model`, `agent.*.model` and `.pr_agent.toml`
      `fallback_models`
- [x] [AC4] Declare a catalog on the pin (`catalog: true`) instead of inferring
      it from `[]`, since `fallback_models` is a list and routing at once
- [x] [AC2] Route the two ids the new pins needed: `tiers.mid.gemini =
      gemini-3.7-flash-high` and `services.advisor = claude:fable`; no chain
      changes
- [x] [AC9] [AC10] Sweep tests and the per-pin dead-id injection test
- [x] [AC12] Doctor: a rotted deployed locator fails, and no pass line follows
- [x] Finding 5: rewrite `pi-default-model`'s `why` from the file's history
- [x] Finding 7: reword AC1 (no schema file, owner decision)
- [ ] Round-2 independent review (the last under the two-round cap)

## Closing

- [x] Every acceptance criterion is covered by at least one test or a recorded
      scenario run
- [x] Every acceptance criterion has an entry in `features.json` with a
      non-vacuous verification command
- [x] `go build`, `go vet`, and `GOOS=windows go vet` all pass
- [x] `go test ./...` — 18/18 packages
- [x] `golangci-lint run` on the pinned 2.12.2 — 0 issues
- [x] The check was run against the live machine and its findings recorded
- [x] No unrelated changes in the diff — staged by explicit path, because this
      worktree also carries another session's uncommitted work (#1244)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder (#1256)
- [ ] Independent adversarial review before archive (`dotf spec review`) — the
      implementing session cannot be the reviewer

## Out of this PR, recorded rather than dropped

- **Generation** for the three surfaces the pipeline owns — phase 2 on #902.
- **`ai/pi/settings.json` is never generated**: seed-if-missing (#754).
- **Repairing deployed drift**: an open disposition question, like #1243's.
- **`harness/reviewer-pool.json`**: its unroutable `agy/gemini-3.1-pro-high` is
  filed as **#1253**.
- **`HIVE_EMBED_BASE_URL`**: an endpoint, not a model id — #1231 stays open.

## Machine-readable features

`features.json` is the harness-facing contract. The agent may not write
`"state": "passing"`; only the harness may, after running `verification` and
capturing exit code 0.
