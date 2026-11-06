Lane: one issue, one branch, one worktree, one PR. Never pile issues onto a branch.

Your issue: #{{.Number}} {{.Title}}. This worktree already exists, on branch {{.Branch}}, created from the base declared in .dotf/lane.json.{{if .Remaining}} Behind it in the queue:{{range .Remaining}} #{{.}}{{end}}.{{end}} The queue is the committed .dotf/lane.json, in priority order.

Per issue: reproduce, write the failing test, fix, and run the repository's verify target. Assign yourself the issue when you start. Open the PR as a draft, mark it ready when it is finished, record the review triage on it, then `dotf pr triage-queue --pr <N> && dotf pr land <N>`: the scoped gate answers for your PR alone, so another lane's untriaged PR does not block yours.

Land one PR at a time. Do not touch other lanes' branches or PRs.

When you land, do NOT delete your worktree (it closes your pane and your session). Hand off, run `dotf lane next` to start the successor, and stop.

Read AGENTS.md before your first tool call.
