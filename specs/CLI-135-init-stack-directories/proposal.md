---
id: "CLI-135-init-stack-directories"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-04"
issue: "mlorentedev/dotfiles#3093"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# CLI-135-init-stack-directories

## Why

`dotf init`'s structure step creates `src/`, `tests/` and `scripts/` for every stack, because its directory list is fixed and never consults `--stack`. A repository whose content is GitHub Actions workflows (`mlorentedev/ci-workflows`) has no use for `src/` or `scripts/`, and they were removed by hand after scaffolding it (#2965). The owner chose option (A) on #3093: each stack declares its directories, in the table that already picks its ci template.

## What

- The stack table becomes the one place a stack is defined: its ci template and its code directories. `ciTemplates` and the fixed code half of `scaffoldDirs` fold into it.
- A new stack, `workflows`, for a repository of reusable workflows and composite actions. It gets `tests/`, because ci-workflows keeps bats shape tests there. It gets no `src/`, no `scripts/` (logic lives in `dotf`, ci-workflows ADR-001 D1) and no `ci.yml`. `lint.yml` (actionlint) is still written for every stack, as it is today.
- `go`, `python`, `node`, `ts`, `none` (the default) and an unknown stack keep exactly the directories they get today. Nobody's default changes.
- The directories every stack shares (`specs/`, `docs/` and its subfolders, `.claude/`) are unchanged.
- `StackInit` treats `workflows` as a no-op, as it does `none`, not as an unknown stack. The `--stack` help lists it.

## Out of scope

- The workflows stack's CI shape is #3163. It covers a bats `test` job, a shellcheck pass over composite actions, a release-please caller pinned by SHA and a final `gate` job. All of it references ci-workflows rather than copying it.
- Validating `--stack` values. An unknown stack keeps today's behaviour.
- Changing anything in ci-workflows. This was coordinated with its sessions on 2026-11-04: they keep ci-workflows, and this spec stays in dotfiles.

## Risks / open questions

- `ScaffoldOpts` gains a `stack` parameter, so `orchestrator.go:53` changes. #3155 (CLI-129, another session) also edits `orchestrator.go`. Whichever lands second rebases a one-line conflict.
- No open questions.

## Acceptance criteria

- [ ] AC1: each stack's code directories are pinned by a test, one row per stack, `workflows` included.
- [ ] AC2: `dotf init --stack workflows` creates `tests/`, `specs/`, `docs/` and `lint.yml`, and creates no `src/`, `scripts/` or `ci.yml`.
- [ ] AC3: `--stack none` (the default), every existing stack and an unknown stack create `src/`, `tests/` and `scripts/`, as before.
- [ ] AC4: `StackInit(root, "workflows")` performs nothing and reports no "unknown stack" skip.

## References

- Issue: mlorentedev/dotfiles#3093 (owner's choices, 2026-11-04: option A, as a new `workflows` stack)
- Origin: #2965 (actionlint for every stack). Follow-up: #3163 (the workflows stack's CI)
- Layout facts: the ci-workflows sessions, 2026-11-04
