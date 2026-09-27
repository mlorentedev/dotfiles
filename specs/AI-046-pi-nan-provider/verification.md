---
tags: [spec, verification, templates]
created: "2026-09-26"
---

# Verification - AI-046-pi-nan-provider

## Evidence

- [x] AC1 -> `ai/pi/packages.json` entry; `bats tests/pi-packages.bats` (pin, uniqueness and `why` contracts) green.
- [x] AC2 -> `tests/pi-nan-package.bats`, run by the `pi-nan-package` CI job (PR-B1). It lists with the package alone, with no `models.json` in the agent dir, which is stricter than the criterion's wording: with our `models.json` present pi lists the union, and every id would pass on our own definition. Mutations caught: a ghost id in `enabledModels` (tests 2 and 3), a ghost `defaultModel` (3), an opencode window changed from 262144 to 262000 (5), a ghost pool member (4). Without `PI_BIN` the file skips; with `PI_NAN_PACKAGE_REQUIRED=1` a skip fails.
- [ ] AC3 -> PR-B.
- [x] AC4 -> `tests/pi-config.bats` "pi-nan-provider's media MCP bridge is deployed OFF, merged into the package's state file". It fails on either mutation: `mediaMcp: true`, or the entry without `strategy: merge`.
- [x] AC5 -> measured 2026-09-26, below.
- [ ] AC6 -> PR-B.

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

## Test status

- `bats tests/pi-config.bats tests/pi-packages.bats` -> 0 failures.
- `PI_BIN=~/.local/bin/pi bats tests/pi-nan-package.bats` -> 5/5 (PR-B1). Without `PI_BIN`: 5 skipped; with `PI_NAN_PACKAGE_REQUIRED=1` and no `PI_BIN`: fails.

## Decisions made during implementation

- The media bridge is kept off through the package's own state file, not with `NAN_MEDIA_MCP=0`. Three wrappers (`.bashrc`, `.zshrc`, `profile.ps1`) and any other launch path would each have to carry an env var, while the package reads the file on every launch. `merge` leaves `webSearch` to the box.
- (A) is split into two PRs. PR-A is additive and changes no model behaviour. PR-B removes definitions that seven consumers read, and it needs the test-deletion evidence ledger.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-046-pi-nan-provider/` -> `specs/archive/AI-046-pi-nan-provider/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
