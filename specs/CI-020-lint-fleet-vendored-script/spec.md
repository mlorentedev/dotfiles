---
id: CI-020-lint-fleet-vendored-script
type: spec
status: active
created: "2026-11-16"
owner: manu
issue: "dotfiles#3537"
---
# CI-020: lint fleet reports a vendored hygiene script

Lane: Fast-Track Development (one rule on an existing command).

## Intent

The ci-workflows `hygiene` composite (CI-005, ci-workflows#31) replaces two scripts each repository vendors: `scripts/check-lessons.sh` by `dotf lessons fmt --check`, and `scripts/check-actions-pinned.sh` by `dotf lint fleet --rule fleet/sha-pin`. Its last criterion is that `dotf lint` reports a repository that still carries one, so a copy cannot linger once the composite runs. No fleet rule looks at files outside the workflows.

## Scope

- `fleet/vendored-script`: one finding per vendored script present at the repository root's `scripts/`, on line 1, naming the command that replaces it. A symlink counts. A stat failure other than absence is an error, as everywhere in `fleetlint`.
- It is a fleet rule like the others: `.github/lint-exceptions.yml` silences it per file, a stale exception is a `fleet/exception` finding, and `--rule fleet/vendored-script` selects it alone.

Out of scope:

- `scripts/pin-actions.sh`. It rewrites pins rather than checking them, and nothing in dotf replaces it yet.
- Copies under other names or paths. The fleet's copies all sit at these two paths (ci-workflows#31's count).

## Checklist

- [x] Failing tests first: each script yields its finding with the replacement named; none without them; a symlink counts; an unreadable `scripts/` is an error; an exception silences it and goes stale once the file is gone; `--rule` selects it.
- [x] `fleetlint`: the rule, registered with the others so exceptions and `--rule` know it.
- [x] Runbook `docs/runbooks/guide-fleet-lint.md`: the rule's row.
- [x] This repository carries `scripts/check-actions-pinned.sh`: adopted. `repo-hygiene.yml` runs `dotf lint fleet --rule fleet/sha-pin` from the checkout, as it already runs `lessons fmt`, and the script and its bats guard are deleted. The guard's case (#1855, a single workflow file calling a reusable workflow) is covered by `fleet/sha-pin`: pinned exits 0, `@v1` exits 1 naming line 4.
- [x] `go test ./...`, `go vet`, `golangci-lint` (also `GOOS=windows`); mutation checks on a copy; finding counts on real checkouts.

## Evidence

- `go test ./...` and `go vet ./...` pass; `golangci-lint run ./internal/fleetlint/...` reports 0 issues on darwin and with `GOOS=windows`.
- New tests: `TestVendoredHygieneScriptsAreFindings`, `TestVendoredScriptMayBeExceptedAndGoesStale`, `TestVendoredScriptFailsClosed` (a symlink, a dangling one, a mode-000 `scripts/`), `TestVendoredScriptSelectedAlone`. `TestRuleFixtures` and `TestEveryRuleMayBeExcepted` skip the tree rules, which have no workflow fixtures.
- Mutations on a copy, each caught:

```
rule not run: caught
lessons script dropped: caught
pin script dropped: caught
pin-actions added: caught
stat error swallowed: caught
symlink not followed as present: caught   (missed at first; the dangling-symlink case was added)
not exceptable: caught
message without command: caught
```

- Each repository's default-branch tree (`git archive origin/HEAD`), findings with dotf from main against this branch, and what `--rule fleet/vendored-script` names:

```
pdf-modifier: main 20; branch 21; vendored: scripts/check-actions-pinned.sh
pollex: main 20; branch 21; vendored: scripts/check-actions-pinned.sh
yt-metrics-cli: main 10; branch 11; vendored: scripts/check-actions-pinned.sh
mail2markdown: main 6; branch 7; vendored: scripts/check-actions-pinned.sh
iris: main 22; branch 23; vendored: scripts/check-actions-pinned.sh
hive: main 20; branch 21; vendored: scripts/check-actions-pinned.sh
garsync: main 4; branch 5; vendored: scripts/check-actions-pinned.sh
ts-bridge: main 0; branch 1; vendored: scripts/check-actions-pinned.sh
web: main 0; branch 1; vendored: scripts/check-actions-pinned.sh
svqtriana.github.io: main 10; branch 11; vendored: scripts/check-actions-pinned.sh
kubelab: main 60; branch 62; vendored: scripts/check-actions-pinned.sh scripts/check-lessons.sh
ci-workflows: main 0; branch 0; vendored: none
```

- On this branch's tree, `--rule fleet/sha-pin` and `--rule fleet/vendored-script` both print `[OK] no fleet lint findings`.

## Next

ci-workflows CI-005 AC6 once a dotf release carries this; the adopters lose their finding as their AC5 PRs delete `check-actions-pinned.sh`.

## Promotion candidates

- [x] Lesson? `no`: one thing surprised, local checkouts behind their remote showed copies already deleted; evidence reads the remote tree instead, which the spec records
- [x] ADR? `no`: a rule under an existing command
- [x] Pattern? `no`: no cross-project method
