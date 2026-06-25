---
tags: [spec, tasks, secrets, catalog, doctor]
created: "2026-06-25"
---

# Tasks - OPS-017-secrets-tooling

> TDD order: failing test → implementation → green. One task = one focused commit. Two commits in this PR: (1) npm catalog source + bw entry, (2) dotf doctor secrets-tooling check.

## Setup

- [x] Branch `feat/secrets-tooling-provisioning` created from `main`
- [x] #577 self-assigned (bitácora → In Progress)
- [x] `proposal.md` complete; acceptance criteria testable
- [x] **Evidence — neither tool is a clean `github-release` binary** (verified via `gh release` 2026-06-25):
  - `age` v1.3.1 assets: `age-v1.3.1-linux-amd64.tar.gz`, `age-v1.3.1-windows-amd64.zip` (archives, inner `age/age`); verification is sigsum `.proof`, **no `checksums.txt`**.
  - `bw` latest tag `cli-v2026.5.0` (**not** `v{version}`); assets `bw-linux-2026.5.0.zip` etc. (zip); **no sha256 manifest**; primary dist is npm `@bitwarden/cli`.
  - Conclusion: catalog's `github-release` installer (raw binary + `v{version}` tag + mandatory sha256 manifest) fits neither → add `source.type: "npm"` for bw; leave age imperative.
- [x] Local toolchain confirmed: `go 1.25.6`, `npm 11.17.0`, `bw 2026.5.0`, `age` present.

## Implementation — commit 1: npm catalog source + bw

- [ ] **RED**: `install_test.go` — npm cases (fresh installs, at/above-pin skips, below-pin upgrades, npm-absent error) with faked `Run` + version probe seams.
- [ ] **GREEN**: `catalog.go` — add `Source.Package` field (npm package name).
- [ ] **GREEN**: `install.go` — add `Run` command seam (default `exec`); dispatch `Install` on `Source.Type`; `installNpm` (probe PATH version → `decideAction` → `npm install -g pkg@version`); reuse `decideAction`/`Result`. No behaviour change to the `github-release` path (sops tests stay green).
- [ ] **GREEN**: `cmd/tools.go` — `list` renders npm tools as `npm:<package>` (not "(no build for this platform)").
- [ ] `packages.json` — add `bw` (`@bitwarden/cli`, pinned `2026.5.0`, profile `full`, `source.type: npm`).
- [ ] `go test ./internal/tools/... && go build ./...` green.

## Implementation — commit 2: dotf doctor secrets-tooling check

- [ ] **RED**: `checks_secrets_tooling_test.go` — table over {bw,age present/absent} × {age key present/absent}: present→PASS, absent bin→FAIL, absent key→WARN.
- [ ] **GREEN**: `checks_secrets_tooling.go` — `checkSecretsTooling(sys, cfg, rep)`: `sys.has("bw")`/`sys.has("age")`; age key at `$AGE_KEY_PATH` or `~/.config/age/key.txt`.
- [ ] Wire `checkSecretsTooling` into `doctor.Run` (non-quick sweep), next to `checkSecrets`.
- [ ] `go test ./internal/doctor/...` green.

## Closing

- [ ] Full `go test ./...` + `go build ./...` green
- [ ] `verification.md` filled with command evidence (test output, `dotf tools list`, a `dotf doctor` excerpt)
- [ ] PR opened referencing this spec folder + issue #577 (no auto-merge)
- [ ] On merge: move to `specs/archive/OPS-017-secrets-tooling/`
