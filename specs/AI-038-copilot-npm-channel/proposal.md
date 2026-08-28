---
id: "AI-038-copilot-npm-channel"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-28"
issue: "mlorentedev/dotfiles#1321"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, copilot, install-channels, adr-036]
template_version: "1.0"
---

# AI-038-copilot-npm-channel

## Why

ADR-036 listed `copilot` under "no cross-OS channel" and left it to the OS
package manager: `setup-windows.ps1` installed it through winget with no pin,
`setup-linux.sh` installed nothing (detect-and-act), and `ai/copilot/config.json`
left `autoUpdate` on its default. Measured on the Windows work box on
2026-08-27: GitHub publishes the CLI as `@github/copilot` on npm (1.0.81), the
binary had self-updated behind winget's registry (1.0.78 recorded vs 1.0.80
running), no `dotf tools version` row existed, and the shadowed-install WARN
could not apply because the tool was not in the catalog. Issue #1321; decision
taken by the owner on 2026-08-27: npm on both OSes.

## What

`copilot` is a `packages.json` catalog tool (`npm`, `@github/copilot`, pin as
floor) that `dotf tools install` converges on every OS. The winget row is gone
from `setup-windows.ps1`; both setup scripts keep only the config-deploy block,
which runs when the binary is on PATH. `ai/copilot/config.json` sets
`"autoUpdate": false` so the pin owns updates. `dotf doctor` gains a
"GitHub Copilot CLI" section: version probed once in Go, matched against the
catalog pin (PASS at or above, FAIL below, SKIP when absent), and a leftover
winget/scoop copy is reported by the existing shadowed-install WARN. ADR-036's
table is amended with the date and the measurement.

## Out of scope

- The BUG-003 "no auto-install on Linux" policy for boxes without Node.js: the
  npm channel needs Node, which `setup-linux.sh` does not provision yet (#1312).
  Until then Linux boxes without Node keep the SKIP.
- `requires_command: copilot` gates in the harness manifest, lesson 157 and
  `verify-setup.bats`'s "not deployed when absent" case: still true where the
  binary is absent; only their prose was refreshed.
- Copilot's own `config.json` key audit (#1322) and the `-NoProfile` env gap
  (#1324).

## Risks / open questions

- A box with both the winget and the npm copy resolves whichever PATH entry
  wins; doctor WARNs, the operator removes the winget copy (ADR-036 §4/§5). On
  the work box the winget copy was removed as part of verification.
- `copilot --version` prints "GitHub Copilot CLI 1.0.80." — a trailing dot after
  the semver; `tools.ProbeVersion` takes the first semver in the output.

## Acceptance criteria

- [x] AC1 — `packages.json` declares `copilot` (npm, `@github/copilot`, 1.0.81) and no
  setup script carries a copilot install block (winget row removed; Linux
  comment/message name the catalog).
- [x] AC2 — `ai/copilot/config.json` sets `autoUpdate: false`; the bats guard pins it.
- [x] AC3 — `dotf doctor` reports copilot: PASS at/above the pin, FAIL below, WARN on a
  version-less output, SKIP when absent; one row per branch, asserted by status.
- [x] AC4 — ADR-036's table moves `copilot` to the npm class with a dated amendment.
- [x] AC5 — on the Windows work box: `dotf tools install` installs the npm copy,
  `dotf tools version copilot` reports it, doctor WARNs on the leftover winget
  copy until it is removed, then reports the pin match; `copilot -p` answers.

## References

- Bitácora board: #1321
- ADR-036 (install channels), AI-034 (#1294, the opencode precedent), CLI-029 (`packages.json`, `dotf tools`)
