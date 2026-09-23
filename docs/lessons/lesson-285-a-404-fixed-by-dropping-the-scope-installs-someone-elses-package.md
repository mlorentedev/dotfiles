---
id: lesson-285
type: lesson
status: active
created: "2026-09-23"
owner: manu
tags: [lesson, supply-chain, npm, tooling, silent-failure, verification]
---

# 285 — A 404 "fixed" by dropping the scope installs someone else's package

## What happened

Setup installed `@vorillaz/obsidian-cli` from npm, and it returned 404. The fix
(#122, recorded as correct in lesson 065) dropped the scope and installed
`obsidian-cli`. The name was one character group shorter and the install succeeded.
`obsidian --version` printed `obsidian-cli v0.5.1`, a semver, so the catalog's
version probe parsed it and every later check passed. When OPS-042 moved it into
`packages.json`, `dotf tools install` began installing it on every machine,
unconditionally.

For four months, nobody looked at what it was. It is an unrelated 2020 tool from
a different author that "imports given test results into Obsidianqa.com", and it
registers a global `obsidian` binary. The `obsidian` this repo actually drives is
the CLI built into the Obsidian desktop app. The npm copy never ran for real on the
Linux box only because the desktop AppImage came first on PATH. Where npm's `bin`
comes first, it shadowed the real CLI. It surfaced only because `dotf doctor`
warned that `obsidian` resolved from two directories, and the advice attached
("remove the other channel's copy") would have deleted the desktop launcher.

## Why it happens

A scope is part of a package's **identity**, not a spelling detail. `@a/x` and `x`
are two different packages, owned by two different people, and nothing connects
them. Every check that ran asked "does something answer to this name?":
`npm view`, the install exit code, a semver in `--version`. Each of them was true.
None of them asked "is this the software we meant?". The version probe was the
most convincing false signal: output shaped like a version reads as confirmation
even when it is a stranger's version.

## The rule

When a package name 404s, the fix is to find **where the intended software is
actually published**: the project's own README or releases page. It is never the
nearest name that resolves. Before a registry name enters an installer or a
catalog, confirm its identity: the author or repository matches the project, and
`--help` describes the tool you meant. An install that succeeds and a `--version`
that parses prove only that *something* is there.

This is typosquatting's mechanism, arrived at by accident. The hazard does not
depend on anyone's intent: whatever is published under the resolving name gets
global install rights on every machine.

## Guard

`tests/setup-linux.bats` refutes any `obsidian` / `obsidian-cli` npm entry in
`packages.json`, and it goes red when the old entry is restored. It pins this
incident only. It is not a general identity check, because the catalog carries no
field a test could compare a package's identity against.

Refs: #1615, lesson 065 (the fix this corrects), OPS-042 (amended AC1).
