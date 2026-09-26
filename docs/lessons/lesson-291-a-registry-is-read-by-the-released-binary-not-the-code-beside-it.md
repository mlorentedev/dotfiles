---
id: lesson-291
type: lesson
status: active
created: "2026-09-24"
owner: manu
tags: [lesson, secrets, release, compatibility, silent-failure]
---

# 291 — A registry is read by the released binary, not by the code beside it

## What happened

On 2026-09-24 the personal plane got a Bitwarden folder, `Dotfiles/personal`. The change was built as one commit: the Go validator accepted the new folder, and `secrets/registry.yaml` declared it on six entries. The unit tests, the full suite, lint and the live `reconcile --apply` (run with `go run` from the worktree) all passed.

Then the installed binary read the same registry:

```
$ dotf version
dotf version 0.57.0
$ dotf secrets ls
Error: secret "ZOHO_APP_PASSWORDS": bw.folder "Dotfiles/personal" is not in the ratified taxonomy (Dotfiles/apps, Dotfiles/infra)
```

Once merged, every `dotf secrets` command run by the installed release against `main` would have failed, including `hive.service`, which starts through `dotf secrets run`. CI would have been green, because CI builds `dotf` from the same commit.

## Why it happens

The registry is data in the repository, and its consumer is whatever `dotf` is installed on the machine: a release, pinned in `versions.conf`, updated only when someone installs one. The code that validates the data and the data itself ship on different schedules. Every check in the change's own loop runs the new code, so it can only ever confirm that new code accepts new data.

## The rule

1. **A registry value that needs new code lands in two changes (expand, then contract).** First the reader accepts it and ships in a release that is installed. Then the registry declares it. Here that is #1673, then a release, then #1674 (held as a draft until then).
2. **Before declaring a change safe, run the installed binary against the new data.** `dotf secrets ls` in the worktree needs no vault and fails loudly on a registry the release cannot parse.
3. **The mechanical version is #1675:** a CI step that installs the pinned release and parses the PR's registry with it.

Refs: #586, #1673, #1674, #1675.
