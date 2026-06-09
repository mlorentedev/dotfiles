---
tags: [spec, tasks]
created: "2026-06-09"
---

# Tasks - AI-025-pi-coding-agent

Implemented on branch `ai-025-pi-coding-agent`, stacked on `refactor-011-version-manifest`
(#293, which makes `setup-linux.sh` source the manifest — pi needs `$PI_VERSION`).

## Setup
- [x] GitHub issue #296 (integration) + #297 (Windows verification); #296 self-assigned.
- [x] `proposal.md` complete; acceptance criteria testable.
- [x] Worktree off the #293 branch.

## Implementation
1. [x] `ai/pi/models.json` — literal `apiKey` -> `{env:NAN_API_KEY}` (deploy-time substitution, SDD-009 pattern; **not** runtime `$VAR`, which would regress SDD-009's silent-401 bug class).
2. [x] `ai/pi/settings.json` — curated `enabledModels` (4 NaN + 3 most-powerful free + 3 paid non-OpenAI/Google/Anthropic); default `nan/qwen3.6` (matches opencode); omits volatile `lastChangelogVersion`.
3. [x] `ai/pi/README.md` — setup doc.
4. [x] `versions.conf` — `PI_VERSION=0.78.1`; `versions-conf.bats` asserts it set + semver.
5. [x] `setup-linux.sh` — pi block: npm-pinned install (guarded) + `models.json` via `substitute_env_placeholders` + canonical `AGENTS.md` + `settings.json` seed-if-missing.
6. [x] `setup-windows.ps1` — parity block: npm-guarded install + `Substitute-EnvPlaceholders` + `Deploy-File` + settings seed.
7. [x] `scripts/healthcheck.{sh,ps1}` — pi presence + `PI_VERSION` match + "no `{env:}` left in deployed models.json", folded into the OpenCode section (title kept "OpenCode" so the existing section guards in `opencode.bats`/`healthcheck-ps1.bats` stay green).
8. [x] `AGENTS.md` — pi entry in *Per-Provider Overlays*.
9. [x] `tests/pi-config.bats` — guard: no literal key, uses `{env:NAN_API_KEY}`, valid JSON, no `lastChangelogVersion`, no big-3 OpenRouter.

## Closing
- [x] `features.json` harness contract added (8 features).
- [x] `bats tests/*.bats` green (25 new/touched pass; only pre-existing env failures remain); `shellcheck --severity=error` clean; `.ps1` ASCII-only in changed lines.
- [x] Secret-substitution smoke: a deployed copy of `models.json` has no `{env:}` placeholder (NAN_API_KEY injected from the age store).
- [x] `verification.md` filled.
- [ ] PR opened (base = `refactor-011-version-manifest`, `Closes #296`).
- [ ] Windows runtime verified (#297) — deferred to a Windows box.

## Follow-ups (tracked)
- opencode model-picker curation to the same free+NaN set (separate atomic PR; per the atomic-PR rule).
- **Rotate the previously-committed NaN key** (was plaintext in `ai/pi/models.json` and this session's transcript).
