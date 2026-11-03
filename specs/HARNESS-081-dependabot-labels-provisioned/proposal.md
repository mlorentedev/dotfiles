---
id: "HARNESS-081-dependabot-labels-provisioned"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-02"
issue: "mlorentedev/dotfiles#1222"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# HARNESS-081-dependabot-labels-provisioned


## Why

<!-- from issue #1222: HARNESS-081: the 'dependencies' label does not exist, so the spec-gate skip for Dependabot has never fired -->

`.github/dependabot.yml` names a `dependencies` label that nothing creates. Dependabot does not create a custom label, and `check-spec-gate.sh` gates the only spec-gate skip a bot author has on that label, so the skip never fired until someone created the label by hand (measured 2026-11-02: it exists now). A fresh repository in the OPS-002 rollout would have the same gap. The owner chose option B on #1222 (2026-11-02): a `dotf` step that `bitacora-rollout.sh` calls, so the fix travels to every repo and the script does not grow.

## What

- `dotf init github` gains a second GitHub repo default beside `delete_branch_on_merge`. It reads the target repository's own `.github/dependabot.yml` (or `.yaml`) through the contents API, collects every `labels:` entry across `updates`, and creates each one the repository lacks. Labels compare case-insensitively, as GitHub does.
- Idempotent by reading before it writes. A second run creates nothing and reports `labels present`. A label that exists with another color or description is left alone.
- A created label gets color `0366d6` and the description "Dependency updates opened by Dependabot", which matches the label created by hand.
- A repository with no `dependabot.yml`, or one that declares no labels, has nothing to do.
- `--dry-run` lists the labels it would create and creates none.
- A label that cannot be created fails the command with a non-zero exit. That is the issue's "fail loudly".
- `dotf init`, which runs the `github` step, gets the labels default too.
- `scripts/bitacora-rollout.sh` gains one step in its provisioning half: `dotf init github --repo <owner>/<repo>`, with `--dry-run` under `--check`. A failure or a `[WARN]` skip is an `err`, and a would-change is a `chg`. `--backfill-only` does not run it.

## Out of scope

- A scheduled reconciler. Labels are durable repository state, and this provisions them where repositories are provisioned: at `dotf init` and in the rollout. Re-running the rollout heals a deleted label. A schedule would need a token with label write in every repo, and `bitacora-reconcile.yml` is ruled out by its own header.
- The spec-gate skip itself (`check-spec-gate.sh`). It already works now that the label exists; this spec only provisions the label.
- Gitea or Forgejo. Dependabot is GitHub's, so the step lives in `dotf init github` and is GitHub-only by construction.
- The issue's static guard, a test that every label `dependabot.yml` declares is one the repo provisions. The step reads the declaration at run time, so there is no second list that could diverge from it.

## Risks / open questions

- **The rollout now applies `delete_branch_on_merge` to every repo it touches**, because `dotf init github` applies both defaults. The owner approved this on 2026-11-02.
- **The rollout's token needs label write on each repository.** It already writes workflow files there with the same token, so no new scope is expected. If one is needed, the step fails loudly, which is the intended mode.
- **Telling a missing file from an unreadable repository.** A 404 on the contents API means no `dependabot.yml`. Any other failure to read is a `[WARN]` skip, the same as the existing default, and the rollout turns that into an `err`.

## Acceptance criteria

- [x] AC1: Against a repository whose `dependabot.yml` declares labels that are missing, `dotf init github --repo R` creates each missing label once and leaves the present ones alone. A second run creates nothing.
- [x] AC2: `--dry-run` lists the missing labels and creates none.
- [x] AC3: A repository with no `dependabot.yml`, or with no `labels:`, causes no label call beyond the read.
- [x] AC4: A failed `gh label create` makes `dotf init github` exit non-zero, naming the label and the repository.
- [x] AC5: `bitacora-rollout.sh` runs `dotf init github --repo <owner>/<repo>` in a provisioning run, adds `--dry-run` under `--check`, does not run it under `--backfill-only`, and reports a failing or warning step as an error.

## References

- Issue #1222, and the decision comment on it.
- `cli/internal/initrepo/github.go` (`ApplyDeleteBranchOnMerge`).
- `scripts/bitacora-rollout.sh`, and its provisioning/backfill split (OPS-023, #809).
