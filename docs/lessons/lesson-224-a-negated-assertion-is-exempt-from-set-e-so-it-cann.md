# Lesson 224 — A negated assertion is exempt from `set -e`, so it cannot fail a test

**Date:** 2026-08-23
**Area:** tests / bats / guards
**Severity:** high — 53 assertions across the suite could not fail, and one was hiding a real violation

## What happened

bash exempts any command preceded by `!` from `set -e`. bats runs a `@test` body
under `set -e` and takes the body's exit status from its **last** command. Put
those together and `! grep -q X file` in the middle of a test is not an
assertion — it is a comment that executes.

```bash
@test "the script names no reviewer" {
    ! grep -q 'coderabbitai' "$SCRIPT"   # violated: passes anyway
    grep -q 'reviewers' "$REGISTRY"      # this line alone decides the verdict
}
```

Measured on 2026-08-23 over `tests/*.bats`: **139** lines whose first token is
`!`, in 30 files. **53** of them were in a position where they could not fail
(not the last statement of their `@test`, and not rescued by an `|| return 1`).
The other 86 worked only because nothing had been appended below them yet.

The proof is a falsification, not an argument. Take one real assertion —
`tests/setup-windows.bats`, "does NOT eager-source load-secrets" — and make it
false by changing the pattern to a string the file certainly contains:

| Form | Result |
|---|---|
| `! grep -qF 'param' "$PS1_SCRIPT"` (old, line 81 of 84) | `ok 1` |
| `refute_grep_fixed 'param' "$PS1_SCRIPT"` (new, same line) | `not ok 1` |

Same file, same position, same falsified claim. One reports green.

This is not theoretical damage. `tests/check-review-attestation.bats` asserted
that `scripts/check-review-attestation.sh` names no reviewer of its own. The
script **did** name one, in a comment. The suite was green for as long as the
violated line was not the last one in its test.

## Why it survives inspection

Three reasons, and each one is worth knowing on its own:

**A bare `false` in the same position IS caught.** The exemption is specific to
the `!` form, so the obvious sanity check — "does `set -e` work in a bats body?"
— answers yes and moves on. The rule is narrower than the check.

**It is the shape a careful person reaches for.** `! grep -q` reads as "assert
absent", which is exactly the intent. Nothing about it looks like a no-op.

**It degrades on edit, silently.** An assertion on the last line works today.
Appending one more assertion below it — the most ordinary edit there is — kills
it, with no diff to the line that died.

There is a fourth, nastier variant. Inside a loop, only the last iteration can
decide anything:

```bash
for plugin in github code-simplifier … feature-dev; do   # 9 plugins
    ! grep -qE "\"${plugin}@…\"" "$DOTFILES_DIR/setup-linux.sh"
done
```

The `for` is the last statement, so it does carry a status — but that status is
the last iteration's. Eight of the nine plugins were unchecked, in a test whose
name promises all nine.

## The fix, and the two vacuous passes hiding inside it

Assertions moved to `tests/lib/refute.bash` (`load 'lib/refute'`), which returns
non-zero explicitly and prints the offending line with its number. Writing it
surfaced two more ways to pass without asserting, both inherited by any hand-
rolled `! grep`:

- **A grep error is not an absence.** An unreadable file or a pattern invalid in
  the chosen dialect exits `2`, and `! grep` reads that as "not found". This
  matters precisely when converting: `(` is a literal in BRE and an unterminated
  group in ERE, so a pattern moved from `grep -q` to `grep -qE` without thought
  starts erroring — and erroring reads as passing. The helper treats any status
  above 1 as a failure.
- **A pattern beginning with `-` is a pattern.** `--vault knowledge` was being
  handed to grep as options. The helper passes the pattern after `--`.

The dialect is in the function name — `refute_grep` (extended regex) and
`refute_grep_fixed` (literal) — so the call site has to say which one it means.
For the non-grep cases, `run cmd` plus an explicit `[ "$status" -eq 1 ]` says the
same thing; prefer `-eq 1` to `-ne 0`, because `-ne 0` also passes on 127, the
status of a helper whose name you typo'd.

## The rule

**Never open a statement in a bats test with `!`.** Not in the middle, not on
the last line — the last line is one appended assertion away from the middle,
and nothing tells you when it moves.

`tests/guard-bats-negation.bats` enforces this over the whole suite. It carries
a quarantine list of the files not yet converted with their **exact** counts, and
fails when a count moves in either direction: a conversion must lower its entry
in the same commit, and a regression cannot hide behind an entry that
over-counts. The list is emptied file by file; when the last entry goes, so does
the list.

The guard also tests its own detector against a fixture with a hand-counted
answer. A guard that silently matches nothing reports a clean suite forever —
the same failure it exists to catch, one level up.

## Relation to Lessons 212 and 220

[212](lesson-212-an-invalid-instrument-is-indistinguishable-from-an.md) — an
invalid instrument is indistinguishable from an absent guard.
[220](lesson-220-four-defects-one-shape-a-thing-verified-by-a-proxy.md) — a
thing verified by a proxy that lives somewhere else, with its diagnostic
question: *what would this still pass on if the thing it checks were broken?*

This is that question answered for an entire suite: **the assertion would still
pass on anything at all.** The instrument was not merely aimed at a proxy — it
had no subject. The generalisable move is the one both lessons end on: falsify
the check deliberately and confirm it goes red. A test never observed failing is
a claim, not a check.

## Evidence

- `grep -rnE '^\s+!\s' tests/*.bats | wc -l` → 139, in 30 files (2026-08-23)
- Falsification pair above: `ok 1` on the old form, `not ok 1` on the new one
- `tests/check-review-attestation.bats` AC5 — the real violation it hid
- Full suite after conversion: 1453 tests, 0 failures
- `#1034` — the ticket
