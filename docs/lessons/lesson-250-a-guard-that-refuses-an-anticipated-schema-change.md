# 250 - A guard that refuses an anticipated schema change should name its own fix, because the person who reads the refusal is the one who caused it

**Date:** 2026-08-31
**Area:** harness, guards, spec-driven development

## What happened

`specs/HARNESS-046/check-roster-consistency.py` compares each persona's declared
skills against `ROSTER.md`. It parsed `skills:` with `^skills:\s*\[(.*?)\]` and
fell back to `[]`.

HARNESS-045 was going to introduce a second frontmatter form carrying per-skill
severity:

```yaml
skills:
  - id: audit
    enforce: warn
```

Under that form the regex finds nothing, `[]` is returned, and the guard compares
*"declares no skills"* against the roster — reporting **exit 0** while checking
nothing. The drift guard would have been disarmed by a schema change it never
noticed. This was foreseen on 2026-08-27 and the guard was changed to **raise**
on a `skills:` key that is present but unparseable, rather than return `[]`.

Four days later the migration landed. Measured: the pre-change guard, run from
`git show HEAD:` against the migrated vault, exits 1 and names the record. It
refused instead of passing silently, which is exactly what it was built to do.

## Why this is the lesson and not just a bug

The refusal was the cheap half. The half that mattered was that its message said
**what to do instead**:

> Teach this guard the new form, or use `dotf harness resolve-skills`.

and its docstring said **why not to do the obvious thing**:

> Parsing the block form here would make this the THIRD hand-rolled frontmatter
> reader in the repository; refusing loudly instead means whoever migrates the
> definitions has to come back and do it once, in `dotf harness resolve-skills`,
> where the parser already lives.

The migrator hitting that refusal is not a stranger debugging a mystery. They are
the person who just made the change that caused it, and they are the only one in
a position to fix it correctly. A refusal that says only *"cannot parse"* sends
them to the fastest local repair — a second regex — and the repository ends up
with the fourth reader the comment was trying to prevent. The message is the
design decision's only surviving carrier at the moment it is needed.

Here it worked as written: the fix was to delegate to
`dotf harness resolve-skills`, which reads both forms through a real YAML parser,
and no new parser was written.

## What this does not license

**Delegation must relocate the loud failure, never dissolve it.** The rewritten
guard raises on a non-zero exit, a missing binary, and a timeout alike; nothing
degrades to `[]`. An unreadable skill list and an empty one produce the same
downstream behaviour and mean opposite things — the whole reason the exception
type exists. Verified against the migrated form: a planted divergence exits 1, a
planted unparseable key exits 1 naming the YAML error and its line, and a restore
exits 0.

Delegating also improved the diagnostic by accident: the old refusal said "a form
this guard cannot read", the new one surfaces the parser's actual complaint with
a line number.

## Rule

When a guard is taught to refuse a schema change you intend to make, write the
refusal for the person who will trigger it — name the sanctioned fix and the
alternative you are ruling out, in the message and not only in the commit that
added it. Then, when you make the change, verify the refusal fires against the
new form before you replace it: a guard update nobody proved was necessary is
indistinguishable from one that quietly removed a check.
