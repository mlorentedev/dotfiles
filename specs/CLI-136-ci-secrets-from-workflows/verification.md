---
tags: [spec, verification, templates]
created: "2026-11-10"
id: "CLI-136-ci-secrets-from-workflows"
type: spec-verification
status: active
owner: manu
---

# Verification - CLI-136-ci-secrets-from-workflows

## Evidence

- [x] AC1 (per-repository missing, excess and unknown names, no values, non-zero exit) -> test `TestRunCICheckReportsNamesAndFails`, and the live fleet run below (exit 1, 16 of 25 repositories)
- [x] AC2 (inherit caller without `PR_AGENT_ANTHROPIC_API_KEY`; a mapping reports the name it maps from) -> tests `TestCheckRepo`, `TestInheritedCalls`, `TestWorkflowReads` (`THEIR_NAME: ${{ secrets.MINE }}` yields `MINE`). Live: `mlorentedev/web` maps the key explicitly and the repository has no such secret
- [x] AC3 (declared but unread is excess) -> test `TestCheckRepo` (`OLD_TOKEN`)
- [x] AC4 (whole-context read is unknown with file and expression, never a pass) -> tests `TestWorkflowReadsUnknown`, `TestCheckRepo`, `TestCheckRepoCleanAndUnknownOnly`
- [x] AC5 (absent `required: false` callee secret is missing, marked optional, naming the callee) -> tests `TestCalleeSecrets`, `TestCheckRepo`
- An unreadable callee fails loudly and names it -> test `TestCheckRepoCalleeUnreadable`

Mutation check (2026-11-10). Each change was applied on its own and the new tests run against it:
- the `GITHUB_TOKEN` exclusion removed -> killed
- the optional-missing branch disabled -> killed
- the callee cache bypassed -> killed
- the `inputs.secrets` guard removed -> survived at first; `TestWorkflowReads` gained an `inputs.secrets` row, then killed

Live run over the fleet (2026-11-10, `go run ./cmd/dotf secrets ci-check`, 2m26s, REST only):

```
mlorentedev/boilerplates: ok
mlorentedev/ci-workflows: FAIL
  excess   HIVE_WORKER_API_KEY
mlorentedev/cloud-architecture-coursework: ok
mlorentedev/dotfiles: FAIL
  excess   HIVE_WORKER_API_KEY
mlorentedev/embedded-systems-coursework: ok
mlorentedev/emva1288-sensor-characterization: ok
mlorentedev/garsync: FAIL
  excess   HIVE_WORKER_API_KEY
mlorentedev/hive: FAIL
  missing  RELEASE_PLEASE_TOKEN
  excess   HIVE_WORKER_API_KEY
mlorentedev/iris: FAIL
  excess   HIVE_WORKER_API_KEY
mlorentedev/kasa-provisioner: ok
mlorentedev/knowledge: ok
mlorentedev/kubelab: FAIL
  missing  BRANCH_PROTECTION_TOKEN
  missing  DOCKERHUB_TOKEN
  missing  DOCKERHUB_USERNAME
  missing  N8N_DEPLOY_TOKEN
  missing  N8N_WEBHOOK_URL
  missing  NOTIFY_WEBHOOK_SECRET
  missing  RELEASE_PLEASE_TOKEN
  excess   HIVE_WORKER_API_KEY
mlorentedev/kubelab-devnode-fixture: FAIL
  missing  BITACORA_PAT
mlorentedev/leaving-denver: FAIL
  missing  BITACORA_PAT
  missing  CLOUDFLARE_API_TOKEN
  missing  SELLER_PHONE
  missing  SELLER_SEALED
  excess   HIVE_WORKER_API_KEY
mlorentedev/mail2markdown: FAIL
  excess   HIVE_WORKER_API_KEY
mlorentedev/mlorentedev: ok
mlorentedev/nan-video-pipeline: FAIL
  missing  BITACORA_PAT
mlorentedev/pdf-modifier: FAIL
  missing  DOCKERHUB_TOKEN
  missing  DOCKERHUB_USERNAME
  excess   HIVE_WORKER_API_KEY
mlorentedev/pollex: FAIL
  missing  RELEASE_TOKEN
  excess   HIVE_WORKER_API_KEY
mlorentedev/svqtriana.github.io: FAIL
  excess   HIVE_WORKER_API_KEY
mlorentedev/telecom-engineering: ok
mlorentedev/ts-bridge: FAIL
  missing  RELEASE_PLEASE_PAT
  excess   HIVE_WORKER_API_KEY
mlorentedev/vicon-fpga-face-detection: ok
mlorentedev/web: FAIL
  missing  DOCKERHUB_TOKEN
  missing  DOCKERHUB_USERNAME
  missing  KUBELAB_DISPATCH_TOKEN
  missing  PR_AGENT_ANTHROPIC_API_KEY
  missing  RELEASE_PLEASE_PAT
  excess   HIVE_WORKER_API_KEY
mlorentedev/yt-metrics-cli: FAIL
  missing  RELEASE_TOKEN
  excess   HIVE_WORKER_API_KEY
Error: 16 of 25 repositories disagree with the registry or could not be read
```

Spot check against the repositories' own secret names (`gh secret list`, names only): `kubelab` holds `DOCKERHUB_TOKEN`, `DOCKERHUB_USERNAME`, `N8N_DEPLOY_TOKEN`, `RELEASE_PLEASE_TOKEN` set outside the registry, and has no `BRANCH_PROTECTION_TOKEN`, `N8N_WEBHOOK_URL` or `NOTIFY_WEBHOOK_SECRET`. `web` has no `PR_AGENT_ANTHROPIC_API_KEY`.

## Test status

- `cd cli && go test ./...` -> every package ok, no failures
- `go vet ./...` and `GOOS=windows go vet ./...` -> clean; `gofmt -l` -> nothing
- `golangci-lint run` on `internal/secrets` and `internal/cmd`, host and `GOOS=windows` -> `0 issues.`
- No regressions in the existing suite: yes

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **`HIVE_WORKER_API_KEY` is excess in 14 repositories, and that is correct.** The registry's `NAN_API_KEY` entry exposes two vars (one credential, two consumer names), and `SelectCI` uploads both to every `ci:` consumer, while no workflow reads the hive alias. The check reports it rather than special-casing it; narrowing the upload is #2306 (CLI-103), already open.
- **"Missing" means not declared in the registry**, not absent from the repository: most kubelab names exist as hand-set secrets. Whether a declared secret was synced stays #1596.
- Only `${{ }}` expressions are scanned, so a comment or a YAML key named `secrets:` is never read as a secret reference.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-136-ci-secrets-from-workflows/` -> `specs/archive/CLI-136-ci-secrets-from-workflows/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
