---
id: "OPS-017-secrets-tooling"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-25"
issue: "dotfiles#577"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, secrets, bitwarden, age, catalog, doctor]
template_version: "1.0"
---

# OPS-017-secrets-tooling

> **Naming**: file lives at `<repo>/specs/OPS-017-secrets-tooling/proposal.md`. Phase 0 of the ADR-028 secrets-governance rollout: provision the two tools the model depends on (`bw`, `age`) and make `dotf doctor` verify them.

## Why

<!-- from issue #577: OPS-017 — provision Bitwarden CLI (bw) via setup / package catalog -->

[ADR-028](../../docs/adr/adr-028-secrets-two-tier-bitwarden-age.md) makes Bitwarden the live secrets SSOT and `age` the DR/bootstrap floor, both behind a future `dotf secrets` facade. Nothing downstream works until both binaries are reliably present on every machine:

- **`age`** is installed today by an imperative `curl | tar` block in `setup-linux.sh` (and ad-hoc on Windows). It works but is invisible to the declarative catalog.
- **`bw`** has *no* provisioning at all — it is installed by hand (scoop on this machine). ADR-028's phased plan (step 0) and #577 call for provisioning it through the **declarative package catalog** (`packages.json`, the CLI-029 mechanism), the same data-driven path `sops` already uses.

A blocking discovery (evidence in `tasks.md` Setup): **neither tool fits the catalog's existing `github-release` installer.** It expects a *raw* binary from a `v{version}` tag, verified against a *mandatory sha256 manifest*. But `age` ships tar.gz/zip archives with sigsum `.proof` files (no sha256 manifest), and `bw` ships zip archives under a `cli-v{date}` tag with no manifest either. Forcing both through an extended `github-release` path means archive extraction + a configurable tag prefix + an alternative verification for the missing manifest — a rewrite of CLI-029's security gate, multiple PRs before any value lands.

`bw`'s *first-class, official* distribution is **npm** (`@bitwarden/cli`), and `node`/`npm` are already core tools in this repo. So the smallest, lowest-risk path that satisfies #577 is to teach the catalog one new `source.type: "npm"` rather than to rebuild the github-release installer.

## What

Three changes, one cohesive Phase-0 slice ("provision + verify"):

1. **`source.type: "npm"` in the package catalog** (`cli/internal/tools`). A new install path that pins via `npm install -g <package>@<version>`, reusing the existing reconcile policy (`decideAction`: skip at/above pin, install when absent, upgrade when below — never downgrade). The version probe runs `<name> --version` on PATH (npm globals land on PATH, not in `~/.local/bin`), so a `bw` already installed by scoop/choco is detected and *not* re-installed. Command execution and the version probe are behind seams so the path is unit-testable with no network and no real npm.

2. **`bw` catalog entry** in `packages.json` → `{ name: bw, version: 2026.5.0, profile: full, source: { type: npm, package: "@bitwarden/cli" } }`. `setup-{linux,windows}` already run `dotf tools install`, so this provisions `bw` cross-platform with no new setup code.

3. **`dotf doctor` secrets-tooling check**. A new section verifying `bw` and `age` resolve on PATH and the `age` identity key exists (`$AGE_KEY_PATH` or `~/.config/age/key.txt`) — the governance hook ADR-028 §Mitigations and the runbook both name. Absent `bw`/`age` → FAIL (the secrets model can't run); absent age key → WARN (a fresh machine before key restore).

**`age` stays on its current imperative install** (Option A, chosen 2026-06-25). It already works and the github-release catalog path is blocked by age's missing sha256 manifest; catalog-ifying it would force the same gate rewrite this slice avoids. The doctor check covers it either way.

## Out of scope

- **Extending `github-release` to archives / sigsum / `cli-v` tags** — explicitly rejected for this slice (the rewrite above). Tracked as a possible follow-up if a future tool needs it.
- **`age` into the catalog** — deferred with the same rationale; its imperative install is untouched.
- **`dotf secrets` facade, registry, `run --`, killing the ambient export** — Phase 1+ (#493/#378), separate PRs.
- **Migrating secrets bw↔age, rotation, DR escrow** — Phases 3-4.
- **Windows `age` provisioning hardening** — the doctor check will surface any gap; fixing it is a separate ticket if it FAILs.

## Risks / open questions

- **R1 — npm global installs don't pin transitive deps like a sha256-verified binary does.** A `-g` install trusts the npm registry + `@bitwarden/cli`'s own integrity, not a checksum we control. *Mitigation:* accepted trade-off for an official first-party CLI; the version is pinned, `dotf doctor` flags drift, and this is strictly better than today's "install by hand". Hardening (lockfile/`--ignore-scripts`) is a follow-up if warranted.
- **R2 — `npm` may be absent when `dotf tools install` runs.** *Mitigation:* the npm path returns an error that `installAll` already logs best-effort (one tool's failure never aborts setup); and if `bw` is already on PATH (scoop/choco), the version probe SKIPs before npm is ever invoked.
- **R3 — `bw --version` output shape.** It prints a bare `2026.5.0`; the shared `semverRE` (`\d+\.\d+\.\d+`) matches it. *Mitigation:* covered by a unit test asserting the parse.
- **R4 — Windows `npm` resolves as `npm.cmd`.** *Mitigation:* Go 1.20+ `exec` honours `PATHEXT`; and on Windows `bw` is typically already present via scoop, so the probe SKIPs. Not a setup blocker (best-effort).
- **R5 — version drift between the pin and the latest bw.** A monthly-cadence CLI will outrun the pin. *Mitigation:* the pin is a *minimum* (never downgrades a newer install); bumping it is a one-line `packages.json` edit, same as `sops`.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1** — `packages.json` lists `bw` with `source.type: "npm"` and `package: "@bitwarden/cli"`, pinned, and parses as valid JSON. *Verify:* `jq` parse + `dotf tools list` shows `bw`.
- [ ] **AC2** — the catalog installer dispatches on `source.type`: an npm tool runs `npm install -g <pkg>@<version>` only when absent/below-pin, and SKIPs when a PATH binary is at/above pin. *Verify:* table-driven `go test` over fresh / below-pin / at-pin / above-pin with faked Run + version seams (no network).
- [ ] **AC3** — `github-release` installs (sops) are unchanged — the existing install_test suite stays green. *Verify:* `go test ./internal/tools/...`.
- [ ] **AC4** — `dotf tools list` renders npm tools legibly (e.g. `npm:@bitwarden/cli`) instead of "(no build for this platform)". *Verify:* unit assertion / manual `dotf tools list`.
- [ ] **AC5** — `dotf doctor` adds a "Secrets tooling" section: `bw`/`age` present → PASS, absent → FAIL; age key present → PASS, absent → WARN. *Verify:* table-driven `go test` over the four states with the `System` seams.
- [ ] **AC6** — full `go test ./...` and `go build ./...` green; no regression in the existing doctor sweep. *Verify:* CI + local.

## References

- Issue: dotfiles#577 (work-gate, OPS-017)
- ADR: [adr-028](../../docs/adr/adr-028-secrets-two-tier-bitwarden-age.md) (the two-tier model + phased plan), [adr-020](../../docs/adr/adr-020-tooling-cli-go-convergence.md) (Go owns tooling), [adr-021](../../docs/adr/adr-021-harness-render-pipeline.md)/CLI-029 (the package catalog)
- Runbook: `docs/runbooks/guide-secrets-governance.md` (names the `dotf doctor` bw/age check)
- Code: `cli/internal/tools/{catalog,install}.go`, `cli/internal/cmd/tools.go`, `cli/internal/doctor/`, `packages.json`
- Related: #493 (Phase 1 `dotf secrets`), #378 (facade), #321 (token split)
