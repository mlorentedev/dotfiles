---
id: "lesson-461-a-post-condition-that-reads-upstream-turns-upstream-lag-into-a-failure"
type: lesson
status: active
title: "A post-condition that reads upstream turns upstream lag into a failure"
created: "2026-10-16"
owner: manu
tags: [converge, mise, tools, cli]
---

# A post-condition that reads upstream turns upstream lag into a failure

**Context:** The converge `tools` step keeps the CLIs marked `# mise: latest` at their newest release. It runs `mise upgrade` on what `mise outdated` lists, then re-plans. Before this change, a tool still listed was a hard failure, in the apply and again in the step's probe (#2526).

**Problem:** A fresh macOS runner failed `from-zero` with `still behind their newest release: claude`. `mise upgrade claude` had exited 0, a re-run passed, and claude's newest release was about 24 hours old. Since 2026.9, mise skips any release younger than `minimum_release_age`, which defaults to 24h as supply-chain protection. Measured on the Mac on 2026-10-16: `mise upgrade --dry-run claude` warned `newer claude release 2.1.302 … ignored by minimum_release_age (24h)` and would install 2.1.301. Near that cutoff, the read that lists a release and the read that installs one can disagree. The machine was correct; the check asked upstream a question upstream could not yet answer consistently.

**Solution:** For a `latest` tool, the apply's post-condition is now "the upgrade ran and left nothing older than it found". A tool still behind is reported as `Behind` ("still behind their newest release, retried next run"). The next converge upgrades it again. The probe re-plans with `CheckUpgrades` off, as doctor already did, so it checks the machine and never the network. Pinned tools keep the strict check. The 24h delay stays: it is mise's own guard against a compromised release, and "latest" now means the latest release mise considers safe.

**Why:** A reconciler's post-condition must be something the machine can make true. "Nothing upstream is newer" depends on timing and on upstream's view of itself, so it turns lag into red builds, and it did so on a PR that touched none of this. Check what the apply controls (it ran, nothing regressed), report what it does not, and let a level-triggered loop retry. When a "transient" failure sits at a round number such as 24h, look for a policy before you blame a cache.
