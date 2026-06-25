---
tags: [spec, verification, secrets, catalog, doctor]
created: "2026-06-25"
---

# Verification - OPS-017-secrets-tooling

> Evidence for the Phase-0 acceptance criteria. Commits on `feat/secrets-tooling-provisioning`.

## Commits

- `1cebf90` docs(spec): scaffold OPS-017 secrets-tooling spec
- `7f84bc4` feat(tools): add npm catalog source type and provision bw
- `f2eb911` feat(doctor): add secrets-tooling check (bw, age, age key)

## AC1 — bw catalog entry (npm), valid JSON

`packages.json` lists `bw` with `source.type: "npm"`, `package: "@bitwarden/cli"`, pinned `2026.5.0`.

```
$ DOTFILES_DIR=<repo> dotf tools list
NAME  VERSION   PROFILE  ASSET (windows/amd64)
sops  3.13.1    full     sops-v3.13.1.amd64.exe
bw    2026.5.0  full     npm:@bitwarden/cli
```

## AC2 — installer dispatches on source type; npm reconcile policy

`go test ./internal/tools/ -run TestInstallNpm -v` — all green (no network, faked Run + version seams):

```
--- PASS: TestInstallNpm_Fresh                 (npm install -g @bitwarden/cli@2026.5.0 run exactly once)
--- PASS: TestInstallNpm_UpgradeWhenBelowPin   (2026.4.0 → Upgraded)
--- PASS: TestInstallNpm_SkipWhenAtOrAbovePin  (2026.5.0 / 2026.6.0 / 2027.0.0 → Skipped, npm NOT run)
--- PASS: TestInstallNpm_RunFailure            (npm error → Install error)
--- PASS: TestInstallNpm_MissingPackage        (empty package → error, npm NOT run)
```

The skip-at/above-pin case proves a scoop/choco-installed `bw` is detected on PATH and not re-installed.

## AC3 — github-release path unchanged (sops)

`go test ./internal/tools/...` → `ok` (all pre-existing sops install tests stay green; only the source-type dispatch was refactored, no behaviour change).

## AC4 — `dotf tools list` renders npm tools legibly

See AC1: `bw … npm:@bitwarden/cli` (was previously "(no build for this platform)" for any tool with no per-OS asset map).

## AC5 — `dotf doctor` Secrets tooling section

`go test ./internal/doctor/ -run TestSecretsTooling -v` — all green across the four states:

```
--- PASS: TestSecretsTooling_AllPresent
--- PASS: TestSecretsTooling_BwMissing                 (bw absent → 1 FAIL)
--- PASS: TestSecretsTooling_AgeMissing                (age absent → 1 FAIL)
--- PASS: TestSecretsTooling_AgeKeyMissingIsWarnNotFail (key absent → WARN, 0 FAIL)
--- PASS: TestSecretsTooling_AgeKeyPathOverride        (AGE_KEY_PATH honoured)
```

Live render on this machine (bw via scoop, age + age key present):

```
[Secrets tooling]
  (3 checks, all ok)
```

(The full `dotf doctor` sweep hangs later on the unrelated harness/deploy-drift checks, which shell out to `compile-harness.sh`/git in this environment — not introduced by this change. The Secrets-tooling section prints before that point.)

## AC6 — package suites green, build OK

- `go build ./...` → OK
- `go test ./internal/{tools,cmd,doctor}/...` → `ok` (the packages this PR touches)
- Pre-existing, UNRELATED failures: `TestEmbeddedTemplatesMatchVault` in `internal/{initrepo,spec,vault}` — embedded templates drifted from the vault SSOT; this PR does not touch any template and does not re-vendor them (separate maintenance task).

## Not done here (deferred by design)

- `age` into the catalog (blocked by its missing sha256 manifest; imperative install retained) — out of scope per `proposal.md`.
- `dotf secrets` facade / `run --` / killing the ambient export — Phase 1 (#493).
