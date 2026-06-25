---
tags: [spec, verification, secrets, registry]
created: "2026-06-25"
---

# Verification - CLI-024-secrets-registry

> Phase 2a of ADR-028. Branch `feat/secrets-registry` (off main). Issue #583.

## AC1 — registry round-trips env-mapping.conf

`TestRealRegistry_RoundTripsEnvMapping` loads the **committed** `secrets/registry.yaml`
and `sensitive/env-mapping.conf` and asserts `Registry.Entries()` == `ParseMapping()`
as a set (var → {age-file, IsFile, Dest}). Covers `github-token`'s two names
(`GITHUB_PERSONAL_ACCESS_TOKEN`+`RELEASE_TOKEN`), the `dockerhub-token` two-field
collapse, the 7 `x-twitter` per-var sources, and all 7 `@VAR` file secrets. 33 entries
both sides. The drift guard fails the build if either file diverges (until env-mapping
dies in PR-C).

## AC2 — `show <id>`: value, no trailing newline

- `TestSecretsShow_SingleEnv_ScrubbedNoTrailingNewline` (fake decryptor → `"the-value\n"`,
  output `"the-value"`).
- Smoke (real age): `dotf secrets show nan-api-key` → `bytes=25`, `last_char=[A]` (no `\n`).
- Equality with injection: `run --only nan-api-key -- pwsh -c '$env:NAN_API_KEY.Length'` = `25`
  == the `show` byte count (same scrub path via `Loader.EnvFor`).

## AC3/AC5 — `show` rejects file/multi; non-zero exit

- `TestSecretsShow_RejectsFileAndMultiAndUnknown`.
- Smoke: `show x-twitter` → `exit=1` `"x-twitter" exposes 7 vars; use run …`; `show kubelab-kubeconfig`
  → `exit=1` `… is a file secret; use run …`; `show nope` → `exit=1`; `show nan-api-key` → `exit=0`.

## AC4 — `ls` lists ids + plane + vars, no values

- `TestSecretsLs_ListsIdsAndVars_NoValues`.
- Smoke: `dotf secrets ls` → 26 ids, each `id  plane  VAR[,VAR…]`, no values printed.

## AC5 — `run` reads the registry; `--only` id or env name

- `TestResolveOnly_IdSelectsAllVars_NameSelectsOne` (`x-twitter` id → 2 vars; `X_BEARER_TOKEN`
  name → 1; empty → nil; unknown → error).
- Smoke: `run --only nan-api-key` (id) injects `NAN_API_KEY`; `run --only x-twitter` (id) →
  `7` `X_*` vars in the child; `run --only bogus` → `exit=1`. The opencode/pi/agy wrappers use
  no `--only` (full mapped set) and are unaffected (`buildChildEnv` over `Registry.Entries()`).

## AC6 — parser fail-fast on malformed registry

`TestParseRegistry_Validation`: bad version / duplicate id / unknown backend / both env+file /
missing source → error. Plus `TestParseRegistry_Shapes` (all 3 env shapes + file + 3 backends)
and `TestRegistry_Lookup_IdThenVar`.

## AC7 — suite green; env-mapping untouched

- `go test ./internal/secrets/ ./internal/cmd/` → **ok** (both packages). `gofmt`/`go vet` clean.
- `sensitive/env-mapping.conf` is **unchanged** in this PR (twins-only legacy; nothing in the Go
  path reads it after the rewire — `buildChildEnv` now goes through the registry).
- **Pre-existing, unrelated failures** (not touched by this PR): `internal/spec`, `internal/vault`,
  `internal/initrepo` `TestEmbeddedTemplatesMatchVault` — vendored templates drifted from the local
  `Workspace/knowledge` vault. Independent of secrets; a separate re-vendor task.

## Dependency

`go.yaml.in/yaml/v3 v3.0.4` promoted from transitive (go.sum) to a direct require — the canonical
YAML lib, already in the module graph. `go mod tidy` clean.

## Out of scope (follow-ups)

- `bw serve` live read + item migration → ADR-028 Phase 3.
- Migrate setup-{linux,windows} eager-load + `nan-*` to `dotf secrets`; delete the
  `load-secrets.{sh,ps1}` twins **and** `env-mapping.conf` (their only remaining consumers) → #493 PR-C.
