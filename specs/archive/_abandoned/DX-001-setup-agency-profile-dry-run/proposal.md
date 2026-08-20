---
id: "DX-001-setup-agency-profile-dry-run"
type: spec
status: abandoned # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
review: waived
review_waived_reason: "Abandoned in owner triage 2026-08-20. Never started: setup-linux.sh has no --dry-run flag and the tasks.md still holds the untouched template placeholders. Cold since 2026-05-27."
---

# DX-001-setup-agency-profile-dry-run

> **Naming**: file lives at `<repo>/specs/DX-001-setup-agency-profile-dry-run/proposal.md`. `DX-001-setup-agency-profile-dry-run` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: *(P2, queued 2026-05-21 from session post-AI-014)*: Setup scripts (`setup-{linux.sh,windows.ps1}`) install every tool unconditionally inside their guards (`if not present, install`). User asked 2026-05-21 for agency without breaking non-interactive flow (CI / VM bootstrap / scripted installs). **Two complementary surfaces (~40 LOC each):** (a) **`-Profile minimal|standard|full`** flag selects a tool subset declared in a new `profiles.json` SSOT (minimal = core git/zsh/jq/age; standard = today's set; full = + DevOps stack docker/kubectl/helm/terraform/ansible). Defaults to `standard` so existing automation is byte-equivalent. (b) **`-DryRun`** flag lists "would install X, Y, Z; already present A, B" without side effects — pure read-only audit, complements healthcheck doctor. **Why both:** profile gives strategic agency (one decision per machine); dry-run gives discoverability (what would change before committing). **Anti-scope:** interactive `[Y/n]` prompts per tool (rejected — apt-style interactive default is the #1 complaint about Debian/Ubuntu in CI, industry has moved to non-interactive-default). **Surface:** new `profiles.json` (~30 LOC), `-Profile` switch + foreach filter in both setup scripts (~25 LOC each), `-DryRun` switch (~15 LOC each), bats parity (~20 LOC). **Anti-scope 2:** do NOT add an env-var skip-list in this ticket (orthogonal, future ticket if profiles are not granular enough). -->

`setup-linux.sh` and `setup-windows.ps1` install every tool inside an `if-not-present → install` guard. That works but leaves no agency: on a minimal CI runner or a constrained VM, the user pays for the full DevOps stack whether or not they need it. After AI-014 shipped (2026-05-21), the user explicitly requested per-machine agency without breaking non-interactive flows (CI / VM bootstrap / scripted installs). Two complementary surfaces solve this without compromising the non-interactive contract.

## What

Two new switches on both setup entrypoints, plus a `profiles.json` SSOT at repo root:

- **`-Profile minimal|standard|full`** — selects which tools install. Default `standard` keeps current behavior byte-equivalent.
  - `minimal`: core git/zsh/jq/age + AI configs only.
  - `standard`: today's set (Antigravity, OpenCode, Claude Code, tmux, etc.).
  - `full`: standard + DevOps stack (docker, kubectl, helm, terraform, ansible).
- **`-DryRun`** — lists "would install X, Y, Z; already present A, B, C" without side effects. Pure read-only audit; complements `doctor`/`healthcheck`.

`profiles.json` is the single source of truth: name → tool list. Both setup scripts read it and filter their install loop accordingly.

## Out of scope

- **Interactive `[Y/n]` prompts per tool** — explicitly rejected. apt-style interactive default is the #1 complaint about Debian/Ubuntu in CI; industry has moved to non-interactive-default.
- **Env-var skip-list** (e.g., `DOTFILES_SKIP=docker,kubectl`) — orthogonal; future ticket if profiles prove too coarse.
- **Per-tool version pinning beyond `versions.conf`** — pinning lives in `versions.conf`; this ticket only filters which tools install.
- **Profile inheritance / composition** — `full = standard + X` resolved at JSON-read time; no recursive merge engine.

## Risks / open questions

- **R1**: `profiles.json` schema. Flat or nested? Pick flat for v1 (`{ "standard": ["git", "zsh", ...] }`). Schema must be bats-testable.
- **R2**: Drift between Linux and Windows tool names (e.g., `git` vs `Git.Git`). `profiles.json` lists *abstract* tool names; each setup script maps them to its package manager's IDs internally. Bats parity test asserts every tool in every profile has a mapping in both scripts.
- **R3**: `-DryRun` interaction with `versions.conf`. Should dry-run print exact versions that would install? Yes — it's the high-value audit moment.
- **R4**: Backwards compatibility. Existing CI invocations pass no profile → must resolve to `standard` automatically.

## Acceptance criteria

- [ ] `profiles.json` at repo root with `minimal`, `standard`, `full` keys.
- [ ] `setup-linux.sh` accepts `--profile <name>` and `--dry-run` (long flags, POSIX style).
- [ ] `setup-windows.ps1` accepts `-Profile <name>` and `-DryRun` (PowerShell style).
- [ ] No-flag invocation → behavior byte-identical to today (drift detector confirms).
- [ ] `-DryRun` exits 0 without modifying any deployment target.
- [ ] Bats coverage: parity test asserts profile lookup + dry-run no-side-effect.
- [ ] README "Quick Start" documents the two flags.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → DX-001.
- Predecessor request: 2026-05-21 session post-AI-014.
- Related (deferred): env-var skip-list pattern (no ticket yet).
