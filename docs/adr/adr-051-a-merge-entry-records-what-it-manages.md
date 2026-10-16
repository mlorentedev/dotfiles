---
id: "ADR-051-a-merge-entry-records-what-it-manages"
type: adr
status: proposed
owner: manu
date: "2026-10-16"
supersedes: []
extends: [adr-012-deploy-strategy-copy-with-drift-assertion]
issue: mlorentedev/dotfiles#2570
tags: [architecture, decision, deploy, merge, state]
created: "2026-10-16"
---

# ADR-051: A merge entry records what it manages, and retires only what it still finds as it wrote it

## Context

A `strategy: merge` entry in `ai/deploy.json` owns the keys and list elements its source declares and nothing else (#2013 H4). The destination is shared with the tool that reads it: herdr saves its Settings screen into `config.toml`, Copilot rewrites `config.json`, and the user edits all of them by hand.

When a key or element leaves the source, the merge has no way to tell "the repo stopped managing X" from "the user added X". Both look the same: a value in the destination that the source does not name. So nothing was ever removed. A herdr binding deleted from `ai/herdr/config.toml` stayed bound on every machine (#2570). `listKeys` (#2455) replaces an edited element in place, but a removed one went unnoticed.

The owner set the constraint on 2026-10-16: the record is the decision, meaning where it lives and how a lost record degrades. A missing record must mean "remove nothing", never "remove everything not declared".

## Decision

1. **Where it lives.** Each merge entry records the source it last merged in `<StateDir>/deploy/merged/<name>.json` (`env.StateDir`: `$XDG_STATE_HOME/dotfiles` or `~/.local/state/dotfiles`). The record holds `{version, dst, source}`. `source` is the text of the source after `paths` rendering, decoded with the codec that merged it, so a TOML int64 or datetime compares exactly. The record is per machine, as the destination is. It holds repo content and resolved local paths only, because a merge entry cannot render secrets (`ParseManifest` refuses `merge` with `render`).
2. **What gets removed.** A value the record holds and the current source no longer declares is removed from the destination only while the destination still holds the value the record has (`jsonEqual`). One the user changed since is theirs and stays. This holds for:
   - table keys, recursively; a table the removal empties goes too;
   - union-list elements, by value;
   - `listKeys` elements, by identity. An element whose identity is still declared is replaced in its slot by `mergeByIdentity`, not removed.
3. **How a lost record degrades.**
   - These all mean remove nothing, and the run writes a fresh baseline: no record, a record for another `dst`, or a record version this binary does not read.
   - A record that exists and does not parse is an error naming the file. Its message says that deleting the file means "remove nothing, record again on the next run". Reading it as no record would be safe, but would hide that the history is gone.
4. **When it is written.** After a merge lands, and also when one is already in sync. The in-sync write is what lets a box deployed before records existed, or a fresh box, take its baseline on its first run: a no-op deploy writing a file is deliberate. A dry run writes nothing. The write is atomic, and an unchanged record is not rewritten.
   - If the destination was installed but the record write failed, the error says "deployed …, but could not record". The operator then knows the deploy landed and only the next retirement is at risk.
5. **One predicate.** `dotf deploy`, converge's `configs` reconciler and doctor's deploy check all read the same records (`deploy.RunOptions.Records`, `deploy.PlanConfigWith`). A retired key is drift in doctor exactly when deploy would remove it.
6. **A state directory that cannot be placed** fails a run only when one of its targets merges (`deploy.MergeRecords`). A replace-only run behaves as it did before records existed.

## Consequences

- Removing a key, a table or a binding from a merge source now converges every machine on its next deploy, with no manual step.
- "Managed" means "declared by the source at a past deploy on this machine". Suppose a union-list element or a key the user already had, with the same value, before the repo declared it. It is indistinguishable from one the repo added, so when the repo retires it, it is removed. This is the accepted edge of the equality rule, not a defect: the alternative, recording the destination's prior state, would need a snapshot of user data in dotf's state.
- The first deploy after upgrading retires nothing. It records the baseline, and only a later source change removes anything. So a value removed from a source before a machine's first recording run stays on that machine until removed by hand: no record ever held it. Measured on 2026-10-16, herdr has none (no binding has ever left `ai/herdr/config.toml`, and the Mac's only binding is the declared `prefix+d`). A Claude Code permission rule dropped from the template earlier may linger; the ones Claude Code rejects are removed by `dotf doctor --fix` (#2350).
- Deleting `<StateDir>/deploy/merged/` is always safe. It costs one round of retirements, never removes a value.

## Alternatives considered

- **Record in the destination** (a marker key or comment). Rejected: a JSON tool that rewrites its file drops unknown keys or comments, Copilot's header included, and the record would be lost silently on the tool's next save.
- **Record in the repo** (the previous source from git history). Rejected: the deployed state is per machine, and a box that skipped commits would retire against the wrong baseline.
- **Remove whatever the source does not declare.** Rejected by the owner's constraint: it deletes every user key on a merge destination.
