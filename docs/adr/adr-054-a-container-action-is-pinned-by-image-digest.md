---
id: "ADR-054-a-container-action-is-pinned-by-image-digest"
type: adr
status: accepted
owner: manu
date: "2026-10-20"
supersedes: []
extends: [adr-042-shared-ci-is-referenced-not-copied]
issue: mlorentedev/dotfiles#2830
tags: [architecture, decision, ci, supply-chain, pr-agent]
created: "2026-10-20"
---

# ADR-054: A container action is pinned by image digest

## Context

ADR-042 §3 pins third-party actions by commit SHA, because a tag is mutable and belongs to upstream. PR-Agent was pinned that way (`The-PR-Agent/pr-agent@8e5a9295…  # v0.47.0`). Its `action.yaml` is `runs: using: docker` with `image: Dockerfile.github_action_dockerhub`, and that Dockerfile is a single line, `FROM pragent/pr-agent:github_action`. That is a rolling Docker Hub tag. Measured 2026-10-20: on 2026-10-16 it moved to the v0.48.0 build (`sha256:80edef71…`), while our commit pin still named v0.47.0. So the commit pin fixed only the wrapper. The code that holds `NAN_API_KEY`, the Anthropic key and `pull-requests: write` was whatever upstream last pushed to the tag. The audited contract (`harness/pr-agent-upstream-contract.json`) also described v0.47.0 sources. Those were not the sources that ran.

Upstream's `SECURITY.md` recommends `uses: docker://pragent/pr-agent@sha256:<digest>`, and its release workflow (`publish.yml`) signs a build-provenance attestation for every image.

## Decision

1. **A container action whose image is not built from the pinned commit is pinned by image digest:** `uses: docker://<image>@sha256:<digest>  # vX.Y.Z`. A commit pin of its wrapper is not a pin. The rest of ADR-042 §3 still holds for JavaScript and composite actions.
2. **The digest is the only pin.** For PR-Agent, the contract records it as `image_digest`, and a hermetic test fails when the workflow's digest differs.
3. **The commit behind the digest is derived, never typed in.** `audited_ref` is the `sourceRepositoryDigest` that `gh attestation verify --repo The-PR-Agent/pr-agent --signer-workflow …/publish.yml` reports for the digest, on a `refs/tags/v*` ref. The network leg of `tests/pr-agent-config.bats` re-runs that check, then compares each audited blob at `audited_ref`.
4. **Everything that needs an upstream ref reads `audited_ref` from the contract** at `github.workflow_sha`. The empty-diff filter does this when it fetches upstream's extension and lockfile lists. Nothing else parses the `uses:` line.

## Consequences

- An upstream push to the tag no longer changes what runs. A bump changes the digest, so the binding test makes it a contract re-audit.
- `with:` inputs of the wrapper (`artifact_path`, `artifact_instructions`) are no longer reachable. Every setting already reached PR-Agent through `env:`, which `docker://` passes the same way.
- A bump PR touches `.github/**`, so the owner lands it. That was already true for the commit pin.
- The trust root moves from "GitHub serves this commit" to "Sigstore and GitHub attest this digest was built from this commit by upstream's release workflow". Without that attestation, `audited_ref` could not be derived, and the network leg would fail closed.
