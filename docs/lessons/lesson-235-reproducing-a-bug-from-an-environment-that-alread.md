# Lesson 235 — reproducing a bug from an environment that already works measures the environment, not the bug

**Date:** 2026-08-27
**Context:** #1283 — `pi` reported "No models available" in fresh terminals.
**Category:** shell, PATH, diagnosis, false negatives

## What happened

`pi` failed to start with *"No models available"*. It was intermittent in the
way that matters most: it failed in some terminals and worked in others, with no
edit in between.

Four explanations were proposed, each tested, each discarded:

| Hypothesis | How it was tested | Result |
|---|---|---|
| Stale terminal holding an old function | opened a new terminal, re-ran | "works" |
| Duplicate `pi` under nvm shadowing the real one | `which -a pi`, removed the duplicate | "works" |
| Secrets vault locked, so injection failed | `dotf secrets unlock`, re-ran | "works" |
| Retired `OLLAMA_API_KEY` in the `--only` scope | checked the registry, already fixed | "works" |

Four rounds, four false negatives. **The tell was that they all agreed** — a
system that is genuinely broken does not return "works" to four unrelated probes.

## The actual defect

The agent-wrapper block in `.zshrc` / `.bashrc` is guarded by:

```sh
if command -v dotf >/dev/null 2>&1; then
    pi() { dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi "$@"; }
fi
```

The guard is evaluated at **line ~87**. `~/.local/bin` — where `dotf` lives — is
prepended to PATH at **line ~129**. In `.bashrc` the line before that prepend
*replaces* `PATH` outright.

So the guard passed only when the parent process happened to already export
`~/.local/bin`. A terminal spawned from a configured shell inherited it and
wrapped the agents. A fresh desktop terminal, an IDE, or an ADE did not — the
whole block was skipped, `pi` ran unwrapped without its JIT credentials, and
found no providers.

**No error, because a guard that fails is silence.** `if` evaluating false is
not an exceptional condition; it is the construct working as designed.

## Why the measurements lied

Every reproduction attempt was run from a shell that had already sourced a
working rc file, or was launched by one. **The instrument shared the very
variable under test.** Asking that shell to reproduce a PATH bug is asking a
thermometer held in your fist to measure the room.

The failure only appeared once PATH was *controlled* rather than inherited:

```
$ env -i PATH=/usr/local/bin:/usr/bin:/bin zsh -ic 'type pi'
pi is /home/manu/.nvm/versions/node/v24.16.0/bin/pi      # unwrapped — the bug

$ env -i PATH=/usr/local/bin:/usr/bin:/bin zsh -ic 'type pi'   # with the fix
pi is a shell function                                    # wrapped
```

Two commands, differing only in the rc file, and the ambiguity was gone.

## The fix

Make the guard independent of the thing it runs before:

```sh
if command -v dotf >/dev/null 2>&1 || [ -x "$HOME/.local/bin/dotf" ]; then
```

Only the **guard** is evaluated at source time. The function **body** runs at
call time, when PATH is complete and plain `dotf` resolves normally — so the
PATH block does not need to move, and nothing about ordering has to be
remembered by the next person to edit these files.

## The lesson

**"I cannot reproduce it" is a statement about the instrument before it is a
statement about the system.** When a report says a thing is broken and your
measurement says it is fine, the disagreement is data. Enumerate what your
measurement inherits from its environment — PATH, cwd, env vars, an open
session, a warm cache, an unlocked credential — and neutralise each one before
concluding the report was wrong.

Two corollaries earned here:

**Anything read before the block that satisfies it is a latent ordering bug.**
In an rc file, "before" is invisible: there is no import graph and no compiler to
complain. Grep for guards that reference a command, and check where the PATH
granting it is set.

**The red herring was what made the bug findable.** The duplicate nvm install —
chased for an hour as the culprit — is the only reason `pi` resolved to
*something* when unwrapped. Without it the unwrapped call would have been
`command not found`, which names the problem in its first line. A second copy of
a tool converts a loud failure into a quiet wrong answer.

## Related

- `lesson-227` — a test suite inherits the developer's installed applications,
  and PATH is the door. Same variable, opposite direction: there the inherited
  environment made a test pass that should have failed.
- `lesson-230` / `lesson-231` / `lesson-232` — the "shape is not effect" family.
  This is its diagnostic sibling: a *measurement* whose shape is right and whose
  effect is nil.
