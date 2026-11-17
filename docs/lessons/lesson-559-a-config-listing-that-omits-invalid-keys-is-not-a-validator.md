---
id: "lesson-559-a-config-listing-that-omits-invalid-keys-is-not-a-validator"
type: lesson
status: active
title: "A config listing that omits invalid keys is not a validator"
created: "2026-11-05"
---

# A config listing that omits invalid keys is not a validator

## Context

GitHub Copilot Desktop refused every session before repository discovery because
`~/.copilot/settings.json` contained the managed-policy-only local key
`telemetry.disabled`. Both the installed CLI and the Desktop-bundled CLI answered
`copilot config --list --json` with `{}` and exit 0 against the same synthetic home.

## Problem

The listing command exposed the normalized configuration the CLI accepted, not the
invalid source entries the runtime would reject during session initialization. Treating
an empty successful listing as schema validation turned omission into a false green and
left one user setting able to disable every workspace.

## Solution

Validate the file that the runtime reads, against an explicit checkout-owned policy.
Report file and key paths without values, fail on malformed or unreadable input, and
repair retired keys through the same backup and atomic-write path as deployment. A
configuration command is validation evidence only when its contract explicitly rejects
unknown or unsupported source keys with a non-zero exit.
