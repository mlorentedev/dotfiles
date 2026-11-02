---
id: "BUG-110-spec-home-repo-resolver"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-01"
issue: "mlorentedev/dotfiles#1906"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-110-spec-home-repo-resolver

## Why

`dotf spec audit` cannot run in a repository whose origin is on Gitea. It reports "no origin remote" even when there is one, and `--repo` must be typed on every run (#1906, observed in `teledyne/projects-toolkit`). The obvious fix, parsing the Gitea URL into `owner/name`, is wrong by construction: the audit asks GitHub through `gh api repos/<slug>/…` with no hostname, so a Gitea slug would be looked up on github.com. Three commands also decide "which repository holds our issues" three different ways. `spec init` reads `--bitacora-repo`, then `DOTF_BITACORA_REPO`, then origin. `spec audit` reads `--repo`, then origin. The doctor check reads origin only.

## What

One resolver in `package spec` answers two separate questions for every caller (`spec init`, `spec audit` and the `spec-issue-state` doctor check):

- **Issue home:** the GitHub `owner/name` that completes bare `#N` links and hosts the work-gate issue. It comes from the first of these that is set:
  1. the command's flag (`--bitacora-repo` on init, `--repo` on audit);
  2. a committed `specs/.bitacora-repo` holding one `owner/name`;
  3. `DOTF_BITACORA_REPO`;
  4. a github.com origin.

  If none applies, the command fails closed. The error names the origin's forge class (`gitea`, `other` or `absent`) and its host, never the URL, which may embed a token. It also names the two ways to declare the issue home.
- **Archive home:** the GitHub `owner/name` that holds the specs themselves, which is the github.com origin or nothing. On any other forge, the zombie-archive check reports that it could not be asked. It never queries github.com for a repository that does not live there.

A malformed `specs/.bitacora-repo` is an error that names the file, never a silent fallback. This matches `specs/.wip-limit` (ADR-043).

## Out of scope

- Asking a Gitea API for issue state or archive contents. Issues stay on GitHub, the bitácora (ADR-018), and the archive check on a non-GitHub forge stays "could not be asked".
- Any other `OriginRepo` caller: `init`'s GitHub defaults and `secrets sync`. They configure GitHub itself, so a GitHub-only parse is correct for them.
- Detecting Gitea by URL shape. `initrepo.ClassifyOrigin` already classifies, and its classification is reused as-is.

## Risks / open questions

- **Precedence of the committed file over the env var.** Decided: the file wins. `DOTF_BITACORA_REPO` is machine-wide, and a value exported for one repository must not redirect another that has declared its own home. A repository without the file behaves exactly as today.
- **Behaviour change for `spec audit`.** It now reads `DOTF_BITACORA_REPO` when no flag and no file are set. Previously it ignored the variable and used origin. This is the drift being removed. Before release, check whether any machine exports the variable and runs `spec audit`.
- **Archive home on a GitHub origin with an issue home elsewhere.** Today the archive lookup uses the issue home. Under an override that is the wrong repository, so the archive home is always the origin.

## Acceptance criteria

- [ ] AC1: In a repository with a github.com origin and no override, `spec init`, `spec audit` and the doctor check resolve the same issue home and archive home as today. The existing tests stay green.
- [ ] AC2: With a Gitea origin (HTTPS form and SSH form) and no override, `spec audit` and `spec init` exit non-zero before any forge call. The message names the forge class and the host, not the URL, and names `--repo`/`--bitacora-repo` and `specs/.bitacora-repo`. The doctor check WARNs with the same reason.
- [ ] AC3: `specs/.bitacora-repo` holding `owner/name` sets the issue home for all three callers. A malformed file (empty, two values, not `owner/name`) is an error naming the file.
- [ ] AC4: Precedence is flag, then file, then env, then origin, and is proven by one table test on the resolver. A test shows `spec init` and `spec audit` agree on the issue home for the same repository and environment.
- [ ] AC5: With a Gitea origin and an override, `spec audit` grades every spec against the override. A zombie keeps FAIL with "could not be checked", and no `gh api repos/<gitea-slug>/contents/…` call is made. The fake runner records every call, and the test asserts none names the Gitea slug.
- [ ] AC6: With a github.com origin and an override naming another repository, the zombie-archive lookup queries the origin, not the override.

## References

- Bitácora board: mlorentedev/dotfiles#1906 (BUG-110)
- Related ADR: `docs/adr/adr-043-a-wip-limit-on-active-specs.md` (the `specs/.wip-limit` precedent), ADR-018 (issues live on the GitHub bitácora)
- Code: `cli/internal/cmd/spec_audit.go`, `cli/internal/cmd/spec.go` (`workGateRepo`), `cli/internal/doctor/checks_spec_issue_state.go`, `cli/internal/initrepo/forge.go` (`ClassifyOrigin`)
