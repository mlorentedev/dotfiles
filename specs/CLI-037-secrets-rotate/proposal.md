---
id: "CLI-037-secrets-rotate"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-15"
issue: "mlorentedev/dotfiles#996"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-037: `dotf secrets rotate`, the C7 primitive

## Why

Rotating one credential was five manual steps (issue #996, walked live on 2026-08-15 while rotating a leaked DockerHub PAT). Two of them failed silently. Without a `bw serve` sync, the write succeeded and every read kept serving the old value. A liveness probe returning 200 could not tell a rotated token from an old one that was never revoked. The registry declares `rotate: 90d` on most entries, and `secrets_set.go` names rotate (C7, ADR-029) as a primitive it composes, but no command did it.

## What

`dotf secrets rotate <id> [var]` replaces a registry secret's value and proves the replacement took:

1. fingerprint the current value (sha256, first 12 hex), never printing it;
2. read the new value from stdin or a hidden prompt;
3. refuse a no-op (same value) and refuse to create (rotate replaces; `set` provisions);
4. write through the same path as `set`;
5. sync the backend the write went to, so the read path stops serving a stale cache;
6. re-read through the normal read path and require a changed fingerprint;
7. run the entry's `validate:` liveness probe when it declares one;
8. with `--push-ci`, upload the new value to every `ci:<repo>` consumer's GitHub Actions secrets, through the same path and skip rules as `secrets sync ci`.

`--dry-run` reports the current fingerprint, the probe and the repos a real run would push to, and writes nothing.

## Out of scope

- **Reading the registry's `rotate: 90d` policy** (reminding or refusing when a secret is past its age). #996 names it as the motivation, not as a criterion; it needs a last-rotated timestamp the vault does not record today.
- **Probes other than `github-token`.** A declared `validate:` with no implementation is reported as "liveness unverified", never as a pass.

## Risks / open questions

- None open. The no-`BW_SESSION` dependency (#993) closed with #1007, which moved the write path onto `bw serve`.

## Acceptance criteria

From #996, plus its `--push-ci` flag:

- [x] **AC1.** `rotate` writes, syncs the backend, and re-resolves through the read path in one command.
- [x] **AC2.** Output reports before/after fingerprints and never the value; a rotation whose read-back fingerprint did not change fails loudly.
- [x] **AC3.** A new value identical to the current one is refused as a no-op, and nothing is written.
- [x] **AC4.** An entry declaring `validate:` is probed with the new value after the write, and a failing probe fails the rotation.
- [x] **AC5.** `--dry-run` reports the intended action, the current fingerprint, the probe and the CI repos that would be pushed, writing nothing.
- [x] **AC6.** Works with no ambient `BW_SESSION` through `bw serve` (#993, fixed by #1007).
- [x] **AC7.** `--push-ci` uploads the rotated value to every `ci:<repo>` consumer and to nothing else; without the flag nothing is uploaded; a secret with no CI consumer says so and still succeeds.

## References

- Issue: #996. Related: #993 / #1007 (write path on `bw serve`), ADR-029 (C7), `cli/internal/cmd/secrets_set.go`.
