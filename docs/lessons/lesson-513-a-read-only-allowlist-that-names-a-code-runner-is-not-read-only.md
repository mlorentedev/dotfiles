---
id: "lesson-513-a-read-only-allowlist-that-names-a-code-runner-is-not-read-only"
type: lesson
status: active
title: "A read-only allowlist that names a code runner is not read-only"
created: "2026-10-22"
---

# A read-only allowlist that names a code runner is not read-only

## Context

`dotf mem resume` renders a thread's `Verify:` commands for the next session and marks any command outside a
read-only allowlist as unverified, because a `MEMORY` file is input the next session did not write (MEMORY-008).
The list matched command prefixes: `git status`, `gh pr view`, and also `go test`, `go vet`, `bats` and
`shellcheck`. It already refused shell control and a `--output`/`-o` argument, and left `cat` and `grep` off
because they take any path.

## Problem

The independent spec review reproduced three commands that rendered as safe: `go test -exec=/tmp/evil-shim ./...`
(the flag was honored and the path executed), `go test --outputdir=/tmp/x`, and `bats /tmp/evil.bats`. A test
runner runs code its arguments choose. A prefix match on its name says nothing about the flag that hands it a
program or the path that names a script outside the checkout. The `-o` denylist had already missed
`--outputdir`, which is the usual fate of a denylist. The reason `cat` was excluded ("it takes any path") applied
to `bats` too, and nobody had applied it.

## Solution

The four code runners now pass only an allowlist of their flags (`-run`, `-count`, `--filter`, ...), so
`-exec`, `-toolexec`, `-vettool`, `--outputdir`, `-cpuprofile` and `--formatter` all render as unverified. Every
positional argument must stay inside the checkout: no absolute path, no `~`, no `..` segment. The segment test
matters, because `./...` contains `..` as a substring and is the most common argument of all. Each reproduced
input is a red case in `TestReadOnlyCommand`, and the controls (`./...`, `-run '^X$'`, `--filter 'a b'`) keep
the rule from over-tightening.

## Rule

An allowlist of "safe" commands is an allowlist of command *shapes*. For a command that executes code chosen by
its arguments (test runners, linters with plugin flags, interpreters), the name proves nothing: allowlist its
flags, contain its paths, and test the escape hatch the reviewer would try first. When a command is excluded for
a reason ("it takes any path"), check which listed commands share that reason.
