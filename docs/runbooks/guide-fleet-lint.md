---
id: "guide-fleet-lint"
type: runbook
status: active
tags: [runbook, ci, lint, fleet, workflows]
created: "2026-11-11"
---

# Runbook: the fleet CI standard (`dotf lint fleet`)

> Every repository's workflows are held to one CI standard, checked by `dotf lint fleet`. Each rule has a stable
> id; a repository that cannot meet one declares an exception with a reason and an issue. Spec:
> [CI-018](../../specs/CI-018-dotf-gate-and-fleet-lint/proposal.md). Issue:
> [#3147](https://github.com/mlorentedev/dotfiles/issues/3147).

## The rules

| Rule | What it asks | Why |
|---|---|---|
| `fleet/timeout` | Every job that runs steps sets `timeout-minutes`. A job that calls a reusable workflow is exempt: it cannot set one. | A hung job holds a runner for six hours, the platform default. |
| `fleet/sha-pin` | Every remote `uses:` is pinned by a full 40-character commit SHA. Local (`./`) and `docker://` references are exempt. | A tag can be moved to other code; a SHA cannot. |
| `fleet/persist-credentials` | Every `actions/checkout` sets `persist-credentials: false`. | Otherwise the job token stays in `.git/config` for every later step to read. |
| `fleet/no-secrets-inherit` | No job passes `secrets: inherit`. | The called workflow gets every secret, not the ones it needs. |
| `fleet/cancel-in-progress` | A workflow that a push to a pull request starts again sets workflow-level `concurrency` with `cancel-in-progress` (an expression counts; `false` does not). A pull-request trigger whose `types:` leave out `synchronize` is exempt: a push does not rerun it. | A new push makes the previous run stale; finishing it only spends runner time. |
| `fleet/os-on-pull-request` | A Windows or macOS job does not run on a pull request unless an `if:` decides it. A literal matrix is read, with `include:`, a nested `matrix.a.b` path, and an `exclude:` on that axis alone; a computed one is conditional. | Those runners cost more and queue longer; change detection should decide. |
| `fleet/skip-cache` | No `*/setup-*` step sets `cache: false`, and golangci-lint-action does not set `skip-cache: true`. A cache left unset is not read. | A switched-off cache pays the download on every run. |
| `fleet/branch-filter` | A `push` or `pull_request` `branches:` filter includes the default branch, read as GitHub reads it: in order, the last pattern that matches decides, `!` excludes, `*` stays within a path segment and `**` crosses a slash, `?` and `+` make the preceding character optional or repeated, `[...]` is a character class and `\` escapes. `branches-ignore` is not read. | A filter that leaves the default branch out means the workflow never checks it. |

Workflows are read under `.github/workflows` and `.gitea/workflows`. The default branch is `--default-branch`,
else what `origin/HEAD` names. A branch filter with neither is an error (exit 2), never a pass.

A clone made by `actions/checkout` has no `origin/HEAD`, so a workflow passes the branch from the event:
`dotf lint fleet --dir . --default-branch "${{ github.event.repository.default_branch }}"`.

## Running it

```bash
dotf lint fleet --dir .                       # one repository
dotf lint fleet --dir . --summary "$GITHUB_STEP_SUMMARY"
dotf lint fleet --fleet ~/Projects            # every git checkout directly under a directory
```

`--fleet` checks each checkout against the branch its own `origin/HEAD` names and prefixes each finding with the
checkout's directory name. A checkout that cannot be read is reported as `[ERROR] <name>: …` and the others are
still checked; the run then exits 2.

| Exit | Meaning |
|---|---|
| 0 | No findings. |
| 1 | Findings, one per line: `path:line: rule: message`. |
| 2 | A workflow or the exceptions file does not parse, the default branch is unknown where a filter needs it, a checkout could not be read, or the flags are wrong. |

## Declaring an exception

One rule on one file, in `.github/lint-exceptions.yml`:

```yaml
exceptions:
  - rule: fleet/skip-cache
    path: .github/workflows/cli.yml
    reason: a lint cache saved on main made every pull request report false SA5011 findings
    issue: "#2834"
```

All four fields are required. The file's own entries are checked, and each of these is a `fleet/exception`
finding that silences nothing:

- an entry missing `rule`, `path`, `reason` or `issue`;
- an entry naming a rule that is not one of the eight above (a typo would otherwise pass as declared);
- an entry that matches no finding: the workflow was fixed or moved, and the exception would silence the
  problem's return. Remove it.

Changing `.github/lint-exceptions.yml` touches `.github/**`, so it lands through the owner, not an agent's merge.

## Adding a rule

1. A constant in `cli/internal/fleetlint`, its id under `fleet/`. The id is a contract: exceptions name it, so a
   rename breaks every repository that declares one.
2. A directory `cli/internal/fleetlint/testdata/<rule>/` with `bad.yml` (exactly one finding of the rule) and
   `good.yml` (the near misses, no finding).
3. The rule in the list exceptions may name. `TestRuleFixtures` walks that list and `TestEveryRuleMayBeExcepted`
   checks it against the fixture directories, so leaving either out fails the tests.
4. A row in the table above and in the command's help.
