---
id: "TOOL-039-forge-pr-draft-ready"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-13"
issue: "mlorentedev/dotfiles#3406"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# TOOL-039-forge-pr-draft-ready

> **Naming**: file lives at `<repo>/specs/TOOL-039-forge-pr-draft-ready/proposal.md`. `TOOL-039-forge-pr-draft-ready` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3406: fix(forge): pr create --draft opens a ready PR on Gitea -->

The "draft while iterating" rule tells every agent to open a pull request as a draft and mark it ready when the change is finished, so reviewers read the finished change rather than each push. `dotf forge pr create --draft` refuses on Gitea (TOOL-030), so on a Gitea-hosted repository agents drop `--draft` and open ready pull requests (three on 2026-11-12), and there is no `dotf forge` verb to mark a draft ready on either forge: agents call `gh pr ready`, which exists only for GitHub. The owner chose one verb for both forges (option A on #3406, 2026-11-14).

## What

- `dotf forge pr create --draft` opens a draft on Gitea too. Gitea has no draft field on create: a draft is a title that starts with one of the instance's work-in-progress prefixes (default `WIP:` and `[WIP]`, case-insensitive). dotf prefixes the title with `WIP: ` and reads the `draft` field of the pull request Gitea returns (present on the instance in use, Gitea 1.27.3, measured 2026-11-14). If Gitea answers `draft: false`, the instance does not recognise the prefix: the command fails with ERROR/WHY/FIX that names the pull request it opened, so the caller knows it exists and is ready.
- A new verb, `dotf forge pr ready <number>`, marks a draft ready on both forges and prints its URL.
  - GitHub: REST can create a pull request with `draft: true` but cannot clear it; the only call that does is the GraphQL mutation `markPullRequestReadyForReview`, which `gh pr ready` also uses. The backend reads the pull request's `node_id` and `draft` over REST through `gh api`, and sends the mutation through `gh api graphql`, the same transport the backend already uses.
  - Gitea: the backend strips a leading `WIP:` or `[WIP]` (case-insensitive, with the spaces after it) from the title through the pull request edit endpoint and reads `draft` back; still `true` means the instance uses another prefix, and the command fails saying so.
  - A pull request that is already ready is not an error on either forge: nothing is written and its URL is printed.

## Out of scope

- Switching the skills and the harness text from `gh pr ready` to `dotf forge pr ready`. That edits `harness/**`, which is the owner's; it is prepared as a follow-up ticket, not landed here.
- Reading an instance's configured `WORK_IN_PROGRESS_PREFIXES`: Gitea exposes no API for it. The read-back of `draft` is the verification instead.
- Converting a ready pull request back to a draft.

## Risks / open questions

- On Gitea the draft is verified after the pull request exists: an instance with custom prefixes ends with a ready pull request whose title starts with `WIP: `, and an error that says so. Accepted: no API reads the prefixes beforehand, and failing loudly with the URL is the honest outcome.
- The GraphQL mutation draws on the GraphQL quota parallel sessions exhaust (#2710). One call per pull request, the same call `gh pr ready` makes today, so the load does not change.
- No live Gitea write in CI: Gitea is exercised through an `httptest` server that answers like 1.27.3. The real instance is checked once by hand on a scratch pull request before the spec archives (verification.md records it).

## Acceptance criteria

- [ ] AC1: On Gitea, `pr create --draft` sends the title prefixed `WIP: ` and succeeds when the answer says `draft: true`; when the answer says `draft: false` it fails with an error that names the pull request URL.
- [ ] AC2: On Gitea, `pr ready <n>` removes a leading `WIP:`/`[WIP]` prefix through the edit endpoint and succeeds when the answer says `draft: false`; still `true` fails; an already-ready pull request is printed without a write.
- [ ] AC3: On GitHub, `pr ready <n>` reads `node_id` and `draft` over REST and sends `markPullRequestReadyForReview` with that id through `gh api graphql`; an already-ready pull request is printed without the mutation.
- [ ] AC4: `dotf forge pr ready` exists, takes exactly one positive number, and its help names both forges; `pr create --help` no longer says `--draft` is GitHub-only.

## References

- Issue #3406, decision comment of 2026-11-14 (option A).
- `specs/TOOL-030-forge-write-surface/proposal.md` (the `--draft` refusal this replaces), ADR-052.
- `cli/internal/forgewrite/github.go`, `cli/internal/forgewrite/gitea.go`, `cli/internal/cmd/forge_write.go`.
