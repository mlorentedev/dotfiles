---
tags: [spec, verification, templates]
created: "2026-08-19"
---

# Verification - OPS-026-age-root-file-authority

## Evidence

Produced in the session of 2026-08-19, not recalled.

- AC1 -> `go test ./internal/secrets/ -run 'FileAuthority|AcceptsExactlyValidBackends'` — 8/8
- AC2 -> `dotf secrets ls` — `AGE_KEY_PERSONAL  floor  AGE_KEY_PERSONAL`
- AC3 -> `dotf secrets verify` — **34 ok, 0 missing, 0 failed** (baseline before the
  change: 33 ok, 0 missing, 0 failed)
- AC4 -> mutations observed live, real root never touched (scratch registry, throwaway key):

  ```
  mode 0600            -> 1 ok, 0 missing, 0 failed
  chmod 0644 (stat: 644 confirmed before the run)
                       -> 0 ok, 0 missing, 1 failed   (non-zero exit)
  file removed (absent confirmed)
                       -> 0 ok, 1 missing, 0 failed   (MISSING, not FAILED)
  ```

- AC5 -> **deferred**, see `features.json` f5 and `tasks.md`
- AC6 -> `TestFileAuthority_ResolveRefuses` + `TestResolversCoverEveryValidBackend`
- AC7 -> `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md:146`

## Test status

- `go build ./... && go vet ./... && go test ./...` — 17 packages ok, 0 FAIL
- `golangci-lint run` at the pinned 2.12.2 (matches `versions.conf`) — 0 issues
- No regressions: the 33 pre-existing secrets report exactly as before

## Decisions made during implementation

- **The root is its own backend, not an exemption in the validator.** A named
  carve-out in `checkAgeSources` was the smaller diff and was rejected: the next root
  (a second machine, a rotated identity) would ask for the same special case again.
- **`Resolve` refuses rather than returning the key.** Materializing the root through
  the same facade as the secrets it protects widens the blast radius. The refusal is
  pinned by a test, because it is the kind of thing a later change "fixes" helpfully.
- **Absent is MISSING, wrong mode is FAILED.** A fresh checkout has no key yet and
  must not look broken; a present key readable by others is a real defect.
- **`bw:` without a folder.** The ratified taxonomy has no `floor` folder and its
  comment says floor secrets carry no `bw:` block at all. ADR-028 distinguishes
  authority from convenience copy, and the comment does not. Rather than amend a
  ratified taxonomy unilaterally, the block is unfoldered and the tension is recorded.

## Promotion candidates

- The shape "a check that cannot answer the real question yet must say so where the
  check lives" — `fileAuthorityResolver`'s comment about the deferred drift
  comparison. Same family as lesson 212.
