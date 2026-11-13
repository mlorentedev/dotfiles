---
id: CI-019-lint-fleet-rule-selector
type: spec
status: active
created: "2026-11-12"
owner: manu
issue: "dotfiles#3386"
---
# CI-019: lint fleet rule selector

Lane: Fast-Track Development (one new flag on an existing command, one more file kind for one rule).

## Intent

The ci-workflows `hygiene` composite (CI-005) is meant to replace the fleet's vendored `scripts/check-actions-pinned.sh` with dotf. Two gaps stop it:

- `dotf lint fleet` cannot run one rule. On pollex it exits 1 with 20 findings, none of them a pin, so a composite step calling it would go red on every adopter for reasons that are not pinning. Grepping the report in YAML would put rule selection in ci-workflows, which its ADR-001 §1 forbids.
- `fleet/sha-pin` reads only workflows. The vendored script also reads local composite action manifests, which can call third-party actions themselves. Deleting the script would drop that check.

## Scope

- `--rule <id>`, repeatable: report only findings of the named rules. Exit 1 means a finding of a selected rule; findings of other rules neither print nor count. An unknown id is a usage error (exit 2) that lists the known ids. Works with `--dir` and `--fleet`, and the `--summary` table holds only the selected findings.
- Selection happens after exceptions are applied, so an exception for an unselected rule still silences its finding and is still checked for staleness; a stale one shows only when `fleet/exception` is selected.
- `fleet/sha-pin` also reads `runs.steps[].uses` in `.github/actions/<name>/action.yml` and `action.yaml` (and `.gitea/actions/<name>/`, as workflows are read on both forges). Exceptions name such a file by its path, as for a workflow.

Out of scope:

- The other step rules (`persist-credentials`, `skip-cache`) on action manifests. They would add findings across the fleet in a change meant to add none; a follow-up ticket if wanted.
- Nested directories under `actions/<name>/`, and Docker actions' `image:`.

## Checklist

- [x] Failing tests first: a `--rule` selection, an unknown id, `--rule` with `--fleet`, and an action manifest pinned by tag (bad and good fixtures).
- [x] `fleetlint`: read action manifests, apply the pin check to their steps; export the known rule ids.
- [x] `lint fleet`: the `--rule` flag, its validation and the filter; help text.
- [x] Runbook `docs/runbooks/guide-fleet-lint.md`: the flag and the manifests.
- [x] `go test ./...`, `go vet`, `golangci-lint` (also `GOOS=windows`); mutation checks on a copy.
- [x] On pollex: `dotf lint fleet --rule fleet/sha-pin` exits 0 where the full run exits 1.

## Evidence

- `go test ./...` passes; `go vet` and `golangci-lint run` report nothing on darwin and with `GOOS=windows`.
- New tests: `TestActionManifestsArePinChecked`, `TestActionManifestFindingMayBeExcepted`, `TestUnreadableActionManifestIsAnError`, `TestRulesListsEveryFindingRule`, `TestLintFleetRuleSelector`, `TestLintFleetRuleSelectorOverCheckouts`, `TestLintFleetUnknownRule`.
- Ten mutations on a copy, each caught: no action directories read; `.gitea/actions` dropped; `action.yaml` dropped; a manifest parse error ignored; the manifest pin check off; no `--rule` filter; the filter inverted; no `--rule` validation; the selection not passed to the lint; `fleet/exception` missing from `Rules()`.
- Real checkouts, finding counts before (main) and after: dotfiles 26/26, ci-workflows 0/0, pollex 20/20, kubelab 57/57, hive 22/22. With `--rule fleet/sha-pin` each exits 0, pollex included, where the full run exits 1.
- A copy of ci-workflows with `actions/setup-node` in `.github/actions/setup-node/action.yml` moved to `@v9`: `--rule fleet/sha-pin` exits 1 with `.github/actions/setup-node/action.yml:58: fleet/sha-pin: actions/setup-node@v9 is not pinned by a full commit SHA`.

## Next

After a release carries this, ci-workflows CI-005 AC3: the hygiene composite's pinned-actions input.

## Promotion candidates

- [x] Lesson? `no`: nothing failed along the way that a lesson would prevent
- [x] ADR? `no`: a flag and one more file kind under an existing rule
- [x] Pattern? `no`: no cross-project method
