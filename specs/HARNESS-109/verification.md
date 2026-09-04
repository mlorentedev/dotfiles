---
tags: [spec, verification, templates]
created: "2026-09-03"
---

# Verification - HARNESS-109

## Evidence

- [x] **AC1** — a dispatch writes a `name → subagent_type` entry, and nothing more —
      commit `503e04b`, tests `TestTheDispatchMapNeverStoresAnythingButTwoIdentifiers`,
      `TestRecordDispatchRoundTrips`. Also observed live on the real hook path: after the AC6
      dispatch, `~/.local/state/dotfiles/gate/<session>.dispatch.json` read verbatim
      `{"types":{"harness109-probe":"reviewer"}}`.
- [x] **AC2** — a later call with `agent_type: <name>` resolves through the map; a control run
      without the dispatch still fails open — commit `503e04b`, tests
      `TestANamedDispatchIsGatedLikeAnUnnamedOne` (asserts the named and unnamed rows AGREE, not
      merely that the named one is non-empty) and `TestWithoutTheDispatchTheChildStillFailsOpen`.
- [x] **AC3** — a direct persona name resolves without the map, unchanged — `503e04b`,
      `TestLoadGatePersonaSaysSoWhenARoleDoesNotResolve/a_role_that_resolves_is_silent`.
- [x] **AC4** — a built-in agent records `no-role`, emits no `ENFORCEMENT IS OFF`, keeps the raw
      name in `role_requested` — `503e04b`, `TestAWitnessedBuiltInAgentIsQuietNotAFault`
      (both the named and unnamed routes, which take different branches).
- [x] **AC5** — names are validated and no other tool-input key is read or persisted — `503e04b`,
      `TestAnUnusableDispatchNameIsNotWritten`, `TestValidDispatchNameMatchesTheToolsOwnConstraint`,
      `TestOnlyTheDispatchPrimitiveIsReadForDispatchArguments`. The security assertion is on the
      BYTES: a `ghp_`-shaped value in the dispatch's `prompt` argument is absent from both the map
      file and every journal record.
- [x] **AC6** *(owner-gated, authorised 2026-09-03)* — reproduced live. See the table below.
- [x] **AC7** — the interactive hooks carry a timeout, asserted through the real emission path —
      `503e04b`, `TestEveryInteractiveHookIsBounded`.

### AC6, observed

One named dispatch of a real persona, on the live `PreToolUse` hook, against a locally built binary
carrying `503e04b`. The journal, verbatim, from
`~/.local/state/dotfiles/gate/<session>-aharness109-1146ec45.decisions.jsonl`:

```json
{"ts":"2026-09-04T01:30:26.166335627Z","harness":"claude","agent_type":"harness109-probe",
 "agent_id":"aharness109-probe-7d551507f3fa68b0","tool":"Bash",
 "role_requested":"harness109-probe","role_resolved":"reviewer","outcome":"warn","allowed":true,
 "warned":["adversarial-review","audit","cyclomatic-complexity","verification-before-completion"]}
```

Against #1434's own table, with the row this spec adds:

| dispatch | payload `agent_type` | `role_resolved` | outcome |
|---|---|---|---|
| `subagent_type: reviewer`, no name | `reviewer` | `reviewer` | `warn` — 4 skills |
| `subagent_type: reviewer`, `name: gate-probe-agent-type` *(before)* | `gate-probe-agent-type` | *(none)* | **`role-unresolved`** |
| `subagent_type: reviewer`, `name: harness109-probe` *(after)* | `harness109-probe` | **`reviewer`** | **`warn` — the same 4 skills** |

The four warned skills match the unnamed row exactly, which is the claim: naming a subagent no
longer changes its gate. `role_requested` still carries the raw name, so the record says which
dispatch this was.

## Test status

- `go build ./... && go vet ./...` → clean.
- `GOOS=windows go vet ./...` → clean. Run because the Windows leg of CI compiles the same tree and
  a Linux-only loop cannot see a Windows build error.
- `go test ./...` → all packages ok, no failures, no regressions.
- `golangci-lint run` → **0 issues**, on the pinned `2.12.2` from `versions.conf` (verified against
  the pin before running, per BUG-071 — a local binary on a different major reports 0 on code CI
  rejects).
- Manual smoke, before installing anything: `dotf harness gate --help`, `dotf harness suggest
  --help`, and a synthetic payload through the built binary — exit 0.
- Live probe: one named subagent dispatch, evidence above. The shared
  `~/.local/bin/dotf` was backed up first and **restored byte-identically afterwards** (`cmp`
  confirmed), so the machine is not left running a `dev` build of an unmerged branch — the trap
  #1469 (CLI-074) describes. The `timeout` fields are **repo-side only**: `compile-harness.sh
  --deploy` was deliberately NOT run, so no session's live `~/.claude/settings.json` was rewritten
  mid-flight. They go live with the next ordinary deploy.

## Decisions made during implementation

- **The ticket's first candidate was refuted, not deferred.** Reading the Claude Code 2.1.260
  executable's own payload builder settled it: `session_id, transcript_path, cwd, scratchpad_dir,
  prompt_id, permission_mode, agent_id, agent_type, effort` is the whole base field set, so there is
  no second field carrying the true type and there never was. Attempting that direction would have
  produced a fallback that silently never fires.
- **An unnamed dispatch is recorded under its own type**, which looks redundant and is what removes
  the need for a hardcoded list of built-in agents anywhere. Presence in the map means "the gate
  witnessed this and knows what it really is"; a witnessed non-persona is a correct quiet answer,
  and only a dispatch the gate never saw stays loud.
- **`no-role` was reused rather than adding a ninth `Outcome`.** The vocabulary is closed and pinned
  by tests on purpose, the fact is genuinely the same one, and `role_requested` keeps the raw name
  so nothing is lost.
- **The AC7 guard asserts the class and immediately found a second instance** nobody was looking
  for: agy's `gate` on `BeforeTool` was also unbounded. Both now carry `timeout: 5`. This is the
  incident-to-guard rule paying out rather than being recited — a guard written for one instance
  finding another in the same file.
- **The `timeout` assertion had to go through `MergeHooks`.** The first version marshalled
  `HookCommand` directly and failed against a correct manifest, because `bind.go` omits `timeout`
  from the emitted JSON when zero and Go's field names are not the emitted keys. A declaration-level
  test would have passed on a value that never reaches the harness — the exact form #1455's review
  called insufficient.
- **The measured count was narrowed after a peer challenged its provenance.** The 271/274 figure
  survives (schema uniform, the two resolving records interleaved with the unresolved ones rather
  than from a later binary), but every `agent_type` record falls in one 3.5-hour window on
  2026-09-02. Stated in the proposal as large-in-calls, small-in-sessions rather than left to read
  as a broad sample.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? **yes** — a hook timeout's blocking semantics are not
      derivable from the documentation and had to be read out of the executable; the fact that a
      timed-out hook returns `blocked: false` unless `timeoutFailsClosed` (set only for a
      cloud-served call) is what makes bounding a hook safe, and the next person to bound one will
      need it. Filed with this spec.
- [ ] ADR-worthy decision? **no** — this implements the gate ADR-027/HARNESS-045 already decided; it
      changes no contract.
- [ ] New pattern candidate for `00_meta/patterns/`? **no** — the mechanism is specific to this
      harness's payload shape. It does not recur across projects.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-109/` -> `specs/archive/HARNESS-109/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
