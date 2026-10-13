---
id: "lesson-403-a-gh-command-without-repo-acts-on-the-checkout-it-runs-in"
type: lesson
status: active
title: "A gh command without --repo acts on the checkout it runs in, and a rename rewrites that checkout's remote"
created: "2026-10-12"
---

# A gh command without --repo acts on the checkout it runs in, and a rename rewrites that checkout's remote

## Context
On 2026-10-12 the owner asked a session to rename the `pdf-modifier-mcp` repository to
`pdf-modifier`. The session's working directory was the shared dotfiles checkout, where it was
also preparing the consumer change (#2360). It ran `gh repo rename pdf-modifier --yes` without
`--repo`.

## The Trap
`gh repo rename` renames "the current repository" by default, and the current repository is
the one the working directory's git remote points at. So the command renamed
`mlorentedev/dotfiles` to `mlorentedev/pdf-modifier`. It then rewrote `origin` in the shared
`.git/config` to the new URL, so every worktree of the checkout followed the rename.

Nothing failed. GitHub redirects the old name, so pushes and fetches kept working. The first
visible symptom came later, in a different session: `gh issue comment 2356` posted to
`pdf-modifier#2356`, and the acceptance evidence of #2356 landed on the wrong issue. Every
unqualified `gh` command in every session on that checkout now resolved to a repository with
the wrong name. Those commands succeeded, so nothing reported the error.

## The Solution
Both renames were reversed with explicit targets: `gh repo rename dotfiles --repo
mlorentedev/pdf-modifier`, then `gh repo rename pdf-modifier --repo
mlorentedev/pdf-modifier-mcp`. The shared remote was restored with `git remote set-url origin
https://github.com/mlorentedev/dotfiles.git`, and the duplicate comment was deleted. The
repository ids confirm which repository holds which name: dotfiles is 882120129 and
pdf-modifier is 1133951816.

## Takeaways
- An agent passes `--repo OWNER/REPO` to every `gh` command that writes: rename, comment,
  edit, merge, close. The working directory is shared by worktrees and by parallel sessions.
  It says where the shell is, not which repository the request is about.
- A rename is the worst case of this. It changes the target, and it also rewrites the
  remote, so every later unqualified command inherits the mistake.
- After a cross-repository operation, check where the remote points with
  `git remote get-url origin`, and compare it with the repository id, not its name
  (`gh api repos/OWNER/REPO --jq .id`). A redirect makes the old name keep working, so the
  name alone is not proof.
