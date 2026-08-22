# Lesson 221 — An allow-list merge makes every new template key a silent no-op

**Date:** 2026-08-22
**Area:** setup / settings deployment
**Severity:** medium — config that reads as deployed but never was

## What happened

`ai/claude/settings.json` carried `env.CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS = "1"`.
The deployed `~/.claude/settings.json` had **no `env` key at all**. The flag had
never been live on any machine that was not bootstrapped from scratch.

The cause is `merge_claude_settings` in `setup-linux.sh`: its per-key policy is
an explicit **allow-list**. It merges `model`, `effortLevel`, `outputStyle`,
`permissions.allow`, `hooks.SessionStart`, `hooks.SessionEnd` and
`enabledPlugins`. Anything else in the template is dropped on the floor, without
a warning, on every existing installation. `setup-windows.ps1` has the same list
as an if-chain, with the same hole.

This is the **second** time this class landed. The first was `outputStyle`, and
the code comment written after that incident describes the mechanism exactly.
The mechanism was documented; the recurrence was not prevented.

## Why the existing guard missed it

`tests/claude-settings-template.bats` did have a test named *"every
dotfiles-owned top-level key is named in both merge policies"*. It iterated a
**hand-written** list:

```bash
for key in model effortLevel outputStyle; do
```

and its comment asserted that the structured keys — naming `env` explicitly —
"have their own merge semantics asserted below and are handled by name in both
scripts". That sentence was false for `env`, and nothing checked it. The test's
name promised template-wide coverage; its body covered three keys somebody had
remembered to type.

This is [Lesson 220](lesson-220-four-defects-one-shape-a-thing-verified-by-a-proxy.md)'s
shape again: **a thing verified by a proxy that lives somewhere else.** The
proxy was a literal list in the test; the thing was the template. They drift
apart the moment anyone edits the template, which is the only moment that
matters.

## The fix

- Both merge policies now handle `env` (object merge, so a machine-local flag
  survives) and `advisorModel` (template wins).
- The guard derives its key list **from the template**:

```bash
while IFS= read -r key; do
    if [ "$key" = "\$schema" ]; then continue; fi
    grep -qF -- "\$tmpl.$key" "$DOTFILES_DIR/setup-linux.sh" || return 1
    grep -qF -- "ContainsKey('$key')" "$DOTFILES_DIR/setup-windows.ps1" || return 1
done < <(jq -r 'keys[]' "$SETTINGS_TEMPLATE")
```

Adding a key to the template now fails CI until both policies name it. Proven
fail-first: injecting a `someNewKey` into the template makes the test fail with
`setup-linux.sh merge policy never mentions 'someNewKey'`.

`$schema` is exempt **by name** — a stated decision, not a gap.

### The guard's first version had the defect it was built to prevent

Review caught it. The greps above originally scanned the **whole scripts**, and
`setup-windows.ps1` says `$tool.ContainsKey('Version')` in its winget loop, far
from `Merge-ClaudeSettings`. So a template key named `Version` would satisfy the
Windows half of the guard while the merge ignored it. Measured, on that exact
state — template carries `Version`, `merge_claude_settings` handles it,
`Merge-ClaudeSettings` does not:

```
OLD GUARD (whole-file grep): PASSES   <-- the hole
NEW GUARD (scoped to bodies): not ok 5
                              # Merge-ClaudeSettings never mentions 'Version'
```

The fix extracts both merge function bodies with `sed` and greps inside those.
The point is not the `sed`: it is that **a guard is a claim about a specific
region of code, and grepping a whole file quietly widens the region until the
claim is no longer the one you meant.** Writing a test against the thing you
care about is not enough; it has to be scoped to it, or the surrounding file
answers on its behalf. Same shape as the bug in this lesson, one level up.

## The generalisable rule

**A test that enumerates what it checks cannot catch the item nobody thought to
enumerate.** Where the set under test is data that lives in the repo, derive the
set from that data. A hand-written list is a second source of truth whose only
job is to agree with the first one, and it will stop agreeing silently.

Ask of any guard: *what would this still pass on if the thing it checks were
broken?* This one passed on a template with an undeployable key in it — which is
precisely the state it was named after preventing.

## Related

- The concrete flag that surfaced this: `CLAUDE_CODE_ENABLE_EXPERIMENTAL_ADVISOR_TOOL`,
  which Claude Code reads from its **own** process environment (`settings.env`
  is merged into `process.env` at startup). Without it the `/advisor` command is
  hidden and `advisorModel` is read but never attaches — a second silent no-op
  stacked on the first.
- `model` sits in the same file with a `template wins` policy, so every setup
  run resets whatever `/model` last saved. A 1M-context default chosen
  interactively survived exactly until the next deploy. The template now carries
  `opus[1m]` so the durable record matches the intent.
