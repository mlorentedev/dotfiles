---
tags: [spec, verification, templates]
created: "2026-11-15"
id: "TOOL-042-renovate-pins"
type: spec-verification
status: active
owner: manu
---

# Verification - TOOL-042-renovate-pins

## Evidence

- [x] AC1 -> `TestVersionsConf_EveryPinNamesItsUpstreamOrWhyItHasNone`, `TestVersionsConf_RenovateFindsEveryAnnotatedPinAtItsValue` (the second compiles the regex `renovate.json` runs)
- [x] AC2 -> `TestRenovate_NoGroupCanOutgrowTheSpecGate`, which reads `THRESHOLD=` from `scripts/check-spec-gate.sh`. Groups today: cli 18 pins, k8s 11, lint 11, runtime 6 + `cli/go.mod`, catalog 8.
- [x] AC3 -> `TestVersionsConf_NpmCLIsTrackThePackageDotfInstalls` (reads `npmBackend`)
- [x] AC4 -> `renovate-config-validator --strict` with `ghcr.io/renovatebot/renovate:44.199.2` (the engine the central run pins): `Config validated successfully against 1 file(s)`. A `RENOVATE_PLATFORM=local RENOVATE_DRY_RUN=extract` run over a copy of the pin files extracted 55 deps (versions.conf 46, cli/go.mod 1, packages.json 8), each at its current value and depType; jq and kustomize carried their own `extractVersion`, the rest `^v?(?<version>.+)$`. The GitHub-backed deps report `github-token-required` locally only: the run had no token, and the central run uses `RENOVATE_TOKEN`.
- [x] AC5 -> `dotf tools sync --dry-run --versions <file>` printed the same 48 lines for `HEAD:versions.conf` and the annotated file. The 28 bats files that reference versions.conf: 570 tests, all ok. The PowerShell readers in ci.yml select `^KEY_VERSION=` lines, which the annotations do not touch.
- [x] AC6 -> `TestRenovate_GoMovesByPatchTogetherWithGoMod` (also asserts `GO_VERSION` equals `cli/go.mod`'s go line), `TestVersionsConf_PinsRenovateMustNotMoveAreSkipped`

### Spec-gate measurement (Risk 1)

A plumbing commit moving the 29 pins that were behind on 2026-11-15, checked as Renovate's PR would be (`SDD_PR_AUTHOR=mlorentedev SDD_LABELS=dependencies,renovate`):

```
[INFO] "dependencies" label present but author "mlorentedev" is not a recognised dependency bot — not honouring the skip; evaluating the gate normally.
  Production LOC (added+removed): 58
[FAIL] SDD Discipline Gate violation:
```

### Mutation check

Each deliberate break below, applied alone to `versions.conf` and reverted, failed the named test:

| Break | Failure |
|---|---|
| FD's renovate line deleted | `FD_VERSION has no # renovate: line above it` |
| COPILOT annotated as `github-releases github/copilot-cli` | `copilot installs npm:@github/copilot, but its renovate line is …` |
| every `depType=k8s` set to `cli` | `group "cli" moves 29 pins, 58 lines, and the spec-gate fails at 50` |
| `GO_VERSION=1.26.11` alone | `GO_VERSION is 1.26.11 but cli/go.mod says go 1.26.10` |
| PYTHON given a `python-version` annotation | `PYTHON_VERSION must be # renovate: skip …` |
| `depname=` typo on HELM | `HELM_VERSION is annotated … but Renovate's regex does not match it` |
| KUBECONFORM moved to `depType=misc` | `depType "misc" has no group rule in renovate.json` |

## Test status

- `go build ./... && go vet ./...`, `GOOS=windows go vet ./internal/tools/`, `go test ./internal/tools/ ./internal/converge/`: ok
- `golangci-lint run ./internal/tools/` at the pinned 2.12.2 (go 1.26.10), Linux and `GOOS=windows`: 0 issues
- No regressions: the bats files that read versions.conf pass (above)

## Decisions made during implementation

- Groups by `depType` on the annotation line, so the group lives next to the pin and one test counts it, instead of package lists in `renovate.json`.
- The group cap reads the gate's threshold rather than copying 50.
- PYTHON is skipped until #3541; JAVA, MAVEN and MINIKUBE name `$APPS_HOME` directories nothing installs, so a bump would only break their `_HOME`.
- Release binaries (`github-releases`, `github-tags`) wait three days, as the preset already does for npm, pypi and go: they run in CI with the repository's secrets.
- The fleet preset's `fix(deps):` prefix is kept; whether a pin bump should cut a release is the owner's call (PR body).

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-606-a-bot-skip-keyed-on-the-author-misses-a-bot-running-on-a-personal-token.md
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: Renovate is the fleet's existing decision (ci-workflows#30); this adopts it and records the per-repo shape in the runbook
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: the author-keyed skip exists only in this repo's spec-gate
