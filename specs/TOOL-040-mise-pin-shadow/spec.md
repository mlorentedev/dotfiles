---
id: spec-TOOL-040-mise-pin-shadow
type: spec
status: implementing
created: "2026-11-14"
owner: manu
issue: "mlorentedev/dotfiles#3524"
---
# TOOL-040: name the mise config that shadows a pin

## Intent

mise resolves a tool from the highest-precedence config that pins it. When a file that outranks
`~/.config/mise/conf.d/dotfiles.toml` pins another version, `dotf converge`'s `tools` step installs
dotfiles' pin, mise keeps running the other one, and the step fails with only
`after mise install, not running at their pin: node, python`. On ace2 (2026-11-14) the cause was
`~/.config/mise/config.toml`, written by kubelab's `dev_node` role (kubelab#3559); finding it took
`mise config ls` and `mise ls` by hand, and every later step, `lanes` and `git-hooks` among them, was
skipped.

## Scope

When the post-install pin check fails, the sync asks `mise ls --json <tool>`, one call per failing
tool, which config resolved it. The active entry's `source.path` is compared with the file the sync
owns. A different file is a shadow, and the failure then names that file, the version it pins
(`requested_version`, else `version`) and the fix:

```
after mise install, not running at their pin: node, python; /home/ace/.config/mise/config.toml pins node 24.19.0, python 3.12.13 and outranks /home/ace/.config/mise/conf.d/dotfiles.toml: remove the pin there, or let that file's owner stop pinning a tool versions.conf owns
```

`dotf doctor` reports the same shadowing from its plan, as a FAIL with a `FIX:` line in place of
`(run: dotf tools sync)`, which cannot fix it. Both go through one function, `MiseSync.ShadowNote`.

mise is asked only about tools that already fail, so a clean machine runs no extra command. A failed
`mise ls`, output that is not the array above, no active entry, or dotfiles' own file as the active
one (compared through symlinks) leaves the message exactly as before: the check never claims a cause
it did not read.

Out of scope: whether a shadowed tool should fail only itself instead of stopping every later step.
That is a separate decision (#3524).

## Consequences

- The error text of the converge `tools` step and the doctor `Pinned CLIs (mise)` FAIL changes only
  when a shadow is found; anything that matches the leading `not running at their pin: <tools>` still
  matches.

## Acceptance criteria

1. Table-driven tests through the sync's runner seam (no real mise): shadowed by a config outside
   `conf.d`, one of two tools shadowed, dotfiles' own file active (genuine mismatch), unparseable
   output, no active entry, and `mise ls` failing. Only the first two name a file; the rest keep the
   old message and name no cause.
2. A doctor test: shadowed gives one FAIL with the file, the version and `FIX:`; unparseable keeps
   `run: dotf tools sync`.
3. `ShadowNote` asks mise nothing for a plan with no missing tool, and treats a symlinked spelling of
   dotfiles' own path as not shadowed.
4. `go run ./cmd/dotf doctor` on a machine where nothing shadows stays clean.
