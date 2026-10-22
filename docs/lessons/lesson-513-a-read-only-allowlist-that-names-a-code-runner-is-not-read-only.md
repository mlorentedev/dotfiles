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

The first fix allowlisted each code runner's flags and kept its paths inside the checkout. The next review defeated
it twice: `go test ${IFS}-exec=/tmp/evil` reached the shell as `-exec=` because the check read the literal, and the
listed commands without a flag list still took their writing and interactive flags (`dotf doctor --fix`,
`gh pr view --web`). A quoted `'-exec=/tmp/x'` would have passed as well, since it does not begin with `-`. Each fix
had defended the claim at a narrower point than the one the shell acts on.

What closed it was making the claim only where it is decidable. A command renders as read-only only when it is a
listed prefix followed by plain words: every character in `[A-Za-z0-9_./ -]`, no word starting with `-`, and no
absolute path or `..` segment. A string of those characters contains nothing the shell interprets, so the shell
splits it exactly where `strings.Fields` does, and the check reads what will run. Flags, quotes, `$`, `~`, `=` and
`:` all render unverified, and the session's permission gate still runs them. The cost is real and accepted:
`go test ./... -run X` and `gh pr view 1 --json state` no longer render as safe.

## Rule

An allowlist of "safe" commands is an allowlist of command *shapes*, and a shape is only checkable where the
checker and the shell agree on how to split it. Restrict the verified set to strings with no shell metacharacter,
and refuse flags outright instead of listing the safe ones: every escape hatch three reviews found was a flag or a
character the shell interprets. When each fix narrows the hole instead of closing it, stop patching and move the
claim to where it can be decided.
