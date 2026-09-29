---
tags: [spec, verification, templates]
created: "2026-09-26"
---

# Verification - AI-046-pi-nan-provider

## Evidence

- [x] AC1 -> `ai/pi/packages.json` entry; `bats tests/pi-packages.bats` (pin, uniqueness and `why` contracts) green.
- [x] AC2 -> `tests/pi-nan-package.bats`, run by the `pi-nan-package` CI job (PR-B1). It lists with the package alone, with no `models.json` in the agent dir, which is stricter than the criterion's wording: with our `models.json` present pi lists the union, and every id would pass on our own definition. Mutations caught: a ghost id in `enabledModels` (tests 2 and 3), a ghost `defaultModel` (3), an opencode window changed from 262144 to 262000 (5), a ghost pool member (4). Without `PI_BIN` the file skips; with `PI_NAN_PACKAGE_REQUIRED=1` a skip fails.
- [x] AC3 -> `tests/pi-config.bats` "ai/pi/models.json defines no NaN model: pi-nan-provider owns them (AI-046 AC3)" (PR-B2). Mutation M7 below turns it red.
- [x] AC4 -> `tests/pi-config.bats` "pi-nan-provider's media MCP bridge is deployed OFF, merged into the package's state file". It fails on either mutation: `mediaMcp: true`, or the entry without `strategy: merge`.
- [x] AC5 -> measured 2026-09-26, below.
- [x] AC6 -> measured 2026-09-27 with `measure-ac6.sh`, below. It exits non-zero unless the guard-on arm answers and the control overflows.

## Measurements (pi 0.87.1, package 0.7.0, 2026-09-26)

All runs used an isolated `PI_CODING_AGENT_DIR` with the package installed, `NAN_API_KEY=dummy`, and the model cache (`models-store.json`) removed.

The network was cut by a Node preload that makes `fetch`, `net.connect` and `tls.connect` throw. `unshare -rn` is not permitted in the sandbox, and an unreachable `HTTPS_PROXY` gave the same result.

**Package alone, offline (AC5).** pi starts, exit 0, and lists the build-time snapshot:

```
provider  model              context  max-out  thinking  images
nan       deepseek-v4-flash  1M       384K     yes       yes
nan       gemma4             262.1K   32.8K    yes       yes
nan       glm5.3-flash       1M       131.1K   yes       yes
nan       mimo-v2.5          1.0M     131.1K   yes       yes
nan       mimo-v2.6-flash    1.0M     131.1K   yes       yes
nan       qwen3.6            262.1K   65.5K    yes       yes
nan       qwen3.8-flash      262.1K   131.1K   yes       yes
```

`glm5.3` is absent: it is premium, and the package filters by tier. This is why AC2 was amended.

**Package alone, no key.** pi hides the provider ("No models available"), as its docs say it hides providers without usable auth. So a CI listing needs a dummy key.

**Package plus today's `models.json` (the state PR-A ships).** pi lists the union: the six ids `models.json` declares, plus `mimo-v2.6-flash` from the package. It exits 0, offline. pi `docs/models.md` states that a `models` entry adds or replaces the model with the same id. So until PR-B, our six definitions are the ones in use.

**Live prompt, package alone (PR-B1, 2026-09-26).** Isolated agent dir with the package and a `models.json` holding only `openrouter`, so our `timeoutSeconds: 300`, `compat.supportsDeveloperRole: true` and `authHeader: false` were all absent. Real key through `dotf secrets run --only NAN_API_KEY`, `-p` "Reply with exactly the word PONG", one call per model:

| model | exit | seconds |
|---|---|---|
| glm5.3-flash | 0 | 10 |
| deepseek-v4-flash | 0 | 4 |
| qwen3.8-flash | 0 | 6 |
| qwen3.6 | 0 | 3 |
| mimo-v2.5 | 0 | 6 |
| gemma4 | 0 | 3 |

All six answered (qwen3.8-flash answered "pong — standing by", which is the model, not the transport). No `modelOverrides` are needed. What the three removed keys did, read from pi 0.87.1's bundle:

- `timeoutSeconds` is read nowhere in pi. The only match is a local variable in the OpenAI SDK's `retry-after` handling. It has been dead config since AI-025.
- `authHeader` is a compat flag: when true, pi refuses a request without a resolved key and sends the header itself. `false` is the default, so the key was a no-op.
- `supportsDeveloperRole`: the package's generator sets it `false`, and every model answered without it.

**Key delivery.** Removing the `nan` block removes `apiKey: "${NAN_API_KEY}"`, and nothing changes: pi resolves that reference from its own process environment at request time (`docs/models.md`), and the package's `envApiKeyAuth` reads the same variable from the same process. The `pi` wrapper (`.zshrc`, `.bashrc`, `profile.ps1`) injects it with `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY`, and `~/.pi/agent/auth.json` is empty on msi, so no stored credential takes precedence.

**Model switch past 262K (AC6, PR-B2, 2026-09-27).** `measure-ac6.sh` writes a synthetic session: 30 turns attributed to `nan/deepseek-v4-flash`, each with a `thinking` block and a short answer, 2,000,000 reasoning characters and 300,000 answer characters. It resumes the session with `--model nan/qwen3.6` in an isolated agent dir holding the package, `nan-provider.json` and `compaction.enabled: false`. Compaction is off so a compact-and-retry cannot make both arms pass. One variable changes between the arms: `NAN_THINKING_GUARD`.

```
$ dotf secrets run --only NAN_API_KEY -- bash specs/AI-046-pi-nan-provider/measure-ac6.sh
history: 30 turns, 2000000 thinking chars, 300000 answer chars
guard-on: exit=0 model=nan/qwen3.6 stopReason=stop input=54184 cacheRead=0 error=none
guard-off: exit=1 model=nan/qwen3.6 stopReason=error input=0 cacheRead=0 error=Requested token count exceeds the model's maximum context length of 262144 tokens (estimated 677327 input tokens). NaN's gateway answered HTTP 400 "Invalid requ
AC6: PASS
```

- With the guard, qwen3.6 answered with 54,184 input tokens: the answers and the system prompt, no replayed reasoning.
- Without it, NaN refused the request with a 400, which the package's classifier rewrote as a context overflow. The 677,327 figure is the package's own chars/3.47 estimate. From the small run below, this vocabulary tokenizes at about 6.8 characters per token, which puts the real request near 350K tokens: still over 262,144.
- The verdict is not vacuous. A small history (`AC6_THINK_CHARS=20000 AC6_TEXT_CHARS=4000 AC6_TURNS=4`) fits in both arms, and the script fails: "the control did not overflow, so this history does not test the guard". Guard off used 12,736 input tokens and guard on 9,782.
- qwen3.6 is unmetered (`harness/nan-quotas.json`), so the measurement costs no quota.
- A first background run hung for 10 minutes. In print mode pi reads a non-TTY stdin as more prompt, and the runner's stdin was an open socket. The script now redirects stdin from `/dev/null` (lesson 311). The verdict caught that run: the last assistant entry was still the synthetic deepseek turn.

## Test-deletion ledger (PR-B2)

PR-B2 deletes five tests. Each one read `ai/pi/models.json`'s `nan` block, and this PR removes that block, which is the production change that unlocks them. Each replacement is shown by a mutation on this branch (`M1`-`M7`, 2026-09-27). Each mutation was applied, the named suite run, and the file restored.

| Deleted test | What it could detect | Origin | Stronger proof that remains, shown | Focused command |
|---|---|---|---|---|
| `pi-config.bats` "models.json uses the `${NAN_API_KEY}` placeholder" | a literal NaN key in `models.json` | #1026 | "every provider in ai/pi/models.json takes its key from a `${VAR}` placeholder" (M5: the openrouter key made literal, red), and "the pi wrapper hands NAN_API_KEY to pi" (M6: `NAN_API_KEY` dropped from the `.zshrc` wrapper, red) | `bats -f 'placeholder\|pi wrapper' tests/pi-config.bats` |
| `pi-config.bats` "settings.json nan/* models all resolve to an id in models.json" | an `enabledModels` id nothing registers | #749 | `pi-nan-package.bats` test 2, against the package alone (M1: ghost id added, red) | `PI_BIN=~/.local/bin/pi bats tests/pi-nan-package.bats` |
| `pi-config.bats` "defaultModel resolves to an id in models.json" | a `defaultModel` nothing registers | #749 | `pi-nan-package.bats` test 3 (M2: ghost default, red) | same |
| `reviewer-pool.bats` "every pi member of the pool is a reasoning-class model in ai/pi/models.json" | a pool member that is not reasoning-class | #1372 | `pi-nan-package.bats` test 4, against the package snapshot, for nan members (M3: ghost pool member, red). Round 1 of the review found this narrower than the deleted test: it filters on provider nan, so a pi member of any other provider was checked nowhere. `reviewer-pool.bats` now checks every other pi member against `ai/pi/models.json` (M8, M9 below) | same |
| `opencode.bats` "opencode.jsonc and ai/pi/models.json declare the same context window" | opencode and pi disagreeing on a NaN window | #1772 | `pi-nan-package.bats` test 5, against the package snapshot (M4: qwen3.6 262144 to 262000, red) | same |

The old tests would now pass vacuously: with no `nan` block they compare against nothing. M7 (a NaN model put back into `models.json`) turns the new AC3 test red.

`guard-pi-models-schema.bats`, `render_test.go` and `deploy_test.go` also name `models.json` and were left alone. They check the file generically, or use a synthetic fixture, and none reads the `nan` block.

## Review round 1 (FAIL, agy/gemini-3.1-pro-high, 2026-09-28)

The signed verdict is committed as it was written (`83d156d`). Dispositions:

| Severity | Finding | Disposition | Evidence |
|---|---|---|---|
| Blocker | The replacement of the deleted reviewer-pool check filters on provider nan, so a non-NaN pi member bypasses the reasoning-class check | **applied** | `reviewer-pool.bats` "every pi member the package does not own is a reasoning-class model in ai/pi/models.json". The two tests now partition the pi members by provider. M8: an `openrouter/no-such-model` member turns it red. M9: `deepseek/deepseek-chat`, added as an openrouter member, is green; flipping that model's `reasoning` to false in `models.json` turns it red. |
| Major | The media bridge defaults ON in the package and only `nan-provider.json` turns it off. `dotf pi packages apply` does not deploy that file, so a `pi` run before `dotf deploy` gets an unpinned `npx` bridge | **declined** | Setup runs `dotf deploy` before `dotf pi packages apply`: `setup-linux.sh` line 764 before line 854, and `setup-windows.ps1` line 1199 before line 1281. The bridge is not unpinned either: the package spawns `npx -y nan-mcp-server@1.1.2` (`DEFAULT_NAN_MEDIA_MCP_VERSION`, `src/mcp/nan-media.ts`). A missing file is reported: with it moved aside, `dotf doctor` prints `[WARN] drift: pi-nan-provider — … (run: dotf deploy pi-nan-provider)`. The env-var alternative (`NAN_MEDIA_MCP=0` in the wrappers) was weighed and rejected under Decisions below. |
| Minor | f5's verification ends in `exit 1`, so the harness can never mark it passing | **declined** | f5 records a measurement that needs the network cut, and CI cannot re-run it. `exit 1` states that honestly; `exit 0` after an `echo` would be a check that passes without checking anything, the vacuous-pass class of lesson 309. `dotf spec archive` does not gate on `features.json`. `AI-047-nan-quota-alarm/features.json` uses the same convention for its recorded live measurement. |

## Test status

- `bats tests/pi-config.bats tests/pi-packages.bats` -> 0 failures.
- `PI_BIN=~/.local/bin/pi bats tests/pi-nan-package.bats` -> 5/5 (PR-B1). Without `PI_BIN`: 5 skipped; with `PI_NAN_PACKAGE_REQUIRED=1` and no `PI_BIN`: fails.
- PR-B2, rebased on `c6f3bcf`: `bats tests/*.bats` -> 1684/1685. The one failure is `vendored oh-my-zsh git-plugin snapshot is still fresh`, environmental and tracked in #1641. `cd cli && go build ./... && go vet ./... && go test ./...` green.

## Decisions made during implementation

- The media bridge is kept off through the package's own state file, not with `NAN_MEDIA_MCP=0`. Three wrappers (`.bashrc`, `.zshrc`, `profile.ps1`) and any other launch path would each have to carry an env var, while the package reads the file on every launch. `merge` leaves `webSearch` to the box.
- (A) is split into two PRs. PR-A is additive and changes no model behaviour. PR-B removes definitions that seven consumers read, and it needs the test-deletion evidence ledger.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? yes: lesson 311, pi's print mode waits on an inherited non-TTY stdin.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: who owns the `nan` provider id is decided in `proposal.md` (owner, option A) and is local to pi's config.
- [x] New pattern candidate for `00_meta/patterns/`? no: nothing here recurs outside this repo yet.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-046-pi-nan-provider/` -> `specs/archive/AI-046-pi-nan-provider/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
