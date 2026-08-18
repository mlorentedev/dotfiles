---
id: lesson-128-a-new-top-level-dir-backing-a-dotf-runtime-read-mu
type: lesson
status: active
created: "2026-06-25"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 128: A new top-level dir backing a dotf runtime read must be deployed by setup

**Context**: #584 added secrets/registry.yaml and made deployed `dotf secrets {ls,show,run}` read it from $DOTFILES_DIR/secrets/registry.yaml. 0.19.0 shipped it.

**Problem**: setup-{linux,windows} only deployed sensitive/ and scripts/ into ~/.dotfiles -- never the new secrets/ dir. So on a 0.19.0 machine, `dotf secrets run` (and the opencode/pi/agy wrappers that call it) failed `read registry: ... cannot find the path`. A post-deploy smoke caught it; otherwise it would have broken the AI-CLI wrappers silently.

**Solution**: B2 (#591) deploys secrets/registry.yaml to $DOTFILES_DIR/secrets/, mirroring sensitive/. Stopgap-copied on the current machine to unbreak it immediately.

**Rule**: When a `dotf` subcommand reads a file from $DOTFILES_DIR at runtime, setup MUST deploy that file/dir -- adding it to the repo is not enough. Always smoke a deployed binary after a release; the smoke is what catches deployed-vs-source drift (redeploy is part of "done").
