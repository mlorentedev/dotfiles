---
id: "AI-025-pi-coding-agent"
type: spec
status: archived
created: "2026-06-09"
issue: "mlorentedev/dotfiles#296"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, ai, pi, agent]
template_version: "1.0"
---

# AI-025-pi-coding-agent

## Why

`pi` (the `@earendil-works/pi-coding-agent`) is installed and working on this machine,
but its config lives only in `~/.pi/agent/` — unmanaged by dotfiles — and the source
`ai/pi/models.json` carries a **plaintext NaN API key**. That means a new machine cannot
reproduce the setup, the key is at risk, and pi is not wired to the cross-agent `AGENTS.md`
SSOT the way opencode is. This makes pi a first-class managed agent like opencode: config
deployed reproducibly, key handled via the established deploy-time substitution, and a
consistent free+NaN model environment.

## What

After this PR, `setup-{linux,windows}` install pi pinned to `PI_VERSION` and deploy its
config to `~/.pi/agent/`: `models.json` with the NaN provider (key injected at deploy time
from the age store, never committed), the canonical `AGENTS.md` as pi's instructions, and a
seed `settings.json` curating the model picker to NaN + free OpenRouter models only. The
healthcheck reports pi presence and version. opencode and pi then expose the same curated
model set.

## Out of scope

- opencode's own model-picker curation (sibling change to `ai/opencode/opencode.jsonc`; same
  free+NaN list, separate atomic PR).
- pi `skills/` management and `auth.json` (runtime/secret state, never versioned).
- Switching opencode's secret handling to runtime `{env:}` (rejected — regresses SDD-009).

## Risks / open questions

- **Config path**: assumes pi reads `~/.pi/agent/{models.json,AGENTS.md,settings.json}`
  (confirmed via pi.dev docs + the live install on this box). If a future pi version changes
  the path, deploy targets must follow.
- **npm dependency**: pinned install needs Node/npm. setup guards on `npm` availability and
  warns (does not fail) when absent.
- **settings.json drift**: pi mutates `settings.json` at runtime (`lastChangelogVersion`).
  Mitigation: deploy **seed-if-missing**, never reconcile, and omit the volatile field from
  the versioned copy.
- **Windows runtime**: ps1 authored on Linux, not runnable here — tracked by #297.

## Acceptance criteria

- [ ] `ai/pi/models.json` contains no literal `sk-` key; uses `{env:NAN_API_KEY}`.
- [ ] `versions.conf` defines `PI_VERSION`; `setup-{linux,windows}` install pi pinned to it.
- [ ] setup deploys `models.json` (key substituted), `AGENTS.md`, and seeds `settings.json` to `~/.pi/agent/`.
- [ ] `settings.json` `enabledModels` is curated to NaN + 3 most-powerful free OpenRouter + 3 paid OpenRouter (none from OpenAI/Google/Anthropic).
- [ ] `healthcheck.{sh,ps1}` report pi presence + version against `PI_VERSION`.
- [ ] bats guard fails if `ai/pi/models.json` re-introduces a literal key.
- [ ] `AGENTS.md` registers pi in Per-Provider Overlays + model-tier.
- [ ] `bats tests/*.bats` green; `shellcheck --severity=error` clean on changed scripts.

## References

- GitHub: `mlorentedev/dotfiles#296` (integration), `#297` (Windows verification)
- pi docs: https://pi.dev/ , https://github.com/earendil-works/pi
- Related: `specs/archive/SDD-009-opencode-deploy-time-secrets/` (deploy-time secret pattern reused here)
- Related: `specs/AI-011-opencode-bootstrap/`, `ai/opencode/opencode.jsonc` (the mirrored agent pattern)
- Stacked on: #293 (REFACTOR-011 — makes `setup-linux.sh` source the manifest)

<!-- archived 2026-06-09 — PR: https://github.com/mlorentedev/dotfiles/pull/298 -->
