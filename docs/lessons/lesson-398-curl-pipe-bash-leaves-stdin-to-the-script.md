---
id: "lesson-398-curl-pipe-bash-leaves-stdin-to-the-script"
type: lesson
status: active
title: "curl | bash leaves stdin to the script, not to the person"
created: "2026-10-11"
---

# curl | bash leaves stdin to the script, not to the person

## Context
#2013 D12: a full `dotf converge` ends by walking the identity restore when a person is at a
terminal (`bw login`, `gh auth login`, the vault clone), and prints the steps when nobody is.
The usual entry to a new machine is `curl -fsSL …/install.sh | bash`, which hands the machine to
`dotf converge`.

## The Trap
The obvious test, "stdin is a terminal", answers no on exactly the run the walk exists for. Under
`curl … | bash`, bash reads the script from stdin, so stdin is the pipe, and every child
inherits it. The person is at the keyboard and stdout is the terminal, but a guide that asks
stdin concludes nobody is there and silently prints the plan instead. And a child such as
`bw login` that reads its prompts from that inherited stdin would read the rest of the script,
or EOF, as the answer.

## The Solution
The walk counts as interactive when stdout is a terminal, and then reads the person from stdin
only if stdin is a terminal too, else from the controlling terminal, `/dev/tty`. The same file is
handed to every command it runs, so `bw` and `gh` prompt the person, not the pipe. Windows has
no `/dev/tty`, so there only a terminal stdin counts. CI has neither a terminal stdout nor a
controlling terminal, so it gets the plan and never a prompt.

## Takeaways
- "Is anyone there?" is a question about stdout and the controlling terminal, not about stdin.
  `curl | bash`, `ssh host cmd < file` and a here-doc all take stdin and leave the person.
- A child process inherits the parent's stdin. A prompt-driven tool started from a piped script
  needs its stdin set explicitly, or it reads the script.
- Test the decision through a seam (`identityTerminal` in `cli/internal/cmd/identity.go`): a
  real pipe stands in for the terminal, and no test needs a TTY.
