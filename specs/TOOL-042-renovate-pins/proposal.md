---
id: "TOOL-042-renovate-pins"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-15"
issue: "mlorentedev/dotfiles#3284"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# TOOL-042-renovate-pins

## Why

<!-- from issue #3284: Bump packages.json and versions.conf pins automatically with Renovate -->

`versions.conf` and `packages.json` are the version SSOT, and nothing moves them. On 2026-11-15 a machine ran `mise upgrade --bump` by hand: 28 tools left their pins and `~/.config/mise/config.toml` shadowed three of them, so the box no longer matched the repository (lesson 605). The fleet already runs Renovate centrally (`mlorentedev/ci-workflows`, weekly, autodiscover over `mlorentedev/*`, `RENOVATE_REQUIRE_CONFIG: required`), and this repository has no config, so it is skipped.

## What

- A root `renovate.json` extends the fleet preset and enables only two custom managers: a regex manager for `versions.conf` and a jsonata manager for `packages.json`. Dependabot keeps gomod, github-actions and docker; the managers do not overlap.
- Every `*_VERSION=` pin in `versions.conf` carries a `# renovate:` line above it (above the `# mise:` marker when there is one) naming its datasource, package and update group, or `# renovate: skip <reason>` for a pin Renovate must not move.
- Pin bumps arrive as reviewed PRs, grouped so that no group can exceed the spec-gate threshold, and `dotf update` converges machines to the merged pins.

## Out of scope

- Changing `scripts/check-spec-gate.sh` so a version-only diff skips the gate. That widens a gate and is the owner's to decide; it is raised on #3284 as an option.
- Moving `PYTHON_VERSION`: about 55 venvs and the `hive-vault` uv tool point at `python/<version>`, and the tools-step orphan prune holds nothing. Ticketed separately.
- The `dependabot[bot]` actor guards in `.github/workflows/pr-agent.yml` and `add-to-project.yml` (owner-only paths).

## Risks / open questions

- **Spec-gate.** Renovate's PRs are authored by the PAT owner, so the bot skip never applies. Measured: a 29-pin bump is 58 LOC and fails the gate (verification.md). Resolved by groups of at most 24 pins, asserted by a test.
- **Go.** CI builds with `cli/go.mod`'s `go` line; `GO_VERSION` must never move alone, and golangci-lint 2.12.2 panics under Go 1.27. Resolved: patch-only, and the same manager matches `cli/go.mod` so both move in one PR.
- **Annotation correctness.** `mise registry` drifts with mise; the npm-backend CLIs take their package name from `npmBackend` in `cli/internal/tools/mise.go`, and a test asserts the two agree.
- **Tag formats.** kustomize tags `kustomize/vX`, jq tags `jq-X`; per-pin `extractVersion` handles both.
- **release-please.** The preset's `fix(deps):` prefix makes every merged bump a fix release candidate. Kept; flagged to the owner.

## Acceptance criteria

- [ ] AC1: Every `*_VERSION=` line in `versions.conf` has a `# renovate:` annotation within the two lines above it, either `datasource=… depName=… depType=<group>` or `skip <reason>`; a test fails on a pin without one.
- [ ] AC2: No update group holds more than 24 pins, so a grouped minor/patch PR stays under the 50-LOC spec-gate threshold; a test counts them.
- [ ] AC3: Each npm-backend CLI's annotation names the package `npmBackend` routes it to; a test fails when they disagree.
- [ ] AC4: `renovate-config-validator` accepts `renovate.json`, and a local `--platform=local` extract finds every annotated pin with the right datasource and current value (verification.md).
- [ ] AC5: The annotations change nothing a reader sees: `dotf tools sync` renders a byte-identical `conf.d/dotfiles.toml`, and `bats tests/versions-conf.bats` passes.
- [ ] AC6: Go is patch-only and moves `versions.conf` and `cli/go.mod` together; PYTHON, JAVA, MAVEN, MINIKUBE, GIT, DOTF and the `# mise: latest` pins are `skip` with a reason.

## References

- Bitácora: mlorentedev/dotfiles#3284; lesson 605; ADR-044 (mise layer), ADR-036 (cache-gc)
- Fleet preset: `github>mlorentedev/ci-workflows//renovate/default#v0.8.0`
