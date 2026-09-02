# 256 - A cleanup block's own description of what it deletes is not evidence that the thing is dead

**Date:** 2026-09-02
**Area:** setup scripts, harness, guards

## What happened

OPS-040 (#1333) set out to delete ~200 lines of one-time migrations from both
setup scripts. One of them looked like the easiest deletion in the batch:

```sh
# SDD-007 one-time migration: gemini-cli -> agy. Remove the legacy
# ~/.gemini/GEMINI.md identity file so it doesn't linger as an orphan
# pointing to the retired binary. Safe to repeat (no-op if absent).
if [ -f "$GEMINI_HOME/GEMINI.md" ]; then
    rm -f "$GEMINI_HOME/GEMINI.md"
    log_info "Removed legacy GEMINI.md (SDD-007 migration: agy replaces gemini-cli)"
fi
```

Self-describing, dated, obviously finished. Classified "harmless cleanup" from
reading it, and deleted.

Then the targets were probed on the machine, as a formality:

```
$ stat -c '%y %s' ~/.gemini/GEMINI.md
2026-09-01 19:34:11  12029 bytes
```

Not an orphan. **12 KB of harness-generated cross-agent doctrine**, carrying a
`sha256:` provenance marker, written the previous evening.

`harness/manifest.json` declares it, with the reason spelled out:

> `.gemini/GEMINI.md` — *"Antigravity reads global rules from
> `~/.gemini/GEMINI.md` and caps EACH rules file at 12000 characters… The file is
> shared with Gemini CLI, so injection is **append-and-replace-in-place, never an
> overwrite**."*

And `tests/compile-harness.bats` asserts precisely that a user's own rules
written into that file survive a deploy.

So both setup scripts had been `rm -f`-ing a live doctrine deploy target on every
run, while the manifest promised users their content in it would be preserved.
`tests/setup-windows.bats` held a test asserting the removal happened, which is
how it stayed green.

## Why nothing caught it

Only ordering hid the conflict. Setup deleted the file early; the single
`compile-harness.sh --deploy` call near the end rewrote it. The destruction was
invisible as long as nothing changed — a run that stopped in between, a skipped
deploy, or a reordering would have silently dropped whatever the user had put
around the generated region.

Two artefacts agreed with each other and were both wrong: the block's comment
("legacy orphan pointing to the retired binary") and the test that pinned it
("Without explicit cleanup the orphan GEMINI.md lingers"). The comment was
written when it was true — before the harness claimed the path — and neither was
re-checked when the manifest started deploying there. **The description aged; the
code did not.**

## What to do instead

**Probe the target before deleting the cleanup.** Not to decide whether the
deletion is safe — a `rm -f` of an absent file is safe either way — but to find
out what the block is actually pointed at. One `stat` was the entire difference
between "harmless cleanup, deleted" and "live doctrine target, this is a bug".

Concretely, for any block that removes a path:

1. Look at the path on a real machine. Present is a question, not an answer.
2. Ask who *writes* it, not only who reads it. Here the writer was
   `compile-harness.sh`, a subsystem the setup script never mentions.
3. Check whether it is declared anywhere as an output. A manifest, a deploy
   manifest, an SSOT — a file that is someone's declared target cannot also be
   someone else's orphan.

## Guard

`tests/guard-doctrine-target-not-deleted.bats` resolves `.doctrine.deploy[]`
from `harness/manifest.json` and refuses any removal command naming a declared
target in either setup script.

It reads the manifest rather than naming `GEMINI.md`, so a target declared later
is covered the day it is declared. Verified fail-first: red against the pre-fix
scripts at `setup-linux.sh:503`, green after. It SKIPs with its reason when the
manifest yields no targets, per C15 — a check that cannot answer must not pass.

## Related

- #1333 — the purge this was found inside
- #1431 — the other block in the same batch that had never converged, for a
  different reason: it asserts against a proxy (`settings.json`) instead of the
  end-state, so a platform change moved the file out from under it
- Lesson 247 — a pointer into a deletable location keeps every check green until
  the deletion
- Lesson 252 — the dangerous constructs are the ones that return an answer that
  reads as a finding
