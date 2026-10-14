---
id: "lesson-415-an-ssh-command-reads-no-profile"
type: lesson
status: active
title: "An ssh command reads no profile: a health check run that way fails for the environment, not the machine"
created: "2026-10-14"
---

# An ssh command reads no profile: a health check run that way fails for the environment, not the machine

## Context
On 2026-10-13 msi's firewall acceptance ran `ssh msi 'dotf doctor'` from the Mac. Doctor
reported five FAILs: `APPS_HOME`, `JAVA_HOME`, `MAVEN_HOME`, `GO_HOME` and `MINIKUBE_HOME`
"is not set". The same binary under `zsh -lic` reported none.

## The Trap
`ssh host cmd` runs `cmd` through a non-login, non-interactive shell. zsh then reads only
`.zshenv`, bash reads nothing, and every variable the profile exports is absent. A check that
reads the environment cannot tell "this machine lacks the variable" from "this process never
read the file that sets it", so a remote run reports drift that a person at the machine would
never see. The quoting makes it worse: in `ssh host "$SHELL" -lic ...` the local shell
expands `$SHELL`, so the remote side gets the Mac's shell path.

## The Solution
Read the evidence that the profile ran before reading what it exports. The generated paths
file always exports `DOTFILES_DIR`; when it and every tool home are unset, doctor says once
that no profile reached the process and names both remedies (#2429). To run a check remotely
with the profile, quote for the remote shell: `ssh -t host 'exec $SHELL -lic "dotf doctor"'`.
Any check that reads the environment should ask first whether the environment was loaded.
