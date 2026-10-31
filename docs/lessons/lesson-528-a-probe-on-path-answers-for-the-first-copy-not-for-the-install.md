---
id: "lesson-528-a-probe-on-path-answers-for-the-first-copy-not-for-the-install"
type: lesson
status: active
title: "A probe on PATH answers for the first copy, not for the install"
created: "2026-10-31"
---

# A probe on PATH answers for the first copy, not for the install

## Context

On Windows, hive installs itself (`self_install` in `packages.json`). dotf probed it with `hive --version` on PATH,
then on the registered PATH, and ran `hive self-upgrade <pin>` only below the pin. ADR-036's rule called this
"probed where the installer puts it".

## Problem

A box that ran dotf 0.77.0 against main's catalog got a `uv tool` hive-vault at the pin, beside hive's own layout
(#2955). The PATH probe read the uv copy as the install, so the self install never ran, and the box stayed on the
layout hive ADR-019 rules out. #2956 read the same two copies the other way round and deleted hive's own launcher.
Both treated "the first thing PATH runs" as "the thing we installed". With one copy on the machine those are the
same, which is why every test passed.

## Solution

Declare where the installer puts the tool (`self_install.launcher`) and probe that absolute path only (#2991).
Report the other copy in `dotf doctor`, and never delete it: dotf did not install it. Every reader of the version
goes through one function (`tools.ProbeTool`), so converge, `dotf tools version` and the setup scripts agree on a
machine that holds two copies. A test for an install probe needs a second copy on PATH. Without one it cannot tell
the install from whatever happens to come first.
