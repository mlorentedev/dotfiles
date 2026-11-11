---
id: "CI-018-dotf-gate-and-fleet-lint"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-11"
issue: "mlorentedev/dotfiles#3147"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# CI-018-dotf-gate-and-fleet-lint

## Why

mlorentedev/ci-workflows CI-008 (its #12, spec merged at e783aa3) folds the cheap pull-request checks
into one `pr-checks` composite, giving back one runner slot per check per event in every adopter; runner
wait is the account's bottleneck (ci-workflows#14: Ubuntu wait p90 691 s in dotfiles). The composite runs
the checks through dotf, and the spec and knowledge gates exist only as dotfiles shell scripts
(`scripts/check-knowledge-gate.sh`, `check-spec-gate.sh`, `spec-gate-pr.sh`) that read the pull request
through `gh`, so no other repository and no Gitea runner can run them. #3147 also asks for the fleet CI
rules of ADR-001 as `dotf lint`, because a standard nothing checks is violated everywhere (109 of 193 jobs
without a timeout, 22 refs pinned by tag). CI-008 PR 1 waits on this issue.

## What

The command surface CI-008 fixed (ci-workflows `specs/CI-008-pr-checks/proposal.md`):

```
dotf gate knowledge --event <path> --base <ref> --head <ref> [--summary <file>] [--explain]
dotf gate spec      --event <path> --base <ref> --head <ref> [--adjacency <tsv>] [--summary <file>]
dotf lint fleet     --dir <repo> [--summary <file>]
```

- `--event` is a pull-request event file. The gates read body, labels and author from it and make no API
  call; the composite refreshes the file first (BUG-066 is handled there, CI-008 AC3). An event with no
  `pull_request` is no PR context: exit 2, as the scripts refuse an unset body today. A `null` body is
  an empty body, as GitHub sends it.
- Exit 0 passes, 1 fails, 2 is a usage or environment error; failures name the file and line or the
  missing section on stderr. `--summary` appends a Markdown section.
- **Same semantics as the scripts, proven by their own tests.** Each existing bats suite
  (`tests/check-knowledge-gate.bats`, then the spec-gate suites) gains a second run through a shim that
  turns its `SDD_*` environment into an event file and calls the Go command. Both implementations pass
  the same cases until the scripts are deleted.
- **One markdown stripper** (fences, HTML comments, code spans, read left to right as GitHub does) in Go,
  used by both gates: the Go half of REFACTOR-018 (#2407).
- **Fleet rules with stable ids** (`fleet/timeout`, `fleet/sha-pin`, `fleet/persist-credentials`,
  `fleet/no-secrets-inherit`, `fleet/cancel-in-progress`, `fleet/os-on-pull-request`, `fleet/skip-cache`,
  `fleet/branch-filter`), each with a good and a bad fixture. **Exceptions** are declared in the
  repository the rule applies to, in `.github/lint-exceptions.yml`: rule id, path, reason and issue are
  required, and an exception missing any of them is itself a finding. dotf never names a repository.
  dotfiles declares `fleet/skip-cache` for cli.yml with #2834.

## Delivery (one PR each, under the 300-line cap)

1. `internal/gate`: event reader, markdown stripper, `dotf gate knowledge`; knowledge-gate bats parity.
2. `dotf gate spec` core (tier, escape label and rationale, archive rules); spec-gate bats parity.
3. `dotf gate spec --adjacency`, `--summary` for both gates.
4. `dotf lint fleet` reader and the first four rules, exceptions file.
5. The remaining rules and `--fleet` report over the owner's repositories.

Deleting the shell scripts and switching dotfiles' workflows is CI-008's adoption PR, not this spec.

## Out of scope

- The refresh of the event file (CI-008's composite owns the API call).
- The `.gitlab-ci.yml` reader the issue leaves open.
- "Nothing is permanently red, disabled or orphaned" from #3147's rule list: it needs run history, so an
  API call, which `--dir` lint does not make. Filed as a follow-up when PR 5 lands.

## Risks / open questions

- The shell gates have grown many edge cases (CRLF bodies, fences inside comments, unclosed fences,
  placeholder reasons). The parity runs are what make the port safe; a case the bats suites do not cover
  is a case the port can silently change, so any behaviour the port reads differently gets a bats case
  first.
- `dotf gate spec` is about 900 lines of shell; if its parity does not fit PR 2's cap it splits again.
- Release: CI-008's composite installs dotf at a pinned version, so adoption waits for a release that
  carries these commands. Cutting it is the owner's.

## Acceptance criteria

- [ ] AC1: `dotf gate knowledge` passes every case of `tests/check-knowledge-gate.bats` through the event shim.
- [ ] AC2: `dotf gate spec` passes every case of the spec-gate suites through the event shim.
- [ ] AC3: The gates make no network call: they run with no `gh` and no token in the environment.
- [ ] AC4: Each fleet rule has a stable id, a good and a bad fixture, and a declared exception silences only
  its rule and path; an exception without reason or issue is a finding.
- [ ] AC5: `dotf lint fleet --fleet` lists findings per repository; the rules are documented in one runbook
  section linked from ci-workflows ADR-001.

## References

- #3147; mlorentedev/ci-workflows#8, #12, CI-008; ADR-001 (ci-workflows)
- BUG-066 (#877), HARNESS-160 (knowledge gate), REFACTOR-018 (#2407), dotfiles#2834
