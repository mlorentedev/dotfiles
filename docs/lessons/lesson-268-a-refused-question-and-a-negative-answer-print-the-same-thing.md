# Lesson 268 — A refused question and a negative answer print the same thing

**Date:** 2026-09-04
**Context:** Seven instances in one evening, across two sessions, through five different mechanisms

## What happened

Every one of these was a query whose *failure* and whose *negative answer* produced identical
output, and in every case the negative answer was the thing being looked for — so silence read as
a finding.

### 1. A grep over a command that had already failed

```bash
gh pr checks 1489 | grep -iE 'test-windows|fail|pending'
```

Nothing came back, which reads as "`test-windows` is no longer pending". It was still running.
`gh pr checks` had exited with `GraphQL: API rate limit already exceeded`, and that string matches
none of the three patterns. The grep was searching an error message.

### 2. A watcher reporting the job did not exist

A poll loop on the other session printed `(no test-windows yet)` on each refusal. That phrasing is
a claim about GitHub. What it actually recorded was that the API declined to answer.

### 3. A board query truncated at its own limit

```bash
gh project item-list 1 --owner mlorentedev --format json --limit 200
```

→ `#490 not found on the board`. Raising the limit found it. The tool had capped the page and the
absence was the cap.

### 4. The same query, at 400, answering about a different repository

`--limit 400` returned **exactly** 400 items — the ceiling again — and this time it *did* find a
`#490`: `NET-001: Migrate all Tailscale IP references to MagicDNS`, which is **kubelab's** #490.
The board is cross-repo, so an issue number identifies nothing on its own. First query said "absent"
and was wrong; second said "present, status Backlog" and was wrong in the other direction.

The answer took one call once the question was posed correctly — **ask the issue which project it
is in**, rather than paging the project looking for the issue:

```bash
gh api graphql -f query='{ repository(owner:"…", name:"…") { issue(number:490) {
  projectItems(first:5){ nodes { project { number } fieldValues(first:20){ … } } } } } }'
```

→ `project #1 Bitácora | Status = In Progress`. Enumerating a large set to find one element makes
every limit of the tool look like a property of the data.

### 5. The instrument reporting a different budget than the one being spent

```bash
gh api rate_limit --jq .resources   # → core 5000/5000, graphql 5000/5000
```

…printed at the same moment `gh pr checks` was refusing with *"rate limit already exceeded"*. The
primary REST budget genuinely was untouched; what was exhausted was GraphQL's **secondary** limit,
which that endpoint does not report at all.

So the diligent move — *check whether you are rate limited before believing you are* — returned a
confident, precise, wrong answer. This one is the worst of the seven, because the check was
performed *specifically* to avoid the error, and its output was used to conclude the error had
cleared.

### 6. `${PIPESTATUS[0]}` under zsh

```bash
golangci-lint run 2>&1 | tail -20; echo "LINT_EXIT=${PIPESTATUS[0]:-done}"
```

zsh spells it `$pipestatus[1]` and indexes from 1, so the bash form expands to **nothing** and the
`:-` fallback fired. Printed `LINT_EXIT=done`. This is row 12 of `.claude/CLAUDE.md`'s prohibited
patterns, used by the agent that had read the table.

### 7. An assertion that could not fail

```bash
grep -qE 'warning|fail|action|stale' "$(_go_log)" || true
```

Green forever. Removing `|| true` turned it **red** — and the red was correct: `grep -qE` is
case-sensitive and the log lines are `[WARNING]`. The twin counts with `grep -ciE`; the test had
lost the `-i`. So the guard that made the assertion unfailable was hiding a defect *in the
assertion itself*.

### 8. The SHA's check list, answering about runs that no longer matter

```bash
gh api "repos/OWNER/REPO/commits/$SHA/check-runs?per_page=100"
```

This is the REST workaround for instance 1, and it has its own version of the same flaw: it returns
entries from **every** run on that SHA, superseded ones included. A PR showed `integration` and
`test-windows` as `cancelled` and it read as a failed build; they were an earlier run cancelled by a
later push, and the real run was still in progress.

**`cancelled` from a superseded run and `cancelled` from a timeout are spelled identically**, and
the field that separates them is not in the row. Read the run — `gh run view <id>` for the run you
mean — rather than the SHA's accumulated check list. Found by the peer session on its own PR while
this document was being written.

## The lesson

**Design every query so that "no" and "I could not answer" are different outputs.** They are the
same string by default, and the default is what everyone writes.

The danger is not uniform. It concentrates exactly where a **negative result is the finding**:

- "Is anything still pending?" → nothing printed → *nothing is pending*
- "Is #490 on the board?" → not in the list → *it is not on the board*
- "Did the linter fail?" → no exit code → *it did not fail*
- "Are there issue lines in the log?" → assertion passed → *the behaviour is correct*

In each, the reassuring reading and the broken reading produce identical evidence, and the
reassuring one requires no further work. That asymmetry is what makes this class expensive: a false
positive costs an investigation, a false negative costs nothing today and is discovered later by
someone else.

**Checking the instrument is not enough — the instrument must report on the thing that can fail.**
Instance 5 is the sharp case. `gh api rate_limit` was working perfectly, answering truthfully about
a budget that was not the constraint. A health check aimed at the wrong subsystem is more dangerous
than no health check, because it converts "I do not know" into "I verified".

## What to do instead

- **Separate the exit status from the content, always.** `if ! out=$(cmd); then …; fi` before
  grepping `$out`. A pipeline that greps a command's output has already discarded the one bit that
  distinguishes the two cases.
- **Make the absent case say "absent".** `grep … || echo "(none)"` costs four characters and turns
  a silence into a statement. Every command in this repo's own verification blocks that does this
  is readable; every one that does not is a coin flip.
- **When a limit is involved, prove you did not hit it.** A result of exactly `--limit N` is not
  data, it is the ceiling. Assert `len(items) < limit` or page properly.
- **Ask the specific object, not the containing set.** "Which project is this issue in" is one call
  and cannot truncate. "Is this issue somewhere in the project" is a scan, and every scan has a
  horizon that looks like an answer.
- **Never `|| true` an assertion.** If it is there so the step does not abort, the step is not an
  assertion; move it. Instance 7 is proof the suppressor hides defects in the assertion, not just
  in the code under test.
- **In a shared-quota environment, do not poll at all.** Five sessions on one account share 5000/hr;
  a 30-second watcher is 120 calls an hour that make *everyone's* checks unreadable rather than
  slow. Wait on wall-clock, check once.

## Relation to the neighbouring lessons

`.claude/CLAUDE.md` already carries a table of shell constructs that "fail **silently**" — returning
empty rather than erroring — with the note that *empty reads as a finding*. This lesson is that
same principle one layer up, at the tooling boundary, and instance 6 shows the table's own rows
still being tripped by someone who had read them.

Lesson 267 is the adjacent case: a mutation harness must prove the mutation **landed**, because a
mutation that silently failed to apply produces a passing test — a no-op indistinguishable from a
success. This lesson is its mirror: a query that silently failed to run produces an empty result —
a no-op indistinguishable from a negative. Both are *absence* being read as *information*.

Lessons 265 and 266 are the measurement-shaped members of the same family: a correct measurement
answering the wrong question, and two samples agreeing read as reproducibility. The through-line
across all four is that **the reassuring reading is always the one that requires no further work**,
and nothing in the tooling will ever prompt for the other one.

## Seen again (2026-09-05, #1534)

Not a query this time but a **reason string**. The harness gate's allow path returned
`"all blocking skills consumed"` for three states: every obligation satisfied, every warn-level
obligation skipped, and no severities declared at all. The sentence was never false. "Blocking"
means `enforce: block`, and no persona carries one, so the predicate held in all 11 526 decisions
ever recorded on this machine and discriminated none of them; 123 of 123 `warn` records carried it
beside a non-empty `warned` array.

This is the same defect as instance 7 with the polarity flipped: there a guard that could not fail
hid a broken assertion; here a reason that could not be false hid an inert gate. A vacuous truth is
worse than a wrong answer because it survives review, and four shipped persona records had already
built doctrine on it ("a merged migration cannot be trusted as an enforced one"). The fix is the
same rule as above, applied to output rather than input: **every distinguishable state gets a
distinguishable sentence**, and the test asserts pairwise distinctness rather than three literals
(`TestGateAllowReasonsAreDistinguishable`), so a rephrasing is a chore and a collapse is a failure.
