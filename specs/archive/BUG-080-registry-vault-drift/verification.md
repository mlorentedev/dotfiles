---
tags: [spec, verification]
created: "2026-08-15"
---

# Verification - BUG-080-registry-vault-drift

## Evidence

| AC | Proof |
|---|---|
| **AC1** | `secrets/registry.yaml` `DOCKERHUB_TOKEN` → `field: PAT`. Live probe, both credentials, 2026-08-15: `PAT -> HTTP 200`, `pass -> HTTP 200` against `hub.docker.com/v2/repositories/<user>/`. The comment in the registry records this so the change is not reverted on the assumption the old field was broken. |
| **AC2** | `TestCheckBWMapping_MissingItemFails` — asserts the item name, the naming secret ids, and the `without --only` consequence all appear. |
| **AC3** | `checkBWMapping` calls only `sys.BWItemNames()`; `BWServeReader.ItemNames` hits `/list/object/items` and returns `it.Name` only. No field access, no `Field(...)` call, no value in any code path. |
| **AC4** | `TestCheckBWMapping_UnavailableVaultSkips` (error → SKIP, 0 failures) and `TestCheckBWMapping_NoBwSecretsSkips` (asserts the vault is **not** touched, and that the output is a SKIP rather than a PASS). |
| **AC5** | Live, below. |

## Test status

```
go build ./...   OK
go vet ./...     OK
go test ./...    OK (all packages)
```

## AC5 — observed against real state, not a fixture

`dotf doctor`, built from this branch, against the live vault:

```
[Bitwarden mapping (registry -> vault)]
  [FAIL] dockerhub: no such item in the vault, named by DOCKERHUB_TOKEN, DOCKERHUB_USERNAME — every `dotf secrets run` without --only fails on it, including `dotf spec review`
```

Independently corroborated by the failure this check exists to explain:

```
$ dotf secrets run -- true
Error: bw resolve dockerhub/password: bw item not found: bw serve item "dockerhub": not found

$ dotf secrets run --only NAN_API_KEY -- true ; echo $?
0
```

Scoped resolution works; unscoped dies. That is the whole blast radius in two commands.

## Decisions made during implementation

- **The vault rename is not in this PR.** `DockerHub` → `dockerhub` is what clears the current instance, and it was authorised — but the CR overlay's precondition could not be met: `dotf secrets backup` fails with `Vault is locked` because `bw export` uses the CLI session while the daemon holds a separate unlocked one. Mutating a live password vault with no working escrow is not a trade worth making for a rename that takes an operator 30 seconds in the UI. The tooling gap is its own finding.
- **A bare `"bw"` literal in the new check**, deliberately: #984 introduces `secrets.BackendBW` and will absorb it with the others. This PR must be mergeable alone, because it is what unblocks that one.
- **SKIP, not PASS, when there is nothing to compare.** "Nothing to check" and "checked, all good" are different statements and only one is evidence — the same distinction the PAT-expiry check got wrong for months (#972).

## Promotion candidates

- [ ] Lesson for `docs/lessons.md` — deferred to the deeper fix. The transferable rule ("a fail-fast set-resolution turns any single mapping error into a total outage of everything that resolves the set") belongs with the decision on `dotf secrets run`'s tolerance, not with the detector.
- [ ] ADR — no. ADR-028 already governs; this conforms.
- [ ] Vault pattern — no. Single-repo so far.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/BUG-080-registry-vault-drift/`
- [ ] #985 closed with the PR link and the operator action (item rename) confirmed done
- [ ] `/adversarial-review` run and `review.md` signed by a pool model — possible only once this very fix has landed, since it is what makes the reviewer launchable
