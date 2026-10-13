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

Nothing failed. The repository kept its id, its issues and its history, and GitHub redirects
the old name, so pushes, fetches and comments kept reaching it. The rename was visible only as
a name. In a different session, `gh issue comment 2356` printed a
`github.com/mlorentedev/pdf-modifier/issues/2356` URL. That looked like the wrong repository,
so the session posted the evidence a second time. Both comments were on dotfiles #2356, under
the repository's new name. Meanwhile the repository the owner meant was still
`pdf-modifier-mcp`. The name it was meant to take now belonged to dotfiles, and the old name
`dotfiles` was free for anything to claim.

## The Solution
The wrong rename was reversed with an explicit target: `gh repo rename dotfiles --repo
mlorentedev/pdf-modifier`. The requested rename then ran against the right repository:
`gh repo rename pdf-modifier --repo mlorentedev/pdf-modifier-mcp`. The shared remote was
restored with `git remote set-url origin https://github.com/mlorentedev/dotfiles.git`, and the
duplicate comment was deleted. The repository ids confirm which repository holds which name:
dotfiles is 882120129 and pdf-modifier (formerly pdf-modifier-mcp) is 1133951816.

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
