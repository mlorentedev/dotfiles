---
id: "lesson-440-an-explicit-home-must-scope-every-path-resolved-under-it"
type: lesson
status: active
title: "An explicit home must scope every path resolved under it, or a test writes the live file"
created: "2026-10-15"
owner: manu
tags: [harness, bind, env-contract, testing, hermeticity]
---

# An explicit home must scope every path resolved under it, or a test writes the live file

**Context:** `dotf harness bind`, the `records-bind` converge step and doctor's hook-binding check wrote Claude's hooks into `agents.bind[].file`, `.claude/settings.json` joined to HOME. `configs-deploy` writes the same file at `{CLAUDE_CONFIG_DIR}/settings.json` and Claude Code reads it there (#2279). With `CLAUDE_CONFIG_DIR` set elsewhere, bind wrote a file Claude never reads, and converge and doctor checked that same file and reported `hooks current`.

**Problem:** Two traps sat on the obvious fix of spelling `file` as `{CLAUDE_CONFIG_DIR}/settings.json` and resolving it with `env.ResolvePath`.

1. The manifest and the binary ship separately. A released `dotf` older than the fix joins `file` to the home verbatim, so it would have created `~/{CLAUDE_CONFIG_DIR}/settings.json` and reported it current until the next release reached the machine.
2. The first cut resolved the token with `env.ResolvePath` while the cmd tests passed `--home <tmp>`. The developer's shell exports `CLAUDE_CONFIG_DIR=$HOME/.claude` unconditionally (`.zshrc`), and the environment wins the cascade. Four tests therefore rewrote the live `~/.claude/settings.json`, pointing every dotf hook (the gate included) at a temp binary that no longer existed. The installed `dotf harness bind --harness claude` repaired it a few minutes later.

**Solution:** The resolved spelling is a new key, `dst`, that older binaries never read; `file` stays as their fallback. `LoadBindTargets` refuses a `{VAR}` token in `file` or in a retired file, and `TestBindDstMatchesItsFileUnderTheContractDefault` requires both spellings to name one file under the contract default on linux, darwin and windows. Bind resolves `dst` with `deploy.ExpandDst`, the expander configs-deploy uses, and an unresolvable `dst` is an error, never a fallback to `file`. An explicit `--home` other than the user's own resolves through `env.ResolvePathUnder(name, home)`: that home's `machine.json` and contract defaults, without the process environment. A `--home` naming the user's own home (compared with `os.SameFile`, through symlinks) keeps the full cascade, since `CLAUDE_CONFIG_DIR` is where Claude reads there; the PR reviewer caught that case. The test that leaked is now a guard: with `CLAUDE_CONFIG_DIR` pointed elsewhere, a bind under `--home` must write nothing outside it.

**Why:** A flag that names a home promises that everything happens inside it. The process environment describes the real home, so any path resolved through it under an explicit home can escape, and in a developer's shell it escapes to the live file. When a path gains a resolution step, the resolver has to take the same home as the caller. The guard is a test that sets the variable to a temp directory and asserts that directory stays empty; a review will not catch the escape, because the code looks correct. Separately, a manifest key that older binaries read cannot change meaning: put the new meaning in a new key and test that the two keys agree.
