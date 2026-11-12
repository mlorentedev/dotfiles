---
id: "BUG-169-reproduce-control-run"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-12"
issue: "mlorentedev/dotfiles#3333"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-169: a probed reproduction needs the clean head as its control

## Why

<!-- from issue #3333: pr reproduce: a check that fails before reaching the test reads as a confirmed finding -->

`dotf pr reproduce --probe` classifies the check by its exit alone. A check that fails before it reaches the code under test exits 1 the same way the defect does. Examples are a wrong directory, a package that does not exist, or a missing fixture. The record then reads `reproduced`, a confirmed finding that `dotf pr land` trusts. That happened for #3325: `go run -C cli` moved the check into `cli/`, where `go test -C cli` failed with `chdir cli: no such file or directory` and exit 1. The test never ran.

## What

When the check reproduces under `--probe`, `pr reproduce` runs it once more on the clean head as its control (the issue's option A). The reproduction stands only when the control refutes it. A check that reads the same both ways is `could-not-run`, and its output says the control read the same and shows the control's output. The printed row names the control's reading and exit. A refutation runs no control, and a run without `--probe` is unchanged, since a committed failing test has no clean head to compare with.

## Out of scope

- Option B, refusing on output signatures such as `chdir` or `no such file`.
- A new state name. `could-not-run` already means "no verdict", and `prtriage` reads only the `reproduced` and `did-not-reproduce` rows.
- The `pr-review-triage` skill text (`harness/**`, owner-landed). The command's help now says where the check runs.

## Risks / open questions

- The control costs one more run of the check, and only for a probed reproduction. Resolved: accept, as the issue does.
- A flaky check could pass the control and still be wrong. That was already true of one run, and the control only removes false confirmations. Resolved.

## Acceptance criteria

- [ ] AC1: With `--probe`, a check that fails both with and without the probe is `could-not-run`, its output names the clean-head reading, and the tree is left clean. The same holds for an inverted check (`--defect-when-passes`) that passes both ways.
- [ ] AC2: A check that fails only with the probe stays `reproduced`, with control `did-not-reproduce`. A refutation runs no control.
- [ ] AC3: The row printed for the triage names the control's reading and exit when a control ran, and nothing about a control otherwise.

## References

- #3333, #3325 (the incident), TOOL-017 (reproduce), #2926 (inverted checks)
- `cli/internal/reproduce/reproduce.go`, `cli/internal/cmd/pr_reproduce.go`
