---
id: "CLI-133-init-actionlint-every-stack"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-02"
issue: "mlorentedev/dotfiles#2965"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# CLI-133-init-actionlint-every-stack

## Why

<!-- from issue #2965: CLI-133: dotf init has no stack for a workflows-only repository and emits no actionlint -->

`dotf init` lints nothing a repository's workflows say. `ciTemplates` maps only `go`, `python`, `node` and `ts`, so `--stack none` gets no CI at all, and no stack gets a workflow linter, although every scaffolded repository holds workflows from its first commit (`ci.yml` itself). Scaffolding `mlorentedev/ci-workflows` showed it: its `lint.yml` (actionlint) had to be written by hand. The owner chose option B on #2965 (2026-11-02): actionlint for every stack, rather than a new `workflows` stack.

## What

- Every `dotf init`, any stack and `none` included, writes `lint.yml` beside `ci.yml` in the forge's workflow directory (`WorkflowDir`). It runs actionlint with shellcheck over every workflow, on `pull_request` and on push to the default branch, with `permissions: contents: read`.
- The template is the `lint` job of `mlorentedev/ci-workflows`' `lint.yml`: actionlint installed from its release tarball and checked against a pinned SHA-256, and `actions/checkout` pinned to a commit SHA with its tag as a comment.
- Skip-if-present, like `ci.yml`: an existing `lint.yml` in any forge's workflow directory is left alone and reported `skipped`.
- The init report gains a `lint` row (`lint.yml created|skipped`), and `--dry-run` lists it without writing.
- One place for the pins: a test fails when the template's `ACTIONLINT_VERSION` differs from `versions.conf`, or when its `actions/checkout` SHA differs from the one this repository's own workflows use.

## Out of scope

- The structure step still creates `src/`, `tests/` and `scripts/` for every stack, which does not fit a workflows-only repository. Option B does not address it: #3093.
- The existing `ci-*.yml` templates use tag refs (`actions/checkout@v5`), not SHA pins. Pinning them changes what three stacks emit: #3094.
- A `test` job like the one in ci-workflows' `lint.yml`: that job runs that repository's own bats suite.

## Risks / open questions

- **The checksum cannot be checked offline.** A test can tie `ACTIONLINT_VERSION` to `versions.conf`, but not prove `ACTIONLINT_SHA256` is that release's checksum. A version bump without the checksum fails loudly at `sha256sum -c` in the generated repository, never silently; the drift test's message says to update both.
- **Gitea and Forgejo.** `lint.yml` goes to the same `WorkflowDir` as `ci.yml`, and uses only `actions/checkout`, which Gitea runners resolve from GitHub by default. Same exposure as `ci.yml` today; no new risk.
- **A repository with no workflow besides these two** still lints them, so the job always has something to check.

## Acceptance criteria

- [ ] AC1: `dotf init . --stack <s> --dry-run` on an empty directory lists `lint.yml` for every stack, `none` included.
- [ ] AC2: `dotf init` writes `lint.yml` with the trigger on the repository's default branch, and leaves an existing `lint.yml` in any forge's workflow directory untouched.
- [ ] AC3: the emitted `lint.yml` passes `actionlint` and `scripts/check-actions-pinned.sh`.
- [ ] AC4: a test fails when the template's actionlint version differs from `versions.conf`, or its checkout SHA from this repository's workflows.

## References

- Issue #2965 and the decision comment on it.
- Reference workflow: `.github/workflows/lint.yml` in `mlorentedev/ci-workflows` (ADR-042).
- `cli/internal/initrepo/ci.go` (`WriteCIOpts`, `WorkflowDir`, `DefaultBranch`).
