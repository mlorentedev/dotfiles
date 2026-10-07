---
id: "lesson-343-a-build-that-skips-on-failure-turns-a-compile-error-into-a-green-job"
type: lesson
status: active
title: "A build that skips on failure turns a compile error into a green job"
created: "2026-10-07"
---

# A build that skips on failure turns a compile error into a green job

## Context
Five bats files ran the real `dotf` binary and each compiled its own copy per file run. Three of
them wrote the build as `go build ... || skip "go build failed"` (`dotf-agent-run.bats`,
`compile-harness-real.bats`, `knowledge-crystallize-go-parity.bats`). The CI duplication audit
(#2059) found them while counting redundant builds.

## The Trap
The `skip` was written for a legitimate case: a shell-only checkout with no Go toolchain, where the
rest of the suite should still run. But the line guards the *build*, not the *toolchain*. A
toolchain that is present and fails to compile the tree took the same exit, so a compile error in
`cli/` turned about thirty tests into skips and the `test` job stayed green. A skip reports as `ok`
in TAP, so nothing on the page said a check had not run. (`cli.yml` still caught the error on PRs
that touched `cli/`, which is why it went unnoticed: the net was elsewhere.) Two sibling files had
already written it correctly (`|| return 1`), so the right pattern existed in the repo and was
not the one copied.

## The Solution
Separate the two causes and give each its own outcome, in one helper rather than five copies
(`tests/lib/dotf-bin.bash`):

- `DOTF_BIN` set: use it, and fail if it is not an executable. CI builds once and exports it.
- No toolchain: skip on a developer machine, fail when `$CI` is set.
- A build that fails: fail everywhere. A tree that does not compile is a defect, not an environment.

`tests/dotf-bin-helper.bats` drives the helper with a fake toolchain and pins each outcome, and a
detector fails any bats file that has a `go build` followed within three lines by a `skip`; its
negative control proves the detector can see the pattern. Reproduce before trusting a fix like
this: drop a syntax error into `cli/cmd/dotf/` and run the files. On `main` they report
`ok ... # skip dotf failed to build`; with the helper they report `not ok`.
