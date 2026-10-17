---
id: "ADR-052-one-forge-write-surface-in-dotf"
type: adr
status: accepted
owner: manu
date: "2026-10-16"
supersedes: []
extends: [adr-048-one-permission-baseline-for-every-agent, adr-050-one-lane-contract-for-every-repo-agent-and-forge]
issue: mlorentedev/dotfiles#2635
tags: [architecture, decision, forge, permissions, agents]
created: "2026-10-16"
---

# ADR-052: Agents write to a forge through one `dotf forge` surface, with the body in a file

## Context

Agents reached forges through whatever was at hand: `gh` with the prose on the command line, or ad-hoc Python against a REST API. Measured on 2026-10-16:

- Four Bash calls in one session were refused with "Permission to use Bash with command ... has been denied". All four carried the literal text `dotf secrets show` in a commit or issue body, and the deny rule `Bash(*dotf secrets show*)` matches that text anywhere in a command line (#2634). The same `gh issue create` without the phrase had passed twice. PR #2629 added an allow rule for it on the wrong premise: a deny resolves before an allow, so that rule could not have helped.
- The auto-mode classifier refuses writes to the private Gitea (`gitea.kubelab.live`) as "External System Writes" when they arrive as ad-hoc Python plus a token to an unknown host (the imagesensortool session). The classifier is a Claude Code mechanism; the other harnesses never see it.
- ADR-048 gives every harness one trust and deny baseline and keeps allow lists per harness. It cannot make a new tool reachable by itself: each tool still needs a rule in each dialect.

## Decision

1. **One write surface.** `dotf forge issue comment` today; `issue create|edit` and `pr create|comment|edit` follow in the spec TOOL-030. It is Go in `cli/internal/forgewrite`, behind a `Backend` interface (GitHub through the `gh` REST endpoint, as `lane.REST` reads; Gitea through a REST client). A third forge is a new `Backend` and one entry in `forge/hosts.json`.
2. **The body is only read from a file or stdin** (`--body-file`), never from an argument. Prose cannot reach a permission glob, whatever the harness or its dialect.
3. **No merge, no close, no delete.** Merging stays `dotf pr land` behind the merge grant (ADR-047); closing stays with the owner. A test fails if a verb of that kind appears on the interface or the command tree.
4. **The forge comes from `origin`.** `github.com` is built in; any other host must be declared in `forge/hosts.json` with its kind and the registry secret holding its token. An undeclared host is an error that says how to fix it, never a guess: writing with the wrong forge's token is the failure this refuses to risk.
5. **The token is read in-process at call time** through the secrets resolver, never by running `dotf secrets show`, and every error path redacts it. A test drives 4xx, 5xx, malformed and transport failures and asserts the token is absent from each error.
6. **One narrow allow set per harness replaces a rule per tool.** It is declared with the other allow lists under ADR-048, and the owner lands it (ADR-048 clause 6). `tea` stays a tool for the human, installed through the package catalog; agents do not call it.

## Consequences

- The classifier's answer to a `dotf forge` write to Gitea is unmeasured at the time of writing. The first PR records it on #2635 before any allow rule is added; the dev-binary result is evidence, and the binding measurement is repeated once with the released binary.
- ADR-048's deploy can only add. Reverting #2629 therefore removes the entry from the template but not from machines that already hold it; removing it there is an owner action, recorded in the PR that does it.
- ADR-048's deny intents are rendered into each harness's syntax. The secret-exposure deny must match the command position, not any substring, or the renderer spreads BUG-152 to every harness (#2634).

## Alternatives rejected

- **A per-tool allow rule for `tea` (and one for each CLI after it).** It fixes one forge in one harness, leaves prose on the command line, and needs an OS-specific install.
- **`autoMode.environment` prose naming the Gitea host.** It still passes through the classifier and exists only in Claude; kept as a fallback if the measurement shows the classifier refuses `dotf forge`.
- **Narrowing the deny globs and calling it done.** It fixes BUG-152 and nothing else; the classifier and the per-tool rule sprawl remain.
