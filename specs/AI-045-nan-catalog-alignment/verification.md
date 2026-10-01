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
- [ ] AC4 -> PR 2
- [ ] AC5 -> PR 2

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

## Decisions made during implementation

- The chain is declared twice (workflow and toml) and a test holds them equal, rather than the preflight reading the toml. PR-Agent reads the toml from the default branch, so a PR changing the model would otherwise probe the old one.
- The "exactly one secret" test now counts distinct secret names instead of references. The preflight reuses `NAN_API_KEY` in its own step: a second use of one credential, not a second credential.
- The push gate's sparse checkout became one checkout of both scripts, since the preflight must run on every event and the gate only on `synchronize`.
- `doctor`'s NaN quota test served a fixture `/v1/models` that the real model map must match; it now lists `mimo-v2.6-flash`.

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
