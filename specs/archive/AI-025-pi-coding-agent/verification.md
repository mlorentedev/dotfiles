---
tags: [spec, verification]
created: "2026-06-09"
verified: "2026-06-09"
---

# Verification - AI-025-pi-coding-agent

Implemented + verified on branch `ai-025-pi-coding-agent` (stacked on #293). Windows parity
authored on Linux; the runtime check is tracked by #297.

## Evidence per acceptance criterion

- [x] **No literal key in `ai/pi/models.json`** — `apiKey` is `{env:NAN_API_KEY}`. `pi-config.bats`
      tests 2-3 assert no `"apiKey": "sk-` and that the placeholder is present.
- [x] **`PI_VERSION` in manifest + install pinned** — `versions.conf` adds `PI_VERSION=0.78.1`
      (the installed `pi --version`). setup-{linux,windows} install
      `@earendil-works/pi-coding-agent@$PI_VERSION` via npm (guarded; latest fallback when unset).
      `versions-conf.bats` test #19 asserts it set; the semver-all test covers format.
- [x] **Deploy targets** — setup deploys `models.json` (key substituted), the canonical
      `AGENTS.md`, and seeds `settings.json` to `~/.pi/agent/` (Linux) /
      `%USERPROFILE%\.pi\agent\` (Windows).
- [x] **Curated `enabledModels`** — 10 entries: `nan/{qwen3.6,gemma4,deepseek-v4-flash,mimo-v2.5}`
      + free `qwen3-coder:free, kimi-k2.6:free, nemotron-3-ultra-550b-a55b:free`
      + paid `deepseek-v4-pro, qwen3-coder-plus, minimax-m3` (none OpenAI/Google/Anthropic).
- [x] **healthcheck pi** — `healthcheck.{sh,ps1}` report pi presence, `PI_VERSION` match, and that
      the deployed `models.json` has no `{env:}` placeholder left.
- [x] **bats guard** — `pi-config.bats` fails on a re-introduced literal key, big-3 OpenRouter, or a
      `lastChangelogVersion` leak into the versioned settings.
- [x] **`AGENTS.md` overlay** — pi listed under *Per-Provider Overlays*.

## Test runs (worktree, 2026-06-09)

- `shellcheck --severity=error setup-linux.sh scripts/healthcheck.sh` -> clean.
- `bash -n` on changed shell scripts -> OK.
- `.ps1` changes ASCII-only (`grep -P '[^\x00-\x7F]'` flags no pi lines).
- `bats tests/pi-config.bats tests/versions-conf.bats tests/versions-no-hardcode.bats` -> 25/25 pass.
- `bats tests/*.bats` -> 0 new failures; the same 9 pre-existing env failures as #293
  (6x pwsh-absent `claude-mem-heal-ps1`, 3x sandbox `shell-profile`), reproduced on the base branch.
- **Secret-substitution smoke**: `substitute_env_placeholders` on a temp copy of `models.json`
  resolves `{env:NAN_API_KEY}` -> no `{env:}` token left, `apiKey` length > 10 (real value
  injected from the age store). Proves the cross-OS self-contained-deploy mechanism on Linux.

## Windows-empirical (tracked by #297)

The Windows code is authored but not runnable here (no pwsh/winget). To verify on a Windows box:
`npm i -g --ignore-scripts @earendil-works/pi-coding-agent@$PI_VERSION`, `Substitute-EnvPlaceholders`
on the deployed `models.json` (no `{env:` left), `AGENTS.md` + seeded `settings.json` in place,
`healthcheck.ps1` reports the version match, and `pi` authenticates against NaN.

## Decisions

- **Deploy-time secret substitution, not runtime `$VAR`** — keeps pi self-contained cross-OS and
  consistent with opencode/SDD-009 (runtime env propagation is a known silent-401 bug class).
- **`settings.json` seed-if-missing** — pi mutates it at runtime (`lastChangelogVersion`), so setup
  never reconciles/overwrites it; the versioned copy omits the volatile field.
- **Default `nan/qwen3.6`** — aligned with opencode so the two agents start identically.

## Follow-ups
- opencode model-picker curation to the same free+NaN set (separate atomic PR).
- Rotate the previously-committed NaN key.

## Promotion candidates
- Lesson for `docs/lessons.md`? Maybe — "reuse the existing deploy-time secret helper for every new
  agent; do not adopt each tool's native runtime-env syntax, or you re-open the SDD-009 bug class."
- New pattern? No (extends an existing repo pattern).
