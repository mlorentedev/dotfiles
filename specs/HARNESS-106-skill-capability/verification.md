---
tags: [spec, verification]
created: "2026-09-01"
---

# Verification - HARNESS-106-skill-capability

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] **AC1** -> test `TestShippedMapGrantsTheSkillCapabilityWhereItExists`, pinned to the
      **shipped** map rather than a fixture. Observed:
      `dotf harness resolve-capabilities read,search,shell,skill --harness claude`
      -> `tools: Read, Glob, Grep, Bash, Skill`
- [x] **AC2** -> test `TestCapabilityMapFailsLoudWhenUnreadable`, seven cases, two of them new
      (a verb both mapped and `unsupported`; an `unsupported` verb outside the vocabulary).
      Observed: `dotf harness resolve-capabilities read,skill --harness opencode`
      -> `permission: {list: allow, read: allow}` on stdout, and on **stderr**
      `[capabilities] opencode declares no native equivalent for skill — omitted from the value, not granted`
- [x] **AC3** -> two guards, at the two layers the defect fell between.
      `TestEveryPersonaDeclaringSkillsCanInvokeThem` asserts the **record** declares the
      capability; it failed red on the **real** defect, not a planted one, naming all seven
      personas and pointing at the vault SSOT rather than the generated files.
      `verify-setup.bats` *"every deployed persona can invoke the skills its own gate demands"*
      asserts the thing the capability exists to produce — a **deployed** agent whose `tools:`
      names `Skill` — inside the integration container, on a fresh machine.
      Red/green proven against two real fixtures: this machine's pre-fix `~/.claude/agents`
      (**RED**, `checked=7`, all seven named) and a deploy from this tree (**GREEN**,
      `checked=7`). Neither is vacuous.
- [ ] **AC4** -> **still open, with a named blocker rather than a vague one.** The deployed
      `~/.claude/agents/reviewer.md` carries `Skill` after the deploy, and a dispatched reviewer
      nonetheless reported its tools as `Read, Bash, advisor`. That matches the roster loaded at
      the *dispatching session's* start, not the file on disk. **Agent definitions are frozen at
      session start; hooks are re-read.** The earlier measurement that hooks reload does not
      generalise to personas, and assuming it did is what made this look ready. AC4 needs a
      session started after the deploy — nothing more, and nothing in this change.
- [x] **AC5** -> test `TestEveryGateDecisionLeavesADurableRecord`, eight cases, one per path
      the command can take — including the three that return before any persona is loaded and
      the block path, which was untestable in process until it stopped calling `os.Exit`.
      Confirmed live: 43 records on this machine across `no-role`, `warn` and `role-unresolved`.
- [x] **AC6** -> test `TestAWarnIsReadableAfterTheSessionEnds`, and measured live. An unnamed
      `reviewer` dispatch produced `outcome: warn` naming all four skills
      (`adversarial-review`, `audit`, `cyclomatic-complexity`, `verification-before-completion`)
      readable from the journal alone, with nothing but the file path shared with the writer.
- [x] **AC7** -> test `TestGateRecordCarriesAgentType`, and **measured on a real dispatch** —
      the criterion's whole point. `agent_type: "reviewer"` arrived, resolved to the persona,
      and was recorded. It also immediately caught a defect nothing else could see: see below.
- [x] **AC8** -> two consecutive `compile-harness.sh --deploy` runs into a scratch `$HOME`,
      `diff -rq` byte-identical. Only the vault SSOT (7 records) and the repo map were edited;
      no generated file was hand-edited.
- [x] **AC9** -> red direction observed for both new loader guards by neutering each condition
      (`if false && …`, so the code still compiles and the variables stay used) and re-running:
      exactly the two new cases failed, the other five stayed green. Guard restored, zero diff.

## Test status

- Test suite: `go build ./... && go vet ./... && go test ./...` -> clean;
  `GOOS=windows go vet ./...` -> clean; `golangci-lint run` -> 0 issues;
  `shellcheck scripts/*.sh setup-linux.sh` -> clean.
- Manual smoke test: deployed into a scratch `$HOME` and read back all seven rendered records.
  Every one carries `Skill` in `tools:`; `reviewer` is `Read, Glob, Grep, Bash, Skill` (no
  `Edit`/`Write`, as its record intends).
- No regressions in existing test suite: yes, with one disclosed exception —
  `tests/install-dotf.bats` 628–633 fail locally because `install_dotf` deliberately refuses to
  overwrite a `dev` source build and the tests read the ambient `PATH` `dotf`. Fixture leakage,
  not a regression; filed as **#1429**. Green in CI, which has no `dev` build on PATH.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **`unsupported` is a declaration, not an omission.** The loader requires every harness to
  cover the whole vocabulary and must never fall back to a permissive default (C15). Adding a
  verb therefore left only bad options — invent a native name for a harness that lacks the
  concept, or drop the verb from the vocabulary and lose it everywhere. The third answer is to
  let a harness *answer* "no equivalent", which is skipped when resolving, reported on stderr,
  and never rendered as a grant.
- **The contradiction check runs before the coverage arithmetic**, deliberately: a verb in both
  blocks would otherwise surface as a coverage error, making the map's meaning depend on which
  check ran first. The new fixture carries both defects at once to pin that ordering.
- **`UnsupportedFor` is a separate exported function** rather than a second return value from
  `ResolveCapabilities`, keeping I/O and reporting out of a pure resolution function.
- **The omission is reported on stderr, never stdout.** The caller substitutes stdout directly
  into a frontmatter line, so a note there would corrupt the file it is warning about.
- **Map version 2 breaks any older `dotf`** — it fails closed with `NO agent deployed`. That is
  correct under C15, but the message blamed the (correct) map, so `compile-harness.sh` now
  probes `resolve-capabilities` to tell "stale binary" from "bad map" and names its own fix.
- **AC4 was reclassified as deferred rather than met.** A scratch deploy proves a config file
  contains a key, which is precisely what AC4 says is not proof.

## What the record caught on its first day

Two dispatches of the SAME persona, differing only in whether the caller supplied a name:

| dispatch | payload `agent_type` | `role_resolved` | outcome |
|---|---|---|---|
| `subagent_type: reviewer`, no name | `reviewer` | `reviewer` | `warn`, 4 skills named |
| `subagent_type: reviewer`, `name: gate-probe-agent-type` | `gate-probe-agent-type` | *(none)* | **`role-unresolved`** |

`agent_type` carries the **caller-supplied name** when one is given, not the persona type. So
naming a dispatch makes the role unresolvable and **turns its gate off** — allowing, exiting 0,
and indistinguishable from health by every other means. Under `enforce: block` that is a bypass
with a one-word opt-out. Filed as **#1434**, and it is a hard precondition on promoting any
skill to `block`.

This is the strongest available argument for the record itself: the defect was found on the day
it shipped, by looking at it, and was invisible to the exit code, the stderr and `dotf doctor`
alike. It also generalises lesson 255 — **both** identity fields are caller-influenced, not just
`agent_id`.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? **yes** — a guard can be individually correct at
      every layer and still miss the requirement, when the requirement is a *relationship*
      between two keys that no single check spans.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — `unsupported` is an
      extension of ADR-027's capability chain, not a new position.
- [x] New pattern candidate for `00_meta/patterns/`? **candidate** — "an escape hatch needs a
      guard, or it becomes the permissive default it was added to avoid". Only if it recurs in
      a second project; do not promote from one instance.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-106-skill-capability/` -> `specs/archive/HARNESS-106-skill-capability/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)

> **Not archivable yet.** AC4–AC7 are open, so #1420 stays open and this PR references it
> without a closing keyword. The archive gate additionally requires an independent adversarial
> review by a model that is not the implementer's.
