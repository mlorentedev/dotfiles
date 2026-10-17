---
tags: [spec, tasks, templates]
created: "2026-10-16"
---

# Tasks - TOOL-030-forge-write-surface

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main for each PR below (the spec PR itself is `docs/tool-030-forge-write-surface`)
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] The open question about the classifier is answered by PR 1, not assumed

## Implementation

> Four PRs, sequenced. PR 1 measures before anything widens authority.

**PR 0: `tea` for the human, on every OS**
- [x] [P] Verify the package id per manager (brew, apt, winget) by installing it, not by recalling it; record any manager that does not package it and what `dotf tools` does there (a stated skip, never a silent one). Measured 2026-10-17: brew `tea` is Gitea's CLI (installed, 0.16.0); winget `Gitea.tea` is in winget-pkgs (0.16.0), with the install unmeasured until the Windows session; apt's `tea` is a graphical text editor (msi), so Linux names no package and `dotf tools install` prints `tea is not installed on linux by this catalog; skipping`.
- [x] Add `tea` to `packages.json` and a catalog test; `dotf tools sync` twice reports `changed=0`. The catalog installs system entries with `dotf tools install` (`tools sync` is mise's): the first run installed through brew, the second printed `tea already installed; skipping`. Test: `TestTheRepoCatalogDeclaresGiteaTeaWithoutApt` (mutation: adding `apt: tea` fails it).
- [ ] Log in from the secrets store through `dotf secrets run`, token never printed (shares the registry entry PR 1 adds for the kubelab Gitea)

**PR 1: the interface and the measurement (no allow rules)**
- [x] [AC1] [AC3] Write failing tests for forge detection from origin and for body-from-file only
- [x] [AC1] [AC3] Add `cli/internal/forgewrite` with a `Backend` interface (not `Forge`: `initrepo.Forge` is a type), detection reusing `initrepo.ClassifyOrigin`, and the GitHub backend over the `gh` REST endpoint
- [x] [AC4] Add the Gitea REST backend. The token is read through the internal `cli/internal/secrets` resolver, never by running `dotf secrets show`. Test with a fake token that a 4xx, a 5xx and a malformed response leave it out of stdout, stderr and every `error.Error()`
- [x] [AC1] Register `dotf forge issue comment` next to `protection` in `cli/internal/cmd/forge.go`. One verb only; `issue create` moves to PR 2 because it drags labels and the NNN convention
- [x] Write the ADR for the contract (ADR-052) (one write surface, bodies from files, no merge or close, Gitea through dotf rather than `tea`)
- [x] [AC6] Measure the classifier (protocol in the proposal) and record it on #2635. Recorded as `classifier-result: allowed dotf forge issue comment` with its caveats (dev binary, one verb); repeated with the released binary before PR 4

**PR 2: the remaining verbs** (split three ways at build time, each under the size guideline: 2a landed as #2655, 2b is `pr create` and `pr edit`, 2c is `--board`)
- [x] [P] [AC1] [AC2] `issue create`, `issue edit` and `pr comment`, each with a test. `--label` takes names (Gitea maps them to IDs and refuses an unknown one); the caller computes any AREA-NNN title, `dotf forge` does not scan for the next number. `pr comment` is the `issue comment` endpoint on both forges, so it adds no backend method
- [x] [P] [AC1] [AC2] `pr create` and `pr edit`, each with a test. `pr edit` is `issue edit` under the pr name (same endpoint on both forges); `pr create` takes `--base` as required and `--head` defaulting to the current branch
- [x] [AC2] Test that no `merge` or `close` verb is registered, walking the whole tree
- [x] `--board` for GitHub Project fields (PR 2c), on `issue create` only. GitHub only, and not part of `Backend`: an optional `BoardWriter` the GitHub backend satisfies, so a Gitea origin gets a stated refusal. The project is declared per host under `boards` in `forge/hosts.json`, never guessed. Field and option ids are resolved from the live board at run time, as the `new-ticket` skill does, never pasted
- [x] A `--board` run that cannot proceed (Gitea origin, undeclared board, unknown option, missing `--type`) exits with ERROR/WHY/FIX and writes nothing, not even the issue; resolution happens before the create. A failed card placement names the created issue and says not to create it again
- [x] f1 and f2 in `features.json` named tests that never existed (they were written before the code); both now run the real `cmd` tests and exit 0, as do f3 and f4. A verification that names a missing test passes `-run` vacuously, so each entry must be re-run, not just read
- [ ] Every verb PR records that its verbs are unmeasured against the classifier; the released-binary measurement stays before PR 4

**PR 3: the guard**
- [x] [AC5] Extend ADR-048's session-level allow lists (`harness/capability-map.json` and its leaves) with `dotf forge` and let that guard assert it, one case per harness, instead of a second file; if ADR-048's renderer is not there yet, a `tests/harness-permits-dotf.bats` stands in and is folded into it later. Passes under bash 3.2 and zsh Delivered as the stand-in: `harness/capability-map.json` has no session-level section yet (ADR-048 decision 3 is unbuilt), so `tests/harness-permits-dotf.bats` ships now (#2651) and is folded into the map guard when that lands.

**PR 4: skills and the allow set (owner lands)**
- [x] [AC7] `new-ticket`, `pr-review-triage` and `autopilot` call `dotf forge` (vault commit `4b7f3deb`, then `compile-harness.sh --refresh`). Only the writes that carried prose moved: claim and park comments, the triage record, the PR body and the issue body. `gh issue edit --add-assignee` stays, because `dotf forge issue edit` sets title and body only
- [x] [AC7] The four `dotf forge` allows are in the Claude template in this PR, which the owner lands (#2178, ADR-048 clause 6). `tests/forge-skills-use-dotf-forge.bats` asserts the skills and the allows together, and that no merge or close verb is allowed
- [ ] [AC7] Owner decision, not in this PR: #2629 (`Bash(gh issue create:*)`) stays in the template for now. It covers the tickets that still go through `gh` (a Gitea issue needs `dotf forge`; a GitHub one works with either). Removing it later is a template edit, and ADR-048's deploy can only add, so a machine that holds it keeps it until the owner removes it by hand
- [ ] Verb coverage: only `issue comment` is measured against the classifier (released 0.73.0, comment 2709). `issue create`, `pr create`, `pr comment` and `pr edit` are allowed in the template on the strength of the rule shape, not of a measurement; the first unattended use of each is the measurement and goes on #2635

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/TOOL-030-forge-write-surface/features.json`):

```json
[
  {
    "id": "TOOL-030-forge-write-surface-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
