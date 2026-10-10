---
id: "lesson-391-a-re-planning-probe-needs-every-outcome-the-apply-skips"
type: lesson
status: active
title: "A re-planning probe needs every outcome the apply skips"
created: "2026-10-10"
---

# A re-planning probe needs every outcome the apply skips

## Context
`dotf converge` gained the `packages.json` catalog in its `tools` step, so a Mac, where no setup
script runs, installs its brew packages, release binaries and npm and uv tools. A converge step's
probe re-runs the plan after the apply and fails if anything is left to do.

## The Trap
`tools.Installer` already skipped two cases at apply time without telling its plan: an apt package
behind a sudo password (it prints the command and moves on) and an npm tool with no npm on `PATH`.
The plan still said `install` for both. A probe built on that plan reads every such skip as an
unfinished install, so each unattended Linux `dotf update` would have gone red on a state converge
cannot change. #1892 was the same gap for refusals.

The opposite trap sat in the test double. The fake installer put a tool on `PATH` the moment it
installed it, so the step looked converged. On a fresh Mac, `dotf` runs by absolute path with no
rc deployed: mise lands in `~/.local/bin` and uv in mise's install dir, and neither is on the
process's `PATH`. The step then reported OK without running the sync, left the uv tools "waiting
on uv", and a second run reported 0 changes. That would have made the from-zero CI check pass
without testing anything (lesson 328).

## The Solution
- The plan knows every outcome the apply skips: `missing-manager` covers npm as well as uv, and
  `needs-sudo` asks the apply's own classifier (`sudo -n true`, asked once per installer) up front.
- The probe tells a real wait from a PATH gap. A wait is accepted only when nothing on this machine
  installs the manager: npm, until something installs node. A manager the tool layer does install
  (mise from the catalog, uv as a mise pin) that is still out of reach fails the probe and is named.
- The second pass walks only the entries that waited, so a failure is attempted and reported once.
- Tests include a machine where an installed tool does not land on `PATH`.

## Takeaways
- A step whose probe re-plans inherits every disagreement between plan and apply. Before wiring a
  probe, list what the apply skips and make the plan name each case.
- A wait is legitimate only for a dependency nothing will provide. Once the system provides it,
  "still waiting" is a defect, so report it as one.
- A fake that puts a tool on `PATH` as soon as it installs it hides the fresh-machine case. Model
  where the tool lands separately from what the process can reach.
