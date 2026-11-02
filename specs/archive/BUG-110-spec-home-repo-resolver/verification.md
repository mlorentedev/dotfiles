---
tags: [spec, verification, templates]
created: "2026-11-01"
---

# Verification - BUG-110-spec-home-repo-resolver

## Evidence

- [x] AC1 (GitHub origin unchanged): `TestResolvePrecedence/origin_alone` (spechome), plus the existing `TestSpecAuditExitIssueState`, `TestSpecInit*` and `TestCheckSpecIssueState`, which stay green. The audit fixture now carries a GitHub origin, because the archive is asked of the origin (AC6).
- [x] AC2 (Gitea origin fails closed): `TestResolveFailsClosedOffGitHub` (HTTPS with an embedded token, SSH, absent) and `TestResolveNamesAGiteaCheckout`. At the command level: `TestSpecAuditGiteaOriginFailsClosed` (no forge call is made) and `TestSpecInitGiteaOriginFailsClosed` (nothing is scaffolded). In doctor: `TestCheckSpecIssueStateGiteaOrigin` (WARN, no forge call, no URL leak).
- [x] AC3 (`specs/.bitacora-repo`): `TestResolvePrecedence/file_beats_env`, `TestResolveBitacoraRepoFileMalformed` and `TestResolveBitacoraRepoFileMalformedEvenUnderAFlag`.
- [x] AC4 (precedence, one resolver): `TestResolvePrecedence` and `TestSpecInitAndAuditAgreeOnTheIssueHome`. The latter sets a file and a conflicting env var; init gates on the file's repository, and the audit asks exactly that issue.
- [x] AC5 (Gitea origin with override): `TestSpecAuditGiteaOriginWithOverride`. A recording runner shows no call names the Gitea repository, and the zombie keeps FAIL with "could not be checked".
- [x] AC6 (archive uses the origin): `TestSpecAuditArchiveUsesOrigin`.

## Test status

- `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./...` -> clean
- `go test ./...` -> exit 0, 52 packages ok
- `golangci-lint run` (pinned 2.12.2) -> 0 issues
- `dotf lessons fmt --check` -> `[OK] lessons are in their canonical form`
- No regressions in the existing test suite: yes. One fixture changed on purpose (AC6, above).

## Decisions made during implementation

- **The resolver lives in its own leaf package, `spechome`, not in `spec`.** `spec` cannot import `initrepo`: initrepo imports vault, vault imports mem, and mem imports spec, so the import is a cycle.
- **`initrepo` exports `OriginURL`, `OriginHost` and `ClassifyURL`.** The doctor check reads git through its `System` seam, so the resolver takes the URL as input instead of running git itself.
- **The committed file beats `DOTF_BITACORA_REPO`.** The variable is set per machine, while the file is a repository's own declaration.

## Review dispositions (round 1, PASS-WITH-GAPS, nan/mimo-v2.6-flash)

| Finding | Disposition |
|---------|-------------|
| Major: the empty-home guard in `GHArchiveLookup` is untested (mutation M3 stays green) | **Fixed** in the archive PR. `TestArchiveGHLookupWithoutAGitHubHomeNeverRunsGH` fails if gh runs at all, and `TestSpecAuditGiteaOriginWithOverride` now fails on any `/contents/` call. With the guard deleted, both turned red in this session. |
| Minor: `go test ./...` fails `TestEmbeddedTemplatesMatchVault` on a machine with the vault | **Already ticketed** as #3050 and #461. This is vault template drift that predates this change and that CI cannot see. The diff touches no template. |
| Minor: a malformed `DOTF_BITACORA_REPO` fails `spec audit` even beside a valid github.com origin | **Intended.** A declared value that is wrong fails closed rather than being silently skipped for a lower source, the same rule as the file and the flag. Tested by `TestResolveRejectsAMalformedFlagOrEnv`. Nothing sets the variable today (proposal, Risks). |
| Minor: the audit harness falls through to the real `gh` | **Declined: misread.** `stubSpecAuditGH` never runs gh. For an unknown key it returns its own canned error, whose text is `dial tcp: lookup api.github.com: no such host`, so the "observed" dial error is the stub's string, not a network call. `specAuditRun` is replaced for the whole test. |
| Minor: `spec init` before-any-forge-call is only partly asserted (HTTPS only, no call recording) | **Fixed** in the archive PR. `TestSpecInitGiteaOriginFailsClosed` runs both the HTTPS and SSH forms, with a `gh` stub that leaves a mark if it is ever run. |
| Question: `--force-no-gate` exits 0 on a Gitea origin | **Intended.** That flag skips the work-gate on every forge, and `issue:` stays empty, so no issue home is needed. AC2 is about the gated path. |

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-535-a-name-parsed-from-a-url-is-only-valid-on-the-host-it-came-from.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: `specs/.bitacora-repo` follows ADR-043's `.wip-limit` precedent. The precedence is documented in the resolver and in `spec audit --help`.
- [x] New pattern candidate for `00_meta/patterns/`? no: one occurrence. The lesson carries it.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/BUG-110-spec-home-repo-resolver/` -> `specs/archive/BUG-110-spec-home-repo-resolver/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
