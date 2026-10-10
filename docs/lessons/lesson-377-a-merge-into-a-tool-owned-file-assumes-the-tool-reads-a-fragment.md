---
id: "lesson-377-a-merge-into-a-tool-owned-file-assumes-the-tool-reads-a-fragment"
type: lesson
status: active
title: "A merge into a tool-owned file assumes the tool reads a fragment"
created: "2026-10-09"
---

# A merge into a tool-owned file assumes the tool reads a fragment

## Context
The owner sized the Mac's Colima VM at 4 CPU and 8 GiB, declared from the repo (#2013 P5b). The
obvious shape was a deploy entry merging `cpu` and `memory` into `~/.colima/default/colima.yaml`,
the way herdr's `config.toml` is merged: the repo owns two keys and colima owns the rest.

## The Trap
A merge into a missing destination writes the fragment alone, and colima does not read a fragment.
Its loader calls a config empty when `runtime` is unset (`Config.Empty()` is `Runtime == ""`), so on
first start it ignores the file, boots at the 2 CPU / 2 GiB defaults and saves a full file over the
fragment. A fragment that also names `runtime` is read, but then every key it omits, `disk`
included, is a zero value rather than a default: the repo would have to restate colima's defaults
to avoid that. Nothing in the deploy tests could show this, because they check the file that was
written, not what the tool makes of it.

## The Solution
Doctor owns the size instead. Colima writes its own file on first start, and *Colima VM size*
compares the declared values with that file and with `colima list --json`. `--fix` rewrites the two
top-level lines in place and reads the result back. It then restarts the Homebrew service and
reports the repair only once the VM reports the declared size. A YAML node round trip was tried
first and dropped colima's blank lines, so the edit is line-level and keeps the file byte for byte.

## Takeaways
- **Before declaring a merge entry, read how the tool loads a partial file.** "Merge" assumes that
  missing keys mean the tool's defaults; a loader can instead read them as zero, or the whole file
  as empty.
- **A fix that restarts something proves the restarted state, not the restart.** A stopping VM can
  still answer `docker info`, so the engine answering does not show the new size.
