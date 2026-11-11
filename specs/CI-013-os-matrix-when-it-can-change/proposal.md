---
id: "CI-013-os-matrix-when-it-can-change"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-11"
issue: "mlorentedev/dotfiles#3149"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# CI-013-os-matrix-when-it-can-change

## Why

The Windows and macOS legs are this repository's largest CI cost: `ci/test-windows`
(p50 5.1 min), `cli/test (windows)` (4.3 min) and `ci/test-macos` (3.1 min) together
cost about 22k runner-minutes a month (issue #3149, part of
mlorentedev/ci-workflows#8, ADR-001 practice 4: pull requests run what they touch).
On a pull request that changes only prose, `test-windows` still starts a Windows
runner and checks out the tree before every step skips, and `cli/test` and
`cli-smoke` run their full Windows leg, though nothing they test can differ.

## What

On a pull request whose diff is documentation only, the Windows replicas of `cli/test`
and `cli-smoke` do not start, and every required check still reports green. Any other
pull request, and every push, runs the full matrix as today.

- **One definition of "documentation only"**: `.github/filters/docs-only.yml`, a
  dorny/paths-filter file whose single filter `code` is `**` minus the docs-only paths,
  read with `predicate-quantifier: every`. `tests/ci-path-filtering.bats` reads its
  docs-only set from that file instead of from an inline dict, so the test that proves
  "every tracked top-level entry is code or declared docs-only" and the filter the
  workflows apply are one list. ci.yml keeps its own `code` allow-list: the bats test
  proves the two are complements.
- **`ci/test-windows` is unchanged.** It is a required check, and CI-004's rule
  (`tests/lib/check-workflow-contexts.py`) forbids a job-level `if` on one: a skipped
  required check passes without running. Its steps already skip on a
  documentation-only diff (HARNESS-041), so what remains is the runner start and the
  checkout. The path to skipping the job is the released `gate` composite
  (mlorentedev/ci-workflows `.github/actions/gate`, #9; ADR-001 practice 3), in the
  two steps its README gives: a PR adds a `gate` job (`if: always()`, needing every
  blocking job, test-windows included) beside the current required checks; once it
  has reported on main, the owner makes `gate` the only required CI check, keeping
  review-attestation. test-windows can then carry a job-level `if`, since a skipped
  need passes the gate and the required check always runs. Follow-up, not this spec.
  The new structural test exempts required checks for this reason.
- **`cli/test`**: the OS list becomes an output of `cli-changes`, `ubuntu-latest` on a
  documentation-only pull request and both OSes otherwise. Every push and tag gets both
  without consulting the filter. The Linux leg still runs every Go test on every diff,
  which is what #1888 requires.
- **`cli-smoke`**: the same shape, from a new `cli-smoke-changes` job. The workflow
  stays unfiltered at the trigger, because a path-filtered workflow never reports
  (CI-004, #1739).
- **`ci/integration`**: the container is built with `docker/build-push-action` and a
  `type=gha` layer cache, so the layers before the repository `COPY` (apt, Node, Go,
  age, bats: about 53 s measured on main run 42964129736) come from cache. Every run
  reads the cache; only the push to main writes it, because the layers after the
  `COPY` change on every commit and a write per pull-request run would only evict
  other caches.

`ci/test-macos` and the `from-zero` matrix are already gated at job level; they are
unchanged.

## Out of scope

- **Removing `skip-cache: true` from the golangci-lint steps** (the issue's third
  bullet). #2834 added it: a cache saved on main made every pull request report four
  SA5011 findings that a clean run does not. The job takes about 15 s uncached, below
  the noise of the jobs this spec is about. Declined, not deferred.
- **The nightly schedule** the issue names. It belongs to the gate restructure
  (mlorentedev/ci-workflows#8); today push to main is the full run.
- **A narrower OS-sensitive set** than "not documentation only" (for example, leaving
  `cli/**` changes off the Windows runtime tests because the `GOOS=windows` lint runs
  on Linux). That is a policy decision for the owner, listed below.

## Risks / open questions

- **Owner question, not blocking:** should the Windows runtime legs skip on more than
  documentation? This spec draws the line where ci.yml already draws it.
- **AC3 as written cannot be met.** The warm-build target of under a minute assumed the
  build is mostly cacheable. Measured on main run 42964129736: the cacheable prefix is
  about 53 s; after the `COPY` of the tree come `go build` (22 s), `setup-linux.sh`
  (100 s) and the image export (26 s). Those are the subject under test and change on
  every commit. AC3 is restated below as what the cache can deliver.
- **The saving is modest, and the issue's figures do not come from documentation-only
  pull requests.** 15 of the last 150 merged pull requests were documentation only,
  and none of the 40 most recent green pull-request runs was (its shortest
  `test-windows` took 212 s, so its steps ran). This spec removes the Windows legs of
  `cli/test` (about 95 s on such a diff, per cli.yml) and `cli-smoke` on that tenth
  of pull requests. The large numbers in #3149 are code pull requests, which only the
  narrower set below would reach.
- `cli/test` gains `needs: cli-changes`. A failing `cli-changes` now skips the tests, and
  `cli-gate` goes red on the failure, so this fails closed. The OS list is a string
  every branch of the expression sets, so it is never empty.
- `predicate-quantifier` is step-wide in paths-filter, so the docs-only filter runs in
  its own step, never merged into cli.yml's `go` filter.
- AC1 cannot be shown on this pull request, which touches `.github/**` and so is not
  documentation only. The bats tests pin the logic; the first documentation-only pull
  request after merge is the live proof, recorded in `verification.md`.

## Acceptance criteria

- [ ] AC1: On a documentation-only pull request, no Windows or macOS job starts except
  the required `test-windows`, whose steps skip, and every required check reports
  success. Pinned by bats over the parsed workflows; proven live on the first such
  pull request after merge.
- [ ] AC2: A push to `main` (and a tag push, for cli) runs the full OS matrix regardless
  of the filter. Pinned by bats; proven by the push run after merge.
- [ ] AC3 (restated): A warm integration build takes every layer before the repository
  `COPY` from the gha cache (`CACHED` in the build log). Proven on the first
  pull-request run after the merge's push to main has written the cache, with the
  build step's time recorded against main run 42964129736.
- [ ] AC4: The docs-only set lives in one file, read by the workflows and by
  `tests/ci-path-filtering.bats`.

## References

- Issue #3149; mlorentedev/ci-workflows#8 (ADR-001 practice 4)
- #1888 (Go tests unfiltered), CI-004 / #1739 (filtered workflows never report),
  #2834 (golangci cache), HARNESS-041 (ci.yml path filtering)
