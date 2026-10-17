Dotfiles fix lane: bugs that other projects discover, without blocking the other lanes. Rule: one bug = one fix/<slug> branch = one worktree = one PR. Never pile bugs onto a branch.

Your bug: #{{.Number}} {{.Title}}. This worktree already exists, on branch {{.Branch}}, created from origin/main.{{if .Remaining}} Behind it in the queue:{{range .Remaining}} #{{.}}{{end}}.{{end}} If another project reports a bug, it enters the queue (.dotf/lane.json) by priority.

Per bug: reproduce, write the failing test, fix, mutate, `go build/vet/test`, `GOOS=windows go vet`, golangci-lint (the pin). Assign yourself the issue when you start. PR with the skip-sdd label, a "## SDD skip rationale" and a "## Knowledge" section (a new lesson in docs/lessons, normalised with `dotf lessons fmt`; take the next free number by looking at origin/main and the open PRs). Draft, ready, triage, then `dotf pr triage-queue --pr <N> && dotf pr land <N>`: the scoped gate answers for your PR alone, so another lane's untriaged PR does not block yours.

First push of a branch with no PR: `SDD_LABELS=skip-sdd SDD_PR_BODY=... git push` (the spec-gate only reads the PR once it exists).

PR queue between sessions: land one at a time and only the head of the queue updates its branch (dotf pr land refuses BEHIND). Check it with `gh pr list --state open`. Merge permission for your own PRs: coordinated with the other sessions, CI green, no conflicts and reviewer comments attended. Do not touch other lanes' branches or PRs.

When you land, do NOT delete your worktree (it closes your pane and your session). Run /handoff, run `dotf lane next` to start the successor, and stop.

Read AGENTS.md before your first tool call. Reply to the user in Spanish; every durable artifact stays in English.
