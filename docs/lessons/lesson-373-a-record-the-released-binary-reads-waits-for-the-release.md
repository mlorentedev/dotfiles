---
id: "lesson-373-a-record-the-released-binary-reads-waits-for-the-release"
type: lesson
status: active
title: "A record the released binary reads waits for the release"
created: "2026-10-09"
---

# A record the released binary reads waits for the release

## Context
#2062 makes Python >= 3.11 a hard dependency of `dotf`, provided through mise. The plan was
one PR: teach `ParseMisePins` a second marker, `# mise: python-package`, and mark
`PYTHON_VERSION` and `PYYAML_VERSION` in `versions.conf` in the same change. Every test in the
tree passed, because the tests build `dotf` from the tree.

## The Trap
The machines do not run the tree's `dotf`. Setup and doctor run the binary `DOTF_VERSION` pins,
and that binary reads the tree's `versions.conf`. Up to v0.65.0 its parser rejects every
`# mise:` comment except `# mise: cli`, by design: a near-miss marker must not silently mark
nothing. So the new marker would have turned every machine's `dotf tools sync` and
`dotf doctor` into a parse error, from the merge until a release carrying the new parser was
the pinned one.

Marking only `PYTHON_VERSION` was not safe either. The old binary would install mise's Python
without the packages, and its shim would shadow the system Python that the suite imports
`yaml` from.

Neither failure shows in CI: the integration job and the bats suite both build `dotf` from the
tree under test.

## The Solution
Two PRs. PR1 ships the reader, the sync and the doctor check with `versions.conf` byte-identical.
PR2 marks the pins once `DOTF_VERSION` carries the parser. `tests/versions-conf.bats` refuses
the new marker, and a marked `PYTHON_VERSION`, until PR2 relaxes it. The order is a test, not
a promise.

## Takeaways
- **Before changing a record (`versions.conf`, `packages.json`, a deploy manifest), ask which
  binary reads it.** If it is the released `dotf`, the change waits for a release that
  understands it. The same rule split P5a from P5b (#1814 class).
- **The record need not be new to break the old reader.** A plain `# mise: cli` pin, a
  marker every release reads, broke the same way for kubectl, helm, argocd and four more
  (#2013): the released binary probes `--version`, which they reject, so the sync fails
  naming them. Ask what the reader *does* with the entry, not only whether it parses it.
- **A strict parser is what makes the window dangerous.** Rejecting unknown input is right,
  and it means a new keyword is a breaking change for every older reader.
- **mise refuses `python@3.12.6`.** It verifies GitHub artifact attestations for
  python-build-standalone by default, and builds that old have none. Pick a pin with
  attestations (3.13.16 here); do not turn the verification off.
