---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - CLI-007-dot-spec-init

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (test name or observed behavior). See `features.json` for the executable verification per feature.

- [x] Scaffold with open issue sets `issue: "dotfiles#N"` + Why comment (exit 0) -> `TestSpecInitWithOpenIssueSetsFrontmatter` (cmd), `TestRenderSubstitutesAndFixesIssueFrontmatter` (spec)
- [x] Missing/closed/nonexistent gate -> non-zero, no spec dir, names the gate -> `TestSpecInitMissingGateFails`, `TestSpecInitClosedIssueFails` (cmd); `TestGateClosedIssueFails`, `TestGateMissingIssueFails` (spec)
- [x] `--force-no-gate` scaffolds without `gh`, leaves `issue: ""` -> `TestSpecInitForceNoGateScaffolds`
- [x] Feature-id grammar parity (sub-id `SDD-012b`, date form; rejects junk) -> `TestValidateID`
- [x] No-clobber + archive warning -> `TestScaffoldWritesFilesAndGuardsClobber`, `TestScaffoldWarnsOnArchivedID`
- [x] Templates embedded; renders with no vault -> `TestRender*` (read `//go:embed` FS, not `$VAULT_PATH`); smoke binary scaffolded in a bare tempdir
- [x] Drift guard fails on divergence, skips when vault absent -> `TestEmbeddedTemplatesMatchVault` (RUN+PASS locally; `t.Skip` path documented for CI)

## Test status

- Go suite: `cd cli && gofmt -l . && go vet ./... && go test ./... && go build ./...` -> all green (`ok cmd/dot`, `ok internal/cmd`, `ok internal/spec`; `internal/review` unchanged, no test files)
- Drift guard: `go test ./internal/spec/ -run TestEmbeddedTemplatesMatchVault -v` -> `--- PASS` (vault present locally; embedded templates byte-identical to vault SSOT)
- Manual smoke: built `dot`, ran `dot spec init DEMO-001-smoke --force-no-gate` in a bare tempdir (no vault) -> scaffolded 3 files, `created:` stamped, `issue: ""` left empty (no fabricated issue without a verified gate)
- No regressions: existing `cmd/dot` + `internal/cmd` (version, review, root) tests unchanged and green; only `root.go` gained one `AddCommand` line

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Templates embedded via `//go:embed`, vendored byte-identical from the vault SSOT** (user decision). A Go test (`TestEmbeddedTemplatesMatchVault`) is the drift guard; it `t.Skip`s when `$VAULT_PATH` is absent because the dotfiles CI has no vault (ADR-013). The skip is the Go analogue of the bats teardown-skip trap fixed in PR #357 — written so a skipped run is never failure-shaped.
- **`issue:` frontmatter set only when a number is known, Why comment only when a title is known.** Under `--force-no-gate` (no `gh` call) the number may be absent, so the frontmatter stays `issue: ""` rather than fabricating an unverified link — strictly better than the shell, which never set it at all.
- **`RepoRoot` walks up for `.git` in pure Go** (file *or* dir) instead of shelling to `git rev-parse`, so it works inside worktrees and needs no git in tests.
- **Clock injected via a package `now` var** so `created:` is deterministic in tests.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? no — the embed+drift-guard-skips-in-CI pattern is recorded in this spec and the test's own comment; it is structurally guarded, not just remembered
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — ADR-020 governs the CLI; embedding templates is an implementation choice within it
- [ ] New pattern candidate for `00_meta/patterns/`? maybe — "vendor SSOT into the binary + drift guard that skips where the SSOT is absent" could recur (every future `dot` twin that reads vault assets). Revisit on the second occurrence; not promoting on n=1.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-007-dot-spec-init/` -> `specs/archive/CLI-007-dot-spec-init/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
