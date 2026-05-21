---
id: "WIN-002-windows-smoke-sweep"
type: spec
status: implementing # this PR ships partial closure; full clean-machine sweep remains
created: "2026-05-21"
tags: [spec, proposal, windows, smoke-sweep, healthcheck]
template_version: "1.0"
---

# WIN-002-windows-smoke-sweep

> Multi-PR umbrella for the Windows post-setup smoke sweep. **This PR ships partial closure**: tightens `scripts/healthcheck.ps1` so the FAILs it emits reflect what `setup-windows.ps1` actually deploys, not the Linux-deploy expectations. Full WIN-002 closure (clean admin Windows machine sweep, BUG-N tickets per finding) is tracked by subsequent work in the same spec.

## Why

<!-- from 11-tasks.md: Run `setup-windows.ps1` + `healthcheck.ps1` on a clean admin Windows machine, log every fail as inline BUG-N entries, fix trivial ones in place. Closes when healthcheck reports 0 fails. -->

After WIN-001 (PR #71) merged, the first empirical `healthcheck.ps1` run on this Windows machine reported 17 FAILs. Closer inspection revealed **most of them are not bugs in `healthcheck.ps1` itself** — they're a scope mismatch between what `healthcheck.sh` checks on Linux (where setup deploys a full dev toolchain via `$APPS_HOME`) and what `setup-windows.ps1` actually installs on Windows (a much narrower winget list: age, eza, jq, gh, zoxide, copilot). The healthcheck was ported 1:1 to PowerShell but never re-scoped to Windows install reality, so it flags as FAIL many tools and env vars that dotfiles never promised to deploy on Windows.

Result: a fresh `setup-windows.ps1 + healthcheck.ps1` run looks alarming ("18+ FAILs!") when in fact the install is doing exactly what was designed. This breaks the canary value of healthcheck — if users learn to ignore healthcheck FAILs because most are false, they'll miss real ones (the kind WIN-002 was supposed to surface in the first place).

## What

Three observable behavior changes after this PR:

1. **Section 1 (Core Tools in PATH)** now lists only the tools `setup-windows.ps1` actually installs via winget plus the bootstrap prerequisites: `git, pwsh, curl, jq, eza, gh, zoxide`. Workflow-dependent tools (`node, npm, docker, kubectl, terraform, direnv`) move to section 6 (Optional Tools) — present means PASS, absent means SKIP (not FAIL).
2. **Section 3 (Version Match)** gates the entire body on `$env:APPS_HOME` being explicitly set. On Linux that's set by `setup-linux.sh`; on Windows it's not (language toolchains come from winget, not `$APPS_HOME/jdk-VERSION` dirs). When `$env:APPS_HOME` is unset, the section emits a single SKIP line explaining the rationale, instead of 5 individual FAILs for absent `jdk-21.0.4/`, `apache-maven-3.9.4/`, etc.
3. **Section 5 (Environment Variables)** splits the var list into REQUIRED (currently only `DOTFILES_DIR`, set by `powershell/profile.ps1`) and OPTIONAL (`APPS_HOME, JAVA_HOME, MAVEN_HOME, PYTHON_HOME, GO_HOME, MINIKUBE_HOME`). Optional vars emit SKIP when unset (with explanation), not FAIL.

**Empirical result on local Windows after the change**: from 60 PASSED / 17 FAILED / 15 SKIPPED → 60 PASSED / 2 FAILED / 26 SKIPPED. The 2 remaining FAILs are legitimate: `Obsidian CLI not in PATH` (real `setup-windows.ps1` gap — see BUG-013 below) and `Linter lintOnSave disabled` (user-side Obsidian config drift — same cross-OS check as Linux, behavior matches healthcheck.sh).

## Out of scope (deferred to subsequent WIN-002 PRs)

- **Full clean-machine sweep.** Running `setup-windows.ps1` end-to-end on a **fresh** Windows VM (no pre-existing tools, no pre-existing config) and logging every FAIL inline. This needs a separate session on a clean machine; this PR works from the daily-driver Windows install.
- **BUG-013-obsidian-cli-windows-parity** (to be opened in vault): `setup-windows.ps1` does not install the Obsidian CLI (`@vorillaz/obsidian-cli` via npm) the way `setup-linux.sh` does. Surfaced by the remaining `FAIL: Obsidian CLI not in PATH` and confirmed against `setup-windows.ps1` (no npm install block for it).
- **Linter lintOnSave drift fix.** The `FAIL: Linter lintOnSave disabled` is the same check Linux's healthcheck performs and is a user-side state finding (Obsidian config in vault, not in dotfiles). Cross-OS parity says keep the FAIL; user resolves by toggling lintOnSave in Obsidian linter plugin. No script change needed.

## Risks / open questions

**Risk: removing tools from sec 1 hides real install regressions.** Mitigation: the moved tools (node/npm/docker/kubectl/terraform/direnv) all go to sec 6 (Optional Tools), still checked, just with SKIP semantics instead of FAIL. A user who needs Docker and finds it missing still sees the SKIP line — same diagnostic value, less alarm.

**Risk: cross-OS healthcheck divergence.** Mitigation: this is OS-conditional behavior inside a single script, not a fork. `healthcheck.sh` keeps its broader Linux contract (because setup-linux.sh genuinely installs that toolchain); `healthcheck.ps1` reflects Windows install reality. The 12-section numbering parity is preserved (bats parity asserts still pass).

**Open question (resolved): should `gh` be required or optional?** `setup-windows.ps1` installs `gh` via winget. Decision: required (in sec 1 list). Same logic as the other 5 winget-installed tools.

## Acceptance criteria

- [x] `scripts/healthcheck.ps1` sec 1 core list reduced to `git, pwsh, curl, jq, eza, gh, zoxide`
- [x] Moved tools (`node, npm, docker, kubectl, terraform, direnv`) appear in sec 6 optional list
- [x] `scripts/healthcheck.ps1` sec 3 gates entire body on `$env:APPS_HOME`; emits single SKIP line when unset
- [x] `scripts/healthcheck.ps1` sec 5 distinguishes required vs optional env vars; optional unset → SKIP not FAIL
- [x] PSScriptAnalyzer clean (Error+Warning)
- [x] PowerShell parse clean
- [x] Empirical: on this Windows machine, healthcheck FAIL count drops from 17 → 2 (Obsidian CLI gap + Linter lintOnSave drift, both deferred per Out of Scope)
- [ ] CI bats suite still green (no asserts on the moved tool lists; structural greps unaffected)
- [ ] Vault entry BUG-013-obsidian-cli-windows-parity opened (P2, ~20 LOC fix for follow-up)

## References

- Vault: `10_projects/dotfiles/11-tasks.md` "WIN-002-windows-smoke-sweep" backlog entry
- Sibling: WIN-001-healthcheck-ps1 (PR #71) — the artifact this PR scopes
- Discovered: BUG-013-obsidian-cli-windows-parity (opened in same session)
- Reference: `setup-windows.ps1` lines 262-269 (the canonical winget array — source of truth for what Windows core tools are)
