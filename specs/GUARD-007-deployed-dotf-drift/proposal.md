---
id: "GUARD-007-deployed-dotf-drift"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-01"
issue: "mlorentedev/dotfiles#1158"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# GUARD-007-deployed-dotf-drift

## Why

<!-- from issue #1158: GUARD: nothing detects that the deployed dotf predates the repo, so the gates can run on a binary that is not this tree -->

Every gate in this repository runs through `dotf` — `spec review`, `spec archive`, `doctor`, `pr triage-queue`, `secrets run`, `harness gate` — and nothing establishes that the `dotf` on PATH was built from the tree those gates are meant to defend. `checkOptionalTools` compares the reported semver against the `versions.conf` pin, which answers "is the declared version the pinned one" and not "was this binary built from this tree". `dotf` is unlike every other tool in that list: the others are third-party releases, and `dotf` **is this repository**, so between two releases the version string is constant while the source moves on every merge.

Measured twice. 2026-08-21: the deployed `dotf spec init` created three files where a build of `origin/main` created four, because the binary predated #1127. 2026-09-01: the deployed binary read `0.52.0`, `versions.conf` read `0.52.0`, and the binary was two feature merges stale — it predated #1410, so `dotf doctor` rendered `Results: 158 passed, 2 failed` with the entire `Persona skill enforcement` section **absent**. A report missing a check is indistinguishable from one where the check ran and passed.

## What

`dotf doctor` gains a `dotf provenance` section that answers whether the deployed binary was built from this checkout, and says so when it cannot answer.

The binary carries its own commit: `.goreleaser.yaml` stamps `-X main.commit={{ .FullCommit }}`, and `dotf version --commit` prints that bare value. The default `dotf version` output is unchanged, byte for byte.

Observable outcomes by state:

| State | Report |
|---|---|
| built from a commit current with HEAD for `cli/` | PASS |
| behind HEAD on `cli/` | WARN naming the count and both SHAs |
| built from a commit not an ancestor of HEAD | WARN naming it as a branch build, not as staleness |
| built from a commit this checkout lacks | WARN naming it as unfetched |
| source build (empty stamp) | WARN — legitimate, but provenance unestablished |
| binary predates the `--commit` flag | WARN — provenance unestablished |
| not inside a checkout, or no `dotf` on PATH | SKIP |

## Out of scope

- **Making it a FAIL.** Running a released binary from inside a checkout is legitimate and is what every non-CLI session on this machine does. The defect is the drift being invisible, not the drift. Failing the health command over a normal state trains the reader to ignore the line, which is the failure one layer up.
- **Auto-reinstalling on drift.** `doctor --fix` is not extended. The remedy names itself in the message; performing it would replace a running binary out from under concurrent sessions.
- **Windows/PowerShell parity beyond compilation.** `GOOS=windows go vet` passes; no `.ps1` changes.
- **The stale binary currently on this machine.** Converging it is a separate act (`scripts/install-dotf.sh` after v0.53.0 publishes), tracked in the session handoff, not here.

## Risks / open questions

- **The installers' idempotence skip is the one thing that must not break.** `install-dotf.{sh,ps1}` regex `(\d+\.\d+\.\d+|dev)` out of the *merged* streams of `dotf version` to decide whether to skip. A second line in the default output risks it, and the skip failing open is silent — every machine would reinstall on every setup run. **Resolved:** the commit went behind `--commit`; the default output is pinned to one line by a test.
- **A pre-stamp binary cannot answer.** `dotf version --commit` on an old binary exits non-zero with `unknown flag`. **Resolved:** that is treated as the provenance answer for a pre-stamp binary, not as a broken probe. It is the state on this machine today and is what the check reports there.
- **The check cannot verify itself on the release path until v0.53.0+ publishes**, because the stamp lands only in a goreleaser build. Verified instead against a locally stamped binary (evidence in `verification.md`). Residual risk: a `.goreleaser.yaml` template error would surface at release time. `goreleaser snapshot` runs on every PR touching `cli/**` and covers it.

## Acceptance criteria

- [ ] **AC1** — `.goreleaser.yaml` stamps the full commit, and `dotf version --commit` prints that bare value with nothing else on the line.
- [ ] **AC2** — The default `dotf version` output remains exactly one line beginning `dotf version `, so both installers' idempotence skip is unaffected.
- [ ] **AC3** — `dotf doctor` WARNs, naming the count and both SHAs, when the deployed binary is behind HEAD on `cli/`.
- [ ] **AC4** — Commits that do not touch `cli/` do not count as staleness, and the `cli/` pathspec is root-relative so the count cannot silently read 0 from a subdirectory.
- [ ] **AC5** — A binary that is not an ancestor of HEAD, one built from an unknown commit, a source build, and a pre-stamp binary are each reported distinctly rather than collapsed into "behind".
- [ ] **AC6** — Outside a checkout, or with no `dotf` on PATH, the check SKIPs rather than passing (C15), and it never FAILs in any state.

## References

- Bitácora board: mlorentedev/dotfiles#1158 (In Progress)
- ADR-020 — tooling CLI Go convergence; C15 (a check that cannot answer must say so)
- Sibling shape: `cli/internal/doctor/checks_agent_skills.go` (#1410) — a check whose absence was indistinguishable from a clean result, one layer up
- Related: #1154 (a linter reporting 0 issues on unformatted code) — same shape, one layer up
