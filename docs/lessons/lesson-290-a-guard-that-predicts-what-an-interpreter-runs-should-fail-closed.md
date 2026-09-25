---
id: lesson-290
type: lesson
status: active
created: "2026-09-24"
owner: manu
tags: [lesson, secrets, security, guard, shell, verification]
---

# 290 — A guard that predicts what an interpreter runs should fail closed, not emulate it

## What happened

`dotf secrets run` refuses a shell snippet that dumps the environment, such as
`sh -c env`. To inspect the snippet, the guard has to know which argument the
shell will run as its command string. Four adversarial reviews in one day each
found a way the guard got that wrong:

1. It read the argument after anything containing `c`. `bash -c -- env`,
   `bash -c -i env` and `bash +c env` all ran uninspected.
2. It was replaced by a POSIX option parser: options end at `--` or the first
   operand, `-o` consumes an argument, `+c` sets c. Then `zsh -ovi -c env` ran,
   because zsh bundles `-o`'s argument into the flag. So did `bash -c + env`,
   because bash skips a bare `+`.

Each fix was correct for the case it named, and each review found the next
dialect's rule. bash, zsh and dash do not share one option grammar, so
emulating "the shell" was always one rule short.

## Why it happens

A guard that must predict an interpreter's behaviour is a second implementation
of that interpreter. It is correct only where it matches every real one, and
its failures are silent: a wrong prediction does not error, it lets the
command through. The reviewer, meanwhile, needs only one real shell that
disagrees.

## The rule

1. **Do not decide what you cannot decide exactly; inspect every candidate.**
   Once any argument sets the c flag, the guard inspects every argument after
   it. Which one the shell runs no longer matters, so there is no grammar left
   to get wrong. The cost is refusing a word the shell would only pass on as
   `$1`. For a tripwire, that is the right direction to err.
2. **Test against the real interpreter, and assert the property, not the
   mechanism.** `TestSnippetGuard_RefusesEveryShapeTheRealShellRuns` runs each
   argv shape through the installed bash, zsh, sh and dash, proves the shell
   executes the marked operand, and then requires the guard to refuse that
   shape. A new bypass becomes one more row, found by the shells rather than by
   belief.
3. **Say what the guard is.** Three rounds rated bypasses as Blockers because
   the spec never said the guard was a tripwire and that the redactor protects
   the values. Stating the boundary in the proposal lets a review measure the
   guard against what it claims.

Refs: SEC-001, #1626. Relates to lesson 289 (a list tested by looping over
itself cannot see a missing member) and lesson 287 (a guard that skips is a
guard that passes).
