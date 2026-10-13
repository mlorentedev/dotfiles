---
id: "lesson-411-a-merge-dst-needs-a-literal-extension"
type: lesson
status: active
title: "A merge entry's dst needs a literal extension, or every dotf refuses the manifest"
created: "2026-10-13"
---

# A merge entry's dst needs a literal extension, or every dotf refuses the manifest

## Context
herdr's config lives in a different place per OS (`~/.config/herdr` on Linux and macOS,
`%APPDATA%\herdr` on Windows), and herdr reads `HERDR_CONFIG_PATH` to override it. The
plan for the `herdr` deploy entry (#2013 H4) was one entry with `"dst": "{HERDR_CONFIG_PATH}"`,
the variable declared in `env-contract.json`.

## The Trap
`dotf deploy` merges TOML or JSON, and it picks the codec from the destination's extension.
`ParseManifest` checks, for every merge entry, that the source and destination formats
agree, and it does so on the dst as written, before any `{VAR}` is resolved. A dst that is
a bare token has no extension, so the check fails, and it fails for the whole manifest:
`ParseManifest` returns an error and no entry deploys. The released 0.68.0 runs the same
check, so the entry would have broken `dotf deploy` on every machine, not only the herdr
config. The first sign was 12 unrelated Go tests failing at once, each loading the shipped
`ai/deploy.json`.

## The Solution
- Keep the file name literal and put only the directory behind the token:
  `"dst": "{HERDR_CONFIG_DIR}/config.toml"`. `HERDR_CONFIG_DIR` in the contract holds
  herdr's default directory per OS, and herdr's own `HERDR_CONFIG_PATH` stays unset, so the
  file dotf writes is the file herdr loads.
- A contract variable that names a file cannot be a merge dst. Name its directory instead.
- When an entry's shape is new, run the whole Go suite, not only the entry's own test: the
  tests that load the shipped manifest are what catch a manifest-wide refusal.
