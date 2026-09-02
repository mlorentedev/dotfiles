---
id: "CLI-070-doctor-next-steps"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-02"
issue: "mlorentedev/dotfiles#1442"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-070-doctor-next-steps

> **Naming**: file lives at `<repo>/specs/CLI-070-doctor-next-steps/proposal.md`. `CLI-070-doctor-next-steps` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

A setup run that ends with `dotf doctor` exiting non-zero hands the operator the scrolled check list and nothing else. Both `setup-windows.ps1` and `setup-linux.sh` print a hardcoded "Next steps" block at the very end of every run, unrelated to whatever doctor actually found. Observed live on 2026-09-02: after converging a stale Windows checkout, doctor FAILed with `Bitwarden session is gone ... run \`bw login\`` — a one-line fix already spelled out in the FAIL message — but the run's final output said nothing about it; the remedy was buried under ~40 other check lines the operator had to scroll back through.

## What

After `Results:`, `dotf doctor` prints a `Next steps:` block listing the remedy command from every FAIL line whose message names one (`run`, `re-run`, `recover with`, or `upgrade with` followed by a backtick-quoted command — the convention 34 of the ~38 backtick-carrying FAIL/WARN messages in the package already follow). Each command is listed once, in first-seen order. A run with no such FAIL prints no block — unchanged output shape from today.

## Out of scope

- Changing `Report`'s message API (no severity/hint field added to `Fail()`/`Warn()`) — the block is built by scanning the already-rendered transcript, not by adding structured data to every one of the package's ~100 `Fail()` call sites.
- WARN remedies. Next steps is about what is actually driving the non-zero exit; a WARN is advisory by the package's own contract (`Status` doc comment in `report.go`) and stays visible in the check list itself.
- A FAIL whose remedy is phrased without one of the four verbs above (e.g. prose that doesn't say "run"). That is a message-authoring convention gap, not something this feature's parser chases; fixing individual messages to use the convention is a separate, unbounded cleanup.
- Rewriting either setup script's own hardcoded "Next steps" block. Both still print their generic first-run orientation (restart the shell, `project-init ...`) below doctor's output; this feature is additive, not a replacement, since removing them was never proposed or reviewed as part of this issue.

## Risks / open questions

- **Coupling to message wording.** The extraction is a regex over free text, so a FAIL message that's reworded to drop its remedy verb silently stops appearing in Next steps with no compile-time signal. Accepted: the alternative (a structured hint field on every check) is a much larger change for a benefit — catching a wording regression at compile time — that a doctor-package code reviewer already catches by eye, since the message and the regex live in the same package.
- **Severity depends on check internals the feature doesn't control.** `checkBitwardenReach` downgrades its unauthenticated-session message from FAIL to WARN when zero registry secrets are bw-backed (`live == 0`). On a machine with bw-backed secrets it's a FAIL and appears in Next steps; on one with none, it's a WARN and correctly does not. This is intentional (severity should scale with actual impact) and not a defect of this feature, but it means "does bw login show up in Next steps" is not a fixed answer — noted so a reviewer doesn't read it as a gap.

## Acceptance criteria

- [x] AC1: `dotf doctor` output whose transcript contains a `[FAIL]` line matching `(run|re-run|recover with|upgrade with) \`<cmd>\`` prints a `Next steps:` block after `Results:` listing `<cmd>`.
- [x] AC2: A `[WARN]` line carrying the same pattern does NOT produce a Next-steps entry.
- [x] AC3: A `[FAIL]` line with a backtick span that is not introduced by one of the four verbs (e.g. a line that only *references* a command, like `` `bw status`: unauthenticated ``) does NOT produce a spurious entry from that span.
- [x] AC4: The same command appearing in more than one FAIL line is listed once.
- [x] AC5: A clean run (no FAIL) prints no `Next steps:` block — output byte-identical in shape to today for that case.
- [x] AC6: `--quick` mode is covered by the same logic with no special-casing (it already skips every section that could produce a FAIL doctor would want surfaced here; the wiring sits after `Summary()` regardless of mode).
- [x] AC7: Color output (`isColorEnabled`) is unaffected — tee-ing the report writer into a transcript buffer must not turn off ANSI color on a real terminal run.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related: `PI_VERSION` / pin-floor fix landed separately as its own PR (#1441) — split out because it was an independent, obvious-cause bug fix bundled with this feature in the same investigation session; kept apart per AGENTS.md's Discipline Gate ("was this one change or two").

<!-- archived 2026-09-02 — PR: https://github.com/mlorentedev/dotfiles/pull/1443 -->
