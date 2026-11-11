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

Live run over the fleet (2026-11-10, `go run ./cmd/dotf secrets ci-check`, 2m26s, REST only): exit 1, `16 of 25 repositories disagree with the registry or could not be read`. This repository is public and most of the fleet is not, so the full output (secret names of private repositories) stays out of the record; per repository it gave:

| Result | Repositories |
|---|---|
| ok | 9 (boilerplates, cloud-architecture-coursework, embedded-systems-coursework, emva1288-sensor-characterization, kasa-provisioner, knowledge, mlorentedev, telecom-engineering, vicon-fpga-face-detection) |
| excess `HIVE_WORKER_API_KEY` only | 6 (ci-workflows, dotfiles, garsync, iris, mail2markdown, svqtriana.github.io) |
| missing names, 1 to 7 per repository | 10 (hive, kubelab, kubelab-devnode-fixture, leaving-denver, nan-video-pipeline, pdf-modifier, pollex, ts-bridge, web, yt-metrics-cli) |
| unknown reads | 0 |
| read errors | 0 |

`HIVE_WORKER_API_KEY` was excess in 14 of the 16 (#2306). In `mlorentedev/web`, `missing PR_AGENT_ANTHROPIC_API_KEY` is the #3188 case: its pr-agent caller maps the key explicitly and the repository has no such secret (`gh secret list`, names only). A spot check of `mlorentedev/kubelab` found 4 of its 7 missing names set by hand outside the registry, and the other 3 absent from the repository too.

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
