---
id: lesson-190-a-bash-case-pattern-is-a-glob-not-a-regex-g-a-z-do
type: lesson
status: active
created: "2026-08-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 190: A bash `case` pattern is a glob, not a regex — `g[a-z]*` doesn't mean what it looks like it means

**Context**: BUG-045, widening `tests/shell-alias-collision.bats`'s g-namespace collision guard past its original 1-4-char cap. The replacement was written as `case "$name" in g[a-z]*) ...` with a comment describing it as "matching any all-lowercase `g[a-z]+` token, uncapped" — regex notation, reasoned about with regex semantics, and never run before being described that way.

**Problem**: bash `case` patterns are shell globs. `g[a-z]*` parses as `g` + exactly one character from the class `[a-z]` + `*` (anything, including nothing). Two consequences the regex reading misses entirely: `g` alone no longer matches (a glob needs at least one more character after the class; the intended "one or more" is what `+` means in a regex, not what a bare `[a-z]` means in a glob), and the trailing bare `*` matches *anything* after that second character — digits, hyphens, uppercase — not just more lowercase letters, so "all-lowercase" was never true of what the pattern actually did. The plugin ships `alias g='git'`, so the missing `g` case was a real coverage regression, not a theoretical one — caught by an advisor review before the branch was pushed, not by running the test, because the mutation that would have caught it (temporarily defining a bare `g()`) hadn't been tried yet.

**Solution**: `g|g[a-z]*` restores the single-character alternative explicitly; the comment was reworded to describe glob semantics ("matches a hyphen, digit or uppercase right after the first letter too") instead of regex ones, and the fix was mutation-tested in both directions (`g()` and a long new g-prefixed name each independently turned the test red, then green again after revert).

**Rule**: never describe or reason about a `case`/`[[ ... ]]` pattern using regex vocabulary (`+`, `\d`, "one or more") — bash glob character classes have no repetition operator; `[a-z]` is exactly one character, and reaching for "more of the same" needs an explicit second alternative or a second bracket, not an intuition carried over from `grep -E`. When a pattern is rewritten to be "broader," mutation-test the boundary that changed (the shortest and longest members of the class you widened) before trusting the new range covers what the comment claims.

**Tags**: `shell`, `bash`, `testing`
