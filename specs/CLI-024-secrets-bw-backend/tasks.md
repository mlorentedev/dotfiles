---
tags: [spec, tasks, secrets, cli, go, bitwarden]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-bw-backend

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/secrets-bw-backend`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] **AC1** — Registry schema: `BW *BWSource{Item, Field}` on `Secret`, `Field` on
  `EnvVar`, per-var `field` parsed in `EnvExpose.UnmarshalYAML`, `checkBwSources`
  wired into `validate()`. Tests: `TestParseRegistry_BwShapes`,
  `TestParseRegistry_BwValidation`.
- [x] **AC2a** — `Entry` gained `Backend`/`Item`/`Field`; `Entries()` emits bw entries
  via `bwEntries`/`ageEntries`, tagged. `AgeBacked()` removed (dead). Test:
  `TestRegistry_Entries_IncludesBwBackend`.
- [x] **AC2b** — `Resolver` interface + `ageResolver` + `bwResolver` (behind
  `BWReader`); `Loader.BW` seam; `EnvFor` dispatches via a resolver map (`""` → age).
  Tests: `TestEnvFor_BwEnvSecret`, `TestEnvFor_BwFileSecret_Materialized`,
  `TestEnvFor_MixedBackends_OnlyFilter`.
- [x] **AC4** — nil reader → "bw backend unavailable"; reader error → wrapped fail-fast.
  Tests: `TestEnvFor_BwNilReader_ClearError`, `TestEnvFor_BwReaderError_FailsFast`,
  `TestEnvFor_UnknownBackend_Errors`.
- [x] **AC3** — `bwReader` seam + `secretLoader()` helper in `secrets.go`;
  `reg.ShowEntry` replaces `showSource` (bw allowed); `ls --pairs` skips bw. Tests:
  `TestSecretsShow_ResolvesBwBackend`, `TestSecretsRun_ResolvesBwBackend`.
- [x] **AC5** — `BWGet` (`bw --nointeraction get item` → JSON → field) in `bw.go`;
  `fieldFromItem` unit-tested (`TestFieldFromItem`); `bw serve` perf upgrade documented.

## Closing

- [x] Every acceptance criterion is covered by ≥1 test (AC5's `BWGet` shell-out excepted — live smoke)
- [x] `features.json` carries non-vacuous verification commands (state left `pending` for the harness)
- [x] `go vet ./...` + `gofmt -l` (staged LF blobs) clean; `go build ./...` clean
- [x] `registry.yaml` unchanged (no secret flipped — confirmed with `git diff`)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See `features.json` (sibling). Each acceptance criterion maps to ≥1 feature with an
executable `verification`. The agent may not set `"state": "passing"` — only the
harness, after capturing exit 0, may.
