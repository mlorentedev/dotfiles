---
tags: [spec, verification, templates]
created: "2026-09-26"
---

# Verification - AI-046-pi-nan-provider

## Evidence

- [x] AC1 -> `ai/pi/packages.json` entry; `bats tests/pi-packages.bats` (pin, uniqueness and `why` contracts) green.
- [ ] AC2 -> PR-B.
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

## Test status

- `bats tests/pi-config.bats tests/pi-packages.bats` -> 0 failures.

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
