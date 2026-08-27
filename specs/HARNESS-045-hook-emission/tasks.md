---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - HARNESS-045-hook-emission

## Done this sitting

- [x] [AC8] Persona loader with declared per-skill severity, on a real YAML
      parser rather than a third hand-rolled frontmatter reader
- [x] [AC8] `Decide()` — the whole policy as a pure function over a
      harness-neutral `ToolCall`, testable with no harness installed
- [x] [AC3] [AC4] `dotf harness gate` — exits 0 or 2 and nothing else, so the
      harnesses' contradictory hook-error semantics never apply
- [x] [AC5] [AC6] `MergeHooks` — find-by-marker emission that coexists with a
      live third-party writer, verified against the REAL deployed settings file
- [x] [AC1] `agents.bind` declared in `harness/manifest.json`, every event name
      measured against the installed harness

## Not done, and deliberately so

- [ ] [AC1] Wire `bind` into setup, replacing `merge_claude_settings`'s
      positional jq paths. The merge is written and tested; nothing writes to a
      live file yet.
- [ ] [AC2] Presence emission. Only the Action half is built.
- [ ] **The pi and opencode TS wrappers.** Declared in the manifest with
      `emit: false` so the gap is visible rather than remembered. They need two
      generated-code templates plus their deploy, which is a sitting of its own —
      splitting it was a decision, not an omission.
- [ ] [AC7] The block-style guards. **Four parsers read `skills:` and three
      break silently on the mapping form** (the fourth is this spec's loader):
      - `specs/HARNESS-046/check-roster-consistency.py:62` — regex on the inline
        form, falls back to `[]`
      - `cli/internal/doctor/checks_agent_tiers.go` `readAgentFrontmatter` —
        skips indented lines by design
      - `scripts/compile-harness.sh:1016` — `skill_field` reads inline only, so
        `skills_line` renders empty
      **The fix already exists in that same file.** `agent_capability_line()`
      detects the block form for `capabilities:` and refuses, on the reasoning
      that an empty allow-list is not "no opinion" but "full default access".
      The identical argument applies to `skills:`, where empty means "nothing
      enforced". Copy that guard.
- [ ] Migrate the 35 skills across 7 personas to declared severity. Blocked on
      the guards above, or the migration silently disarms three readers.

## Closing

- [x] `go test ./...` — 18/18 packages
- [x] `go build`, `go vet`, and `GOOS=windows go vet` all pass
- [x] `golangci-lint run` on the pinned 2.12.2 — 0 issues
- [x] `bats tests/compile-harness.bats` — 72/72, exit 0 (the manifest edit adds a
      key without disturbing the render)
- [x] The gate driven end to end as a binary, on all four harnesses
- [x] Emission verified against the real `~/.claude/settings.json`
- [ ] PR opened referencing this spec folder
- [ ] Independent adversarial review before archive — the implementing session
      cannot be the reviewer

## Machine-readable features

`features.json` is the harness-facing contract. The agent may not write
`"state": "passing"`; only the harness may, after running `verification` and
capturing exit code 0.
