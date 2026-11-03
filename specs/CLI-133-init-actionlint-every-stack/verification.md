---
tags: [spec, verification, templates]
created: "2026-11-02"
id: "CLI-133-init-actionlint-every-stack"
type: spec-verification
status: active
owner: manu
---

# Verification - CLI-133-init-actionlint-every-stack

## Evidence

- [x] AC1 -> `TestWriteLintForEveryStack` (go, python, node, ts, none), `TestInitDryRunListsLint`; `dotf init . --stack none --dry-run` prints `lint [ok] lint.yml created` and `ci [skipped] ci.yml none`
- [x] AC2 -> `TestWriteLintFollowsForgeAndDefaultBranch` (Gitea on `master`: trigger and actionlint directory), `TestWriteLintSkipsAnExistingLintInAnyForgeDir`
- [x] AC3 -> scaffolded `--stack none`: `actionlint -shellcheck=shellcheck .github/workflows/*.y*ml` exit 0; `check-actions-pinned.sh` copied into the scaffolded repo prints `OK (1 workflow files)`, and exits 1 on an unpinned copy (negative control)
- [x] AC4 -> `TestLintTemplatePinsMatchTheRepository`; mutating `ACTIONLINT_VERSION` to 1.7.11 fails it

## Test status

- Test suite: `cd cli && go build ./... && go vet ./... && go test ./... -count=1` -> every package ok; `GOOS=windows go vet ./...` ok; `golangci-lint run` 0 issues
- Manual smoke test: `dotf init` built from this branch, run in empty directories for `none` and `go`, dry-run and real
- No regressions in existing test suite: yes (the existing `TestWriteCI*` and forge tests pass unchanged after the `writeWorkflow` extraction)

## Decisions made during implementation

- The template names the workflow directory for actionlint instead of keeping ci-workflows' no-argument call, because actionlint's default is `.github/workflows` only and a Gitea or Forgejo repository would lint nothing (lesson 547).
- The SHA-256 of the actionlint tarball cannot be tied to `versions.conf` offline; a bump without it fails loudly at `sha256sum -c`, and the test's message says to update both.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-547-actionlint-with-no-arguments-lints-only-github-workflows.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the owner's choice of option B is recorded on #2965; no architecture changed
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: one project, one template

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-133-init-actionlint-every-stack/` -> `specs/archive/CLI-133-init-actionlint-every-stack/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
